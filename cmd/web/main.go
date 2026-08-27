package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"github.com/leandroxt/lit-go/ui"
)

// state is what the handler injects into the React app as its initial state.
type state struct {
	Count int `json:"count"`
}

func main() {
	home := template.Must(template.ParseFS(ui.Files, "html/home.html"))

	static, err := fs.Sub(ui.Files, "static")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		initial, err := json.Marshal(state{Count: 42})
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// template.JS so html/template writes the JSON verbatim instead of
		// re-encoding it as a JS string literal. json.Marshal already escapes
		// <, > and &, so a "</script>" payload cannot break out.
		//
		// Render into a buffer: a failed render must not emit half a page
		// with a 200 already sent.
		var buf bytes.Buffer
		if err := home.Execute(&buf, template.JS(initial)); err != nil {
			log.Printf("render: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		buf.WriteTo(w)
	})

	log.Println("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", mux))
}
