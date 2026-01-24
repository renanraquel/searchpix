package handler

import (
	"encoding/json"
	"net/http"

	"searchpix/internal/service"
)

type PixHandler struct {
	service *service.PixService
}

func NewPixHandler(service *service.PixService) *PixHandler {
	return &PixHandler{service: service}
}

func (h *PixHandler) BuscarPix(w http.ResponseWriter, r *http.Request) {
	inicio := r.URL.Query().Get("inicio")
	fim := r.URL.Query().Get("fim")

	if inicio == "" || fim == "" {
		http.Error(w, "inicio e fim são obrigatórios", http.StatusBadRequest)
		return
	}

	resp, err := h.service.BuscarPorPeriodo(inicio, fim)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
