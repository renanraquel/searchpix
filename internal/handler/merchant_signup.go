package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"

	"searchpix/internal/config"
	"searchpix/internal/repository"
	"searchpix/internal/service"
	"searchpix/internal/validation"

	"golang.org/x/crypto/bcrypt"
)

var merchantSlugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// MerchantSignupHandler cadastro público de loja + primeiro usuário (painel).
type MerchantSignupHandler struct {
	tenants     *repository.TenantRepository
	users       *repository.UserRepository
	emailSender *service.EmailSender
	emailCfg    config.EmailConfig
}

func NewMerchantSignupHandler(tenants *repository.TenantRepository, users *repository.UserRepository, emailCfg config.EmailConfig) *MerchantSignupHandler {
	return &MerchantSignupHandler{
		tenants:     tenants,
		users:       users,
		emailSender: service.NewEmailSender(emailCfg),
		emailCfg:    emailCfg,
	}
}

type merchantSignupRequest struct {
	TenantName string `json:"tenant_name"`
	TenantSlug string `json:"tenant_slug"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	FullName   string `json:"full_name"`
	CPF        string `json:"cpf"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
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

func newVerificationToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashToken(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func (h *MerchantSignupHandler) verificationLink(token string) string {
	base := strings.TrimSuffix(strings.TrimSpace(h.emailCfg.PublicUIOrigin), "/")
	if base == "" {
		base = "https://searchpix-ui.onrender.com"
	}
	return fmt.Sprintf("%s/precos/verificar-email?token=%s", base, url.QueryEscape(token))
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
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	cpfNorm := validation.OnlyDigitsCPF(req.CPF)
	phoneNorm := digitsOnlyPhone(req.Phone)

	if req.TenantName == "" || req.TenantSlug == "" || req.Username == "" || req.Password == "" || req.FullName == "" || req.Email == "" {
		http.Error(w, "Preencha nome da loja, identificador (slug), login, senha, nome completo e e-mail.", http.StatusBadRequest)
		return
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		http.Error(w, "E-mail inválido.", http.StatusBadRequest)
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
	if u, _ := h.users.GetByEmail(req.Email); u != nil {
		http.Error(w, "Este e-mail já está em uso.", http.StatusConflict)
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

	user, err := h.users.CreateWithProfile(tenant.ID, req.Username, string(hash), req.FullName, cpfNorm, phoneNorm, req.Email)
	if err != nil {
		_ = h.tenants.DeleteByID(tenant.ID)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := newVerificationToken()
	if err != nil {
		_ = h.tenants.DeleteByID(tenant.ID)
		http.Error(w, "Erro ao gerar validação de e-mail", http.StatusInternalServerError)
		return
	}
	expiresAt := time.Now().Add(24 * time.Hour)
	if err := h.users.CreateEmailVerificationToken(user.ID, hashToken(token), expiresAt); err != nil {
		_ = h.tenants.DeleteByID(tenant.ID)
		http.Error(w, "Erro ao preparar validação de e-mail", http.StatusInternalServerError)
		return
	}

	link := h.verificationLink(token)
	body := fmt.Sprintf(
		"Olá %s,\n\nConfirme seu e-mail para ativar o acesso ao painel da loja %s.\n\nLink de confirmação:\n%s\n\nEste link expira em 24 horas.\n",
		req.FullName,
		req.TenantName,
		link,
	)
	if err := h.emailSender.Send(req.Email, "Confirme seu e-mail - SearchPix", body); err != nil {
		_ = h.tenants.DeleteByID(tenant.ID)
		http.Error(w, "Não foi possível enviar o e-mail de confirmação. Tente novamente.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Cadastro criado. Verifique seu e-mail para ativar o login.",
		"tenant":  tenant,
	})
}

// VerifyEmail GET /api/public/merchant-signup/verify-email?token=...
func (h *MerchantSignupHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		http.Error(w, "Token obrigatório", http.StatusBadRequest)
		return
	}
	userID, err := h.users.ConsumeValidEmailVerificationToken(hashToken(token), time.Now())
	if err != nil {
		http.Error(w, "Erro ao validar token", http.StatusInternalServerError)
		return
	}
	if userID == "" {
		http.Error(w, "Token inválido ou expirado", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "E-mail confirmado com sucesso. Você já pode fazer login.",
	})
}
