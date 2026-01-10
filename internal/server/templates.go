package server

import (
	"html/template"
	"io"
	"path/filepath"
)

type Templates struct {
	templates *template.Template
}

func NewTemplates() (*Templates, error) {
	tmpl, err := template.ParseGlob(filepath.Join("web", "templates", "*.html"))
	if err != nil {
		return nil, err
	}

	partials, err := filepath.Glob(filepath.Join("web", "templates", "partials", "*.html"))
	if err != nil {
		return nil, err
	}

	if len(partials) > 0 {
		tmpl, err = tmpl.ParseFiles(partials...)
		if err != nil {
			return nil, err
		}
	}

	return &Templates{templates: tmpl}, nil
}

func (t *Templates) Render(w io.Writer, name string, data any) error {
	return t.templates.ExecuteTemplate(w, name, data)
}
