package genpls

import (
	"context"
	"fmt"
	"go/ast"
	"iter"
	"maps"
	"runtime"
	"slices"
	"strings"
	"unicode"

	"github.com/WinPooh32/genpls/internal/xslices"
	"github.com/WinPooh32/genpls/opt"
	"golang.org/x/sync/errgroup"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/packages"
)

const cmdPrefix = "genpls:"

const pkgLoadMode = packages.NeedModule |
	packages.NeedName |
	packages.NeedFiles |
	packages.NeedSyntax |
	packages.NeedTypes |
	packages.NeedTypesInfo

type pkgID string

type Please struct {
	Filename string
	Args     []string
	TS       *TypeSpec
}

type GenFunc func(ctx context.Context, name GeneratorName, pls []Please) ([]File, error)

type GeneratorName string

func (gn GeneratorName) Command() string {
	return cmdPrefix + string(gn)
}

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
func (gen *Generator) Load(ctx context.Context, dir string, patterns ...string) (*Generator, error) {
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

	if len(gen.pkgs) == 0 {
		gen.pkgs = make(map[pkgID]*packages.Package, len(gen.pkgs))
	}

	var errs pkgerrs

	for _, pkg := range pkgs {
		if pkg.Errors != nil || pkg.TypeErrors != nil {
			errs = append(errs, pkg)
			continue
		}

		gen.pkgs[pkgID(pkg.ID)] = pkg
	}

	if errs != nil {
		return gen, &errs
	}

	return gen, nil
}

// Generate runs generator functions on Go's packages loaded AST.
// Returns the stream of generated contents.
// The jobs parameter specifies number of used goroutines for processing, if set as 0 number of cpu cores will be used.
func (gen *Generator) Generate(ctx context.Context, jobs int, gens map[GeneratorName]GenFunc) <-chan opt.Result[File] {
	if jobs <= 0 {
		jobs = runtime.NumCPU()
	}

	resC := make(chan opt.Result[File], jobs*len(gens))

	go func() {
		defer close(resC)

		eg, ctx := errgroup.WithContext(ctx)
		pkgs := slices.Collect(maps.Keys(gen.pkgs))

		for part := range xslices.Split(pkgs, jobs) {
			eg.Go(func() error {
				wrkr := genWorker{
					pkgs:   gen.pkgs,
					gens:   gens,
					pkgIDs: part,
					resC:   resC,
				}

				return wrkr.run(ctx)
			})
		}

		if err := eg.Wait(); err != nil {
			resC <- opt.Err[File](err)
			return
		}
	}()

	return resC
}

type genWorker struct {
	pkgs   map[pkgID]*packages.Package
	gens   map[GeneratorName]GenFunc
	pkgIDs []pkgID
	resC   chan<- opt.Result[File]
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

func (gw *genWorker) scan(pkg *packages.Package) map[string][]Please {
	typs := map[string]*TypeSpec{}

	var syntax []*ast.File

	if isTestPackage(pkg) {
		syntax = selectTestfiles(pkg, pkg.Syntax)
	} else {
		syntax = pkg.Syntax
	}

	in := inspector.New(syntax)

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

	cmds := map[string][]Please{}

	for _, ts := range typs {
		ts.addCMD(cmds, ts.commands(gw.gens))
	}

	return cmds
}

func (gw *genWorker) execGenerators(ctx context.Context, cmds map[string][]Please) error {
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

func (gw *genWorker) sendFile(ctx context.Context, file File) error {
	select {
	case gw.resC <- opt.Ok(file):
	case <-ctx.Done():
		return fmt.Errorf("condext is done: %w", ctx.Err())
	}

	return nil
}

var typeSpecsFilter = []ast.Node{
	new(ast.GenDecl),
	new(ast.TypeSpec),
}

func (gw *genWorker) typeSpecs(pkg *packages.Package, in *inspector.Inspector) iter.Seq[TypeSpec] {
	return func(yield func(TypeSpec) bool) {
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

			ts := TypeSpec{
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

func (gw *genWorker) funcSpecs(in *inspector.Inspector) iter.Seq[FuncSpec] {
	return func(yield func(FuncSpec) bool) {
		in.WithStack(funcSpecsFilter, func(n ast.Node, _ bool, _ []ast.Node) (proceed bool) {
			funcDecl, ok := n.(*ast.FuncDecl)
			if !ok {
				//  Only top-level func decl are interested.
				return false
			}

			fs := FuncSpec{
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

type command struct {
	name string
	args []string
	gen  GenFunc
}

type FuncSpec struct {
	Doc  *ast.CommentGroup
	Decl *ast.FuncDecl
	Type *ast.FuncType
}

type TypeSpec struct {
	Pkg     *packages.Package
	Doc     *ast.CommentGroup
	Spec    *ast.TypeSpec
	Methods []FuncSpec
}

func (ts *TypeSpec) addCMD(cmds map[string][]Please, cmdSeq iter.Seq[command]) {
	filename := ts.Pkg.Fset.Position(ts.Spec.Pos()).Filename

	for cmd := range cmdSeq {
		cmds[cmd.name] = append(cmds[cmd.name], Please{
			Filename: filename,
			Args:     cmd.args,
			TS:       ts,
		})
	}
}

func (ts *TypeSpec) commands(gens map[GeneratorName]GenFunc) iter.Seq[command] {
	return func(yield func(command) bool) {
		for _, line := range ts.Doc.List {
			textOnly := trimCommentPrefix(line.Text)
			textOnly = strings.TrimSpace(textOnly)

			name, args, _ := strings.Cut(textOnly, " ")
			name = strings.TrimPrefix(name, cmdPrefix)

			genf, ok := gens[GeneratorName(name)]
			if !ok {
				continue
			}

			cmd := command{
				name: name,
				args: cleanupArgs(strings.Split(args, " ")),
				gen:  genf,
			}

			if !yield(cmd) {
				return
			}
		}
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
