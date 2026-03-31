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

func decodePageVisitRequest(r *http.Request) (pageVisitRequest, error) {
	if r.Method == http.MethodGet {
		q := r.URL.Query()
		return pageVisitRequest{
			PageKey:    q.Get("page_key"),
			PagePath:   q.Get("page_path"),
			Query:      q.Get("query"),
			TenantSlug: q.Get("tenant_slug"),
		}, nil
	}
	var req pageVisitRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

func writePixelNoContent(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.WriteHeader(http.StatusNoContent)
}

// Create POST/GET /api/public/page-visit
func (h *PageVisitHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	req, err := decodePageVisitRequest(r)
	if err != nil {
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

	if r.Method == http.MethodGet {
		writePixelNoContent(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}
