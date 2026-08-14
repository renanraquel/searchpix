package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"searchpix/internal/auth"
	"searchpix/internal/model"
	"searchpix/internal/nfcepr"
	"searchpix/internal/r2"
	"searchpix/internal/repository"
	"searchpix/internal/service"

	"golang.org/x/crypto/bcrypt"
)

// --- Tenants (listar público; criar protegido) ---

type TenantHandler struct {
	repo            *repository.TenantRepository
	nfceEmitterRepo *repository.NfceEmitterRepository
	userRepo        *repository.UserRepository
	r2              *r2.Client
}

func NewTenantHandler(repo *repository.TenantRepository, nfceEmitterRepo *repository.NfceEmitterRepository, userRepo *repository.UserRepository, r2Client *r2.Client) *TenantHandler {
	return &TenantHandler{repo: repo, nfceEmitterRepo: nfceEmitterRepo, userRepo: userRepo, r2: r2Client}
}

func (h *TenantHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	list, err := h.repo.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// Create novo estabelecimento + primeiro usuário (protegido; qualquer usuário logado pode criar)
func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
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
	exist, _ := h.repo.GetBySlug(req.TenantSlug)
	if exist != nil {
		http.Error(w, "Já existe um estabelecimento com esse slug (tenant_slug)", http.StatusBadRequest)
		return
	}
	tenant, err := h.repo.Create(req.TenantName, req.TenantSlug)
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
		"message": "Estabelecimento e usuário criados. Use o usuário e senha para fazer login.",
		"tenant":  tenant,
	})
}

// NfceEmitters gerencia os CNPJs emitentes aceitos para pontuação por NFC-e.
// GET lista, POST adiciona, DELETE remove.
func (h *TenantHandler) NfceEmitters(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.TenantIDFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		list, err := h.nfceEmitterRepo.ListByTenant(tenantID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"emitters": list})
		return
	case http.MethodPost:
		var req struct {
			NfceEmitterCNPJ string `json:"nfce_emitter_cnpj"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON inválido", http.StatusBadRequest)
			return
		}
		cnpj14, err := nfcepr.NormalizeCNPJ14(req.NfceEmitterCNPJ)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := h.nfceEmitterRepo.Add(tenantID, cnpj14); err != nil {
			if repository.IsUniqueViolation(err) {
				http.Error(w, "CNPJ já cadastrado para este estabelecimento.", http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		list, err := h.nfceEmitterRepo.ListByTenant(tenantID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message":  "CNPJ emissor da NFC-e adicionado com sucesso.",
			"emitters": list,
		})
		return
	case http.MethodDelete:
		var req struct {
			NfceEmitterCNPJ string `json:"nfce_emitter_cnpj"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON inválido", http.StatusBadRequest)
			return
		}
		cnpj14, err := nfcepr.NormalizeCNPJ14(req.NfceEmitterCNPJ)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := h.nfceEmitterRepo.Remove(tenantID, cnpj14); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		list, err := h.nfceEmitterRepo.ListByTenant(tenantID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message":  "CNPJ emissor removido.",
			"emitters": list,
		})
		return
	default:
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
}

