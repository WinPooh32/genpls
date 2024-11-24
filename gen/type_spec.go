package gen

import (
	"go/ast"
	"iter"

	"golang.org/x/tools/go/packages"
)

type Command struct {
	Name string
	Args []string
	Gen  Func
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

func (ts *TypeSpec) AddCMD(cmds map[string][]Please, imports map[PkgPath]PkgName, cmdSeq iter.Seq[Command]) {
	filename := ts.Pkg.Fset.Position(ts.Spec.Pos()).Filename

	for cmd := range cmdSeq {
		cmds[cmd.Name] = append(cmds[cmd.Name], Please{
			Filename: filename,
			Args:     cmd.Args,
			TS:       ts,
			Imports:  imports,
		})
	}
}
