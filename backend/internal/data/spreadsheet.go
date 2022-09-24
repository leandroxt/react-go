package data

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

type TickersResponse struct {
	Results []struct {
		Symbol             string  `json:"symbol"`
		ShortName          string  `json:"shortName"`
		RegularMarketPrice float64 `json:"regularMarketPrice"` // not best type to hold money
	} `json:"results"`
}

type SpreadsheetModel struct{}

func (m SpreadsheetModel) EnrichSpreadsheet(name string) []byte {
	r, _ := http.Get("https://brapi.dev/api/quote/ABCB4,TRPL4,TAEE11,SANB11,EGIE3")
	data, _ := io.ReadAll(r.Body)

	tickers := TickersResponse{}
	if err := json.Unmarshal(data, &tickers); err != nil {
		log.Println("Error when unmarshal json", err.Error())
	}
	defer r.Body.Close()

	time.Sleep(1 * time.Second) // just testing

	byte, _ := json.Marshal(tickers)
	return byte
}
