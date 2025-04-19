package mock

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/WinPooh32/genpls/gen"
)

func Generate(ctx context.Context, name gen.GeneratorName, gp []gen.Please) ([]gen.File, error) {
	var files []gen.File

	buf := bytes.NewBuffer(nil)

	for _, pls := range gp {
		buf.Reset()
		buf.WriteString(gp[0].FormatDoNotEditHeader(name))
		buf.WriteString(gp[0].FormatPkg())

		cfg, err := parseArgs(pls.Args, config{
			Name: strings.ToLower(pls.TS.Spec.Name.String()) + "_gen.go",
			Pkg:  "mocks",
			Dir:  "mocks",
		})
		if err != nil {
			return nil, fmt.Errorf("pasrse command arguments: %w", err)
		}

		if err := generate(buf, pls); err != nil {
			return nil, fmt.Errorf("generate: %w", err)
		}

		files = append(files, gen.File{
			Name: filepath.Clean(filepath.Join(filepath.Dir(pls.Filename), cfg.Filename())),
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

func generate(buf *bytes.Buffer, pls gen.Please) error {
	usedImports := map[gen.PkgPath]gen.PkgName{}

	info, err := analyze(pls, usedImports)
	if err != nil {
		return fmt.Errorf("analyze AST: %w", err)
	}

	if len(usedImports) > 0 {
		genImports(buf, usedImports)
	}

	if err := genBody(buf, info); err != nil {
		return fmt.Errorf("generate body: %w", err)
	}

	return nil
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

func genBody(buf *bytes.Buffer, inf ifaceInfo) error {
	var concrname string

	const proxy = "Mock"

	startChar := string([]rune(inf.name)[0])

	if upper := strings.ToUpper(startChar); startChar != upper {
		if len(inf.name) > 1 {
			concrname = proxy + upper + inf.name[1:]
		} else {
			concrname = proxy + upper
		}
	} else {
		concrname = proxy + inf.name
	}

	data := struct {
		ConcrName      string
		InterfaceName  string
		TypeParamsDecl string
		TypeParams     string
		Methods        []methInfo
	}{
		ConcrName:      concrname,
		InterfaceName:  inf.name,
		TypeParamsDecl: inf.typeParamsDecl,
		TypeParams:     inf.typeParams,
		Methods:        inf.methInfos,
	}

	if err := tmpl.Execute(buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}
