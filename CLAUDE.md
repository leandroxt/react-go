# CLAUDE.md — lit-go

Guidance for working in this repository.

---

## What this project is

A Next.js-shaped skeleton where **Go owns the pages** and **Lit owns the
interactive parts**. Pages under `ui/html/pages/` map to routes and are
rendered by Go templates; Lit components handle client-side behaviour.

The goal is a single Go binary that serves everything, with no Node process
in dev or production and no third-party origins at runtime.

---

## Motivation

### Why not load Lit from a CDN

**The current code does not work.** `ui/html/pages/home.html` contains:

```html
<script>
  import { html, css, LitElement } from 'lit';
```

Two independent failures:

1. **It is not a module.** A plain `<script>` cannot contain `import`. The
   browser throws a `SyntaxError` before the class is ever defined.
2. **`'lit'` is a bare specifier.** Even with `type="module"`, browsers do
   not resolve bare specifiers. They only understand URLs
   (`./x.js`, `/static/x.js`, `https://…`). Without an import map, `'lit'`
   resolves to nothing.

So a decision has to be made now, and CDN is the wrong one:

- **Runtime dependency on someone else's uptime.** Every page load blocks on
  a host you do not control. `unpkg` and `esm.sh` both have had multi-hour
  outages. That is your app down for a reason your logs will not explain.
- **Cross-origin waterfall.** `lit` imports `lit-element`, which imports
  `@lit/reactive-element`, which imports `lit-html`. Each is a separate
  request the browser cannot discover until the previous one parses. Four
  serial round trips to a third party before your first component upgrades.
  Bundled, it is one request to your own origin, already warm.
- **No tree-shaking.** The CDN ships whole packages. Bundling ships only what
  you import — the current component bundles to **15.3 kB unminified-gzip
  territory**, measured, including all of Lit.
- **Cache partitioning killed the shared-cache argument.** Since 2020 every
  major browser partitions the HTTP cache by top-level site. The old "your
  users already have Lit cached from another site" benefit does not exist
  anymore.
- **Version drift and supply chain.** A CDN URL is a live dependency on a
  registry. A compromised or republished package executes in your users'
  browsers with no build step to review.
- **It breaks the single-binary story.** The entire point of `ui/efs.go` and
  `//go:embed` is that `bin/app` is self-contained. A CDN link puts a hole in
  that: the binary is complete but the app still is not.

### Why esbuild's Go API specifically

esbuild is not just a CLI — it ships as a **Go package**
(`github.com/evanw/esbuild/pkg/api`). That means the bundler is `go get`-able,
compiles into this repo's own tooling, and runs in-process.

Consequences worth stating plainly:

- **Node never executes.** Not in dev, not in CI, not in prod. `npm` is used
  exactly once, to *download Lit's source files* into `node_modules/`. Those
  are inputs to esbuild, not a runtime.
- **The toolchain is `go run`.** No `package.json` scripts, no Vite config,
  no plugin ecosystem to keep current. `make build` is two Go commands.
- **It is fast enough to be invisible.** Measured on this bundle: **11 ms**.
  Watch-mode rebuilds are below perceptual threshold.
- **TypeScript comes free.** esbuild strips types natively, no extra tool.

---

## Implementation

### Directory layout

```
cmd/
  web/main.go           the server
  build/main.go         the bundler (new)
ui/
  efs.go                //go:embed "html" "static"
  html/
    base.html
    partials/
    pages/              one .html per route
  components/           Lit components (new) — SOURCE, not served
    index.ts            the registry: imports every component
    greeting.ts
  static/               BUILD OUTPUT — gitignored, embedded at compile time
    app.js
    main.css
tsconfig.json           type-checking config (new)
package.json            declares lit; npm only ever downloads source
```

The important separation: **`ui/components/` is source, `ui/static/` is
output.** Nothing in `ui/components/` is ever served. `//go:embed` picks up
`ui/static/` at compile time, so the bundle must be built *before* `go build`.

### Step 1 — dependencies

```sh
go get github.com/evanw/esbuild/pkg/api
npm install lit
```

Add to `.gitignore`:

```
node_modules/
ui/static/app.js
ui/static/app.js.map
bin/
```

`ui/static/app.js` is generated. Do not commit it. But note the consequence:
**`go build` fails on a clean checkout until the bundle exists**, because
`//go:embed "static"` needs the directory non-empty. Keep a committed
`ui/static/main.css` (already there) and the embed is satisfied.

### Step 2 — `cmd/build/main.go`

