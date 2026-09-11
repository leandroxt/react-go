<div align="center">

# react-go

**Go owns the document. React owns `<main>`. One binary ships both.**

A reference implementation of a full-page React frontend served from a single
Go binary — no Node at runtime, no Node in the build, no CDN, no third-party
origin anywhere in the request path.

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![React](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=black)](https://react.dev)
[![esbuild](https://img.shields.io/badge/esbuild-Go%20API-FFCF00?logo=esbuild&logoColor=black)](https://esbuild.github.io)
[![Node at runtime](https://img.shields.io/badge/Node%20at%20runtime-none-success)](#no-node-anywhere-that-matters)
[![Ships as](https://img.shields.io/badge/ships%20as-1%20binary-blue)](#everything-is-one-binary)

</div>

---

## The idea in one paragraph

Go serves the shell — the `<head>` with its SEO metadata, the site header, the
footer — and injects the page's initial state as JSON. React mounts over
`<main>` and renders that page in full. Each route is its own esbuild entry
point, so it ships its own bundle plus one shared chunk that every route
reuses. The bundler is a 150-line Go program using esbuild's Go API in-process,
and its output is embedded into the server binary with `//go:embed`. The result
is one file you can `scp` to a box and run.

> **The split is at `<main>`, not inside it.** Mixing Go-rendered HTML and a
> small React widget in the same page body is hard to read and harder to
> maintain. Either a page's body is Go's, or it is React's.

---

## Why

The default modern answer to "Go backend, React frontend" is two servers, two
deploys, two dependency trees, and a `node_modules` in production. That is a
lot of moving parts for a site that is mostly documents.

The other default — server-rendered templates with a sprinkle of JavaScript —
gives up the component model exactly when a page gets interactive enough to
need it.

This repo takes a third path:

| | Two-server SPA | Templates + sprinkles | **react-go** |
|---|---|---|---|
| Deploy artifacts | 2 | 1 | **1** |
| Node in production | yes | no | **no** |
| Node in the build | yes | usually | **no** |
| `<head>` / SEO | client-rendered | server | **server** |
| Component model | full | none | **full, per route** |
| Payload for a 2nd route | shared chunk cached | n/a | **~800 bytes** |

---

## Quick start

```sh
npm install          # React + types. Fetched, never executed.
make dev             # esbuild in watch mode + server on :8080
```

| Command | What it does |
|---|---|
| `make dev` | esbuild watcher alongside the server |
| `make build` | bundle, then compile to `bin/app` |
| `make run` | build, then run |
| `make check` | `npx tsgo --noEmit` — the only thing that checks types |

> **Order matters.** `//go:embed` reads `ui/static/` at compile time, so the
> bundle must exist *before* `go build`. That is why `make build` is two steps
> and not one.

---

## How a request works

```
  GET /about
      │
      ▼
┌─────────────────────────────────────────────────────────────┐
│ cmd/web · app.about()                                        │
│   page{ Route: "about", Title: …, Description: …, Props: nil }│
└─────────────────────────────────────────────────────────────┘
      │
      ▼   Route is the single key ───────────────┐
┌──────────────────────────────┐                 │
│ render()                     │                 ▼
│  templates["about"] ─────────┼──→  ui/html/pages/about.html
│  bundles["about"]   ─────────┼──→  /static/build/about-OZUTUGHZ.js
│  execute into a bytes.Buffer │        (from manifest.json)
└──────────────────────────────┘
      │
      ▼
┌─────────────────────────────────────────────────────────────┐
│ HTML on the wire                                             │
│   <head>  title · description · canonical · modulepreload    │
│   <header>                       ← Go's                      │
│   <main></main>                  ← empty, React's to fill    │
│   <script type="application/json" id="state">{…}</script>    │
└─────────────────────────────────────────────────────────────┘
      │
      ▼
┌─────────────────────────────────────────────────────────────┐
│ about-OZUTUGHZ.js  →  mount(About)                           │
│   reads #state → { shared, props }                           │
│   createRoot(<main>).render(<SharedProvider><About/></…>)    │
└─────────────────────────────────────────────────────────────┘
```

A missing template or a missing manifest entry fails at startup or as a logged
500 — never silently, and never as a blank page.

---

## Layout

```
cmd/
  web/main.go            server, handlers, render(), the state template func
  build/main.go          bundler: multi-entry, splitting, hashing, manifest
ui/
  efs.go                 //go:embed "html" "static"
  html/
    base.html            shell: <head>, header, <main>, state
    pages/
      home.html          {{define "main"}}<main></main>{{end}} (+ optional head)
      about.html
  app/
    mount.tsx            createRoot over <main> + shared context
    shared.tsx           the cross-page context
    routes/
      home.tsx           page component; ends with mount(Home)
      about.tsx
  static/
    main.css
    build/               OUTPUT — gitignored: hashed bundles + manifest.json
tsconfig.json            type-checking only (noEmit)
```

`ui/app/` is source. `ui/static/` is output. The whole server is 216 lines and
the whole build system is 150.

---

## Per-route bundles

`cmd/build` globs `ui/app/routes/*.tsx` as entry points with `Splitting: true`.
Anything reachable from more than one entry — React, `mount`, `shared` — is
hoisted into a chunk that every route imports.

Measured in this repo:

| Output | Raw | gzip | Contents |
|---|---:|---:|---|
| `chunk-*.js` | 190.0 KiB | 59.2 KiB | React, ReactDOM, `mount`, `shared` |
| `home-*.js` | 789 B | 442 B | just the Home component |
| `about-*.js` | 1,061 B | 579 B | just the About component |

**A new route costs its own code, not another copy of React.**

Output names are content-hashed, and esbuild's metafile is turned into
`static/build/manifest.json` (`route → /static/build/route-HASH.js`) which Go
reads at startup. That manifest is what lets `base.html` emit the right
`<script>` per route, and the hash doubles as cache busting.

### `immutable` is what makes splitting pay off

`/static/build/` is served `Cache-Control: public, max-age=31536000, immutable`.

Without it, the 190 KiB shared chunk is refetched on every navigation, which
cancels most of the point of splitting. Verified in DevTools: `transferSize`
went from `194847` to `0` on the second route.

> It is only safe **because the filenames are hashed** — a changed file gets a
> new URL, so a stale cache entry can never be served. Never apply this to an
> unhashed asset like `main.css`.

---

## State and the shared context

One `<script type="application/json" id="state">` per page carries both halves:

```json
{
  "shared": { "appName": "go-react", "user": "leandro" },
  "props":  { "count": 42 }
}
```

- **`props`** — page-specific, typed per route, `null` when a page needs none.
- **`shared`** — injected into every page, provided to React through context
  and read with `useShared()`.

Pages are separate documents, so a navigation tears down the React tree.
`shared` is **not** client state that survives a page load — it survives
because **Go re-injects it on every render**, which keeps the server the single
source of truth. If you ever need genuinely client-only state to persist across
navigations, that is `sessionStorage`, and it is a separate decision.

The `state` template func in `cmd/web/main.go` is the only place that marshals.
Centralising it means no handler can forget it, and `json.Marshal` escapes `<`,
`>` and `&`, so a `</script>` payload cannot break out of the block.

---

## SEO — the honest version

`<head>` is server-rendered: title, description, and a per-page `{{block
"head"}}` for canonical, OG tags and JSON-LD. That is real value — it is
exactly what search results and social previews read.

**But the body is client-rendered, so its copy is not reliably indexed.** Google
renders JavaScript on a deferred second-wave crawl with no timing guarantee.
Bing, social scrapers and most LLM crawlers largely do not render at all.

The per-route split is what turns this into a *per-route* decision instead of a
global one:

- A **marketing or docs route** can render its body straight from the Go
  template and ship no bundle at all.
- An **app route behind auth** mounts React over `<main>` and does not care,
  because nothing is indexing it anyway.

Decide per route, in the handler.

---

## Decisions

### No Node anywhere that matters

`cmd/build` uses esbuild's **Go API**, in-process. `go run ./cmd/build` is the
entire frontend toolchain: bundling, splitting, hashing, minification, the
manifest, and watch mode. Cold build: ~30 ms.

npm is used to *fetch* React and the type definitions. Node is never executed
to build or to serve. This is also the reason the stack is React and not
Svelte — esbuild has a built-in loader for `.tsx`, but a `.svelte` file needs
`svelte.compile()`, which is JavaScript, which would put a Node subprocess back
in the middle of the build.

### Everything is one binary

`//go:embed "html" "static"` pulls the templates, the CSS and the hashed
bundles into the executable. `bin/app` is ~11.8 MiB and has no runtime
dependency on the filesystem it was built on.

### No CDN. Not for fonts, not for CSS, not for JS.

A CDN is a runtime dependency on someone else's uptime, a cross-origin
waterfall before first paint, and a hole in the single-binary story. Cache
partitioning (shipped across browsers from 2020) killed the shared-cache
argument that used to justify it. Vendor into `ui/static/` instead.

### `html/template`, never `text/template`

Contextual escaping is what stops user data from becoming stored XSS. The two
packages have identical APIs and one of them is a vulnerability.

### Render into a buffer, then write

A mid-render error must not emit half a page with a `200` already sent. `render()`
executes into a `bytes.Buffer` and only touches the `ResponseWriter` on success.

### `{{block}}`, not `{{template}}`, for optional page blocks

With `{{template "head" .}}` in the layout, every page that omits a `head` block
breaks at render time. `{{block}}` supplies an empty default, so a page with
nothing extra to say is still valid.

### One `<title>`, in `base.html`

Driven by `page.Title`. Do not also set it in a page's `head` block.

### esbuild strips types; it never checks them

A blatant type error bundles clean with exit 0. `make check` (`tsgo --noEmit`)
is the only thing that validates types — run it in CI or a pre-commit hook,
never in the hot dev loop.

### `process.env.NODE_ENV` must be defined at build time

React ships a dev path and a prod path behind that check. Without the esbuild
`define`, the dead-code elimination cannot drop the dev half and you ship both.

---

## Adding a route

Four mechanical steps:

1. **`ui/app/routes/thing.tsx`** — a page component ending in `mount(Thing)`.
2. **`ui/html/pages/thing.html`** — `{{define "main"}}<main></main>{{end}}`.
3. **A props struct** with `json:` tags in `cmd/web/main.go` — skip it if the
   page needs no initial state.
4. **A handler** calling `app.render(w, page{Route: "thing", …})` and a
   `mux.HandleFunc` line.

> **`Route` is the single key** tying all three together: it names the template
> (`html/pages/thing.html`) and the manifest entry (`thing-<hash>.js`). A
> mismatch fails at startup or with a logged 500 — never silently.

---

## Known gaps

Each is small and deliberate, listed so nobody mistakes them for decisions.

| Gap | Shape of the fix |
|---|---|
| **No live reload** — `make dev` rebuilds JS on save, the browser does not refresh | `fsnotify` + an SSE endpoint firing `location.reload()`, ~40 lines |
| **No server timeouts** — `main.go` uses bare `http.ListenAndServe` | an `http.Server` with Read/Write/Idle timeouts |
| **No 404/500 pages** — `http.NotFound` and `http.Error`, plain text | route them through `render()` like any other page |
| **`main.css` is unhashed**, so it revalidates on every page and cannot be `immutable` | route it through the manifest too, once it grows |
| **No CSP** | the inline state block is `type="application/json"` and not executable, so a strict `script-src` is achievable without nonces |

---

<div align="center">
<sub>Deliberately minimal. Every choice points at production.</sub>
</div>
