package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/leandroxt/lit-go/ui"
)

// ---------------------------------------------------------------------------
// Page contract
// ---------------------------------------------------------------------------

// shared is injected into every page. Pages are separate documents, so React
// state does not survive navigation — anything that must be present everywhere
// is server-owned and re-sent on each render.
type shared struct {
	AppName string `json:"appName"`
	User    string `json:"user"`
}

// page is what every handler builds and render() consumes.
//
// Route is both the template name and the manifest key, so /about resolves to
// html/pages/about.html and to the about-<hash>.js bundle.
type page struct {
	Route       string
	Title       string
	Description string
	Props       any // page-specific initial state; nil when the page needs none
}

type application struct {
	templates map[string]*template.Template
	bundles   map[string]string // route -> hashed bundle URL
	shared    shared
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

// view is the data every template sees. state() and bundle() below are the
// only two things templates need from it.
type view struct {
	page
	Shared shared
	Bundle string
}

// state emits the JSON the React entry point reads before mounting.
//
// Centralising the encoding is the point: no handler can forget it, and
// json.Marshal escapes <, > and & so a "</script>" payload cannot break out.
func state(v view) (template.HTML, error) {
	b, err := json.Marshal(struct {
		Shared shared `json:"shared"`
		Props  any    `json:"props"`
	}{v.Shared, v.Props})
	if err != nil {
		return "", fmt.Errorf("state %q: %w", v.Route, err)
	}
	return template.HTML(
		`<script type="application/json" id="state">` + string(b) + `</script>`,
	), nil
}

func (app *application) render(w http.ResponseWriter, p page) {
	ts, ok := app.templates[p.Route]
	if !ok {
		log.Printf("render: no template for route %q", p.Route)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	bundle, ok := app.bundles[p.Route]
	if !ok {
		log.Printf("render: no bundle for route %q — run `go run ./cmd/build`", p.Route)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Buffer first: a failed render must not emit half a page with a 200
	// already sent.
	var buf bytes.Buffer
	v := view{page: p, Shared: app.shared, Bundle: bundle}
	if err := ts.ExecuteTemplate(&buf, "base", v); err != nil {
		log.Printf("render %s: %v", p.Route, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

// ---------------------------------------------------------------------------
// Startup
// ---------------------------------------------------------------------------

func newTemplates() (map[string]*template.Template, error) {
	funcs := template.FuncMap{"state": state}

	pages, err := fs.Glob(ui.Files, "html/pages/*.html")
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no page templates in html/pages")
	}

	cache := map[string]*template.Template{}
	for _, p := range pages {
		route := strings.TrimSuffix(filepath.Base(p), ".html")

		ts, err := template.New(route).Funcs(funcs).ParseFS(ui.Files, "html/base.html", p)
		if err != nil {
			return nil, err
		}
		cache[route] = ts
	}
	return cache, nil
}

// newBundles reads the manifest esbuild wrote, mapping route -> hashed URL.
func newBundles() (map[string]string, error) {
	b, err := ui.Files.ReadFile("static/build/manifest.json")
	if err != nil {
		return nil, fmt.Errorf("read manifest (run `go run ./cmd/build` first): %w", err)
	}

	var bundles map[string]string
	if err := json.Unmarshal(b, &bundles); err != nil {
		return nil, err
	}
	return bundles, nil
}

// immutable marks the hashed build output as cacheable forever.
//
// Without this the shared chunk — React, ~190kb — is refetched on every
// navigation, which cancels most of the benefit of splitting per route. It is
// safe precisely because the filenames are content-hashed: a changed file gets
// a new URL, so a stale cache entry can never be served.
func immutable(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/static/build/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		h.ServeHTTP(w, r)
	})
}

// ---------------------------------------------------------------------------
// Handlers — one per route, each passing its own initial state.
// ---------------------------------------------------------------------------

type homeProps struct {
	Count int `json:"count"`
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	app.render(w, page{
		Route:       "home",
		Title:       "Home",
		Description: "Go renders the shell, React renders <main>, one binary ships both.",
		Props:       homeProps{Count: 42},
	})
}

func (app *application) about(w http.ResponseWriter, r *http.Request) {
	// Props is nil: this page needs no initial state, only the shared context.
	app.render(w, page{
		Route:       "about",
		Title:       "Sobre",
		Description: "Como o bundle por rota e o contexto compartilhado funcionam.",
	})
}

func main() {
	templates, err := newTemplates()
	if err != nil {
		log.Fatalf("templates: %v", err)
	}

	bundles, err := newBundles()
	if err != nil {
		log.Fatalf("bundles: %v", err)
	}

	static, err := fs.Sub(ui.Files, "static")
	if err != nil {
		log.Fatal(err)
	}

	app := &application{
		templates: templates,
		bundles:   bundles,
		shared:    shared{AppName: "go-react", User: "leandro"},
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", immutable(http.StripPrefix("/static/", http.FileServerFS(static))))
	mux.HandleFunc("GET /{$}", app.home)
	mux.HandleFunc("GET /about", app.about)

	log.Println("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", mux))
}
