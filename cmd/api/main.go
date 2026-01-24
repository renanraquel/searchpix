package main

import (
	"log"
	"net/http"

	"searchpix/internal/bb"
	"searchpix/internal/config"
	"searchpix/internal/handler"
	"searchpix/internal/service"

	"github.com/joho/godotenv"
)

func main() {
	tokenCache := &bb.TokenCache{}
	_ = godotenv.Load() // ignora erro em produção
	cfg := config.Load()

	client := bb.NewHTTPClient() // depois vira mTLS
	pixService := service.NewPixService(client, cfg, tokenCache)
	pixHandler := handler.NewPixHandler(pixService)

	//http.HandleFunc("/pix", pixHandler.BuscarPix)
	http.HandleFunc("/pix", enableCORS(pixHandler.BuscarPix))

	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}

func enableCORS(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		h(w, r)
	}
}
