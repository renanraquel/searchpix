package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"searchpix/internal/nfcepr"
	"searchpix/internal/repository"
	"searchpix/internal/service"
)

// PublicNFCeHandler V1: acúmulo de pontos a partir da URL do QR da NFC-e (PR) + CPF.
type PublicNFCeHandler struct {
	tenantRepo    *repository.TenantRepository
	customerRepo  *repository.CustomerRepository
	nfceClaimRepo *repository.NfceClaimRepository
	pointsSvc     *service.LoyaltyPointsService
}

func NewPublicNFCeHandler(
	tenantRepo *repository.TenantRepository,
	customerRepo *repository.CustomerRepository,
	nfceClaimRepo *repository.NfceClaimRepository,
	pointsSvc *service.LoyaltyPointsService,
) *PublicNFCeHandler {
	return &PublicNFCeHandler{
		tenantRepo:    tenantRepo,
		customerRepo:  customerRepo,
		nfceClaimRepo: nfceClaimRepo,
		pointsSvc:     pointsSvc,
	}
}

type nfceClaimRequest struct {
	TenantSlug string `json:"tenant_slug"`
	CPF        string `json:"cpf"`
	QRPayload  string `json:"qr_payload"`
}

// ClaimPoints POST /api/public/nfce-points — chave do QR + valor lido na página SEFAZ-PR (sem valor informado pelo cliente).
func (h *PublicNFCeHandler) ClaimPoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	var req nfceClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}
	req.TenantSlug = strings.TrimSpace(req.TenantSlug)
	req.CPF = strings.TrimSpace(req.CPF)
	req.QRPayload = strings.TrimSpace(req.QRPayload)
	if req.TenantSlug == "" || req.CPF == "" || req.QRPayload == "" {
		http.Error(w, "tenant_slug, cpf e qr_payload são obrigatórios", http.StatusBadRequest)
		return
	}
	tenant, err := h.tenantRepo.GetBySlug(req.TenantSlug)
	if err != nil || tenant == nil {
		http.Error(w, "Estabelecimento não encontrado", http.StatusNotFound)
		return
	}
	chave, err := nfcepr.ExtractAccessKeyFromPayload(req.QRPayload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	emitenteChave, err := nfcepr.ExtractEmitterCNPJFromAccessKey(chave)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(tenant.NfceEmitterCNPJ) == "" {
		http.Error(
			w,
			"Pontos por nota fiscal não estão ativos: o estabelecimento precisa cadastrar o CNPJ emissor da NFC-e no painel (Pontos).",
			http.StatusBadRequest,
		)
		return
	}
	tenantCNPJ, err := nfcepr.NormalizeCNPJ14(tenant.NfceEmitterCNPJ)
	if err != nil {
		http.Error(
			w,
			"CNPJ emissor cadastrado para NFC-e é inválido. Corrija no painel (Pontos).",
			http.StatusBadRequest,
		)
		return
	}
	if emitenteChave != tenantCNPJ {
		http.Error(w, "Esta nota fiscal não foi emitida por este estabelecimento.", http.StatusBadRequest)
		return
	}

	customer, err := h.customerRepo.GetByTenantAndCPF(tenant.ID, req.CPF)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if customer == nil {
		http.Error(w, "CPF não cadastrado no programa de fidelidade.", http.StatusNotFound)
		return
	}
	used, err := h.nfceClaimRepo.Exists(tenant.ID, chave)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if used {
		http.Error(w, "Esta nota fiscal já foi utilizada para acumular pontos.", http.StatusConflict)
		return
	}

	pageURL := strings.TrimSpace(req.QRPayload)
	if !strings.Contains(pageURL, "://") {
		pageURL = "https://" + pageURL
	}
	if !nfcepr.IsPRNFCeConsultaURL(pageURL) {
		http.Error(w, "Esta nota não é de uma consulta NFC-e do Paraná (SEFAZ-PR).", http.StatusBadRequest)
		return
	}
	value, err := nfcepr.FetchValorPagarFromConsultaURL(pageURL)
	if err != nil {
		http.Error(w, "Não foi possível obter o valor da nota na SEFAZ. Tente novamente em instantes.", http.StatusBadGateway)
		return
	}
	if value <= 0 || value > 100_000 {
		http.Error(w, "Valor da nota inválido para acúmulo.", http.StatusBadRequest)
		return
	}

	ref := fmt.Sprintf("NFC-e %s — R$ %.2f", chave, value)
	points, err := h.pointsSvc.EarnPointsWithReference(tenant.ID, req.CPF, value, ref)
	if err != nil {
		if err == service.ErrCustomerNotFound {
			http.Error(w, "CPF não cadastrado.", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.nfceClaimRepo.Insert(tenant.ID, chave, customer.ID, value, points); err != nil {
		if repository.IsUniqueViolation(err) {
			_ = h.customerRepo.SubtractPoints(customer.ID, points)
			http.Error(w, "Esta nota fiscal já foi utilizada.", http.StatusConflict)
			return
		}
		_ = h.customerRepo.SubtractPoints(customer.ID, points)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"points_added": points,
		"new_balance":  customer.PointsBalance + points,
		"access_key":   chave,
		"value_reais":  value,
		"message":      "Pontos lançados com sucesso a partir da NFC-e.",
	})
}
