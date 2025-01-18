package proxy

import "text/template"

//nolint:lll
const tmplText = `// *{{.ConcrName}} implements {{.InterfaceName}}.
type {{.ConcrName}}{{.TypeParamsDecl}} struct {
	v      {{.InterfaceName}}{{.TypeParams}}
	logger interface{Log(string, ...any)}
}

func New{{.ConcrName}}{{.TypeParamsDecl}}(v {{.InterfaceName}}{{.TypeParams}}, logger interface{Log(string, ...any)}) (*{{.ConcrName}}{{.TypeParams}}, error) {
	if v == nil {
		return nil, errors.New("v is nil")
	}
	if logger == nil {
		return nil, errors.New("logger is nil")
	}
	return &{{.ConcrName}}{{.TypeParams}}{
		v:      v,
		logger: logger,
	}, nil
}
{{range .Methods}}
func (p *{{$.ConcrName}}{{$.TypeParams}}) {{.Name}}{{.Sig}} {
	p.logger.Log("Calling {{.Name}}", "arguments", {{.Args}})
	{{if .Ret}}{{.Results}} := p.v.{{.Name}}({{.Args}})
	p.logger.Log("Calling {{.Name}}", "results", {{.Results}})
	return {{.Results}}{{else}}p.v.{{.Name}}({{.Args}})
	p.logger.Log("Calling {{.Name}}", "results"){{end}}
}
{{end}}
`

var tmpl = template.Must(template.New("proxy").Parse(tmplText))
