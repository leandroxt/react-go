# CLAUDE.md — react-go

Reference implementation of the architecture. Deliberately minimal, but every
choice points at production.

---

## The architecture in one paragraph

**Go owns the document, React owns `<main>`.** Go serves the shell — `<head>`
with the SEO metadata, site header, footer — and injects the page's initial
state as JSON. React mounts over `<main>` and renders that page in full. Each
route is its own esbuild entry point, so it ships its own bundle plus a shared
chunk. Everything is embedded into **one Go binary**: no Node in dev or
production, no third-party origins at runtime.

The split is at `<main>`, not inside it. Mixing Go-rendered HTML and a small
React widget in the same page body is hard to read; either a page's body is
Go's or it is React's.

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

`ui/app/` is source, `ui/static/` is output. `//go:embed` reads `ui/static/`
at compile time, so **the bundle must exist before `go build`** — hence the
order in `make build`.

---

## Commands

```sh
make dev      # esbuild in watch mode + server
make build    # bundle, then compile to bin/app
make run      # build, then run
make check    # npx tsgo --noEmit — the only thing that checks types
```

---

## Adding a route

Four steps, all mechanical:

1. `ui/app/routes/thing.tsx` — a page component ending in `mount(Thing)`.
2. `ui/html/pages/thing.html` — `{{define "main"}}<main></main>{{end}}`.
3. A props struct with `json:` tags in `cmd/web/main.go` (skip if the page
   needs no initial state).
4. A handler calling `app.render(w, page{Route: "thing", …})` and a
   `mux.HandleFunc` line.

**`Route` is the single key** tying the three together: it names the template
(`html/pages/thing.html`) and the manifest entry (`thing-<hash>.js`). A
mismatch fails at startup or with a logged 500, never silently.

---

## Per-route bundles

`cmd/build` globs `ui/app/routes/*.tsx` as entry points with `Splitting: true`.
Anything reachable from more than one entry — React, `mount`, `shared` — is
hoisted into a chunk every route imports.

Measured here: shared chunk 190 kb, `home` 789 b, `about` 611 b. A new route
costs its own code, not another copy of React.

Output names are content-hashed and esbuild's metafile is turned into
`static/build/manifest.json` (`route -> /static/build/route-HASH.js`), which Go
reads at startup. That manifest is what lets `base.html` emit the right
`<script>` per route, and hashing doubles as cache busting.

**`/static/build/` is served `Cache-Control: immutable`.** Without it the
shared chunk is refetched on every navigation, which cancels most of the point
of splitting — verified: transferSize went from 194847 to 0 on the second
route. It is only safe because the names are hashed; never apply it to
unhashed assets like `main.css`.

---

## State and shared context

One `<script type="application/json" id="state">` per page carries both:

```json
{ "shared": { "appName": "…", "user": "…" }, "props": { "count": 42 } }
```

- **`props`** — page-specific, typed per route, `null` when the page needs
  none.
- **`shared`** — injected into every page and provided through React context
  via `useShared()`.

Pages are separate documents, so a navigation tears down the React tree:
`shared` is not client state that survives a page load. It survives because
**Go re-injects it on every render**, which also keeps the server the single
source of truth. If you later need genuinely client-only state to persist
across navigations, that is `sessionStorage`, and it is a separate decision.

The `state` template func is the only place that marshals. Centralising it
means no handler can forget it, and `json.Marshal` escapes `<`, `>` and `&`
so a `</script>` payload cannot break out.

---

## SEO

`<head>` is server-rendered: title, description, and a per-page `{{block
"head"}}` for canonical, OG tags, JSON-LD. That is real value — it is what
search results and social previews read.

**But the body is client-rendered, so its copy is not reliably indexed.**
Google renders JS on a deferred second-wave crawl with no timing guarantee;
Bing, social scrapers and most LLM crawlers largely do not.

The per-route split is what makes this a per-route decision rather than a
global one. A marketing or docs route can render its body from the Go template
and ship no bundle at all; an app route behind auth mounts React over `<main>`
and does not care. Decide per route, in the handler.

---

## Conventions

- **`html/template`, never `text/template`.** Contextual escaping is what
  stops user data becoming stored XSS.
- **Render into a buffer**, then write. A mid-render error must not emit half
  a page with a `200` already sent.
- **`{{block}}`, not `{{template}}`, for optional page blocks.** With
  `{{template "head" .}}` in the layout, every page that omits a `head` block
  breaks at render time. `{{block}}` supplies an empty default.
- **One `<title>`, in `base.html`,** driven by `page.Title`. Do not also set it
  in a page's `head` block.
- **No CDN.** Not for fonts, not for CSS, not for JS. It is a runtime
  dependency on someone else's uptime, a cross-origin waterfall, and a hole in
  the single-binary story. Cache partitioning (2020+) killed the shared-cache
  argument. Vendor into `ui/static/` instead.
- **esbuild strips types; it never checks them.** A blatant type error bundles
  clean with exit 0. Run `make check` in CI or a pre-commit hook, never in the
  hot dev loop.
- **`process.env.NODE_ENV` must be defined at build time.** React ships dev and
  prod paths behind that check; without the define, esbuild cannot drop the dev
  half.

---

## Known gaps

Each is small and deliberate — listed so nobody mistakes them for decisions.

- **No live reload.** `make dev` rebuilds JS on save; the browser does not
  refresh. `fsnotify` + an SSE endpoint firing `location.reload()` is ~40
  lines.
- **No server timeouts.** `main.go` uses bare `http.ListenAndServe`. Production
  wants an `http.Server` with Read/Write/Idle timeouts.
- **No 404/500 pages.** `http.NotFound` and `http.Error`, plain text.
- **`main.css` is unhashed** and global, so it revalidates on every page and
  cannot be `immutable`. Route it through the manifest too when it grows.
- **No CSP.** The inline JSON state block is `type="application/json"`, not
  executable, so a strict script-src is achievable without nonces.
