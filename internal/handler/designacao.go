package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"searchpix/internal/auth"
	"searchpix/internal/model"
	"searchpix/internal/repository"
	"searchpix/internal/service"
)

type DesignacaoHandler struct {
	repo *repository.DesignacaoRepository
}

func NewDesignacaoHandler(repo *repository.DesignacaoRepository) *DesignacaoHandler {
	return &DesignacaoHandler{repo: repo}
}

func (h *DesignacaoHandler) tenantID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := auth.TenantIDFromContext(r.Context())
	if id == "" {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return "", false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ListTipos GET /api/desig/tipos
func (h *DesignacaoHandler) ListTipos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	list, err := h.repo.ListTiposParte()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []model.DesigTipoParte{}
	}
	writeJSON(w, http.StatusOK, list)
}

// --- Pessoas ---

func (h *DesignacaoHandler) ListPessoas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	onlyActive := r.URL.Query().Get("ativos") == "1"
	list, err := h.repo.ListPessoas(tenantID, onlyActive)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []model.DesigPessoa{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *DesignacaoHandler) CreatePessoa(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	var req model.DesigPessoa
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}
	if err := validatePessoaInput(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.TenantID = tenantID
	req.Nome = service.NomeToUpper(strings.TrimSpace(req.Nome))
	req.Ativo = true
	if req.Capacidade == "" {
		req.Capacidade = "pleno"
	}
	created, err := h.repo.CreatePessoa(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *DesignacaoHandler) UpdatePessoa(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id é obrigatório", http.StatusBadRequest)
		return
	}
	exist, err := h.repo.GetPessoaByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if exist == nil || exist.TenantID != tenantID {
		http.Error(w, "Pessoa não encontrada", http.StatusNotFound)
		return
	}
	var req model.DesigPessoa
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}
	if err := validatePessoaInput(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.ID = id
	req.TenantID = tenantID
	req.Nome = service.NomeToUpper(strings.TrimSpace(req.Nome))
	if req.Capacidade == "" {
		req.Capacidade = "pleno"
	}
	if err := h.repo.UpdatePessoa(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	updated, _ := h.repo.GetPessoaByID(id)
	writeJSON(w, http.StatusOK, updated)
}

func (h *DesignacaoHandler) DeletePessoa(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id é obrigatório", http.StatusBadRequest)
		return
	}
	if err := h.repo.DeletePessoa(tenantID, id); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Pessoa não encontrada", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

type pessoaValidationError string

func (e pessoaValidationError) Error() string { return string(e) }

func validatePessoaInput(p *model.DesigPessoa) error {
	if strings.TrimSpace(p.Nome) == "" {
		return pessoaValidationError("nome é obrigatório")
	}
	switch p.Tipo {
	case "estudante", "servo", "anciao":
	default:
		return pessoaValidationError("tipo inválido (estudante, servo, anciao)")
	}
	switch p.Sexo {
	case "M", "F":
	default:
		return pessoaValidationError("sexo inválido (M ou F)")
	}
	if p.Capacidade != "" && p.Capacidade != "pleno" && p.Capacidade != "limitado" {
		return pessoaValidationError("capacidade inválida (pleno ou limitado)")
	}
	return nil
}

// --- Semanas ---

func (h *DesignacaoHandler) ListSemanas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	list, err := h.repo.ListSemanas(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []model.DesigSemana{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *DesignacaoHandler) GetSemana(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id é obrigatório", http.StatusBadRequest)
		return
	}
	s, err := h.repo.GetSemanaByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s == nil || s.TenantID != tenantID {
		http.Error(w, "Semana não encontrada", http.StatusNotFound)
		return
	}
	partes, err := h.repo.ListPartesBySemana(s.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if partes == nil {
		partes = []model.DesigParte{}
	}
	writeJSON(w, http.StatusOK, model.DesigSemanaDetalhe{DesigSemana: *s, Partes: partes})
}

func (h *DesignacaoHandler) CreateSemana(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	var req struct {
		Data           string `json:"data"` // qualquer data na semana ou da reunião
		ComPartesFixas bool   `json:"com_partes_fixas"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}
	d, err := time.Parse("2006-01-02", req.Data)
	if err != nil {
		http.Error(w, "data inválida (use YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	inicio, fim, reuniao := service.WeekBoundsFromDate(d)
	dataInicio := inicio.Format("2006-01-02")
	exist, err := h.repo.GetSemanaByInicio(tenantID, dataInicio)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if exist != nil {
		partes, _ := h.repo.ListPartesBySemana(exist.ID)
		if partes == nil {
			partes = []model.DesigParte{}
		}
		writeJSON(w, http.StatusOK, model.DesigSemanaDetalhe{DesigSemana: *exist, Partes: partes})
		return
	}
	s := &model.DesigSemana{
		TenantID:    tenantID,
		DataInicio:  dataInicio,
		DataFim:     fim.Format("2006-01-02"),
		DataReuniao: reuniao.Format("2006-01-02"),
		Rotulo:      service.FormatWeekRotulo(inicio, fim),
	}
	created, err := h.repo.CreateSemana(s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	partes := []model.DesigParte{}
	if req.ComPartesFixas {
		partes, err = h.criarPartesFixas(created.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	writeJSON(w, http.StatusCreated, model.DesigSemanaDetalhe{DesigSemana: *created, Partes: partes})
}

func (h *DesignacaoHandler) criarPartesFixas(semanaID string) ([]model.DesigParte, error) {
	tipos, err := h.repo.ListTiposParte()
	if err != nil {
		return nil, err
	}
	var out []model.DesigParte
	for _, t := range tipos {
		if !t.Fixa {
			continue
		}
		p, err := h.repo.CreateParte(&model.DesigParte{
			SemanaID:    semanaID,
			TipoParteID: t.ID,
			Titulo:      t.Nome,
			Tema:        "",
			DuracaoMin:  t.DuracaoPadraoMin,
			Ordem:       t.Ordem,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, nil
}

func (h *DesignacaoHandler) DeleteSemana(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id é obrigatório", http.StatusBadRequest)
		return
	}
	if err := h.repo.DeleteSemana(tenantID, id); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Semana não encontrada", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// --- Partes ---

func (h *DesignacaoHandler) CreateParte(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	var req struct {
		SemanaID    string `json:"semana_id"`
		TipoParteID string `json:"tipo_parte_id"`
		Titulo      string `json:"titulo"`
		Tema        string `json:"tema"`
		DuracaoMin  int    `json:"duracao_min"`
		Ordem       int    `json:"ordem"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}
	semana, err := h.repo.GetSemanaByID(req.SemanaID)
	if err != nil || semana == nil || semana.TenantID != tenantID {
		http.Error(w, "Semana não encontrada", http.StatusNotFound)
		return
	}
	tipo, err := h.repo.GetTipoByID(req.TipoParteID)
	if err != nil || tipo == nil {
		http.Error(w, "Tipo de parte inválido", http.StatusBadRequest)
		return
	}
	titulo := strings.TrimSpace(req.Titulo)
	if titulo == "" {
		titulo = tipo.Nome
	}
	dur := req.DuracaoMin
	if dur == 0 {
		dur = tipo.DuracaoPadraoMin
	}
	ordem := req.Ordem
	if ordem == 0 {
		ordem = tipo.Ordem
	}
	created, err := h.repo.CreateParte(&model.DesigParte{
		SemanaID:    req.SemanaID,
		TipoParteID: req.TipoParteID,
		Titulo:      titulo,
		Tema:        strings.TrimSpace(req.Tema),
		DuracaoMin:  dur,
		Ordem:       ordem,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *DesignacaoHandler) UpdateParte(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id é obrigatório", http.StatusBadRequest)
		return
	}
	parte, err := h.repo.GetParteByID(id)
	if err != nil || parte == nil {
		http.Error(w, "Parte não encontrada", http.StatusNotFound)
		return
	}
	semana, err := h.repo.GetSemanaByID(parte.SemanaID)
	if err != nil || semana == nil || semana.TenantID != tenantID {
		http.Error(w, "Parte não encontrada", http.StatusNotFound)
		return
	}
	var req struct {
		Titulo     string `json:"titulo"`
		Tema       string `json:"tema"`
		DuracaoMin int    `json:"duracao_min"`
		Ordem      int    `json:"ordem"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Titulo) != "" {
		parte.Titulo = strings.TrimSpace(req.Titulo)
	}
	parte.Tema = strings.TrimSpace(req.Tema)
	parte.DuracaoMin = req.DuracaoMin
	if req.Ordem != 0 {
		parte.Ordem = req.Ordem
	}
	if err := h.repo.UpdateParte(parte); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	updated, _ := h.repo.GetParteByID(id)
	writeJSON(w, http.StatusOK, updated)
}

func (h *DesignacaoHandler) DeleteParte(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	id := r.URL.Query().Get("id")
	parte, err := h.repo.GetParteByID(id)
	if err != nil || parte == nil {
		http.Error(w, "Parte não encontrada", http.StatusNotFound)
		return
	}
	semana, _ := h.repo.GetSemanaByID(parte.SemanaID)
	if semana == nil || semana.TenantID != tenantID {
		http.Error(w, "Parte não encontrada", http.StatusNotFound)
		return
	}
	if err := h.repo.DeleteParte(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// SetDesignacao POST /api/desig/designacoes/set
func (h *DesignacaoHandler) SetDesignacao(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	var req struct {
		ParteID  string `json:"parte_id"`
		PessoaID string `json:"pessoa_id"`
		Papel    string `json:"papel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}
	if req.Papel != "dono" && req.Papel != "ajudante" {
		http.Error(w, "papel deve ser dono ou ajudante", http.StatusBadRequest)
		return
	}
	parte, err := h.repo.GetParteByID(req.ParteID)
	if err != nil || parte == nil {
		http.Error(w, "Parte não encontrada", http.StatusNotFound)
		return
	}
	semana, err := h.repo.GetSemanaByID(parte.SemanaID)
	if err != nil || semana == nil || semana.TenantID != tenantID {
		http.Error(w, "Parte não encontrada", http.StatusNotFound)
		return
	}
	if req.Papel == "ajudante" && !parte.PermiteAjudante {
		http.Error(w, "Esta parte não permite ajudante", http.StatusBadRequest)
		return
	}
	pessoa, err := h.repo.GetPessoaByID(req.PessoaID)
	if err != nil || pessoa == nil || pessoa.TenantID != tenantID {
		http.Error(w, "Pessoa não encontrada", http.StatusNotFound)
		return
	}
	donoSexo := ""
	for _, d := range parte.Designacoes {
		if d.Papel == "dono" {
			dono, _ := h.repo.GetPessoaByID(d.PessoaID)
			if dono != nil {
				donoSexo = dono.Sexo
			}
		}
	}
	if req.Papel == "dono" {
		donoSexo = pessoa.Sexo
	}
	okElig, motivo := service.PessoaElegivelParaParte(*pessoa, parte.TipoCodigo, req.Papel, donoSexo)
	if !okElig {
		http.Error(w, motivo, http.StatusBadRequest)
		return
	}
	ja, titulo, err := h.repo.PessoaJaDesignadaNaSemana(semana.ID, req.PessoaID, req.ParteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if ja {
		http.Error(w, "Já designado nesta semana em: "+titulo, http.StatusBadRequest)
		return
	}
	// Se está designando dono e já havia ajudante de outro sexo, limpar ajudante
	if req.Papel == "dono" {
		for _, d := range parte.Designacoes {
			if d.Papel == "ajudante" {
				aj, _ := h.repo.GetPessoaByID(d.PessoaID)
				if aj != nil && aj.Sexo != pessoa.Sexo {
					_ = h.repo.RemoveDesignacao(req.ParteID, "ajudante")
				}
			}
		}
	}
	desig, err := h.repo.UpsertDesignacao(req.ParteID, req.PessoaID, req.Papel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, desig)
}

func (h *DesignacaoHandler) ClearDesignacao(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	var req struct {
		ParteID string `json:"parte_id"`
		Papel   string `json:"papel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}
	parte, err := h.repo.GetParteByID(req.ParteID)
	if err != nil || parte == nil {
		http.Error(w, "Parte não encontrada", http.StatusNotFound)
		return
	}
	semana, _ := h.repo.GetSemanaByID(parte.SemanaID)
	if semana == nil || semana.TenantID != tenantID {
		http.Error(w, "Parte não encontrada", http.StatusNotFound)
		return
	}
	if err := h.repo.RemoveDesignacao(req.ParteID, req.Papel); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// Candidatos GET /api/desig/candidatos?parte_id=&papel=
func (h *DesignacaoHandler) Candidatos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	parteID := r.URL.Query().Get("parte_id")
	papel := r.URL.Query().Get("papel")
	if papel == "" {
		papel = "dono"
	}
	parte, err := h.repo.GetParteByID(parteID)
	if err != nil || parte == nil {
		http.Error(w, "Parte não encontrada", http.StatusNotFound)
		return
	}
	semana, err := h.repo.GetSemanaByID(parte.SemanaID)
	if err != nil || semana == nil || semana.TenantID != tenantID {
		http.Error(w, "Parte não encontrada", http.StatusNotFound)
		return
	}
	out, err := h.rankCandidatos(tenantID, semana, parte, papel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *DesignacaoHandler) rankCandidatos(tenantID string, semana *model.DesigSemana, parte *model.DesigParte, papel string) ([]model.DesigCandidato, error) {
	pessoas, err := h.repo.ListPessoas(tenantID, true)
	if err != nil {
		return nil, err
	}
	donoSexo := ""
	for _, d := range parte.Designacoes {
		if d.Papel == "dono" {
			dono, _ := h.repo.GetPessoaByID(d.PessoaID)
			if dono != nil {
				donoSexo = dono.Sexo
			}
		}
	}
	var out []model.DesigCandidato
	rotacaoMesmoTipo := parte.TipoCodigo == "presidente" ||
		parte.TipoCodigo == "tesouros" ||
		parte.TipoCodigo == "estudo_biblico"
	for _, p := range pessoas {
		c := model.DesigCandidato{DesigPessoa: p}
		elegivel, motivo := service.PessoaElegivelParaParte(p, parte.TipoCodigo, papel, donoSexo)
		c.Elegivel = elegivel
		c.MotivoInelegivel = motivo
		ja, _, _ := h.repo.PessoaJaDesignadaNaSemana(semana.ID, p.ID, parte.ID)
		c.JaDesignadoNaSemana = ja
		if ja {
			c.Elegivel = false
			c.MotivoInelegivel = "Já designado em outra parte nesta semana"
		}
		if parte.TipoCodigo == "tesouros" && p.Capacidade == "limitado" && c.Elegivel {
			c.SomenteManual = true
		}
		var count int
		var ultima string
		if rotacaoMesmoTipo {
			count, ultima, _ = h.repo.HistoricoTipoParte(
				tenantID, p.ID, parte.TipoCodigo, semana.DataInicio, 7,
			)
		} else {
			count, _, _ = h.repo.ContagemDesignacoesRecentes(tenantID, p.ID, semana.DataInicio, 8)
			ultima, _ = h.repo.UltimaDesignacaoAntes(tenantID, p.ID, semana.DataInicio)
		}
		c.DesignacoesUltimas8Semanas = count
		c.UltimaSemanaDesignado = ultima
		if rotacaoMesmoTipo {
			if ultima == "" {
				c.Alerta = "Nunca fez esta parte — prioridade"
			} else if isSemanaAnterior(ultima, semana.DataInicio) {
				c.Alerta = "Fez esta parte na semana anterior"
			} else {
				c.Alerta = "Última nesta parte: " + formatUltimaRotulo(ultima)
			}
		} else if prev, _ := h.repo.DesignadoNaSemanaAnterior(tenantID, p.ID, semana.DataInicio); prev {
			c.Alerta = "Designado na semana anterior"
		} else if ultima == "" {
			c.Alerta = "Nunca designado — prioridade"
		} else {
			c.Alerta = "Última: " + formatUltimaRotulo(ultima)
		}
		if c.SomenteManual {
			c.Alerta = "Capacidade limitada — designar só por sua decisão"
		}
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Elegivel != b.Elegivel {
			return a.Elegivel
		}
		if a.SomenteManual != b.SomenteManual {
			return !a.SomenteManual
		}
		if a.UltimaSemanaDesignado != b.UltimaSemanaDesignado {
			if a.UltimaSemanaDesignado == "" {
				return true
			}
			if b.UltimaSemanaDesignado == "" {
				return false
			}
			return a.UltimaSemanaDesignado < b.UltimaSemanaDesignado
		}
		if a.DesignacoesUltimas8Semanas != b.DesignacoesUltimas8Semanas {
			return a.DesignacoesUltimas8Semanas < b.DesignacoesUltimas8Semanas
		}
		return a.Nome < b.Nome
	})
	return out, nil
}

func pickAutoCandidato(cands []model.DesigCandidato) *model.DesigPessoa {
	for i := range cands {
		c := &cands[i]
		if c.Elegivel && !c.SomenteManual {
			p := c.DesigPessoa
			return &p
		}
	}
	return nil
}

func isSemanaAnterior(ultima, atual string) bool {
	ultimaData, errUltima := time.Parse("2006-01-02", ultima)
	atualData, errAtual := time.Parse("2006-01-02", atual)
	if errUltima != nil || errAtual != nil {
		return false
	}
	return ultimaData.Equal(atualData.AddDate(0, 0, -7))
}

func formatUltimaRotulo(dataInicio string) string {
	t, err := time.Parse("2006-01-02", dataInicio)
	if err != nil {
		return dataInicio
	}
	meses := []string{"", "jan", "fev", "mar", "abr", "mai", "jun", "jul", "ago", "set", "out", "nov", "dez"}
	return fmt.Sprintf("%02d/%s", t.Day(), meses[int(t.Month())])
}

// Lembretes GET /api/desig/lembretes?date=YYYY-MM-DD
func (h *DesignacaoHandler) Lembretes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		http.Error(w, "date inválida", http.StatusBadRequest)
		return
	}
	semana, err := h.repo.FindSemanaContaining(tenantID, dateStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := model.DesigLembretesResponse{Itens: []model.DesigLembreteItem{}}
	if semana == nil {
		resp.Mensagem = "Nenhuma semana de designação cadastrada para esta data."
		writeJSON(w, http.StatusOK, resp)
		return
	}
	resp.Semana = semana
	partes, err := h.repo.ListPartesBySemana(semana.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, parte := range partes {
		var nomeDono, nomeAjudante string
		for _, d := range parte.Designacoes {
			nome := d.PessoaNome
			if pessoa, _ := h.repo.GetPessoaByID(d.PessoaID); pessoa != nil {
				nome = pessoa.Nome
			}
			if d.Papel == "dono" {
				nomeDono = nome
			} else if d.Papel == "ajudante" {
				nomeAjudante = nome
			}
		}
		msg := service.FormatWhatsAppMessage(semana.Rotulo, parte.Titulo, parte.DuracaoMin, parte.Tema, nomeDono, nomeAjudante)
		for _, d := range parte.Designacoes {
			pessoa, _ := h.repo.GetPessoaByID(d.PessoaID)
			nome := d.PessoaNome
			tel := ""
			if pessoa != nil {
				nome = pessoa.Nome
				tel = pessoa.Telefone
			}
			resp.Itens = append(resp.Itens, model.DesigLembreteItem{
				ParteID:          parte.ID,
				Titulo:           parte.Titulo,
				Tema:             parte.Tema,
				DuracaoMin:       parte.DuracaoMin,
				Papel:            d.Papel,
				PessoaID:         d.PessoaID,
				PessoaNome:       nome,
				Telefone:         tel,
				MensagemWhatsApp: msg,
			})
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// WhatsApp GET /api/desig/whatsapp?parte_id=
// Monta uma única mensagem com dono e, se houver, ajudante.
func (h *DesignacaoHandler) WhatsApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	parteID := r.URL.Query().Get("parte_id")
	parte, err := h.repo.GetParteByID(parteID)
	if err != nil || parte == nil {
		http.Error(w, "Parte não encontrada", http.StatusNotFound)
		return
	}
	semana, _ := h.repo.GetSemanaByID(parte.SemanaID)
	if semana == nil || semana.TenantID != tenantID {
		http.Error(w, "Parte não encontrada", http.StatusNotFound)
		return
	}
	var nomeDono, nomeAjudante, telefone, pessoaNome string
	for _, d := range parte.Designacoes {
		nome := d.PessoaNome
		tel := ""
		if pessoa, _ := h.repo.GetPessoaByID(d.PessoaID); pessoa != nil {
			nome = pessoa.Nome
			tel = pessoa.Telefone
		}
		if d.Papel == "dono" {
			nomeDono = nome
			if telefone == "" {
				telefone = tel
			}
			pessoaNome = nome
		} else if d.Papel == "ajudante" {
			nomeAjudante = nome
			if telefone == "" {
				telefone = tel
			}
			if pessoaNome == "" {
				pessoaNome = nome
			} else {
				pessoaNome = nomeDono + " com " + nomeAjudante
			}
		}
	}
	if nomeDono == "" {
		http.Error(w, "Ninguém designado como dono da parte", http.StatusBadRequest)
		return
	}
	if parte.PermiteAjudante && nomeAjudante == "" {
		http.Error(w, "Designação incompleta: falta o ajudante", http.StatusBadRequest)
		return
	}
	msg := service.FormatWhatsAppMessage(semana.Rotulo, parte.Titulo, parte.DuracaoMin, parte.Tema, nomeDono, nomeAjudante)
	writeJSON(w, http.StatusOK, model.DesigWhatsAppResponse{
		Mensagem:   msg,
		PessoaNome: pessoaNome,
		Telefone:   telefone,
	})
}

// ImportApostila POST /api/desig/apostila/import
// Cola o texto de uma página da apostila, monta a semana e sugere designações.
func (h *DesignacaoHandler) ImportApostila(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	var req struct {
		Texto string `json:"texto"`
		Data  string `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}
	var fallback time.Time
	if strings.TrimSpace(req.Data) != "" {
		d, err := time.Parse("2006-01-02", req.Data)
		if err != nil {
			http.Error(w, "data inválida (use YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		fallback = d
	}
	parsed := service.ParseApostila(req.Texto, fallback)
	if parsed.DataReferencia.IsZero() {
		http.Error(w, strings.Join(parsed.Avisos, " "), http.StatusBadRequest)
		return
	}
	if len(parsed.Partes) == 0 {
		msg := "Não encontrei o programa da semana no texto."
		if len(parsed.Avisos) > 0 {
			msg = strings.Join(parsed.Avisos, " ")
		}
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	inicio, fim, reuniao := service.WeekBoundsFromDate(parsed.DataReferencia)
	dataInicio := inicio.Format("2006-01-02")
	semana, err := h.repo.GetSemanaByInicio(tenantID, dataInicio)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	criou := false
	if semana == nil {
		semana, err = h.repo.CreateSemana(&model.DesigSemana{
			TenantID:    tenantID,
			DataInicio:  dataInicio,
			DataFim:     fim.Format("2006-01-02"),
			DataReuniao: reuniao.Format("2006-01-02"),
			Rotulo:      service.FormatWeekRotulo(inicio, fim),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := h.criarPartesFixas(semana.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		criou = true
	}

	partes, err := h.repo.ListPartesBySemana(semana.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(partes) == 0 {
		partes, err = h.criarPartesFixas(semana.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	avisos := append([]string{}, parsed.Avisos...)
	if err := h.aplicarPartesApostila(semana.ID, partes, parsed.Partes, &avisos); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	partes, err = h.repo.ListPartesBySemana(semana.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	preenchidas, assignAvisos := h.preencherDesignacoesSemana(tenantID, semana, partes)
	avisos = append(avisos, assignAvisos...)

	partes, err = h.repo.ListPartesBySemana(semana.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if partes == nil {
		partes = []model.DesigParte{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"semana":                  model.DesigSemanaDetalhe{DesigSemana: *semana, Partes: partes},
		"avisos":                  avisos,
		"criou_semana":            criou,
		"designacoes_preenchidas": preenchidas,
	})
}

func apostilaTipoRepetivel(codigo string) bool {
	switch codigo {
	case "iniciando_conversas", "cultivando_interesse", "explicando_crencas",
		"fazendo_discipulos", "discurso", "vida_crista_extra":
		return true
	default:
		return false
	}
}

func (h *DesignacaoHandler) aplicarPartesApostila(semanaID string, existentes []model.DesigParte, drafts []service.ApostilaParteDraft, avisos *[]string) error {
	byCodigo := map[string][]*model.DesigParte{}
	used := map[string]int{}
	for i := range existentes {
		c := existentes[i].TipoCodigo
		byCodigo[c] = append(byCodigo[c], &existentes[i])
	}
	for _, d := range drafts {
		tipo, err := h.repo.GetTipoByCodigo(d.Codigo)
		if err != nil || tipo == nil {
			*avisos = append(*avisos, "Tipo não cadastrado: "+d.Codigo)
			continue
		}
		idx := used[d.Codigo]
		lista := byCodigo[d.Codigo]
		if idx < len(lista) {
			p := lista[idx]
			p.Tema = d.Tema
			if d.DuracaoMin > 0 {
				p.DuracaoMin = d.DuracaoMin
			}
			if d.Titulo != "" {
				p.Titulo = d.Titulo
			}
			if err := h.repo.UpdateParte(p); err != nil {
				return err
			}
			used[d.Codigo] = idx + 1
			continue
		}
		if !apostilaTipoRepetivel(d.Codigo) {
			*avisos = append(*avisos, "Não achei a parte fixa “"+tipo.Nome+"” para atualizar.")
			continue
		}
		dur := d.DuracaoMin
		if dur == 0 {
			dur = tipo.DuracaoPadraoMin
		}
		created, err := h.repo.CreateParte(&model.DesigParte{
			SemanaID:    semanaID,
			TipoParteID: tipo.ID,
			Titulo:      d.Titulo,
			Tema:        d.Tema,
			DuracaoMin:  dur,
			Ordem:       tipo.Ordem + idx,
		})
		if err != nil {
			return err
		}
		byCodigo[d.Codigo] = append(byCodigo[d.Codigo], created)
		used[d.Codigo] = idx + 1
	}
	return nil
}

func parteTemPapel(parte model.DesigParte, papel string) bool {
	for _, d := range parte.Designacoes {
		if d.Papel == papel && d.PessoaID != "" {
			return true
		}
	}
	return false
}

func (h *DesignacaoHandler) preencherDesignacoesSemana(tenantID string, semana *model.DesigSemana, partes []model.DesigParte) (int, []string) {
	preenchidas := 0
	var avisos []string
	for _, parte := range partes {
		p := parte
		if !parteTemPapel(p, "dono") {
			n, aviso := h.autoDesignar(&p, semana, tenantID, "dono")
			preenchidas += n
			if aviso != "" {
				avisos = append(avisos, aviso)
			}
		}
		fresh, err := h.repo.GetParteByID(p.ID)
		if err != nil || fresh == nil {
			continue
		}
		p = *fresh
		if p.PermiteAjudante && !parteTemPapel(p, "ajudante") {
			n, aviso := h.autoDesignar(&p, semana, tenantID, "ajudante")
			preenchidas += n
			if aviso != "" {
				avisos = append(avisos, aviso)
			}
		}
	}
	return preenchidas, avisos
}

func (h *DesignacaoHandler) autoDesignar(parte *model.DesigParte, semana *model.DesigSemana, tenantID, papel string) (int, string) {
	cands, err := h.rankCandidatos(tenantID, semana, parte, papel)
	if err != nil {
		return 0, "Falha ao sugerir " + papel + " de “" + parte.Titulo + "”."
	}
	escolha := pickAutoCandidato(cands)
	if escolha == nil {
		return 0, "Sem candidato automático para “" + parte.Titulo + "” (" + papel + "). Preencha na mão."
	}
	if _, err := h.repo.UpsertDesignacao(parte.ID, escolha.ID, papel); err != nil {
		return 0, "Não consegui gravar “" + parte.Titulo + "”: " + err.Error()
	}
	return 1, ""
}
