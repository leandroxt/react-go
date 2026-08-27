# CLAUDE.md — lit-go (React islands)

Guidance for working in this repository.

---

## What this project is

A study for a non-custodial web3 app: users connect a wallet (MetaMask),
the site builds liquidity-pool positions on their behalf on **Base**, and
charges in USDC via **x402**. Assets are never custodied.

Architecturally: **Go owns the pages, React owns the islands.** The
deliverable is a single Go binary — no Node in dev or production, no
third-party origins at runtime.

---

## The rule that matters: SEO

**Anything that must be indexed is rendered by Go into HTML. React roots
contain only interactive chrome, never SEO-critical text.**

Google renders JS on a deferred second-wave crawl with no timing guarantee;
Bing, social preview scrapers and most LLM crawlers largely do not. So pool
names, stats, copy, FAQ answers → Go templates. Connect button, deposit
modal, forms → islands.

The payoff beyond SEO: **there is no hydration.** `hydrateRoot` demands
markup matching React's output exactly, which Go templates cannot reliably
produce, and mismatches are the most painful part of SSR. Content lives
*outside* the React roots, so `createRoot` on a small empty div is enough.

Corollary: prefer HTML that works without JS. The FAQ uses
`<details>`/`<summary>` rather than a JS accordion precisely so the answers
are in the DOM and indexable.

---

## Layout

```
cmd/
  web/main.go            server, handlers, the `island` template func
  build/main.go          the bundler — esbuild's Go API, in-process
ui/
  efs.go                 //go:embed "html" "static"
  html/
    base.html            layout: head / header / main / footer / script
    partials/            header.html, footer.html
    pages/               one .html per route
  app/
    main.tsx             the island registry — the only thing that mounts React
    wallet.ts            EIP-1193 access + the cross-island store
    islands/             one component per island
  static/
    app.js               BUILD OUTPUT — gitignored, embedded at compile time
    main.css
tsconfig.json            type-checking only (noEmit)
```

`ui/app/` is source; `ui/static/` is output. `//go:embed` reads `ui/static/`
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

## Adding an island

1. Write `ui/app/islands/Thing.tsx` exporting a component whose props are a
   plain JSON-shaped type.
2. Register it in `ui/app/main.tsx`: `'thing': Thing`.
3. Define a props struct in `cmd/web/main.go` with `json:` tags.
4. Mount it from a template: `{{island "thing" .Data.Thing}}`.

The `island` template func emits the mount point and the props together:

```go
<div data-island="thing"><script type="application/json">{…}</script></div>
```

Keeping the encoding in one func is the point — no handler can forget it.
`json.Marshal` escapes `<`, `>` and `&`, so a `</script>` payload cannot
break out.

---

## Conventions

- **One handler per page, each injecting its own state.** Handlers build a
  `page{Title, Description, Wallet, Data}` and call `app.render`.
- **`html/template`, never `text/template`.** Contextual escaping is what
  stops user data becoming stored XSS.
- **Render into a buffer**, then write. A mid-render error must not emit half
  a page with a `200` already sent.
- **`{{block}}`, not `{{template}}`, for optional page blocks.** With
  `{{template "script" .}}` in the layout, every page that omits a `script`
  block breaks at render time. `{{block}}` supplies an empty default.
- **One `<title>`, in `base.html`,** driven by `page.Title`. Do not also set
  it in a page's `head` block.
- **No CDN.** Not for fonts, not for CSS, not for JS. It is a runtime
  dependency on someone else's uptime, a cross-origin waterfall, and a hole
  in the single-binary story. Cache partitioning (2020+) killed the shared-
  cache argument. Vendor it into `ui/static/` instead.
- **esbuild strips types; it never checks them.** A blatant type error
  bundles clean with exit 0. Run `make check` in CI or a pre-commit hook,
  never in the hot dev loop.
- **`process.env.NODE_ENV` must be defined at build time.** React ships dev
  and prod paths behind that check; without the define, esbuild cannot drop
  the dev half.

---

## Web3 notes

- **No wallet library.** For a connect button and a chain switch, `window.
  ethereum` (EIP-1193) directly is a few dozen lines; wagmi + viem would be
  more bundle than behaviour. Revisit when contract calls arrive.
- **Islands are separate React roots, so context cannot cross them.** They do
  share a module graph, so `wallet.ts` holds a module-level store read via
  `useSyncExternalStore`. That is how the header's connect button and a
  page's deposit form stay in sync.
- **The chain is server-owned.** `chainID` lives in Go and reaches the client
  as island props, so the client never hardcodes 8453.

---

## Known gaps

- **x402 is not implemented.** `createPosition` in `DepositForm.tsx` is a
  documented stub that resolves to a fake receipt; the UI labels it as a
  simulation. The real flow — POST, 402 with payment requirements, pay USDC
  on Base, retry with the payment header, sign the built transaction — needs
  the actual x402 header format and real contract addresses.
- **Pools are a hardcoded slice** in `main.go`. No chain reads, no database.
- **No live reload.** `make dev` rebuilds JS on save; the browser does not
  refresh. `fsnotify` + an SSE endpoint firing `location.reload()` is ~40
  lines.
- **No cache busting.** `/static/app.js` has no fingerprint, so a deploy can
  serve stale JS.
- **No server timeouts.** `main.go` uses bare `http.ListenAndServe`.
  Production wants an `http.Server` with Read/Write/Idle timeouts.
- **No 404/500 pages.** `http.NotFound` and `http.Error` plain text.
- **One bundle for all islands.** Fine at this size; consider per-route
  splitting past ~300 kB.
- **The module is still named `lit-go`** even though Lit is gone.
