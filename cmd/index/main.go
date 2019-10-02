package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

type Data struct {
	Layouts   [][]byte
	Statics   map[string][]byte
	Templates map[string][]byte
}

const s = `package index

import (
	"github.com/asticode/go-astibob"
	"github.com/julienschmidt/httprouter"
)

type resources struct {
	ls []string
	ss map[string]httprouter.Handle // Static handles indexed by path
	ts map[string]string            // Templates indexed by path
}

func newResources() (r *resources) {
	// Create resources
	r = &resources{
		ss: make(map[string]httprouter.Handle),
		ts: make(map[string]string),
	}

	// Add layouts
	{{ range $_, $v := .Layouts }}r.ls = append(r.ls, string([]byte{ {{ range $_, $b := $v }}{{ printf "%#x," $b }}{{ end }} }))
	{{ end }}
	// Add static handles
	{{ range $k, $v := .Statics }}r.ss["{{ $k }}"] = astibob.ContentHandle("{{ $k }}", []byte{ {{ range $_, $b := $v }}{{ printf "%#x," $b }}{{ end }} })
	{{ end }}
	// Add templates
	{{ range $k, $v := .Templates }}r.ts["{{ $k }}"] = string([]byte{ {{ range $_, $b := $v }}{{ printf "%#x," $b }}{{ end }} })
	{{ end }}
	return
}

func (r *resources) layouts() []string {
	return r.ls
}

func (r *resources) statics() map[string]httprouter.Handle {
	return r.ss
}

func (r *resources) templates() map[string]string {
	return r.ts
}`

func main() {
	// Set logger
	flag.Parse()
	astilog.FlagInit()

	// Parse template
	r := template.New("root")
	t, err := r.Parse(s)
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "main: parsing template failed"))
	}

	// Create data
	d := Data{
		Statics:   make(map[string][]byte),
		Templates: make(map[string][]byte),
	}

	// Walk layouts
	if err := walkLayouts(&d); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: walking layouts failed"))
	}

	// Walk statics
	if err := walkStatics(&d); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: walking statics failed"))
	}

	// Walk templates
	if err := walkTemplates(&d); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: walking templates failed"))
	}

	// Create destination
	dp := filepath.Join("index/resources.go")
	f, err := os.Create(dp)
	if err != nil {
		astilog.Fatal(errors.Wrapf(err, "main: creating %s failed", dp))
	}
	defer f.Close()

	// Execute template
	if err = t.Execute(f, d); err != nil {
		astilog.Fatal(errors.Wrap(err, "main: executing template failed"))
	}
}

func walkLayouts(d *Data) (err error) {
	lp := "index/resources/templates/layouts"
	if err = filepath.Walk(lp, func(path string, info os.FileInfo, e error) (err error) {
		// Check input error
		if e != nil {
			err = errors.Wrapf(e, "main: walking layouts has an input error for path %s", path)
			return
		}

		// Only process files
		if info.IsDir() {
			return
		}

		// Read file
		var b []byte
		if b, err = ioutil.ReadFile(path); err != nil {
			err = errors.Wrapf(err, "main: reading %s failed", path)
			return
		}

		// Add to data
		d.Layouts = append(d.Layouts, b)
		return
	}); err != nil {
		err = errors.Wrapf(err, "main: walking %s failed", lp)
		return
	}
	return
}

func walkStatics(d *Data) (err error) {
	sp := "index/resources/static"
	if err = filepath.Walk(sp, func(path string, info os.FileInfo, e error) (err error) {
		// Check input error
		if e != nil {
			err = errors.Wrapf(e, "main: walking layouts has an input error for path %s", path)
			return
		}

		// Only process files
		if info.IsDir() {
			return
		}

		// Read file
		var b []byte
		if b, err = ioutil.ReadFile(path); err != nil {
			err = errors.Wrapf(err, "main: reading %s failed", path)
			return
		}

		// Add to data
		d.Statics[filepath.ToSlash(strings.TrimPrefix(path, sp))] = b
		return
	}); err != nil {
		err = errors.Wrapf(err, "main: walking %s failed", sp)
		return
	}
	return
}

func walkTemplates(d *Data) (err error) {
	tp := "index/resources/templates/pages"
	if err = filepath.Walk(tp, func(path string, info os.FileInfo, e error) (err error) {
		// Check input error
		if e != nil {
			err = errors.Wrapf(e, "main: walking layouts has an input error for path %s", path)
			return
		}

		// Only process files
		if info.IsDir() {
			return
		}

		// Read file
		var b []byte
		if b, err = ioutil.ReadFile(path); err != nil {
			err = errors.Wrapf(err, "main: reading %s failed", path)
			return
		}

		// Add to data
		d.Templates[filepath.ToSlash(strings.TrimPrefix(path, tp))] = b
		return
	}); err != nil {
		err = errors.Wrapf(err, "main: walking %s failed", tp)
		return
	}
	return
}
