package genpls

import (
	"context"
	"fmt"
	"go/ast"
	"iter"
	"maps"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/WinPooh32/genpls/gen"
	"github.com/WinPooh32/genpls/internal/xslices"
	"github.com/WinPooh32/genpls/opt"
	"golang.org/x/sync/errgroup"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/packages"
)

const pkgLoadMode = packages.NeedModule |
	packages.NeedName |
	packages.NeedFiles |
	packages.NeedSyntax |
	packages.NeedTypes |
	packages.NeedTypesInfo

type pkgID string

// Generator loads go files and runs generators on them.
type Generator struct {
	pkgs map[pkgID]*packages.Package
}

// NewGenerator returns a new initialized [Generator] instance.
func NewGenerator() (*Generator, error) {
	return &Generator{
		pkgs: make(map[pkgID]*packages.Package),
	}, nil
}

// Load loads Go packages  by the given patterns to the [Generator] instance.
//
// Dir parameter is the directory in which to run the build system's query
// tool that provides information about the packages.
// If Dir is empty, the tool is run in the current directory.
func (g *Generator) Load(ctx context.Context, dir string, patterns ...string) (*Generator, error) {
	cfg := &packages.Config{
		Mode:    pkgLoadMode,
		Context: ctx,
		Dir:     dir,
		Tests:   true,
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	if len(g.pkgs) == 0 {
		g.pkgs = make(map[pkgID]*packages.Package, len(g.pkgs))
	}

	var errs pkgerrs

	for _, pkg := range pkgs {
		if pkg.Errors != nil || pkg.TypeErrors != nil {
			errs = append(errs, pkg)
			continue
		}

		g.pkgs[pkgID(pkg.ID)] = pkg
	}

	if errs != nil {
		return g, &errs
	}

	return g, nil
}

// Generate runs generator functions on Go's packages loaded AST.
// Returns the stream of generated contents.
// The jobs parameter specifies number of used goroutines for processing, if set as 0 number of cpu cores will be used.
func (g *Generator) Generate(
	ctx context.Context,
	jobs int,
	gens map[gen.GeneratorName]gen.Func,
) <-chan opt.Result[gen.File] {
	if jobs <= 0 {
		jobs = runtime.NumCPU()
	}

	resC := make(chan opt.Result[gen.File], jobs*len(gens))

	go func() {
		defer close(resC)

		eg, ctx := errgroup.WithContext(ctx)
		pkgs := slices.Collect(maps.Keys(g.pkgs))

		for part := range xslices.Split(pkgs, jobs) {
			eg.Go(func() error {
				wrkr := genWorker{
					pkgs:   g.pkgs,
					gens:   gens,
					pkgIDs: part,
					resC:   resC,
				}

				return wrkr.run(ctx)
			})
		}

		if err := eg.Wait(); err != nil {
			resC <- opt.Err[gen.File](err)
			return
		}
	}()

	return resC
}

type genWorker struct {
	pkgs   map[pkgID]*packages.Package
	gens   map[gen.GeneratorName]gen.Func
	pkgIDs []pkgID
	resC   chan<- opt.Result[gen.File]
}

func (gw *genWorker) run(ctx context.Context) error {
	for _, id := range gw.pkgIDs {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context is done: %w", err)
		}

		pkg, ok := gw.pkgs[id]
		if !ok {
			return fmt.Errorf("the package is not found by ID %s", id)
		}

		if err := gw.execGenerators(ctx, gw.scan(pkg)); err != nil {
			return fmt.Errorf("exec generators: %w", err)
		}
	}

	return nil
}

func (gw *genWorker) scan(pkg *packages.Package) map[string][]gen.Please {
	typs := map[string]*gen.TypeSpec{}

	var syntax []*ast.File

	if isTestPackage(pkg) {
		syntax = selectTestfiles(pkg, pkg.Syntax)
	} else {
		syntax = pkg.Syntax
	}

	in := inspector.New(syntax)

	imports := gw.imports(in)

	for ts := range gw.typeSpecs(pkg, in) {
		typs[ts.Spec.Name.Name] = &ts
	}

	if len(typs) == 0 {
		return nil
	}

	for fs := range gw.funcSpecs(in) {
		if fs.Decl.Recv == nil {
			continue
		}

		name := inspectRecvName(fs.Decl.Recv)

		if ts, ok := typs[name]; ok {
			ts.Methods = append(ts.Methods, fs)
		}
	}

	cmds := map[string][]gen.Please{}

	for _, ts := range typs {
		ts.AddCMD(cmds, imports, commands(ts, gw.gens))
	}

	return cmds
}

func (gw *genWorker) execGenerators(ctx context.Context, cmds map[string][]gen.Please) error {
	if cmds == nil {
		return nil
	}

	for name, gen := range gw.gens {
		pls, ok := cmds[string(name)]
		if !ok {
			continue
		}

		files, err := gen(ctx, name, pls)
		if err != nil {
			return fmt.Errorf("run generator %s: %w", name, err)
		}

		for _, file := range files {
			if err := gw.sendFile(ctx, file); err != nil {
				return err
			}
		}
	}

	return nil
}

