package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"

	"github.com/leandroxt/lit-go/ui"
)

// Base mainnet. The client checks this and asks the wallet to switch.
const (
	chainID   = 8453
	chainName = "Base"
)

// ---------------------------------------------------------------------------
// Domain
// ---------------------------------------------------------------------------

// pool is server-owned data. It is rendered into HTML by Go so crawlers can
// read it, and only the parts a user interacts with are handed to an island.
type pool struct {
	ID     string
	Pair   string
	TokenA string
	TokenB string
	FeeBps int
	TVL    string
	APR    string
	About  string
}

var pools = []pool{
	{
		ID: "usdc-weth-005", Pair: "USDC / WETH",
		TokenA: "USDC", TokenB: "WETH", FeeBps: 5,
		TVL: "$12.4M", APR: "14.2%",
		About: "Par de maior volume na Base. Faixa estreita, rebalanceamento frequente.",
	},
	{
		ID: "usdc-cbbtc-030", Pair: "USDC / cbBTC",
		TokenA: "USDC", TokenB: "cbBTC", FeeBps: 30,
		TVL: "$3.8M", APR: "21.7%",
		About: "Exposição a BTC na Base via cbBTC. Volatilidade maior, fee tier maior.",
	},
	{
		ID: "usdc-aero-100", Pair: "USDC / AERO",
		TokenA: "USDC", TokenB: "AERO", FeeBps: 100,
		TVL: "$1.1M", APR: "38.4%",
		About: "Token nativo do Aerodrome. APR alta acompanhada de impermanent loss alta.",
	},
}

func poolByID(id string) (pool, bool) {
	for _, p := range pools {
		if p.ID == id {
			return p, true
		}
	}
	return pool{}, false
}

// ---------------------------------------------------------------------------
// Island props — these are the ONLY structs that cross into React.
// ---------------------------------------------------------------------------

type walletProps struct {
	ChainID   int    `json:"chainId"`
	ChainName string `json:"chainName"`
}

type depositProps struct {
	PoolID     string  `json:"poolId"`
	Pair       string  `json:"pair"`
	TokenA     string  `json:"tokenA"`
	TokenB     string  `json:"tokenB"`
	MinDeposit float64 `json:"minDeposit"`
	ChainID    int     `json:"chainId"`
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

type page struct {
	Title       string
	Description string
	Wallet      walletProps // header island, present on every page
	Data        any         // rendered into HTML by Go — this is what SEO reads
}

type application struct {
	templates map[string]*template.Template
}

// island emits the mount point for a React island plus its props as JSON.
//
// Centralising it here is the point: no handler can forget the encoding, and
// json.Marshal escapes <, > and & so a "</script>" payload cannot break out.
func island(name string, props any) (template.HTML, error) {
	b, err := json.Marshal(props)
	if err != nil {
		return "", fmt.Errorf("island %q: %w", name, err)
	}
	return template.HTML(fmt.Sprintf(
		`<div data-island="%s"><script type="application/json">%s</script></div>`,
		template.HTMLEscapeString(name), b,
	)), nil
}

func (app *application) render(w http.ResponseWriter, name string, p page) {
	ts, ok := app.templates[name]
	if !ok {
		log.Printf("render: no template %q", name)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Buffer first: a failed render must not emit half a page with a 200
	// already sent.
	var buf bytes.Buffer
	if err := ts.ExecuteTemplate(&buf, "base", p); err != nil {
		log.Printf("render %s: %v", name, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

func newTemplates() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	funcs := template.FuncMap{"island": island}

	// ParseFS fails on a pattern matching zero files, so only include the
	// partials glob once there is a partial to parse.
	partials, err := fs.Glob(ui.Files, "html/partials/*.html")
	if err != nil {
		return nil, err
	}

	pages, err := fs.Glob(ui.Files, "html/pages/*.html")
	if err != nil {
		return nil, err
	}

	for _, p := range pages {
		name := filepath.Base(p)

		patterns := append([]string{"html/base.html"}, partials...)
		patterns = append(patterns, p)

		ts, err := template.New(name).Funcs(funcs).ParseFS(ui.Files, patterns...)
		if err != nil {
			return nil, err
		}
		cache[name] = ts
	}

	return cache, nil
}

// ---------------------------------------------------------------------------
// Handlers — one per page, each injecting its own initial state.
// ---------------------------------------------------------------------------

func wallet() walletProps {
	return walletProps{ChainID: chainID, ChainName: chainName}
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	app.render(w, "home.html", page{
		Title:       "Pools de liquidez sem custódia",
		Description: "Monte posições de liquidez na Base direto da sua carteira. Você mantém a custódia dos ativos o tempo todo.",
		Wallet:      wallet(),
		Data:        pools,
	})
}

func (app *application) pool(w http.ResponseWriter, r *http.Request) {
	p, ok := poolByID(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}

	app.render(w, "pool.html", page{
		Title:       "Pool " + p.Pair,
		Description: p.About,
		Wallet:      wallet(),
		Data: struct {
			Pool    pool
			Deposit depositProps
		}{
			Pool: p,
			Deposit: depositProps{
				PoolID:     p.ID,
				Pair:       p.Pair,
				TokenA:     p.TokenA,
				TokenB:     p.TokenB,
				MinDeposit: 10,
				ChainID:    chainID,
			},
		},
	})
}

func main() {
	templates, err := newTemplates()
	if err != nil {
		log.Fatalf("templates: %v", err)
	}

	static, err := fs.Sub(ui.Files, "static")
	if err != nil {
		log.Fatal(err)
	}

	app := &application{templates: templates}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	mux.HandleFunc("GET /{$}", app.home)
	mux.HandleFunc("GET /pools/{id}", app.pool)

	log.Println("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", mux))
}
