package main

import (
	"flag"
	"os"
	"strings"

	"github.com/leandroxt/react-go/internal/data"
	"github.com/leandroxt/react-go/internal/jsonlog"
)

type config struct {
	port    int
	limiter struct {
		enabled bool
		rps     float64
		burst   int
	}
	cors struct {
		trustedOrigins []string
	}
}

type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 8080, "API server port")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})

	flag.Parse()

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(),
	}

	err := app.serve()
	if err != nil {
		logger.PrintFatal(err, nil)
	}
}
