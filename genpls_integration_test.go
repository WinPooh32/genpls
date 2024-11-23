package genpls_test

import (
	"cmp"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"go/ast"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/WinPooh32/genpls"
	"github.com/WinPooh32/genpls/gen"
	"github.com/WinPooh32/genpls/opt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/cmdinfo.json
var testCmdInfos []byte

func mustLoad(t *testing.T, dir string, patterns ...string) *genpls.Generator {
	t.Helper()

	gen, err := genpls.NewGenerator()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = gen.Load(ctx, dir, patterns...)
	require.NoError(t, err)

	return gen
}

type Info struct {
	Name string `json:"name,omitempty"`
	Doc  string `json:"doc,omitempty"`
}

type CmdInfo struct {
	PkgPath  string   `json:"pkg_path,omitempty"`
	PkgName  string   `json:"pkg_name,omitempty"`
	Filename string   `json:"filename,omitempty"`
	Cmd      string   `json:"cmd,omitempty"`
	Args     []string `json:"args,omitempty"`
	Type     TypeInfo `json:"type,omitempty"`
}

type TypeInfo struct {
	Info
	Fields  []Info `json:"fields,omitempty"`
	Methods []Info `json:"methods,omitempty"`
}

func NewCmdInfo(pls gen.Please, name gen.GeneratorName) CmdInfo {
	cmdi := CmdInfo{
		PkgName:  pls.TS.Pkg.Name,
		PkgPath:  pls.TS.Pkg.PkgPath,
		Filename: pls.Filename,
		Cmd:      string(name),
		Args:     pls.Args,
		Type: TypeInfo{
			Info: Info{
				Name: pls.TS.Spec.Name.Name,
				Doc:  pls.TS.Doc.Text(),
			},
			Fields:  newFieldsInfo(pls.TS),
			Methods: newMethodsInfo(pls.TS),
		},
	}

	return cmdi
}

func newFieldsInfo(ts *gen.TypeSpec) []Info {
	switch node := ts.Spec.Type.(type) {
	case *ast.StructType:
		var infos []Info

		for _, field := range node.Fields.List {
			infos = append(infos, Info{
				Name: field.Names[0].Name,
				Doc:  field.Doc.Text(),
			})
		}

		return infos
	case *ast.InterfaceType:
		return nil
	default:
		panic(fmt.Sprintf("unexpected spec type %T", node))
	}
}

func newMethodsInfo(ts *gen.TypeSpec) []Info {
	var infos []Info

	switch node := ts.Spec.Type.(type) {
	case *ast.StructType:
		for _, method := range ts.Methods {
			infos = append(infos, Info{
				Name: method.Decl.Name.Name,
				Doc:  method.Doc.Text(),
			})
		}
	case *ast.InterfaceType:
		for _, method := range node.Methods.List {
			infos = append(infos, Info{
				Name: method.Names[0].Name,
				Doc:  method.Doc.Text(),
			})
		}
	default:
		panic(fmt.Sprintf("unexpected spec type %T", node))
	}

	return infos
}

func trimDirPrefix(filename string) string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return strings.TrimPrefix(filename, dir)
}

var genmap1 = map[gen.GeneratorName]gen.Func{
	"test": func(ctx context.Context, name gen.GeneratorName, pp []gen.Please) ([]gen.File, error) {
		cmds := []CmdInfo{}

		for i := range pp {
			pls := &pp[i]

			pls.Filename = trimDirPrefix(pls.Filename)

			cmds = append(cmds, NewCmdInfo(*pls, name))
		}

		slices.SortFunc(cmds, func(a CmdInfo, b CmdInfo) int {
			return cmp.Or(
				cmp.Compare(a.PkgPath, b.PkgPath),
				cmp.Compare(a.PkgName, b.PkgName),
				cmp.Compare(a.Filename, b.Filename),
				cmp.Compare(a.Type.Name, b.Type.Name),
				cmp.Compare(a.Cmd, b.Cmd),
				cmp.Compare(fmt.Sprint(a.Args), fmt.Sprint(b.Args)),
			)
		})

		data, err := json.Marshal(&cmds)
		if err != nil {
			return nil, fmt.Errorf("unmarshal json: %w", err)
		}

		return []gen.File{{
			Name: pp[0].FormatGeneratorFileName(name, false),
			Data: data,
		}}, nil
	},
}

func TestGenerator_Generate(t *testing.T) {
	t.Parallel()

	type args struct {
		jobs int
		gens map[gen.GeneratorName]gen.Func
	}

	tests := []struct {
		name string
		gen  *genpls.Generator
		args args
		want []opt.Result[gen.File]
	}{
		{
			name: "simple",
			gen:  mustLoad(t, "internal/_testdata/parsing", "./..."),
			args: args{
				jobs: 1,
				gens: genmap1,
			},
			want: []opt.Result[gen.File]{
				opt.Ok(gen.File{
					Name: "/internal/_testdata/parsing/test_gen.go",
					Data: testCmdInfos,
				}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotResults := []opt.Result[gen.File]{}

			for res := range tt.gen.Generate(context.Background(), tt.args.jobs, tt.args.gens) {
				gotResults = append(gotResults, res)
			}

			f := func(a, b opt.Result[gen.File]) int {
				return cmp.Compare(a.Ok.Name, b.Ok.Name)
			}

			slices.SortFunc(tt.want, f)
			slices.SortFunc(gotResults, f)

			for i := range tt.want {
				wantRes := tt.want[i]
				gotRes := gotResults[i]

				if wantRes.Err != nil {
					assert.Equal(t, wantRes.Err.Error(), gotRes.Err.Error())
				} else {
					assert.Equal(t, wantRes.Ok.Name, gotRes.Ok.Name)
					assert.JSONEq(t, string(wantRes.Ok.Data), string(gotRes.Ok.Data))
				}
			}
		})
	}
}
