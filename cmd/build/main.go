// Command build bundles one JS bundle per route.
//
// This is the entire frontend toolchain: esbuild's Go API, in-process.
//
//	go run ./cmd/build            production
//	go run ./cmd/build -watch     rebuild on change
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

const (
	routesDir = "ui/app/routes" // one .tsx per route — these are the entry points
	outDir    = "ui/static/build"
	manifest  = outDir + "/manifest.json"
)

func main() {
	watch := flag.Bool("watch", false, "rebuild on file change")
	flag.Parse()

	entries, err := filepath.Glob(filepath.Join(routesDir, "*.tsx"))
	if err != nil {
		log.Fatalf("build: %v", err)
	}
	if len(entries) == 0 {
		log.Fatalf("build: no route entry points in %s", routesDir)
	}

	// React ships both a dev and a prod path behind this check. Without the
	// define, esbuild cannot drop the dev half.
	env := `"production"`
	if *watch {
		env = `"development"`
	}

	// Output names are content-hashed, so stale files would accumulate.
	if err := os.RemoveAll(outDir); err != nil {
		log.Fatalf("build: %v", err)
	}

	opts := api.BuildOptions{
		EntryPoints: entries,
		Outdir:      outDir,

		Bundle: true,
		Write:  true,
		Format: api.FormatESModule, // required by Splitting
		Target: api.ES2022,

		JSX:    api.JSXAutomatic, // no `import React` needed
		Define: map[string]string{"process.env.NODE_ENV": env},

		// One bundle per route. Code reachable from more than one entry point
		// — React itself, mount.tsx, shared.tsx — is hoisted into a chunk both
		// routes import, so it is downloaded and parsed once.
		Splitting:  true,
		EntryNames: "[name]-[hash]",
		ChunkNames: "chunk-[hash]",

		// Content hashes are the cache-busting story too: a changed file gets
		// a new URL, so /static/build can be served immutable.
		Metafile: true,

		Plugins:  []api.Plugin{manifestPlugin()},
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
		log.Printf("build: watching %s", routesDir)
		select {}
	}

	opts.MinifyWhitespace = true
	opts.MinifyIdentifiers = true
	opts.MinifySyntax = true

	if r := api.Build(opts); len(r.Errors) > 0 {
		os.Exit(1) // non-zero so make and CI actually stop
	}
}

// manifestPlugin writes route name -> hashed URL after every build, watch
// rebuilds included. Go reads it to know which bundle a route needs.
func manifestPlugin() api.Plugin {
	return api.Plugin{
		Name: "manifest",
		Setup: func(b api.PluginBuild) {
			b.OnEnd(func(r *api.BuildResult) (api.OnEndResult, error) {
				if len(r.Errors) > 0 {
					return api.OnEndResult{}, nil
				}
				if err := writeManifest(r.Metafile); err != nil {
					log.Printf("build: manifest: %v", err)
					return api.OnEndResult{}, err
				}
				return api.OnEndResult{}, nil
			})
		},
	}
}

func writeManifest(metafile string) error {
	var meta struct {
		Outputs map[string]struct {
			EntryPoint string `json:"entryPoint"`
		} `json:"outputs"`
	}
	if err := json.Unmarshal([]byte(metafile), &meta); err != nil {
		return err
	}

	// Only entry points get an entry; shared chunks are pulled in by the
	// entry's own imports.
	routes := map[string]string{}
	for out, o := range meta.Outputs {
		if o.EntryPoint == "" {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(o.EntryPoint), filepath.Ext(o.EntryPoint))
		routes[name] = "/" + strings.TrimPrefix(filepath.ToSlash(out), "ui/")
	}

	b, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(manifest, b, 0o644); err != nil {
		return err
	}

	log.Printf("build: %d route bundles", len(routes))
	return nil
}
