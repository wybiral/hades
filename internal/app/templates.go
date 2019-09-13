package app

import (
	"html/template"

	"github.com/gobuffalo/packr"
)

func newTemplates(path string) (*template.Template, error) {
	tbox := packr.NewBox(path)
	var t *template.Template
	for _, name := range tbox.List() {
		var tmpl *template.Template
		if t == nil {
			t = template.New(name)
			tmpl = t
		} else {
			tmpl = t.New(name)
		}
		txt, err := tbox.FindString(name)
		if err != nil {
			return nil, err
		}
		_, err = tmpl.Parse(txt)
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}
