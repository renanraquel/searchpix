package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"searchpix/internal/db"
	"searchpix/internal/model"
)

type DesignacaoRepository struct {
	db     *sql.DB
	driver string
}

func NewDesignacaoRepository(database *sql.DB, driver string) *DesignacaoRepository {
	return &DesignacaoRepository{db: database, driver: driver}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// --- Tipos de parte ---

var defaultTiposParte = []struct {
	Codigo, Categoria, Nome string
	Fixa, PermiteAjudante   bool
	Ordem, Duracao          int
}{
	{"oracao_inicial", "Oração Inicial", "Oração inicial", true, false, 10, 0},
	{"presidente", "Presidente", "Presidente da reunião", true, false, 20, 0},
	{"tesouros", "Tesouros da Palavra de Deus", "Tesouros da Palavra de Deus", true, false, 30, 10},
	{"joias", "Tesouros da Palavra de Deus", "Joias espirituais", true, false, 40, 10},
	{"leitura_biblia", "Tesouros da Palavra de Deus", "Leitura da Bíblia", true, false, 50, 4},
	{"iniciando_conversas", "Faça Seu Melhor no Ministério", "Iniciando conversas", false, true, 60, 3},
	{"cultivando_interesse", "Faça Seu Melhor no Ministério", "Cultivando o interesse", false, true, 70, 4},
	{"explicando_crencas", "Faça Seu Melhor no Ministério", "Explicando suas crenças", false, true, 80, 5},
	{"fazendo_discipulos", "Faça Seu Melhor no Ministério", "Fazendo discípulos", false, true, 90, 5},
	{"discurso", "Faça Seu Melhor no Ministério", "Discurso", false, false, 100, 5},
	{"vida_crista_extra", "Nossa Vida Cristã", "Parte de Nossa Vida Cristã", false, false, 110, 5},
	{"estudo_biblico", "Nossa Vida Cristã", "Estudo bíblico de congregação", true, false, 120, 30},
	{"oracao_final", "Oração Final", "Oração final", true, false, 130, 0},
}

func (r *DesignacaoRepository) SeedTiposParte() error {
	for _, t := range defaultTiposParte {
		var exists string
		qCheck := db.QueryForDriver(`SELECT id FROM desig_tipos_parte WHERE codigo = $1`, r.driver)
		err := r.db.QueryRow(qCheck, t.Codigo).Scan(&exists)
		if err == nil {
			continue
		}
		if err != sql.ErrNoRows {
			return err
		}
		q := db.QueryForDriver(
			`INSERT INTO desig_tipos_parte (id, codigo, categoria, nome, fixa, permite_ajudante, ordem, duracao_padrao_min)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			r.driver,
		)
		_, err = r.db.Exec(q, newUUID(), t.Codigo, t.Categoria, t.Nome, boolToInt(t.Fixa), boolToInt(t.PermiteAjudante), t.Ordem, t.Duracao)
		if err != nil {
			return fmt.Errorf("seed tipo %s: %w", t.Codigo, err)
		}
	}
	return nil
}

func (r *DesignacaoRepository) ListTiposParte() ([]model.DesigTipoParte, error) {
	q := `SELECT id, codigo, categoria, nome, fixa, permite_ajudante, ordem, duracao_padrao_min FROM desig_tipos_parte ORDER BY ordem`
	q = db.QueryForDriver(q, r.driver)
	rows, err := r.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.DesigTipoParte
	for rows.Next() {
		var t model.DesigTipoParte
		var fixa, aj int
		if err := rows.Scan(&t.ID, &t.Codigo, &t.Categoria, &t.Nome, &fixa, &aj, &t.Ordem, &t.DuracaoPadraoMin); err != nil {
			return nil, err
		}
		t.Fixa = fixa != 0
		t.PermiteAjudante = aj != 0
		list = append(list, t)
	}
	return list, rows.Err()
}

func (r *DesignacaoRepository) GetTipoByID(id string) (*model.DesigTipoParte, error) {
	q := db.QueryForDriver(`SELECT id, codigo, categoria, nome, fixa, permite_ajudante, ordem, duracao_padrao_min FROM desig_tipos_parte WHERE id = $1`, r.driver)
	var t model.DesigTipoParte
	var fixa, aj int
	err := r.db.QueryRow(q, id).Scan(&t.ID, &t.Codigo, &t.Categoria, &t.Nome, &fixa, &aj, &t.Ordem, &t.DuracaoPadraoMin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.Fixa = fixa != 0
	t.PermiteAjudante = aj != 0
	return &t, nil
}

func (r *DesignacaoRepository) GetTipoByCodigo(codigo string) (*model.DesigTipoParte, error) {
	q := db.QueryForDriver(`SELECT id, codigo, categoria, nome, fixa, permite_ajudante, ordem, duracao_padrao_min FROM desig_tipos_parte WHERE codigo = $1`, r.driver)
	var t model.DesigTipoParte
	var fixa, aj int
	err := r.db.QueryRow(q, codigo).Scan(&t.ID, &t.Codigo, &t.Categoria, &t.Nome, &fixa, &aj, &t.Ordem, &t.DuracaoPadraoMin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.Fixa = fixa != 0
	t.PermiteAjudante = aj != 0
	return &t, nil
}

// --- Pessoas ---

func scanPessoa(scanner interface {
	Scan(dest ...any) error
}) (*model.DesigPessoa, error) {
	var p model.DesigPessoa
	var tel sql.NullString
	var ativo, qt, doi, qp int
	err := scanner.Scan(
		&p.ID, &p.TenantID, &p.Nome, &p.Tipo, &p.Sexo, &tel, &ativo,
		&qt, &doi, &qp, &p.Capacidade, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if tel.Valid {
		p.Telefone = tel.String
	}
	p.Ativo = ativo != 0
	p.QualificadoTesouros = qt != 0
	p.DisponivelOracaoInicial = doi != 0
	p.QualificadoPresidente = qp != 0
	return &p, nil
}

func (r *DesignacaoRepository) ListPessoas(tenantID string, onlyActive bool) ([]model.DesigPessoa, error) {
	q := `SELECT id, tenant_id, nome, tipo, sexo, telefone, ativo, qualificado_tesouros, disponivel_oracao_inicial, qualificado_presidente, capacidade, created_at, updated_at
	      FROM desig_pessoas WHERE tenant_id = $1`
	if onlyActive {
		q += ` AND ativo = 1`
	}
	q += ` ORDER BY nome`
	q = db.QueryForDriver(q, r.driver)
	rows, err := r.db.Query(q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.DesigPessoa
	for rows.Next() {
		p, err := scanPessoa(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *p)
	}
	return list, rows.Err()
}

func (r *DesignacaoRepository) GetPessoaByID(id string) (*model.DesigPessoa, error) {
	q := db.QueryForDriver(`SELECT id, tenant_id, nome, tipo, sexo, telefone, ativo, qualificado_tesouros, disponivel_oracao_inicial, qualificado_presidente, capacidade, created_at, updated_at FROM desig_pessoas WHERE id = $1`, r.driver)
	p, err := scanPessoa(r.db.QueryRow(q, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (r *DesignacaoRepository) CreatePessoa(p *model.DesigPessoa) (*model.DesigPessoa, error) {
	id := newUUID()
	q := db.QueryForDriver(
		`INSERT INTO desig_pessoas (id, tenant_id, nome, tipo, sexo, telefone, ativo, qualificado_tesouros, disponivel_oracao_inicial, qualificado_presidente, capacidade)
		 VALUES ($1,$2,$3,$4,$5,NULLIF($6,''),$7,$8,$9,$10,$11)
		 RETURNING id, tenant_id, nome, tipo, sexo, telefone, ativo, qualificado_tesouros, disponivel_oracao_inicial, qualificado_presidente, capacidade, created_at, updated_at`,
		r.driver,
	)
	out, err := scanPessoa(r.db.QueryRow(q,
		id, p.TenantID, p.Nome, p.Tipo, p.Sexo, p.Telefone,
		boolToInt(p.Ativo), boolToInt(p.QualificadoTesouros), boolToInt(p.DisponivelOracaoInicial),
		boolToInt(p.QualificadoPresidente), p.Capacidade,
	))
	return out, err
}

func (r *DesignacaoRepository) UpdatePessoa(p *model.DesigPessoa) error {
	q := db.QueryForDriver(
		`UPDATE desig_pessoas SET nome=$1, tipo=$2, sexo=$3, telefone=NULLIF($4,''), ativo=$5,
		 qualificado_tesouros=$6, disponivel_oracao_inicial=$7, qualificado_presidente=$8, capacidade=$9,
		 updated_at=CURRENT_TIMESTAMP
		 WHERE id=$10 AND tenant_id=$11`,
		r.driver,
	)
	res, err := r.db.Exec(q,
		p.Nome, p.Tipo, p.Sexo, p.Telefone, boolToInt(p.Ativo),
		boolToInt(p.QualificadoTesouros), boolToInt(p.DisponivelOracaoInicial),
		boolToInt(p.QualificadoPresidente), p.Capacidade, p.ID, p.TenantID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *DesignacaoRepository) DeletePessoa(tenantID, id string) error {
	q := db.QueryForDriver(`DELETE FROM desig_pessoas WHERE id = $1 AND tenant_id = $2`, r.driver)
	res, err := r.db.Exec(q, id, tenantID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// --- Semanas ---

func (r *DesignacaoRepository) ListSemanas(tenantID string) ([]model.DesigSemana, error) {
	q := db.QueryForDriver(`SELECT id, tenant_id, data_inicio, data_fim, data_reuniao, rotulo, created_at FROM desig_semanas WHERE tenant_id = $1 ORDER BY data_inicio DESC`, r.driver)
	rows, err := r.db.Query(q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.DesigSemana
	for rows.Next() {
		var s model.DesigSemana
		if err := rows.Scan(&s.ID, &s.TenantID, &s.DataInicio, &s.DataFim, &s.DataReuniao, &s.Rotulo, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.DataInicio = normalizeDate(s.DataInicio)
		s.DataFim = normalizeDate(s.DataFim)
		s.DataReuniao = normalizeDate(s.DataReuniao)
		list = append(list, s)
	}
	return list, rows.Err()
}

func normalizeDate(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

func (r *DesignacaoRepository) GetSemanaByID(id string) (*model.DesigSemana, error) {
	q := db.QueryForDriver(`SELECT id, tenant_id, data_inicio, data_fim, data_reuniao, rotulo, created_at FROM desig_semanas WHERE id = $1`, r.driver)
	var s model.DesigSemana
	err := r.db.QueryRow(q, id).Scan(&s.ID, &s.TenantID, &s.DataInicio, &s.DataFim, &s.DataReuniao, &s.Rotulo, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.DataInicio = normalizeDate(s.DataInicio)
	s.DataFim = normalizeDate(s.DataFim)
	s.DataReuniao = normalizeDate(s.DataReuniao)
	return &s, nil
}

func (r *DesignacaoRepository) GetSemanaByInicio(tenantID, dataInicio string) (*model.DesigSemana, error) {
	q := db.QueryForDriver(`SELECT id, tenant_id, data_inicio, data_fim, data_reuniao, rotulo, created_at FROM desig_semanas WHERE tenant_id = $1 AND data_inicio = $2`, r.driver)
	var s model.DesigSemana
	err := r.db.QueryRow(q, tenantID, dataInicio).Scan(&s.ID, &s.TenantID, &s.DataInicio, &s.DataFim, &s.DataReuniao, &s.Rotulo, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.DataInicio = normalizeDate(s.DataInicio)
	s.DataFim = normalizeDate(s.DataFim)
	s.DataReuniao = normalizeDate(s.DataReuniao)
	return &s, nil
}

func (r *DesignacaoRepository) FindSemanaContaining(tenantID, dateYYYYMMDD string) (*model.DesigSemana, error) {
	q := db.QueryForDriver(
		`SELECT id, tenant_id, data_inicio, data_fim, data_reuniao, rotulo, created_at
		 FROM desig_semanas WHERE tenant_id = $1 AND data_inicio <= $2 AND data_fim >= $3`,
		r.driver,
	)
	var s model.DesigSemana
	err := r.db.QueryRow(q, tenantID, dateYYYYMMDD, dateYYYYMMDD).Scan(&s.ID, &s.TenantID, &s.DataInicio, &s.DataFim, &s.DataReuniao, &s.Rotulo, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.DataInicio = normalizeDate(s.DataInicio)
	s.DataFim = normalizeDate(s.DataFim)
	s.DataReuniao = normalizeDate(s.DataReuniao)
	return &s, nil
}

func (r *DesignacaoRepository) CreateSemana(s *model.DesigSemana) (*model.DesigSemana, error) {
	id := newUUID()
	q := db.QueryForDriver(
		`INSERT INTO desig_semanas (id, tenant_id, data_inicio, data_fim, data_reuniao, rotulo)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 RETURNING id, tenant_id, data_inicio, data_fim, data_reuniao, rotulo, created_at`,
		r.driver,
	)
	var out model.DesigSemana
	err := r.db.QueryRow(q, id, s.TenantID, s.DataInicio, s.DataFim, s.DataReuniao, s.Rotulo).
		Scan(&out.ID, &out.TenantID, &out.DataInicio, &out.DataFim, &out.DataReuniao, &out.Rotulo, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	out.DataInicio = normalizeDate(out.DataInicio)
	out.DataFim = normalizeDate(out.DataFim)
	out.DataReuniao = normalizeDate(out.DataReuniao)
	return &out, nil
}

func (r *DesignacaoRepository) DeleteSemana(tenantID, id string) error {
	q := db.QueryForDriver(`DELETE FROM desig_semanas WHERE id = $1 AND tenant_id = $2`, r.driver)
	res, err := r.db.Exec(q, id, tenantID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// --- Partes ---

func (r *DesignacaoRepository) ListPartesBySemana(semanaID string) ([]model.DesigParte, error) {
	q := db.QueryForDriver(
		`SELECT p.id, p.semana_id, p.tipo_parte_id, p.titulo, p.tema, p.duracao_min, p.ordem, p.created_at,
		        t.codigo, t.nome, t.categoria, t.permite_ajudante
		 FROM desig_partes p
		 JOIN desig_tipos_parte t ON t.id = p.tipo_parte_id
		 WHERE p.semana_id = $1
		 ORDER BY p.ordem, p.created_at`,
		r.driver,
	)
	rows, err := r.db.Query(q, semanaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.DesigParte
	for rows.Next() {
		var p model.DesigParte
		var aj int
		if err := rows.Scan(&p.ID, &p.SemanaID, &p.TipoParteID, &p.Titulo, &p.Tema, &p.DuracaoMin, &p.Ordem, &p.CreatedAt,
			&p.TipoCodigo, &p.TipoNome, &p.Categoria, &aj); err != nil {
			return nil, err
		}
		p.PermiteAjudante = aj != 0
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range list {
		desigs, err := r.ListDesignacoesByParte(list[i].ID)
		if err != nil {
			return nil, err
		}
		list[i].Designacoes = desigs
	}
	return list, nil
}

func (r *DesignacaoRepository) GetParteByID(id string) (*model.DesigParte, error) {
	q := db.QueryForDriver(
		`SELECT p.id, p.semana_id, p.tipo_parte_id, p.titulo, p.tema, p.duracao_min, p.ordem, p.created_at,
		        t.codigo, t.nome, t.categoria, t.permite_ajudante
		 FROM desig_partes p
		 JOIN desig_tipos_parte t ON t.id = p.tipo_parte_id
		 WHERE p.id = $1`,
		r.driver,
	)
	var p model.DesigParte
	var aj int
	err := r.db.QueryRow(q, id).Scan(&p.ID, &p.SemanaID, &p.TipoParteID, &p.Titulo, &p.Tema, &p.DuracaoMin, &p.Ordem, &p.CreatedAt,
		&p.TipoCodigo, &p.TipoNome, &p.Categoria, &aj)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.PermiteAjudante = aj != 0
	desigs, err := r.ListDesignacoesByParte(p.ID)
	if err != nil {
		return nil, err
	}
	p.Designacoes = desigs
	return &p, nil
}

func (r *DesignacaoRepository) CreateParte(p *model.DesigParte) (*model.DesigParte, error) {
	id := newUUID()
	q := db.QueryForDriver(
		`INSERT INTO desig_partes (id, semana_id, tipo_parte_id, titulo, tema, duracao_min, ordem)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		r.driver,
	)
	_, err := r.db.Exec(q, id, p.SemanaID, p.TipoParteID, p.Titulo, p.Tema, p.DuracaoMin, p.Ordem)
	if err != nil {
		return nil, err
	}
	return r.GetParteByID(id)
}

func (r *DesignacaoRepository) UpdateParte(p *model.DesigParte) error {
	q := db.QueryForDriver(
		`UPDATE desig_partes SET titulo=$1, tema=$2, duracao_min=$3, ordem=$4 WHERE id=$5`,
		r.driver,
	)
	res, err := r.db.Exec(q, p.Titulo, p.Tema, p.DuracaoMin, p.Ordem, p.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *DesignacaoRepository) DeleteParte(id string) error {
	q := db.QueryForDriver(`DELETE FROM desig_partes WHERE id = $1`, r.driver)
	res, err := r.db.Exec(q, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// --- Designações ---

func (r *DesignacaoRepository) ListDesignacoesByParte(parteID string) ([]model.DesigDesignacao, error) {
	q := db.QueryForDriver(
		`SELECT d.id, d.parte_id, d.pessoa_id, d.papel, d.created_at, p.nome
		 FROM desig_designacoes d
		 JOIN desig_pessoas p ON p.id = d.pessoa_id
		 WHERE d.parte_id = $1
		 ORDER BY CASE d.papel WHEN 'dono' THEN 0 ELSE 1 END`,
		r.driver,
	)
	rows, err := r.db.Query(q, parteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.DesigDesignacao
	for rows.Next() {
		var d model.DesigDesignacao
		if err := rows.Scan(&d.ID, &d.ParteID, &d.PessoaID, &d.Papel, &d.CreatedAt, &d.PessoaNome); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func (r *DesignacaoRepository) UpsertDesignacao(parteID, pessoaID, papel string) (*model.DesigDesignacao, error) {
	// Remove existing for this papel, then insert
	del := db.QueryForDriver(`DELETE FROM desig_designacoes WHERE parte_id = $1 AND papel = $2`, r.driver)
	if _, err := r.db.Exec(del, parteID, papel); err != nil {
		return nil, err
	}
	id := newUUID()
	ins := db.QueryForDriver(
		`INSERT INTO desig_designacoes (id, parte_id, pessoa_id, papel) VALUES ($1,$2,$3,$4)`,
		r.driver,
	)
	if _, err := r.db.Exec(ins, id, parteID, pessoaID, papel); err != nil {
		return nil, err
	}
	q := db.QueryForDriver(
		`SELECT d.id, d.parte_id, d.pessoa_id, d.papel, d.created_at, p.nome
		 FROM desig_designacoes d JOIN desig_pessoas p ON p.id = d.pessoa_id WHERE d.id = $1`,
		r.driver,
	)
	var d model.DesigDesignacao
	err := r.db.QueryRow(q, id).Scan(&d.ID, &d.ParteID, &d.PessoaID, &d.Papel, &d.CreatedAt, &d.PessoaNome)
	return &d, err
}

func (r *DesignacaoRepository) RemoveDesignacao(parteID, papel string) error {
	q := db.QueryForDriver(`DELETE FROM desig_designacoes WHERE parte_id = $1 AND papel = $2`, r.driver)
	_, err := r.db.Exec(q, parteID, papel)
	return err
}

// PessoaJaDesignadaNaSemana verifica se a pessoa já tem alguma designação na semana (exceto na parte ignoreParteID).
func (r *DesignacaoRepository) PessoaJaDesignadaNaSemana(semanaID, pessoaID, ignoreParteID string) (bool, string, error) {
	q := db.QueryForDriver(
		`SELECT p.titulo FROM desig_designacoes d
		 JOIN desig_partes p ON p.id = d.parte_id
		 WHERE p.semana_id = $1 AND d.pessoa_id = $2 AND p.id <> $3
		 LIMIT 1`,
		r.driver,
	)
	var titulo string
	err := r.db.QueryRow(q, semanaID, pessoaID, ignoreParteID).Scan(&titulo)
	if err == sql.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, titulo, nil
}

// ContagemDesignacoesRecentes conta designações da pessoa nas últimas N semanas (por data_inicio < semanaAtual).
func (r *DesignacaoRepository) ContagemDesignacoesRecentes(tenantID, pessoaID, antesDeDataInicio string, semanas int) (int, string, error) {
	q := db.QueryForDriver(
		`SELECT COUNT(*), MAX(s.data_inicio)
		 FROM desig_designacoes d
		 JOIN desig_partes p ON p.id = d.parte_id
		 JOIN desig_semanas s ON s.id = p.semana_id
		 WHERE s.tenant_id = $1 AND d.pessoa_id = $2 AND s.data_inicio < $3
		   AND s.data_inicio >= $4`,
		r.driver,
	)
	antes, err := time.Parse("2006-01-02", antesDeDataInicio)
	if err != nil {
		return 0, "", err
	}
	limite := antes.AddDate(0, 0, -7*semanas).Format("2006-01-02")
	var count int
	var ultima sql.NullString
	err = r.db.QueryRow(q, tenantID, pessoaID, antesDeDataInicio, limite).Scan(&count, &ultima)
	if err != nil {
		return 0, "", err
	}
	ult := ""
	if ultima.Valid {
		ult = normalizeDate(ultima.String)
	}
	return count, ult, nil
}

func (r *DesignacaoRepository) DesignadoNaSemanaAnterior(tenantID, pessoaID, dataInicioAtual string) (bool, error) {
	inicio, err := time.Parse("2006-01-02", dataInicioAtual)
	if err != nil {
		return false, err
	}
	prev := inicio.AddDate(0, 0, -7).Format("2006-01-02")
	q := db.QueryForDriver(
		`SELECT 1 FROM desig_designacoes d
		 JOIN desig_partes p ON p.id = d.parte_id
		 JOIN desig_semanas s ON s.id = p.semana_id
		 WHERE s.tenant_id = $1 AND d.pessoa_id = $2 AND s.data_inicio = $3
		 LIMIT 1`,
		r.driver,
	)
	var one int
	err = r.db.QueryRow(q, tenantID, pessoaID, prev).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}
