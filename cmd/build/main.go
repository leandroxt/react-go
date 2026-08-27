// Command build bundles the client assets.
//
// This is the entire frontend toolchain: esbuild's Go API, in-process.
//
//	go run ./cmd/build            production
//	go run ./cmd/build -watch     rebuild on change
package main

import (
	"flag"
	"log"
	"os"

	"github.com/evanw/esbuild/pkg/api"
)

func main() {
	watch := flag.Bool("watch", false, "rebuild on file change")
	flag.Parse()

	// React ships both a dev and a prod path behind this check. Without the
	// define, esbuild cannot drop the dev half and the bundle stays large.
	env := `"production"`
	if *watch {
		env = `"development"`
	}

	opts := api.BuildOptions{
		EntryPoints: []string{"ui/app/main.tsx"},
		Outfile:     "ui/static/app.js",

		Bundle: true,
		Write:  true,
		Format: api.FormatESModule,
		Target: api.ES2022,

		JSX:     api.JSXAutomatic, // no `import React` needed
		Define:  map[string]string{"process.env.NODE_ENV": env},

		LogLevel: api.LogLevelInfo,
	}

	if *watch {
		opts.Sourcemap = api.SourceMapLinked
		ctx, err := api.Context(opts)
		if err != nil {
			log.Fatalf("build: %v", err)
		}
		if err := ctx.Watch(api.WatchOptions{}); err != nil {
			log.Fatalf("build: %v", err)
		}
		log.Println("build: watching ui/app")
		select {}
	}

	opts.MinifyWhitespace = true
	opts.MinifyIdentifiers = true
	opts.MinifySyntax = true

	if r := api.Build(opts); len(r.Errors) > 0 {
		os.Exit(1) // non-zero so make and CI actually stop
	}
	log.Println("build: wrote ui/static/app.js")
}
