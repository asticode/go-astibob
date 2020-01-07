package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type Data struct {
	Layouts   [][]byte
	Statics   map[string][]byte
	Templates map[string][]byte
}

const s = `package index

import (
	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astikit"
	"github.com/julienschmidt/httprouter"
)

type resources struct {
	ls []string
	ss map[string]httprouter.Handle // Static handles indexed by path
	ts map[string]string            // Templates indexed by path
}

func newResources(l astikit.SeverityLogger) (r *resources) {
	// Create resources
	r = &resources{
		ss: make(map[string]httprouter.Handle),
		ts: make(map[string]string),
	}

	// Add layouts
	{{ range $_, $v := .Layouts }}r.ls = append(r.ls, string([]byte{ {{ range $_, $b := $v }}{{ printf "%#x," $b }}{{ end }} }))
	{{ end }}
	// Add static handles
	{{ range $k, $v := .Statics }}r.ss["{{ $k }}"] = astibob.ContentHandle("{{ $k }}", []byte{ {{ range $_, $b := $v }}{{ printf "%#x," $b }}{{ end }} }, l)
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
	log.SetFlags(0)

	// Parse template
	r := template.New("root")
	t, err := r.Parse(s)
	if err != nil {
		log.Fatal(fmt.Errorf("main: parsing template failed: %w", err))
	}

	// Create data
	d := Data{
		Statics:   make(map[string][]byte),
		Templates: make(map[string][]byte),
	}

	// Walk layouts
	if err := walkLayouts(&d); err != nil {
		log.Fatal(fmt.Errorf("main: walking layouts failed: %w", err))
	}

	// Walk statics
	if err := walkStatics(&d); err != nil {
		log.Fatal(fmt.Errorf("main: walking statics failed: %w", err))
	}

	// Walk templates
	if err := walkTemplates(&d); err != nil {
		log.Fatal(fmt.Errorf("main: walking templates failed: %w", err))
	}

	// Create destination
	dp := filepath.Join("index/resources.go")
	f, err := os.Create(dp)
	if err != nil {
		log.Fatal(fmt.Errorf("main: creating %s failed: %w", dp, err))
	}
	defer f.Close()

	// Execute template
	if err = t.Execute(f, d); err != nil {
		log.Fatal(fmt.Errorf("main: executing template failed: %w", err))
	}
}

func walkLayouts(d *Data) (err error) {
	lp := "index/resources/templates/layouts"
	if err = filepath.Walk(lp, func(path string, info os.FileInfo, e error) (err error) {
		// Check input error
		if e != nil {
			err = fmt.Errorf("main: walking layouts has an input error for path %s: %w", path, e)
			return
		}

		// Only process files
		if info.IsDir() {
			return
		}

		// Read file
		var b []byte
		if b, err = ioutil.ReadFile(path); err != nil {
			err = fmt.Errorf("main: reading %s failed: %w", path, err)
			return
		}

		// Add to data
		d.Layouts = append(d.Layouts, b)
		return
	}); err != nil {
		err = fmt.Errorf("main: walking %s failed: %w", lp, err)
		return
	}
	return
}

func walkStatics(d *Data) (err error) {
	sp := "index/resources/static"
	if err = filepath.Walk(sp, func(path string, info os.FileInfo, e error) (err error) {
		// Check input error
		if e != nil {
			err = fmt.Errorf("main: walking layouts has an input error for path %s: %w", path, e)
			return
		}

		// Only process files
		if info.IsDir() {
			return
		}

		// Read file
		var b []byte
		if b, err = ioutil.ReadFile(path); err != nil {
			err = fmt.Errorf("main: reading %s failed: %w", path, err)
			return
		}

		// Add to data
		d.Statics[filepath.ToSlash(strings.TrimPrefix(path, sp))] = b
		return
	}); err != nil {
		err = fmt.Errorf("main: walking %s failed: %w", sp, err)
		return
	}
	return
}

func walkTemplates(d *Data) (err error) {
	tp := "index/resources/templates/pages"
	if err = filepath.Walk(tp, func(path string, info os.FileInfo, e error) (err error) {
		// Check input error
		if e != nil {
			err = fmt.Errorf("main: walking layouts has an input error for path %s: %w", path, e)
			return
		}

		// Only process files
		if info.IsDir() {
			return
		}

		// Read file
		var b []byte
		if b, err = ioutil.ReadFile(path); err != nil {
			err = fmt.Errorf("main: reading %s failed: %w", path, err)
			return
		}

		// Add to data
		d.Templates[filepath.ToSlash(strings.TrimPrefix(path, tp))] = b
		return
	}); err != nil {
		err = fmt.Errorf("main: walking %s failed: %w", tp, err)
		return
	}
	return
}
