package stub

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/types"
	"io"
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

func generate(w io.Writer, gp []gen.Please) error {
	for _, pls := range gp {
		if err := stubIface(w, pls); err != nil {
			return err
		}
	}

	return nil
}

func stubIface(w io.Writer, pls gen.Please) error {
	ifacename := pls.TS.Spec.Name.Name
	position := pls.TS.Pkg.Fset.Position(pls.TS.Spec.Pos())

	_, ok := pls.TS.Spec.Type.(*ast.InterfaceType)
	if !ok {
		return fmt.Errorf("%s: type %q must be an interface", position, ifacename)
	}

	object := pls.TS.Pkg.Types.Scope().Lookup(ifacename)
	if object == nil {
		return fmt.Errorf("%s: object %s not found", position, ifacename)
	}

	if _, ok := object.(*types.TypeName); !ok {
		return fmt.Errorf("%v is not a named type", object)
	}

	concrname := ifacename + "Stub"
	rcv := strings.ToLower(string([]rune(concrname)[0]))

	fmt.Fprintf(w, "// *%s implements %s.\n", concrname, ifacename)
	fmt.Fprintf(w, "type %s struct{}\n\n", concrname)

	mset := types.NewMethodSet(object.Type())

	for i := range mset.Len() {
		meth := mset.At(i).Obj()
		sig := types.TypeString(meth.Type(), (*types.Package).Name)

		fmt.Fprintf(w, "func (%s *%s) %s%s {\n\tpanic(\"not implemented!\")\n}\n\n",
			rcv, concrname, meth.Name(),
			strings.TrimPrefix(sig, "func"))
	}

	return nil
}
