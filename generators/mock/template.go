package mock

import "text/template"

//nolint:lll
const tmplText = `// *{{.ConcrName}} implements {{.InterfaceName}}.
type {{.ConcrName}}{{.TypeParamsDecl}} struct {
{{range .Methods}}	{{.Name}}Func func {{.Sig}}
{{end}}
	Calls struct{
{{range .Methods}}		{{.Name}} []struct{ {{.ArgsSig}} 
		}
{{end}}	}
}
{{range .Methods}}
func (mock *{{$.ConcrName}}{{$.TypeParams}}) {{.Name}}{{.Sig}} {
	if mock.{{.Name}}Func == nil {
		panic("nil method {{.Name}} is called!")
	}

	callInfo := struct{ {{.ArgsSig}}
	} { {{.Args}} }

	mock.Calls.{{.Name}} = append(mock.Calls.{{.Name}}, callInfo)

	{{if .Ret}}return mock.{{.Name}}Func({{.Args}}){{else}}mock.{{.Name}}Func(){{end}}
}
{{end}}
`

var tmpl = template.Must(template.New("mock").Parse(tmplText))
