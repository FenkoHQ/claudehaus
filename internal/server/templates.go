package server

import (
	"html/template"
	"io"
	"io/fs"

	claudehaus "github.com/aliadnani/claudehaus"
)

type Templates struct {
	templates *template.Template
}

func NewTemplates() (*Templates, error) {
	tfs := claudehaus.TemplatesFS()

	tmpl, err := template.ParseFS(tfs, "*.html")
	if err != nil {
		return nil, err
	}

	partials, err := fs.Glob(tfs, "partials/*.html")
	if err != nil {
		return nil, err
	}

	if len(partials) > 0 {
		tmpl, err = tmpl.ParseFS(tfs, partials...)
		if err != nil {
			return nil, err
		}
	}

	return &Templates{templates: tmpl}, nil
}

func (t *Templates) Render(w io.Writer, name string, data any) error {
	return t.templates.ExecuteTemplate(w, name, data)
}
