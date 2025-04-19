package mock

import (
	"fmt"
	"go/ast"
	"go/types"
	"strconv"
	"strings"

	"github.com/WinPooh32/genpls/gen"
)

type methInfo struct {
	Name    string
	Sig     string
	Args    string
	ArgsSig string
	Results string
	Ret     bool
}

type ifaceInfo struct {
	name           string
	object         types.Object
	methInfos      []methInfo
	typeParamsDecl string
	typeParams     string
}

func analyze(pls gen.Please, usedImports map[gen.PkgPath]gen.PkgName) (ifaceInfo, error) {
	origIfacename := pls.TS.Spec.Name.Name
	position := pls.TS.Pkg.Fset.Position(pls.TS.Spec.Pos())

	var spec *ast.TypeSpec

	// Handle aliased interface
	if ident, ok := pls.TS.Spec.Type.(*ast.Ident); ok {
		typeSpec, okTypeSpec := ident.Obj.Decl.(*ast.TypeSpec)
		if !okTypeSpec {
			return ifaceInfo{}, fmt.Errorf("%s: type %q expected to be an interface alias", position, origIfacename)
		}

		spec = typeSpec
	} else {
		spec = pls.TS.Spec
	}

	_, ok := spec.Type.(*ast.InterfaceType)
	if !ok {
		return ifaceInfo{}, fmt.Errorf("%s: type %q must be an interface", position, origIfacename)
	}

	ifacename := spec.Name.Name

	object := pls.TS.Pkg.Types.Scope().Lookup(ifacename)
	if object == nil {
		return ifaceInfo{}, fmt.Errorf("%s: object %s not found", position, ifacename)
	}

	if _, ok := object.(*types.TypeName); !ok {
		return ifaceInfo{}, fmt.Errorf("%v is not a named type", object)
	}

	objtyp := object.Type()

	typ, ok := objtyp.(*types.Named)
	if !ok {
		return ifaceInfo{}, fmt.Errorf("unexpected type %T", objtyp)
	}

	pkgAliasFn := alias(pls.TS.Pkg.Types, pls.Imports, usedImports)

	typeParamsDecl, typeParams := typeParams(typ, pkgAliasFn)

	mset := types.NewMethodSet(objtyp)

	methInfos := make([]methInfo, 0, mset.Len())

	for i := range mset.Len() {
		meth := mset.At(i).Obj()

		sig := types.TypeString(meth.Type(), pkgAliasFn)
		sig = strings.TrimPrefix(sig, "func")

		argsSig, _, _ := strings.Cut(sig, ")")
		argsSig = strings.Trim(argsSig, "()")
		argsSig = strings.ReplaceAll(argsSig, ", ", "\n\t\t\t")
		argsSig = "\n\t\t\t" + argsSig

		sigtyp, ok := meth.Type().(*types.Signature)
		if !ok {
			return ifaceInfo{}, fmt.Errorf("unexpected type %T", meth.Type())
		}

		ret := sigtyp.Results().Len() > 0

		methInfos = append(methInfos, methInfo{
			Name:    meth.Name(),
			Sig:     sig,
			Args:    extractArgs(sigtyp),
			ArgsSig: argsSig,
			Results: extractResults(sigtyp),
			Ret:     ret,
		})
	}

	return ifaceInfo{
		name:           origIfacename,
		object:         object,
		methInfos:      methInfos,
		typeParamsDecl: typeParamsDecl,
		typeParams:     typeParams,
	}, nil
}

// extractArgs returns list of parameters names.
func extractArgs(sig *types.Signature) string {
	var args []string

	params := sig.Params()

	for i := range params.Len() {
		param := params.At(i)
		args = append(args, param.Name())
	}

	return strings.Join(args, ", ")
}

// extractResults returns list of results names.
func extractResults(sig *types.Signature) string {
	var args []string

	results := sig.Results()

	for i := range results.Len() {
		args = append(args, "r"+strconv.Itoa(i))
	}

	return strings.Join(args, ", ")
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

func typeParams(typ *types.Named, pkgAliasFn types.Qualifier) (typeParamsDecl string, typeParams string) {
	if typ.TypeParams().Len() > 0 {
		typeParamsDecl = "["
		typeParams = "["

		for i := range typ.TypeParams().Len() {
			param := typ.TypeParams().At(i)
			constraintName := types.TypeString(param.Constraint(), pkgAliasFn)

			if i == 0 {
				typeParamsDecl += param.String() + " " + constraintName
				typeParams += param.String()
			} else {
				typeParamsDecl += ", " + param.String() + " " + constraintName
				typeParams += ", " + param.String()
			}
		}

		typeParamsDecl += "]"
		typeParams += "]"
	}

	return typeParamsDecl, typeParams
}
