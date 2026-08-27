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

	opts := api.BuildOptions{
		EntryPoints: []string{"ui/components/index.ts"},
		Outfile:     "ui/static/app.js",

		Bundle: true,
		Write:  true,
		Format: api.FormatESModule,
		Target: api.ES2022, // native class fields, no transpile tax

		// Honours useDefineForClassFields: false, without which class fields
		// get define semantics and silently shadow Lit's reactive accessors.
		Tsconfig: "tsconfig.json",

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
		log.Println("build: watching ui/components")
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