func (gw *genWorker) sendFile(ctx context.Context, file gen.File) error {
	select {
	case gw.resC <- opt.Ok(file):
	case <-ctx.Done():
		return fmt.Errorf("condext is done: %w", ctx.Err())
	}

	return nil
}

var funcImportSpecFilter = []ast.Node{
	new(ast.ImportSpec),
}

// imports returns the map with a key as a package path and value as an alias of the package name.
func (gw *genWorker) imports(in *inspector.Inspector) map[gen.PkgPath]gen.PkgName {
	m := map[gen.PkgPath]gen.PkgName{}

	in.Nodes(funcImportSpecFilter, func(n ast.Node, _ bool) (proceed bool) {
		spec, ok := n.(*ast.ImportSpec)
		if !ok {
			return true
		}

		if spec.Name == nil {
			return true
		}

		switch spec.Name.Name {
		case "_", ".":
			return true
		}

		pkgPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			panic("failed to unquote package import path " + spec.Path.Value)
		}

		m[gen.PkgPath(pkgPath)] = gen.PkgName(spec.Name.Name)

		return true
	})

	return m
}

var typeSpecsFilter = []ast.Node{
	new(ast.GenDecl),
	new(ast.TypeSpec),
}

func (gw *genWorker) typeSpecs(pkg *packages.Package, in *inspector.Inspector) iter.Seq[gen.TypeSpec] {
	return func(yield func(gen.TypeSpec) bool) {
		in.WithStack(typeSpecsFilter, func(n ast.Node, _ bool, stack []ast.Node) (proceed bool) {
			spec, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}

			var doc *ast.CommentGroup

			if spec.Doc != nil {
				doc = spec.Doc
			} else if decl, ok := stack[len(stack)-2].(*ast.GenDecl); ok {
				doc = decl.Doc
			}

			if doc == nil {
				return false
			}

			ts := gen.TypeSpec{
				Pkg:     pkg,
				Doc:     doc,
				Spec:    spec,
				Methods: nil,
			}

			if !yield(ts) {
				return false
			}

			return false
		})
	}
}

var funcSpecsFilter = []ast.Node{
	new(ast.FuncDecl),
}

func (gw *genWorker) funcSpecs(in *inspector.Inspector) iter.Seq[gen.FuncSpec] {
	return func(yield func(gen.FuncSpec) bool) {
		in.WithStack(funcSpecsFilter, func(n ast.Node, _ bool, _ []ast.Node) (proceed bool) {
			funcDecl, ok := n.(*ast.FuncDecl)
			if !ok {
				//  Only top-level func decl are interested.
				return false
			}

			fs := gen.FuncSpec{
				Doc:  funcDecl.Doc,
				Decl: funcDecl,
				Type: funcDecl.Type,
			}

			if !yield(fs) {
				return false
			}

			return false
		})
	}
}

func selectTestfiles(pkg *packages.Package, syntax []*ast.File) []*ast.File {
	var testfiles []*ast.File

	for _, file := range syntax {
		f := pkg.Fset.File(file.Pos())
		if f == nil {
			continue
		}

		name := f.Name()
		if !strings.HasSuffix(strings.ToLower(name), "_test.go") {
			continue
		}

		testfiles = append(testfiles, file)
	}

	return testfiles
}

func isTestPackage(pkg *packages.Package) bool {
	for _, f := range pkg.GoFiles {
		if strings.HasSuffix(f, "_test.go") {
			return true
		}
	}

	return false
}

func inspectRecvName(recv *ast.FieldList) (name string) {
	node := recv.List[0].Type

	ast.Inspect(node, func(ast.Node) bool {
		switch node := node.(type) {
		case *ast.Ident:
			name = node.Name

		case *ast.StarExpr:
			ast.Inspect(node, func(ast.Node) bool {
				ident, ok := node.X.(*ast.Ident)
				if !ok {
					return true
				}

				name = ident.Name

				return false
			})

			return false
		}

		return true
	})

	return name
}

func commands(ts *gen.TypeSpec, gens map[gen.GeneratorName]gen.Func) iter.Seq[gen.Command] {
	return func(yield func(gen.Command) bool) {
		for _, line := range ts.Doc.List {
			textOnly := trimCommentPrefix(line.Text)
			textOnly = strings.TrimSpace(textOnly)

			name, args, _ := strings.Cut(textOnly, " ")
			name = strings.TrimPrefix(name, gen.CmdPrefix)

			genf, ok := gens[gen.GeneratorName(name)]
			if !ok {
				continue
			}

			if !strings.HasPrefix(line.Text, "//"+gen.CmdPrefix) {
				panic(fmt.Sprintf("no spaces are expected after // at comment line %q", line.Text))
			}

			cmd := gen.Command{
				Name: name,
				Args: cleanupArgs(strings.Split(args, " ")),
				Gen:  genf,
			}

			if !yield(cmd) {
				return
			}
		}
	}
}

func isCommentSlashOrSpace(r rune) bool {
	return r == '/' || unicode.IsSpace(r)
}

func trimCommentPrefix(s string) string {
	return strings.TrimLeftFunc(s, isCommentSlashOrSpace)
}

func cleanupArgs(s []string) []string {
	for i, arg := range s {
		s[i] = strings.TrimSpace(arg)
	}

	s = slices.DeleteFunc(s, func(arg string) bool {
		return arg == ""
	})

	return s
}
