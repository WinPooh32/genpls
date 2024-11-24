package stub

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/types"
	"maps"
	"slices"
	"strings"

	"github.com/WinPooh32/genpls/gen"
)

func Generate(ctx context.Context, name gen.GeneratorName, gp []gen.Please) ([]gen.File, error) {
	var files []gen.File

	buf := bytes.NewBuffer(nil)

	for filename, gp := range gen.IterateFiles(gp) {
		buf.Reset()
		buf.WriteString(gp[0].FormatDoNotEditHeader(name))
		buf.WriteString(gp[0].FormatPkg())

		if err := generate(buf, gp); err != nil {
			return nil, fmt.Errorf("generate: %w", err)
		}

		files = append(files, gen.File{
			Name: gp[0].FormatGeneratorFileName(name, strings.HasSuffix(filename, "_test.go")),
			Data: bytes.Clone(buf.Bytes()),
		})

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context is closed: %w", ctx.Err())
		default:
		}
	}

	return files, nil
}

type methInfo struct {
	name string
	sig  string
}

type ifaceInfo struct {
	name      string
	object    types.Object
	methInfos []methInfo
}

func generate(buf *bytes.Buffer, gp []gen.Please) error {
	usedImports := map[gen.PkgPath]gen.PkgName{}
	infos := make([]ifaceInfo, 0, len(gp))

	for _, pls := range gp {
		info, err := analyze(pls, usedImports)
		if err != nil {
			return fmt.Errorf("analyze: %w", err)
		}

		infos = append(infos, info)
	}

	if len(usedImports) > 0 {
		genImports(buf, usedImports)
	}

	for _, inf := range infos {
		genStubIface(buf, inf)
	}

	return nil
}

func analyze(pls gen.Please, usedImports map[gen.PkgPath]gen.PkgName) (ifaceInfo, error) {
	ifacename := pls.TS.Spec.Name.Name
	position := pls.TS.Pkg.Fset.Position(pls.TS.Spec.Pos())

	_, ok := pls.TS.Spec.Type.(*ast.InterfaceType)
	if !ok {
		return ifaceInfo{}, fmt.Errorf("%s: type %q must be an interface", position, ifacename)
	}

	object := pls.TS.Pkg.Types.Scope().Lookup(ifacename)
	if object == nil {
		return ifaceInfo{}, fmt.Errorf("%s: object %s not found", position, ifacename)
	}

	if _, ok := object.(*types.TypeName); !ok {
		return ifaceInfo{}, fmt.Errorf("%v is not a named type", object)
	}

	mset := types.NewMethodSet(object.Type())

	methInfos := make([]methInfo, 0, mset.Len())

	for i := range mset.Len() {
		meth := mset.At(i).Obj()
		sig := types.TypeString(meth.Type(), alias(pls.Imports, usedImports))

		methInfos = append(methInfos, methInfo{
			name: meth.Name(),
			sig:  strings.TrimPrefix(sig, "func"),
		})
	}

	return ifaceInfo{
		name:      ifacename,
		object:    object,
		methInfos: methInfos,
	}, nil
}

func alias(
	pkg *types.Package,
	imports map[gen.PkgPath]gen.PkgName,
	usedImports map[gen.PkgPath]gen.PkgName,
) types.Qualifier {
	return func(p *types.Package) string {
		if pkg == p {
			// local imports are unqualified.
			return ""
		}

		path := gen.PkgPath(p.Path())
		alias := imports[path]

		// Populate imports used by generated types.
		// Include empty alias too.
		if usedImports != nil {
			usedImports[path] = alias
		}

		if alias == "" {
			return p.Name()
		}

		return string(alias)
	}
}

func genImports(buf *bytes.Buffer, usedImports map[gen.PkgPath]gen.PkgName) {
	pkgs := slices.Sorted(maps.Keys(usedImports))

	buf.WriteString("import (\n")

	for _, pkg := range pkgs {
		buf.WriteByte('\t')

		alias := usedImports[pkg]
		if alias != "" {
			buf.WriteString(string(alias))
			buf.WriteByte(' ')
		}

		buf.WriteByte('"')
		buf.WriteString(string(pkg))
		buf.WriteString("\"\n")
	}

	buf.WriteString(")\n\n")
}

func genStubIface(buf *bytes.Buffer, inf ifaceInfo) {
	concrname := "Unimplemented" + inf.name
	rcv := strings.ToLower(string([]rune(concrname)[0]))

	fmt.Fprintf(buf, "var _ %s = (*%s)(nil)\n\n", inf.name, concrname)
	fmt.Fprintf(buf, "// *%s implements %s.\n", concrname, inf.name)
	fmt.Fprintf(buf, "type %s struct{}\n\n", concrname)

	for _, minf := range inf.methInfos {
		fmt.Fprintf(buf,
			"func (%s *%s) %s%s {\n\tpanic(\"method %s not implemented!\")\n}\n\n",
			rcv, concrname, minf.name, minf.sig, minf.name,
		)
	}
}