```go
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

		// Reads tsconfig.json automatically — see the class-fields note below.
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
```

### Step 3 — the component registry

`ui/components/index.ts` is the only entry point. Importing a component file
runs its `customElements.define()` call, so **this file is the registry**:

```ts
import './greeting.js';
```

Note the `.js` extension on a `.ts` file. That is correct and intentional —
it is what TypeScript's `bundler`/`nodenext` resolution expects, and it keeps
the source valid for tools that do not rewrite specifiers.

Add a component, add a line here. Nothing else registers elements.

### Step 4 — move the component out of the template

`ui/components/greeting.ts`:

```ts
import { LitElement, html, css } from 'lit';

export class SimpleGreeting extends LitElement {
  static styles = css`p { color: blue }`;
  static properties = { name: { type: String } };

  declare name: string; // see the class-fields note below

  constructor() {
    super();
    this.name = 'Somebody';
  }

  render() {
    return html`<p>Hello, ${this.name}!</p>`;
  }
}
customElements.define('simple-greeting', SimpleGreeting);
```

Delete the entire `{{define "script"}}` block from `home.html`. Components do
not live in templates. The page becomes just:

```html
{{define "main"}}
<main>
    <simple-greeting name="World"></simple-greeting>
</main>
{{end}}
```

### Step 5 — load it from `base.html`

```html
<link rel="stylesheet" href="/static/main.css" />

<!-- modulepreload starts the fetch during HTML parsing rather than after,
     which is most of the fix for the flash of undefined elements -->
<link rel="modulepreload" href="/static/app.js" />
<script type="module" src="/static/app.js"></script>
```

`type="module"` is mandatory. `defer` is implied.

`{{template "script" .}}` in `base.html` should be removed, or every page will
break for want of a `script` block.

### Step 6 — serve `/static/` (currently missing)

`cmd/web/main.go` registers no static handler at all, so `main.css` 404s
today. Add:

```go
r.Handle("GET /static/", http.StripPrefix("/static/",
	http.FileServerFS(staticFS)))
```

where `staticFS` is `fs.Sub(ui.Files, "static")`. Also note `base.html` asks
for `/static/css/main.css` but the file is at `ui/static/main.css` — pick one
and make them agree.

### Step 7 — Makefile

```make
.PHONY: dev build run

# Bundler in watch mode alongside the server.
dev:
	@go run ./cmd/build -watch & \
	go run ./cmd/web

# Assets first — they get embedded into the binary.
build:
	go run ./cmd/build
	go build -o bin/app ./cmd/web

run: build
	./bin/app
```

Order matters in `build`. Bundle, then compile.

---

## TypeScript without a build tool

Short answer: **the browser half is impossible; the checking half is free.**

### What is not possible

No browser can execute TypeScript. There is no `type="text/typescript"`, no
flag, no polyfill. `.ts` served to a browser is a syntax error. Something must
strip the types before the code reaches the client — that is not a preference,
it is the runtime's constraint.

Node *can* run TypeScript natively now (verified here on v22: `node file.ts`
works, types stripped, no config). But that only helps server-side JS, and
this project's server is Go. It does nothing for browser code.

### What is possible, and is the actual answer

**esbuild already does the transform, so no *additional* tool is needed.**
Set `EntryPoints` to `.ts` and it strips types with zero configuration. That
is the whole change. TypeScript here costs you nothing you were not already
paying.

The one thing esbuild does **not** do is *check* the types. It strips and
moves on; bad types compile silently. For checking, two options:

**Option A — `tsgo`, no emit.** TypeScript 7 (the Go port) went GA in July
2026. Install it and run:

```sh
npx tsgo --noEmit
```

Put it in CI or a pre-commit hook, never in the hot dev loop. Fast dev
iteration from esbuild, real checking where it matters. It still needs npm to
install, so it does not remove Node from the machine — only from the build.

**Option B — JSDoc in plain `.js` files.** If you want type safety with
literally zero transform:

```js
/** @type {string} */
name = 'Somebody';
```

`tsgo --noEmit` with `"allowJs": true, "checkJs": true` type-checks JSDoc
comments in `.js` files. The files stay valid JavaScript that a browser could
run directly. This is the closest thing to "TypeScript without a build tool"
that actually exists, and it is worth considering given that this project's
component count is small.

**Recommendation:** use `.ts`. esbuild is already in the pipeline, the
ergonomics are better than JSDoc, and Lit's own types are excellent.

### The class-fields footgun — read this before writing any `.ts` component

