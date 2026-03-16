package handler

import (
	"encoding/json"
	"net/http"

	"searchpix/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// BootstrapHandler cria o primeiro tenant e usuário quando não existe nenhum
type BootstrapHandler struct {
	tenantRepo *repository.TenantRepository
	userRepo   *repository.UserRepository
}

func NewBootstrapHandler(tenantRepo *repository.TenantRepository, userRepo *repository.UserRepository) *BootstrapHandler {
	return &BootstrapHandler{tenantRepo: tenantRepo, userRepo: userRepo}
}

func (h *BootstrapHandler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenants, err := h.tenantRepo.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(tenants) > 0 {
		http.Error(w, "Bootstrap já realizado. Já existem estabelecimentos cadastrados.", http.StatusBadRequest)
		return
	}
	var req struct {
		TenantName string `json:"tenant_name"`
		TenantSlug string `json:"tenant_slug"`
		Username   string `json:"username"`
		Password   string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}
	if req.TenantName == "" || req.TenantSlug == "" || req.Username == "" || req.Password == "" {
		http.Error(w, "tenant_name, tenant_slug, username e password são obrigatórios", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 6 {
		http.Error(w, "Senha deve ter no mínimo 6 caracteres", http.StatusBadRequest)
		return
	}
	tenant, err := h.tenantRepo.Create(req.TenantName, req.TenantSlug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Erro ao gerar senha", http.StatusInternalServerError)
		return
	}
	_, err = h.userRepo.Create(tenant.ID, req.Username, string(hash))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Estabelecimento e usuário criados. Use tenant_slug e credenciais para fazer login.",
		"tenant":  tenant,
	})
}
