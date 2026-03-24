package main

import (
	"log"
	"net/http"

	"searchpix/internal/auth"
	"searchpix/internal/bb"
	"searchpix/internal/config"
	"searchpix/internal/db"
	"searchpix/internal/handler"
	"searchpix/internal/seed"
	"searchpix/internal/repository"
	"searchpix/internal/service"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	mux := http.NewServeMux()

	// ---------- Banco de dados (fidelização) ----------
	database, err := db.Open(&cfg.DB)
	if err != nil {
		log.Fatal("Erro ao conectar banco:", err)
	}
	defer database.Close()

	driver := cfg.DB.Driver
	tenantRepo := repository.NewTenantRepository(database, driver)
	userRepo := repository.NewUserRepository(database, driver)
	productRepo := repository.NewProductRepository(database, driver)
	customerRepo := repository.NewCustomerRepository(database, driver)
	pointsRepo := repository.NewPointsTransactionRepository(database, driver)
	redemptionRepo := repository.NewRedemptionRepository(database, driver)

	// Seed: cria tenant ibimassas e usuário ibimassas se o banco estiver vazio (local e produção)
	seed.Run(tenantRepo, userRepo)

	pointsSvc := service.NewLoyaltyPointsService(customerRepo, productRepo, pointsRepo, redemptionRepo)

	tenantHandler := handler.NewTenantHandler(tenantRepo, userRepo)
	productHandler := handler.NewProductHandler(productRepo)
	customerHandler := handler.NewCustomerHandler(customerRepo)
	pointsHandler := handler.NewPointsHandler(customerRepo, pointsSvc)
	redemptionListHandler := handler.NewRedemptionListHandler(redemptionRepo)
	redeemAtCounterHandler := handler.NewRedeemAtCounterHandler(customerRepo, pointsSvc)
	publicRedemption := handler.NewPublicRedemptionHandler(tenantRepo, customerRepo, productRepo, redemptionRepo)
	bootstrapHandler := handler.NewBootstrapHandler(tenantRepo, userRepo)

	// ---------- Rotas públicas (fidelização) ----------
	mux.Handle("/api/tenants", enableCORS(http.HandlerFunc(tenantHandler.List)))
	mux.Handle("/api/tenants/create", enableCORS(auth.LoyaltyAuthMiddleware(http.HandlerFunc(tenantHandler.Create))))
	mux.Handle("/api/tenants/background", enableCORS(auth.LoyaltyAuthMiddleware(http.HandlerFunc(tenantHandler.SetBackground))))
	mux.Handle("/api/auth/login", enableCORS(http.HandlerFunc(auth.LoyaltyLoginHandler(tenantRepo, userRepo))))
	mux.Handle("/api/bootstrap", enableCORS(http.HandlerFunc(bootstrapHandler.Bootstrap)))

	// Public redemption (sem auth) - por tenant slug e cpf
	mux.Handle("/api/public/redemption", enableCORS(http.HandlerFunc(publicRedemption.Get)))
	mux.Handle("/api/public/register", enableCORS(http.HandlerFunc(publicRedemption.RegisterPublic)))
	mux.Handle("/api/public/tenant-background", enableCORS(http.HandlerFunc(publicRedemption.ServeTenantBackground)))
	mux.Handle("/api/public/product-image", enableCORS(http.HandlerFunc(publicRedemption.ServeProductImage)))
	mux.Handle("/api/public/redeem", enableCORS(publicRedemption.RedeemProduct(pointsSvc)))

	// ---------- Rotas protegidas (fidelização) ----------
	mux.Handle("/api/products", enableCORS(auth.LoyaltyAuthMiddleware(http.HandlerFunc(productHandler.List))))
	mux.Handle("/api/products/image", enableCORS(http.HandlerFunc(productHandler.ServeImage)))
	mux.Handle("/api/products/create", enableCORS(auth.LoyaltyAuthMiddleware(http.HandlerFunc(productHandler.Create))))
	mux.Handle("/api/products/update", enableCORS(auth.LoyaltyAuthMiddleware(http.HandlerFunc(productHandler.Update))))
	mux.Handle("/api/products/delete", enableCORS(auth.LoyaltyAuthMiddleware(http.HandlerFunc(productHandler.Delete))))

	mux.Handle("/api/customers", enableCORS(auth.LoyaltyAuthMiddleware(http.HandlerFunc(customerHandler.List))))
	mux.Handle("/api/customers/create", enableCORS(auth.LoyaltyAuthMiddleware(http.HandlerFunc(customerHandler.Create))))
	mux.Handle("/api/customers/update", enableCORS(auth.LoyaltyAuthMiddleware(http.HandlerFunc(customerHandler.Update))))
	mux.Handle("/api/customers/delete", enableCORS(auth.LoyaltyAuthMiddleware(http.HandlerFunc(customerHandler.Delete))))

	mux.Handle("/api/points/customer", enableCORS(auth.LoyaltyAuthMiddleware(http.HandlerFunc(pointsHandler.GetCustomerByCPF))))
	mux.Handle("/api/points/earn", enableCORS(auth.LoyaltyAuthMiddleware(http.HandlerFunc(pointsHandler.Earn))))

	mux.Handle("/api/redemptions", enableCORS(auth.LoyaltyAuthMiddleware(http.HandlerFunc(redemptionListHandler.List))))
	mux.Handle("/api/redemptions/redeem", enableCORS(auth.LoyaltyAuthMiddleware(http.HandlerFunc(redeemAtCounterHandler.Redeem))))

	// ---------- PIX (legado) - só registra se BB configurado ----------
	if cfg.BB.ApiBaseURL != "" && cfg.BB.OAuthURL != "" {
		tokenCache := &bb.TokenCache{}
		client, err := bb.NewHTTPClient()
		if err != nil {
			log.Println("AVISO: BB client não disponível (certificado/env). Rotas /login e /pix desabilitadas:", err)
		} else {
			pixService := service.NewPixService(client, cfg, tokenCache)
			pixHandler := handler.NewPixHandler(pixService)
			mux.Handle("/login", enableCORS(http.HandlerFunc(auth.LoginHandler)))
			mux.Handle("/pix", enableCORS(auth.AuthMiddleware(http.HandlerFunc(pixHandler.BuscarPix))))
			log.Println("Rotas PIX (/login, /pix) registradas")
		}
	}

	log.Println("Servidor rodando na porta", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, mux))
}

func enableCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	})
}
