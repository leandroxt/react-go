package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/stocks/price", app.requestPrices)
	router.HandlerFunc(http.MethodGet, "/ws", app.webSocket)

	return app.metrics(app.recoverPanic(app.enableCORS(app.rateLimit(router))))
}
