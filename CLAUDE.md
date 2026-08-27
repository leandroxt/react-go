# CLAUDE.md — lit-go (React branch)

Guidance for working in this repository.

---

## What this project is

**Go owns the page, React owns the interactive part.** Go renders the HTML
shell from `html/template` and injects initial state as JSON. React mounts
into it on the client.

The deliverable is **a single Go binary**. No Node process in dev or
production, no third-party origins at runtime.

---

## Layout

```
cmd/
  web/main.go       the server (~50 lines)
  build/main.go     the bundler — esbuild's Go API, in-process
ui/
  efs.go            //go:embed "html" "static"
  html/home.html    the page shell
  app/main.tsx      the React app — SOURCE, never served
  static/
    app.js          BUILD OUTPUT — gitignored, embedded at compile time
    main.css
tsconfig.json       type-checking only (noEmit)
package.json        npm only ever downloads source for esbuild to read
```

`ui/app/` is source; `ui/static/` is output. `//go:embed` picks up
`ui/static/` at compile time, so **the bundle must exist before `go build`** —
that is why `make build` runs the bundler first.

---

## Commands

```sh
make dev      # esbuild in watch mode + server
make build    # bundle, then compile to bin/app
make run      # build, then run
make check    # npx tsgo --noEmit — the only thing that checks types
```

---

## Why no CDN, why esbuild's Go API

- **A CDN is a runtime dependency on someone else's uptime**, a cross-origin
  waterfall, and a hole in the single-binary story. Cache partitioning
  (2020+) killed the "already cached from another site" argument.
- **esbuild ships as a Go package** (`github.com/evanw/esbuild/pkg/api`), so
  the bundler is `go get`-able and runs in-process. `npm` is used exactly
  once, to download React's source into `node_modules/`. Those are inputs to
  esbuild, not a runtime.
- **Bundle cost:** React is ~190 kB minified here, versus ~16 kB for the Lit
  version on `main`. That is the price of the migration; it is the framework,
  not the setup.

---

## Conventions

- **`html/template`, never `text/template`.** Contextual escaping is what
  stops user data becoming stored XSS.
- **Render into a buffer**, then write. A mid-render error must not emit half
  a page with a `200` already sent.
- **Server state reaches React as JSON in the page, not via a fetch.** The
  handler marshals a struct into
  `<script type="application/json" id="state">`, and `main.tsx` reads it
  before mounting.
- **`template.JS` is load-bearing.** Passing the marshalled JSON as a plain
  `string` makes `html/template` re-encode it as a *JS string literal*, so
  `JSON.parse` returns a string instead of an object. `json.Marshal` already
  escapes `<`, `>` and `&`, so a `</script>` payload cannot break out.
- **esbuild strips types; it never checks them.** A blatant type error
  bundles clean with exit 0. Run `make check` in CI or a pre-commit hook,
  never in the hot dev loop.
- **`process.env.NODE_ENV` must be defined at build time.** React ships dev
  and prod paths behind that check; without the define, esbuild cannot drop
  the dev half.

---

## Known gaps

- **No live reload.** `make dev` rebuilds JS on save; the browser does not
  refresh. `fsnotify` + an SSE endpoint firing `location.reload()` is ~40
  lines.
- **No cache busting.** `/static/app.js` has no fingerprint, so a deploy can
  serve stale JS.
- **No server timeouts.** `main.go` uses bare `http.ListenAndServe` for
  minimalism. Production wants an `http.Server` with Read/Write/Idle
  timeouts.
- **One route, one template.** No template cache, no layout/partial split,
  no 500 page — all removed to keep the example minimal. Re-add when a
  second page appears.
- **The module is still named `lit-go`** even though Lit is gone on this
  branch.