// SetBackground permite ao estabelecimento logado definir uma imagem de fundo para a tela pública
func (h *TenantHandler) SetBackground(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID := auth.TenantIDFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Erro ao processar formulário", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Campo 'image' é obrigatório", http.StatusBadRequest)
		return
	}
	defer file.Close()
	ct := header.Header.Get("Content-Type")
	if ct == "" || !strings.HasPrefix(ct, "image/") {
		http.Error(w, "Apenas arquivos de imagem são permitidos", http.StatusBadRequest)
		return
	}
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Erro ao ler arquivo", http.StatusInternalServerError)
		return
	}
	if len(data) == 0 {
		http.Error(w, "Imagem vazia", http.StatusBadRequest)
		return
	}
	existing, _ := h.repo.GetByID(tenantID)
	if h.r2 != nil {
		key := r2.TenantBackgroundKey(tenantID, ct)
		sk, url, err := h.r2.UploadWithTimeout(key, data, ct)
		if err != nil {
			http.Error(w, "Falha ao enviar imagem para o R2: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := h.repo.UpdateBackgroundR2(tenantID, sk, url, ct); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if existing != nil && existing.BackgroundStorageKey != "" && existing.BackgroundStorageKey != sk {
			_ = h.r2.DeleteObject(context.Background(), existing.BackgroundStorageKey)
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := h.repo.UpdateBackground(tenantID, data, ct); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Products (protegido, por tenant) ---

type ProductHandler struct {
	repo *repository.ProductRepository
	r2   *r2.Client
}

func NewProductHandler(repo *repository.ProductRepository, r2Client *r2.Client) *ProductHandler {
	return &ProductHandler{repo: repo, r2: r2Client}
}

func isHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func writeStoredImage(w http.ResponseWriter, r *http.Request, publicURL string, data []byte, contentType string) {
	if isHTTPURL(publicURL) {
		http.Redirect(w, r, publicURL, http.StatusFound)
		return
	}
	if len(data) == 0 {
		http.Error(w, "Imagem não encontrada", http.StatusNotFound)
		return
	}
	if contentType == "" {
		contentType = "image/jpeg"
	}
	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}

// ServeImage retorna a imagem do produto (GET /api/products/image?id=xxx).
// Com R2, redireciona para a URL pública.
func (h *ProductHandler) ServeImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id é obrigatório", http.StatusBadRequest)
		return
	}
	meta, err := h.repo.GetImageMetaByProductID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if meta == nil {
		http.Error(w, "Imagem não encontrada", http.StatusNotFound)
		return
	}
	writeStoredImage(w, r, meta.PublicURL, meta.Data, meta.ContentType)
}

func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
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
	json.NewEncoder(w).Encode(list)
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID := auth.TenantIDFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	var description string
	var pointsRequired int
	var imageURL string
	var imageData []byte
	var imageContentType string
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Erro ao processar formulário", http.StatusBadRequest)
			return
		}
		description = strings.TrimSpace(r.FormValue("description"))
		if s := r.FormValue("points_required"); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				pointsRequired = n
			}
		}
		file, header, err := r.FormFile("image")
		if err == nil {
			defer file.Close()
			if ct := header.Header.Get("Content-Type"); ct != "" && strings.HasPrefix(ct, "image/") {
				imageData, _ = io.ReadAll(file)
				imageContentType = ct
			}
		}
	} else {
		var req struct {
			ImageURL       string `json:"image_url"`
			Description    string `json:"description"`
			PointsRequired int    `json:"points_required"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Requisição inválida", http.StatusBadRequest)
			return
		}
		description = req.Description
		pointsRequired = req.PointsRequired
		imageURL = req.ImageURL
	}
	if description == "" || pointsRequired <= 0 {
		http.Error(w, "description e points_required (maior que 0) são obrigatórios", http.StatusBadRequest)
		return
	}
	storageKey := ""
	if len(imageData) > 0 && h.r2 != nil {
		itemID := uuid.New().String()
		key := r2.ProductObjectKey(tenantID, itemID, imageContentType)
		sk, url, err := h.r2.UploadWithTimeout(key, imageData, imageContentType)
		if err != nil {
			http.Error(w, "Falha ao enviar imagem para o R2: "+err.Error(), http.StatusInternalServerError)
			return
		}
		storageKey, imageURL = sk, url
		product, err := h.repo.CreateWithID(itemID, tenantID, imageURL, description, pointsRequired, nil, imageContentType, storageKey)
		if err != nil {
			_ = h.r2.DeleteObject(context.Background(), storageKey)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(product)
		return
	}
	product, err := h.repo.Create(tenantID, imageURL, description, pointsRequired, imageData, imageContentType, storageKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
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
	product, err := h.repo.GetByID(id)
	if err != nil || product == nil || product.TenantID != tenantID {
		http.Error(w, "Produto não encontrado", http.StatusNotFound)
		return
	}
	var description string
	var pointsRequired int
	var imageURL string
	var imageData []byte
	var imageContentType string
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Erro ao processar formulário", http.StatusBadRequest)
			return
		}
		description = strings.TrimSpace(r.FormValue("description"))
		if s := r.FormValue("points_required"); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				pointsRequired = n
			}
		}
		file, header, err := r.FormFile("image")
		if err == nil {
			defer file.Close()
			if ct := header.Header.Get("Content-Type"); ct != "" && strings.HasPrefix(ct, "image/") {
				imageData, _ = io.ReadAll(file)
				imageContentType = ct
			}
		}
		if description == "" {
			description = product.Description
		}
		if pointsRequired <= 0 {
			pointsRequired = product.PointsRequired
		}
	} else {
		var req struct {
			ImageURL       string `json:"image_url"`
			Description    string `json:"description"`
			PointsRequired int    `json:"points_required"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Requisição inválida", http.StatusBadRequest)
			return
		}
		description = req.Description
		pointsRequired = req.PointsRequired
		imageURL = req.ImageURL
	}
	if description == "" || pointsRequired <= 0 {
		http.Error(w, "description e points_required (maior que 0) são obrigatórios", http.StatusBadRequest)
		return
	}
	storageKey := product.StorageKey
	replaceImage := len(imageData) > 0
	if replaceImage && h.r2 != nil {
		key := r2.ProductObjectKey(tenantID, id, imageContentType)
		sk, url, err := h.r2.UploadWithTimeout(key, imageData, imageContentType)
		if err != nil {
			http.Error(w, "Falha ao enviar imagem para o R2: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if product.StorageKey != "" && product.StorageKey != sk {
			_ = h.r2.DeleteObject(context.Background(), product.StorageKey)
		}
		storageKey, imageURL = sk, url
		imageData = nil
	} else if replaceImage && imageURL == "" {
		imageURL = product.ImageURL
	}
	updated, err := h.repo.Update(id, imageURL, description, pointsRequired, imageData, imageContentType, storageKey, replaceImage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
	product, _ := h.repo.GetByID(id)
	if product != nil && product.TenantID != tenantID {
		http.Error(w, "Produto não encontrado", http.StatusNotFound)
		return
	}
	if product != nil && h.r2 != nil && product.StorageKey != "" {
		_ = h.r2.DeleteObject(context.Background(), product.StorageKey)
	}
	if err := h.repo.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Customers (protegido) ---

type CustomerHandler struct {
	repo *repository.CustomerRepository
}

func NewCustomerHandler(repo *repository.CustomerRepository) *CustomerHandler {
	return &CustomerHandler{repo: repo}
}

func (h *CustomerHandler) List(w http.ResponseWriter, r *http.Request) {
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
	json.NewEncoder(w).Encode(list)
}

func (h *CustomerHandler) Create(w http.ResponseWriter, r *http.Request) {
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
		CPF   string `json:"cpf"`
		Name  string `json:"name"`
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}
	if req.CPF == "" || req.Name == "" || req.Phone == "" {
		http.Error(w, "cpf, name e phone são obrigatórios", http.StatusBadRequest)
		return
	}
	customer, err := h.repo.Create(tenantID, req.CPF, req.Name, req.Phone)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(customer)
}

func (h *CustomerHandler) Update(w http.ResponseWriter, r *http.Request) {
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
	customer, _ := h.repo.GetByID(id)
	if customer != nil && customer.TenantID != tenantID {
		http.Error(w, "Cliente não encontrado", http.StatusNotFound)
		return
	}
	var req struct {
		CPF   string `json:"cpf"`
		Name  string `json:"name"`
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}
	if req.CPF == "" || req.Name == "" || req.Phone == "" {
		http.Error(w, "cpf, name e phone são obrigatórios", http.StatusBadRequest)
		return
	}
	updated, err := h.repo.Update(id, req.CPF, req.Name, req.Phone)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (h *CustomerHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
	customer, _ := h.repo.GetByID(id)
	if customer != nil && customer.TenantID != tenantID {
		http.Error(w, "Cliente não encontrado", http.StatusNotFound)
		return
	}
	if err := h.repo.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Redemptions (consulta de resgates - protegido) ---

type RedemptionListHandler struct {
	repo *repository.RedemptionRepository
}

func NewRedemptionListHandler(repo *repository.RedemptionRepository) *RedemptionListHandler {
	return &RedemptionListHandler{repo: repo}
}

func (h *RedemptionListHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID := auth.TenantIDFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	inicio := r.URL.Query().Get("inicio")
	fim := r.URL.Query().Get("fim")
	q := r.URL.Query().Get("q")
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	const pageSize = 8
	offset := (page - 1) * pageSize
	list, total, err := h.repo.ListByTenantFiltered(tenantID, inicio, fim, q, pageSize, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items":       list,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

// --- Efetuar resgate no caixa (protegido) ---

type RedeemAtCounterHandler struct {
	customerRepo *repository.CustomerRepository
	pointsSvc    *service.LoyaltyPointsService
}

func NewRedeemAtCounterHandler(customerRepo *repository.CustomerRepository, pointsSvc *service.LoyaltyPointsService) *RedeemAtCounterHandler {
	return &RedeemAtCounterHandler{customerRepo: customerRepo, pointsSvc: pointsSvc}
}

func (h *RedeemAtCounterHandler) Redeem(w http.ResponseWriter, r *http.Request) {
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
		CPF       string `json:"cpf"`
		ProductID string `json:"product_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}
	cpfNorm := strings.TrimSpace(strings.ReplaceAll(req.CPF, " ", ""))
	if cpfNorm == "" || req.ProductID == "" {
		http.Error(w, "cpf e product_id são obrigatórios", http.StatusBadRequest)
		return
	}
	customer, err := h.customerRepo.GetByTenantAndCPF(tenantID, cpfNorm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if customer == nil {
		http.Error(w, "Cliente não encontrado", http.StatusNotFound)
		return
	}
	redemption, err := h.pointsSvc.Redeem(tenantID, customer.ID, req.ProductID)
	if err != nil {
		if err == service.ErrInsufficientPoints {
			http.Error(w, "Pontos insuficientes para este resgate", http.StatusBadRequest)
			return
		}
		if err == service.ErrProductNotFound {
			http.Error(w, "Produto não encontrado", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"redemption": redemption,
		"message":    "Resgate efetuado com sucesso.",
	})
}

// --- Points (lançar pontos - protegido) ---

type PointsHandler struct {
	customerRepo *repository.CustomerRepository
	pointsSvc    *service.LoyaltyPointsService
}

func NewPointsHandler(customerRepo *repository.CustomerRepository, pointsSvc *service.LoyaltyPointsService) *PointsHandler {
	return &PointsHandler{customerRepo: customerRepo, pointsSvc: pointsSvc}
}

// GetCustomerByCPF retorna cliente por CPF no tenant (para a tela de lançar pontos)
func (h *PointsHandler) GetCustomerByCPF(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID := auth.TenantIDFromContext(r.Context())
	if tenantID == "" {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	cpf := r.URL.Query().Get("cpf")
	if cpf == "" {
		http.Error(w, "cpf é obrigatório", http.StatusBadRequest)
		return
	}
	customer, err := h.customerRepo.GetByTenantAndCPF(tenantID, cpf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if customer == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"found":   false,
			"message": "Cliente não cadastrado. Cadastre o cliente antes de lançar pontos.",
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"found":    true,
		"customer": customer,
	})
}

// Earn lança pontos (valor em R$, 1 real = 1 ponto)
func (h *PointsHandler) Earn(w http.ResponseWriter, r *http.Request) {
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
		CPF        string  `json:"cpf"`
		ValueReais float64 `json:"value_reais"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}
	if req.CPF == "" || req.ValueReais <= 0 {
		http.Error(w, "cpf e value_reais (maior que 0) são obrigatórios", http.StatusBadRequest)
		return
	}
	points, err := h.pointsSvc.EarnPoints(tenantID, req.CPF, req.ValueReais)
	if err != nil {
		if err == service.ErrCustomerNotFound {
			http.Error(w, "Cliente não encontrado. Cadastre o cliente antes de lançar pontos.", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"points_added": points,
		"message":      "Pontos lançados com sucesso.",
	})
}

// --- Public redemption (sem auth, por tenant_slug + cpf) ---

type PublicRedemptionHandler struct {
	tenantRepo     *repository.TenantRepository
	customerRepo   *repository.CustomerRepository
	productRepo    *repository.ProductRepository
	redemptionRepo *repository.RedemptionRepository
}

func NewPublicRedemptionHandler(
	tenantRepo *repository.TenantRepository,
	customerRepo *repository.CustomerRepository,
	productRepo *repository.ProductRepository,
	redemptionRepo *repository.RedemptionRepository,
) *PublicRedemptionHandler {
	return &PublicRedemptionHandler{
		tenantRepo:     tenantRepo,
		customerRepo:   customerRepo,
		productRepo:    productRepo,
		redemptionRepo: redemptionRepo,
	}
}

// RedemptionPublicResponse resposta da tela pública de resgates
type RedemptionPublicResponse struct {
	Tenant      model.Tenant           `json:"tenant"`
	Customer    *model.Customer        `json:"customer,omitempty"` // nil se não encontrado
	Products    []model.Product        `json:"products"`
	Redemptions []model.RedemptionView `json:"redemptions"`
}

func (h *PublicRedemptionHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	slug := r.URL.Query().Get("tenant")
	cpf := r.URL.Query().Get("cpf")
	if slug == "" {
		http.Error(w, "tenant (slug) é obrigatório", http.StatusBadRequest)
		return
	}
	tenant, err := h.tenantRepo.GetBySlug(slug)
	if err != nil || tenant == nil {
		http.Error(w, "Estabelecimento não encontrado", http.StatusNotFound)
		return
	}
	// URL da imagem de fundo: R2 quando já migrada; senão endpoint legado.
	if isHTTPURL(tenant.BackgroundImageURL) {
		// já é URL absoluta
	} else {
		tenant.BackgroundImageURL = "/api/public/tenant-background?tenant=" + slug
	}
	tenant.NfceEmitterCNPJ = "" // não expor CNPJ em endpoint público
	products, _ := h.productRepo.ListByTenant(tenant.ID)
	sort.Slice(products, func(i, j int) bool { return products[i].PointsRequired < products[j].PointsRequired })
	for i := range products {
		if products[i].ImageURL != "" && strings.HasPrefix(products[i].ImageURL, "/api/products/image") {
			products[i].ImageURL = "/api/public/product-image?tenant=" + slug + "&id=" + products[i].ID
		}
	}
	resp := RedemptionPublicResponse{
		Tenant:   *tenant,
		Products: products,
	}
	if cpf != "" {
		customer, _ := h.customerRepo.GetByTenantAndCPF(tenant.ID, cpf)
		resp.Customer = customer
		if customer != nil {
			redemptions, _ := h.redemptionRepo.ListByCustomer(customer.ID)
			resp.Redemptions = redemptions
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ServeProductImage retorna a imagem do produto (público; valida tenant slug + product id)
func (h *PublicRedemptionHandler) ServeProductImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	slug := r.URL.Query().Get("tenant")
	id := r.URL.Query().Get("id")
	if slug == "" || id == "" {
		http.Error(w, "tenant e id são obrigatórios", http.StatusBadRequest)
		return
	}
	tenant, err := h.tenantRepo.GetBySlug(slug)
	if err != nil || tenant == nil {
		http.Error(w, "Estabelecimento não encontrado", http.StatusNotFound)
		return
	}
	meta, err := h.productRepo.GetImageMetaByID(id, tenant.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if meta == nil {
		http.Error(w, "Imagem não encontrada", http.StatusNotFound)
		return
	}
	writeStoredImage(w, r, meta.PublicURL, meta.Data, meta.ContentType)
}

// ServeTenantBackground retorna a imagem de fundo do tenant para a tela pública
func (h *PublicRedemptionHandler) ServeTenantBackground(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	slug := r.URL.Query().Get("tenant")
	if slug == "" {
		http.Error(w, "tenant é obrigatório", http.StatusBadRequest)
		return
	}
	meta, err := h.tenantRepo.GetBackgroundMetaBySlug(slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if meta == nil {
		http.Error(w, "Imagem não encontrada", http.StatusNotFound)
		return
	}
	writeStoredImage(w, r, meta.PublicURL, meta.Data, meta.ContentType)
}

// RedeemProduct resgate por cliente (público - identificado por cpf + tenant)
func (h *PublicRedemptionHandler) RedeemProduct(svc *service.LoyaltyPointsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			TenantSlug string `json:"tenant_slug"`
			CPF        string `json:"cpf"`
			ProductID  string `json:"product_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Requisição inválida", http.StatusBadRequest)
			return
		}
		if req.TenantSlug == "" || req.CPF == "" || req.ProductID == "" {
			http.Error(w, "tenant_slug, cpf e product_id são obrigatórios", http.StatusBadRequest)
			return
		}
		tenant, _ := h.tenantRepo.GetBySlug(req.TenantSlug)
		if tenant == nil {
			http.Error(w, "Estabelecimento não encontrado", http.StatusNotFound)
			return
		}
		customer, _ := h.customerRepo.GetByTenantAndCPF(tenant.ID, req.CPF)
		if customer == nil {
			http.Error(w, "Cliente não encontrado", http.StatusNotFound)
			return
		}
		redemption, err := svc.Redeem(tenant.ID, customer.ID, req.ProductID)
		if err != nil {
			if err == service.ErrInsufficientPoints {
				http.Error(w, "Pontos insuficientes para este resgate", http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(redemption)
	}
}

// RegisterPublic cadastro público no programa de fidelidade (sem auth).
// POST JSON: tenant_slug, name, cpf, phone
func (h *PublicRedemptionHandler) RegisterPublic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		TenantSlug string `json:"tenant_slug"`
		Name       string `json:"name"`
		CPF        string `json:"cpf"`
		Phone      string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}
	slug := strings.TrimSpace(req.TenantSlug)
	name := strings.TrimSpace(req.Name)
	if slug == "" || name == "" {
		http.Error(w, "tenant_slug e name são obrigatórios", http.StatusBadRequest)
		return
	}
	if len(name) < 2 {
		http.Error(w, "Informe o nome completo", http.StatusBadRequest)
		return
	}
	cpfNorm := digitsOnly(req.CPF)
	if len(cpfNorm) != 11 {
		http.Error(w, "CPF deve conter 11 dígitos", http.StatusBadRequest)
		return
	}
	phoneNorm := digitsOnly(req.Phone)
	if len(phoneNorm) < 10 || len(phoneNorm) > 11 {
		http.Error(w, "Telefone inválido (informe DDD + número)", http.StatusBadRequest)
		return
	}
	tenant, err := h.tenantRepo.GetBySlug(slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tenant == nil {
		http.Error(w, "Estabelecimento não encontrado", http.StatusNotFound)
		return
	}
	existing, err := h.customerRepo.GetByTenantAndCPF(tenant.ID, cpfNorm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if existing != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "already_registered",
			"message": "Este CPF já está cadastrado no programa de fidelidade deste estabelecimento.",
		})
		return
	}
	customer, err := h.customerRepo.Create(tenant.ID, cpfNorm, name, phoneNorm)
	if err != nil {
		// Concorrência: unique (tenant_id, cpf)
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "already_registered",
				"message": "Este CPF já está cadastrado no programa de fidelidade deste estabelecimento.",
			})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Cadastro realizado com sucesso.",
		"customer": customer,
	})
}

func digitsOnly(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
