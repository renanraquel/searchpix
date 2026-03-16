package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"searchpix/internal/model"
	"searchpix/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// LoyaltyLoginRequest corpo do login (fidelização) — tenant identificado pelo usuário/senha
type LoyaltyLoginRequest struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// LoyaltyLoginResponse resposta com token e tenant
type LoyaltyLoginResponse struct {
	Token    string       `json:"token"`
	Tenant   model.Tenant `json:"tenant"`
	ExpiresAt time.Time   `json:"expires_at"`
}

// LoyaltyLoginHandler autentica usuário por tenant (DB)
func LoyaltyLoginHandler(tenantRepo *repository.TenantRepository, userRepo *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
			return
		}
		var req LoyaltyLoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Requisição inválida", http.StatusBadRequest)
			return
		}
		if req.User == "" || req.Password == "" {
			http.Error(w, "user e password são obrigatórios", http.StatusBadRequest)
			return
		}

		user, err := userRepo.GetByUsername(req.User)
		if err != nil || user == nil {
			http.Error(w, "Usuário ou senha inválidos", http.StatusUnauthorized)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			http.Error(w, "Usuário ou senha inválidos", http.StatusUnauthorized)
			return
		}

		tenant, err := tenantRepo.GetByID(user.TenantID)
		if err != nil || tenant == nil {
			http.Error(w, "Usuário ou senha inválidos", http.StatusUnauthorized)
			return
		}

		exp := time.Now().Add(8 * time.Hour)
		claims := jwt.MapClaims{
			"sub":       user.ID,
			"tenant_id": tenant.ID,
			"exp":       exp.Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		secret := []byte(os.Getenv("JWT_SECRET"))
		if len(secret) == 0 {
			secret = []byte("default-secret-change-in-production")
		}
		tokenString, err := token.SignedString(secret)
		if err != nil {
			http.Error(w, "Erro ao gerar token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LoyaltyLoginResponse{
			Token:     tokenString,
			Tenant:    *tenant,
			ExpiresAt: exp,
		})
	}
}

// LoyaltyAuthMiddleware exige JWT com tenant_id (fidelização)
func LoyaltyAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			http.Error(w, "Não autorizado", http.StatusUnauthorized)
			return
		}
		tokenString := authHeader[7:]
		secret := []byte(os.Getenv("JWT_SECRET"))
		if len(secret) == 0 {
			secret = []byte("default-secret-change-in-production")
		}
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return secret, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Token inválido", http.StatusUnauthorized)
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Token inválido", http.StatusUnauthorized)
			return
		}
		sub, _ := claims["sub"].(string)
		tenantID, _ := claims["tenant_id"].(string)
		if sub == "" || tenantID == "" {
			http.Error(w, "Token inválido", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ContextKeyUserID, sub)
		ctx = context.WithValue(ctx, ContextKeyTenantID, tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
