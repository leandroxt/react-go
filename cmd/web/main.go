package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/leandroxt/lit-go/ui"
)

type application struct {
	logger        *slog.Logger
	templateCache map[string]*template.Template
}

// counterState is the server-owned initial state for <app-counter>. It reaches
// the component as JSON embedded in the page, not via a fetch.
type counterState struct {
	Count int `json:"count"`
	Step  int `json:"step"`
}

func (app *application) homePage(w http.ResponseWriter, r *http.Request) {
	initial, err := json.Marshal(counterState{Count: 42, Step: 1})
	if err != nil {
		app.logger.Error("marshal initial state", slog.String("err", err.Error()))
		app.render(w, r, http.StatusInternalServerError, "500.html", nil)
		return
	}

	app.render(w, r, http.StatusOK, "home.html", map[string]any{
		"Title": "Hello, World!",
		// template.JS so html/template writes the JSON verbatim rather than
		// re-encoding it as a JS string literal. json.Marshal already escapes
		// <, > and &, so a "</script>" payload comes out as </script>.
		"InitialJSON": template.JS(initial),
	})
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	templateCache, err := newTemplateCache()
	if err != nil {
		logger.Error("cant create template cache",
			slog.String("err", err.Error()))
		os.Exit(1)
		return
	}

	staticFS, err := fs.Sub(ui.Files, "static")
	if err != nil {
		logger.Error("cant open static files",
			slog.String("err", err.Error()))
		os.Exit(1)
		return
	}

	app := application{
		logger:        logger,
		templateCache: templateCache,
	}

	r := http.NewServeMux()
	r.Handle("GET /static/", http.StripPrefix("/static/",
		http.FileServerFS(staticFS)))
	r.HandleFunc("GET /{$}", app.homePage)

	if err = app.serve(r); err != nil {
		logger.Error("failed to serve",
			slog.String("error", err.Error()))
	}
}

func (app *application) serve(r *http.ServeMux) error {
	server := http.Server{
		Addr:         "0.0.0.0:8080",
		Handler:      r,
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Minute,
		IdleTimeout:  time.Minute,
	}

	app.logger.Info("Starting server...", slog.String("addr", server.Addr))

	if err := server.ListenAndServe(); err != nil {
		app.logger.Error("error starting server",
			slog.String("err", err.Error()))
		return err
	}

	return nil
}

func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	// ParseFS fails on a pattern that matches nothing, so only include the
	// partials glob once there is at least one partial to parse.
	partials, err := fs.Glob(ui.Files, "html/partials/*.html")
	if err != nil {
		return nil, err
	}

	// Walk the html/pages directory recursively to find all .html files
	err = fs.WalkDir(ui.Files, "html/pages", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".html" {
			return nil
		}

		relPath, err := filepath.Rel("html/pages", path)
		if err != nil {
			return err
		}
		name := relPath

		patterns := []string{"html/base.html"}
		patterns = append(patterns, partials...)
		patterns = append(patterns, path)

		ts, err := template.New(name).ParseFS(ui.Files, patterns...)
		if err != nil {
			return err
		}

		cache[name] = ts
		return nil
	})

	if err != nil {
		return nil, err
	}

	return cache, nil
}

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

func (app *application) render(w http.ResponseWriter, r *http.Request, status int, page string, data map[string]any) {
	ts, ok := app.templateCache[page]
	if !ok {
		app.logger.Error("template not found", "page", page)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	err := ts.ExecuteTemplate(buf, "base", data)
	if err != nil {
		app.logger.Error("render page error", "message", err.Error())
		if page != "500.html" {
			app.render(w, r, http.StatusInternalServerError, "500.html", nil)
		} else {
			// Last resort: plain text, avoid infinite loop
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	buf.WriteTo(w)
}