This is the one thing that will silently break Lit and cost an afternoon.

With `target: ES2022` and no tsconfig, esbuild emits class fields with
**define semantics**. Verified output:

```js
var Bad = class extends LitElement {
  static properties = { name: { type: String } };
  name = "x";              // ← Object.defineProperty on the instance
```

That instance property **shadows the accessor Lit installs on the prototype**.
Setting `this.name` no longer triggers a re-render. Nothing throws; the
component just stops being reactive.

With `useDefineForClassFields: false` in `tsconfig.json`, esbuild emits:

```js
var Bad = class extends LitElement {
  constructor() {
    super(...arguments);
    this.name = "x";       // ← assignment, hits Lit's setter
```

Which is correct.

So `tsconfig.json` is **not optional** here:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "useDefineForClassFields": false,
    "experimentalDecorators": true,
    "strict": true,
    "noEmit": true,
    "skipLibCheck": true
  },
  "include": ["ui/components/**/*.ts"]
}
```

And pass `Tsconfig: "tsconfig.json"` in `BuildOptions` so esbuild honours it.

Belt and braces: also use `declare name: string;` instead of a plain field
declaration. `declare` emits nothing at all, so it is immune to whichever
semantics are in effect.

---

## Conventions for this repo

- **`html/template`, never `text/template`.** `cmd/web/main.go` currently
  imports `text/template`, which does no contextual escaping. Any user data
  reaching a page is a stored XSS. Fix this before anything else in this
  document.
- **Components never live in templates.** `.ts` files in `ui/components/`,
  one element per file, `customElements.define` at the bottom, an import line
  in `index.ts`.
- **Templates parse once at boot.** `newTemplateCache()` already does this
  correctly. Keep it. Consider `template.Must`-style fatal-on-boot rather than
  a 500 later.
- **Render into a buffer.** Already done via `bufPool`. Keep it — a mid-render
  error must not emit half a page with a 200 already sent.
- **`500.html` must exist.** `render()` falls back to it; it is not in
  `ui/html/pages/`, so the fallback currently hits the "last resort" branch.
- **No `document.getElementById`.** The only DOM access should be a component
  querying its own children. If a page needs imperative DOM code, that is a
  missing component.
- **Server state reaches components as JSON, not fetches.** Handler marshals a
  struct, template writes it into
  `<script type="application/json">{{.InitialJSON}}</script>` inside the
  element, component reads it in `connectedCallback` **before** calling
  `super.connectedCallback()` (Lit's first render replaces the render root's
  children and would eat the script tag). `json.Marshal` escapes `<`, `>` and
  `&` by default, so a `</script>` payload comes out as `\u003c/script\u003e`.
- **Light DOM by default.** `createRenderRoot() { return this }` opts a
  component out of shadow DOM so `main.css` applies directly. Use shadow DOM
  when style isolation genuinely matters; CSS custom properties defined on
  `:root` still pierce the boundary either way, so keep design tokens global
  and component rules local.

---

## Known gaps

- **No live reload.** `make dev` rebuilds JS on save but the browser does not
  refresh. `fsnotify` on `ui/` plus an SSE endpoint firing `location.reload()`
  is roughly 40 lines and is the difference between pleasant and annoying.
- **No cache busting.** `/static/app.js` has no fingerprint, so a deploy can
  serve stale JS. Hash the file contents at startup and pass the query string
  into templates as a variable.
- **No route generation.** Routes are registered by hand in `main.go`. Go
  1.22+ `ServeMux` understands `GET /products/{id}` with `r.PathValue("id")`,
  so a generator only needs to walk `ui/html/pages/`, turn `[id]` into `{id}`,
  and emit the registration list. Worth writing at around the fifth page, not
  before.
- **`homePage` is empty** — it never calls `app.render`, and it is a bare
  function rather than a method on `application`, so it has no access to the
  template cache.

---

## Dropping `node_modules` entirely

`npm install` only fetches source files. To remove Node from the machine
completely:

```sh
npm install lit
mkdir -p ui/vendor
cp -r node_modules/lit node_modules/lit-html node_modules/lit-element \
      node_modules/@lit ui/vendor/
rm -rf node_modules package-lock.json
```

Then point esbuild at it:

```go
opts.NodePaths = []string{"ui/vendor"}
```

Commit `ui/vendor/`. After that the repo builds with `go build` alone, on any
machine with a Go toolchain and nothing else. Trade-off: dependency updates
become a manual copy, which for a dependency as stable as Lit is arguably a
feature.