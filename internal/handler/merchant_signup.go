package handler

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"searchpix/internal/repository"
	"searchpix/internal/validation"

	"golang.org/x/crypto/bcrypt"
)

var merchantSlugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// MerchantSignupHandler cadastro público de loja + primeiro usuário (painel).
type MerchantSignupHandler struct {
	tenants *repository.TenantRepository
	users   *repository.UserRepository
}

func NewMerchantSignupHandler(tenants *repository.TenantRepository, users *repository.UserRepository) *MerchantSignupHandler {
	return &MerchantSignupHandler{tenants: tenants, users: users}
}

type merchantSignupRequest struct {
	TenantName string `json:"tenant_name"`
	TenantSlug string `json:"tenant_slug"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	FullName   string `json:"full_name"`
	CPF        string `json:"cpf"`
	Phone      string `json:"phone"`
}

func digitsOnlyPhone(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Signup POST /api/public/merchant-signup — sem autenticação.
func (h *MerchantSignupHandler) Signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	var req merchantSignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}
	req.TenantName = strings.TrimSpace(req.TenantName)
	req.TenantSlug = strings.TrimSpace(strings.ToLower(req.TenantSlug))
	req.Username = strings.TrimSpace(req.Username)
	req.FullName = strings.TrimSpace(req.FullName)
	req.Password = strings.TrimSpace(req.Password)
	cpfNorm := validation.OnlyDigitsCPF(req.CPF)
	phoneNorm := digitsOnlyPhone(req.Phone)

	if req.TenantName == "" || req.TenantSlug == "" || req.Username == "" || req.Password == "" || req.FullName == "" {
		http.Error(w, "Preencha nome da loja, identificador (slug), login, senha e seu nome completo.", http.StatusBadRequest)
		return
	}
	if cpfNorm == "" || len(cpfNorm) != 11 || !validation.IsValidCPF(cpfNorm) {
		http.Error(w, "CPF inválido.", http.StatusBadRequest)
		return
	}
	if len(phoneNorm) < 10 {
		http.Error(w, "Celular inválido: informe DDD + número (10 ou 11 dígitos).", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 6 {
		http.Error(w, "Senha deve ter no mínimo 6 caracteres", http.StatusBadRequest)
		return
	}
	if !merchantSlugPattern.MatchString(req.TenantSlug) {
		http.Error(w, "Identificador da loja (slug): use apenas letras minúsculas, números e hífens (ex.: minha-padaria).", http.StatusBadRequest)
		return
	}
	if strings.ContainsAny(req.Username, " \t\r\n") {
		http.Error(w, "Login não pode conter espaços.", http.StatusBadRequest)
		return
	}

	if exist, _ := h.tenants.GetBySlug(req.TenantSlug); exist != nil {
		http.Error(w, "Já existe uma loja com esse identificador (slug). Escolha outro.", http.StatusConflict)
		return
	}
	if u, _ := h.users.GetByUsername(req.Username); u != nil {
		http.Error(w, "Este login já está em uso. Escolha outro.", http.StatusConflict)
		return
	}

	tenant, err := h.tenants.Create(req.TenantName, req.TenantSlug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		_ = h.tenants.DeleteByID(tenant.ID)
		http.Error(w, "Erro ao processar senha", http.StatusInternalServerError)
		return
	}

	_, err = h.users.CreateWithProfile(tenant.ID, req.Username, string(hash), req.FullName, cpfNorm, phoneNorm)
	if err != nil {
		_ = h.tenants.DeleteByID(tenant.ID)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Loja e usuário criados. Faça login com seu login e senha.",
		"tenant":  tenant,
	})
}
