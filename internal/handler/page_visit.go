package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"searchpix/internal/repository"
)

type PageVisitHandler struct {
	repo *repository.PageVisitRepository
}

func NewPageVisitHandler(repo *repository.PageVisitRepository) *PageVisitHandler {
	return &PageVisitHandler{repo: repo}
}

type pageVisitRequest struct {
	PageKey    string `json:"page_key"`
	PagePath   string `json:"page_path"`
	Query      string `json:"query"`
	TenantSlug string `json:"tenant_slug"`
}

func firstForwardedIP(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.Split(header, ",")
	return strings.TrimSpace(parts[0])
}

// Create POST /api/public/page-visit
func (h *PageVisitHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	var req pageVisitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}

	pageKey := strings.TrimSpace(req.PageKey)
	pagePath := strings.TrimSpace(req.PagePath)
	if pageKey == "" || pagePath == "" {
		http.Error(w, "page_key e page_path são obrigatórios", http.StatusBadRequest)
		return
	}

	ip := firstForwardedIP(r.Header.Get("X-Forwarded-For"))
	if ip == "" {
		ip = strings.TrimSpace(r.Header.Get("X-Real-IP"))
	}
	referrer := strings.TrimSpace(r.Header.Get("Referer"))
	userAgent := strings.TrimSpace(r.UserAgent())

	if err := h.repo.Create(pageKey, pagePath, req.Query, req.TenantSlug, referrer, userAgent, ip); err != nil {
		http.Error(w, "Erro ao registrar acesso", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}
