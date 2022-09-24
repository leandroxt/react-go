package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var conn *websocket.Conn

func (app *application) requestPrices(w http.ResponseWriter, r *http.Request) {
	tickers := app.models.Spreadsheet.EnrichSpreadsheet("name")
	if err := conn.WriteMessage(1, []byte("Generating...")); err != nil {
		log.Println("Error ")
	}

	go func() {
		if err := conn.WriteMessage(1, tickers); err != nil {
			log.Println("Error ")
		}
	}()

	app.writeJSON(w, http.StatusAccepted, envelope{"data": "requested"}, nil)
}

func (app *application) webSocket(w http.ResponseWriter, r *http.Request) {
	app.upgradeConnection(w, r)
	for {
		// Read message from browser
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		for i := 0; i < 5; i++ {
			responseMessage := []byte(fmt.Sprintf("Server: %s %d", string(msg), i))

			// Write message back to browser
			if err = conn.WriteMessage(msgType, responseMessage); err != nil {
				return
			}

			time.Sleep(500 * time.Millisecond)
		}

	}
}

func (app *application) upgradeConnection(w http.ResponseWriter, r *http.Request) *websocket.Conn {
	var err error
	conn, err = upgrader.Upgrade(w, r, nil)
	if err != nil {
		app.logger.PrintError(err, map[string]string{
			"Message:": err.Error(),
		})

		conn.Close()
	}

	return conn
}
