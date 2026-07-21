package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"searchpix/internal/auth"
	"searchpix/internal/repository"
)

// carouselPublicEnabled controla a tela pública de TV (/carrossel-tv).
// Desligado temporariamente por alto consumo de banda — religar quando houver solução.
const carouselPublicEnabled = false

type CarouselHandler struct {
	repo *repository.CarouselRepository
}

func NewCarouselHandler(repo *repository.CarouselRepository) *CarouselHandler {
	return &CarouselHandler{repo: repo}
}

func writeCarouselPublicDisabled(w http.ResponseWriter) {
	http.Error(w, "Tela do carrossel temporariamente desativada", http.StatusServiceUnavailable)
}

func (h *CarouselHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID := auth.TenantIDFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	list, err := h.repo.ListByTenant(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (h *CarouselHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID := auth.TenantIDFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		http.Error(w, "Erro ao processar formulário", http.StatusBadRequest)
		return
	}
	sortOrder, err := h.repo.NextSortOrder(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	file, header, err := r.FormFile("media")
	if err != nil {
		http.Error(w, "Campo 'media' é obrigatório", http.StatusBadRequest)
		return
	}
	defer file.Close()
	ct := header.Header.Get("Content-Type")
	mediaType, ok := classifyCarouselMedia(ct)
	if !ok {
		http.Error(w, "Envie uma imagem (image/*) ou vídeo (video/*)", http.StatusBadRequest)
		return
	}
	mediaData, err := io.ReadAll(file)
	if err != nil || len(mediaData) == 0 {
		http.Error(w, "Arquivo inválido ou vazio", http.StatusBadRequest)
		return
	}
	item, err := h.repo.Create(tenantID, mediaType, "", sortOrder, mediaData, ct)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(item)
}

func (h *CarouselHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID := auth.TenantIDFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id é obrigatório", http.StatusBadRequest)
		return
	}
	existing, err := h.repo.GetByID(id)
	if err != nil || existing == nil || existing.TenantID != tenantID {
		http.Error(w, "Item não encontrado", http.StatusNotFound)
		return
	}
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		http.Error(w, "Erro ao processar formulário", http.StatusBadRequest)
		return
	}
	sortOrder := existing.SortOrder
	var mediaData []byte
	var contentType, mediaType string
	file, header, err := r.FormFile("media")
	if err == nil {
		defer file.Close()
		ct := header.Header.Get("Content-Type")
		mt, ok := classifyCarouselMedia(ct)
		if !ok {
			http.Error(w, "Envie uma imagem (image/*) ou vídeo (video/*)", http.StatusBadRequest)
			return
		}
		mediaData, err = io.ReadAll(file)
		if err != nil || len(mediaData) == 0 {
			http.Error(w, "Arquivo inválido ou vazio", http.StatusBadRequest)
			return
		}
		contentType = ct
		mediaType = mt
	}
	updated, err := h.repo.Update(id, existing.Title, sortOrder, mediaData, contentType, mediaType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}

func (h *CarouselHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID := auth.TenantIDFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	var req struct {
		ItemIDs []string `json:"item_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}
	if len(req.ItemIDs) == 0 {
		http.Error(w, "item_ids é obrigatório", http.StatusBadRequest)
		return
	}
	list, err := h.repo.ListByTenant(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(req.ItemIDs) != len(list) {
		http.Error(w, "item_ids deve conter todos os itens do carrossel", http.StatusBadRequest)
		return
	}
	known := make(map[string]bool, len(list))
	for _, item := range list {
		known[item.ID] = true
	}
	seen := make(map[string]bool, len(req.ItemIDs))
	for _, id := range req.ItemIDs {
		if !known[id] || seen[id] {
			http.Error(w, "item_ids inválido", http.StatusBadRequest)
			return
		}
		seen[id] = true
	}
	if err := h.repo.Reorder(tenantID, req.ItemIDs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	updated, err := h.repo.ListByTenant(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}

func (h *CarouselHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID := auth.TenantIDFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id é obrigatório", http.StatusBadRequest)
		return
	}
	existing, _ := h.repo.GetByID(id)
	if existing != nil && existing.TenantID != tenantID {
		http.Error(w, "Item não encontrado", http.StatusNotFound)
		return
	}
	if err := h.repo.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CarouselHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID := auth.TenantIDFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	settings, err := h.repo.GetSettings(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(settings)
}

func (h *CarouselHandler) SaveSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID := auth.TenantIDFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	var req struct {
		ImageDurationSeconds int `json:"image_duration_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}
	settings, err := h.repo.UpsertSettings(tenantID, req.ImageDurationSeconds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(settings)
}

// PublicGet lista itens e configurações do carrossel (sem autenticação).
func (h *CarouselHandler) PublicGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	if !carouselPublicEnabled {
		writeCarouselPublicDisabled(w)
		return
	}
	tenantSlug := strings.TrimSpace(r.URL.Query().Get("tenant"))
	if tenantSlug == "" {
		http.Error(w, "tenant é obrigatório", http.StatusBadRequest)
		return
	}
	items, _, err := h.repo.ListByTenantSlug(tenantSlug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	settings, err := h.repo.GetSettingsByTenantSlug(tenantSlug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"items":    items,
		"settings": settings,
	})
}

// PublicServeMedia entrega mídia do carrossel (sem autenticação).
func (h *CarouselHandler) PublicServeMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	if !carouselPublicEnabled {
		writeCarouselPublicDisabled(w)
		return
	}
	id := r.URL.Query().Get("id")
	tenantSlug := strings.TrimSpace(r.URL.Query().Get("tenant"))
	if id == "" || tenantSlug == "" {
		http.Error(w, "id e tenant são obrigatórios", http.StatusBadRequest)
		return
	}
	data, contentType, _, updatedAt, err := h.repo.GetMediaByIDAndTenantSlug(id, tenantSlug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(data) == 0 {
		http.Error(w, "Mídia não encontrada", http.StatusNotFound)
		return
	}
	serveCarouselBytes(w, r, data, contentType, updatedAt)
}

func serveCarouselBytes(w http.ResponseWriter, r *http.Request, data []byte, contentType string, modTime time.Time) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	name := "media.bin"
	switch {
	case strings.HasPrefix(contentType, "video/"):
		name = "media.mp4"
	case strings.Contains(contentType, "png"):
		name = "media.png"
	case strings.HasPrefix(contentType, "image/"):
		name = "media.jpg"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	// ServeContent habilita HTTP Range (206) — necessário para <video> no Safari/iOS.
	if modTime.IsZero() {
		modTime = time.Now()
	}
	http.ServeContent(w, r, name, modTime, bytes.NewReader(data))
}

func classifyCarouselMedia(contentType string) (mediaType string, ok bool) {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if strings.HasPrefix(ct, "image/") {
		return "image", true
	}
	if strings.HasPrefix(ct, "video/") {
		return "video", true
	}
	return "", false
}
