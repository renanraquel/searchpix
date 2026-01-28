package main

import (
	"log"
	"net/http"

	"searchpix/internal/auth"
	"searchpix/internal/bb"
	"searchpix/internal/config"
	"searchpix/internal/handler"
	"searchpix/internal/service"

	"github.com/joho/godotenv"
)

func main() {
	// Carrega variáveis de ambiente
	_ = godotenv.Load()
	cfg := config.Load()

	// Dependências do PIX
	tokenCache := &bb.TokenCache{}
	client, err := bb.NewHTTPClient() // já com mTLS
	if err != nil {
		log.Fatal("Erro ao criar HTTP client:", err)
	}

	pixService := service.NewPixService(client, cfg, tokenCache)
	pixHandler := handler.NewPixHandler(pixService)

	// Mux explícito
	mux := http.NewServeMux()

	// ---------- ROTAS ----------
	// Login (público)
	mux.Handle("/login", enableCORS(http.HandlerFunc(auth.LoginHandler)))

	// PIX (protegido)
	mux.Handle(
		"/pix",
		enableCORS(
			auth.AuthMiddleware(http.HandlerFunc(pixHandler.BuscarPix)),
		),
	)

	log.Println("Servidor rodando na porta", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, mux))
}

/* =======================
   CORS
======================= */

func enableCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	})
}
