package model

// DesigTipoParte catálogo de tipos de parte da reunião do meio de semana.
type DesigTipoParte struct {
	ID                string `json:"id"`
	Codigo            string `json:"codigo"`
	Categoria         string `json:"categoria"`
	Nome              string `json:"nome"`
	Fixa              bool   `json:"fixa"`
	PermiteAjudante   bool   `json:"permite_ajudante"`
	Ordem             int    `json:"ordem"`
	DuracaoPadraoMin  int    `json:"duracao_padrao_min"`
}

// DesigPessoa irmão/irmã cadastrado para designações.
type DesigPessoa struct {
	ID                       string   `json:"id"`
	TenantID                 string   `json:"tenant_id"`
	Nome                     string   `json:"nome"`
	Tipo                     string   `json:"tipo"` // estudante | servo | anciao
	Sexo                     string   `json:"sexo"` // M | F
	Telefone                 string   `json:"telefone,omitempty"`
	Ativo                    bool     `json:"ativo"`
	QualificadoTesouros      bool     `json:"qualificado_tesouros"`
	DisponivelOracaoInicial  bool     `json:"disponivel_oracao_inicial"`
	QualificadoPresidente    bool     `json:"qualificado_presidente"`
	Capacidade               string   `json:"capacidade"` // pleno | limitado
	CreatedAt                FlexTime `json:"created_at"`
	UpdatedAt                FlexTime `json:"updated_at"`
}

// DesigSemana semana da designação (seg–dom), reunião na quinta.
type DesigSemana struct {
	ID          string   `json:"id"`
	TenantID    string   `json:"tenant_id"`
	DataInicio  string   `json:"data_inicio"`  // YYYY-MM-DD
	DataFim     string   `json:"data_fim"`     // YYYY-MM-DD
	DataReuniao string   `json:"data_reuniao"` // YYYY-MM-DD (quinta)
	Rotulo      string   `json:"rotulo"`
	CreatedAt   FlexTime `json:"created_at"`
}

// DesigDesignacao atribuição de pessoa a uma parte.
type DesigDesignacao struct {
	ID        string   `json:"id"`
	ParteID   string   `json:"parte_id"`
	PessoaID  string   `json:"pessoa_id"`
	Papel     string   `json:"papel"` // dono | ajudante
	PessoaNome string  `json:"pessoa_nome,omitempty"`
	CreatedAt FlexTime `json:"created_at"`
}

// DesigParte parte da reunião em uma semana.
type DesigParte struct {
	ID           string            `json:"id"`
	SemanaID     string            `json:"semana_id"`
	TipoParteID  string            `json:"tipo_parte_id"`
	Titulo       string            `json:"titulo"`
	Tema         string            `json:"tema"`
	DuracaoMin   int               `json:"duracao_min"`
	Ordem        int               `json:"ordem"`
	TipoCodigo   string            `json:"tipo_codigo,omitempty"`
	TipoNome     string            `json:"tipo_nome,omitempty"`
	Categoria    string            `json:"categoria,omitempty"`
	PermiteAjudante bool           `json:"permite_ajudante"`
	Designacoes  []DesigDesignacao `json:"designacoes,omitempty"`
	CreatedAt    FlexTime          `json:"created_at"`
}

// DesigSemanaDetalhe semana com partes.
type DesigSemanaDetalhe struct {
	DesigSemana
	Partes []DesigParte `json:"partes"`
}

// DesigCandidato pessoa elegível com métricas de rotação.
type DesigCandidato struct {
	DesigPessoa
	Elegivel                   bool   `json:"elegivel"`
	MotivoInelegivel           string `json:"motivo_inelegivel,omitempty"`
	DesignacoesUltimas8Semanas int    `json:"designacoes_ultimas_8_semanas"`
	UltimaSemanaDesignado      string `json:"ultima_semana_designado,omitempty"`
	Alerta                     string `json:"alerta,omitempty"`
	JaDesignadoNaSemana        bool   `json:"ja_designado_na_semana"`
}

// DesigLembreteItem item da tela de lembretes da semana corrente.
type DesigLembreteItem struct {
	ParteID      string `json:"parte_id"`
	Titulo       string `json:"titulo"`
	Tema         string `json:"tema"`
	DuracaoMin   int    `json:"duracao_min"`
	Papel        string `json:"papel"`
	PessoaID     string `json:"pessoa_id"`
	PessoaNome   string `json:"pessoa_nome"`
	Telefone     string `json:"telefone,omitempty"`
	MensagemWhatsApp string `json:"mensagem_whatsapp"`
}

// DesigLembretesResponse resposta da tela de lembretes.
type DesigLembretesResponse struct {
	Semana   *DesigSemana        `json:"semana,omitempty"`
	Itens    []DesigLembreteItem `json:"itens"`
	Mensagem string              `json:"mensagem,omitempty"`
}

// DesigWhatsAppResponse texto pronto para WhatsApp.
type DesigWhatsAppResponse struct {
	Mensagem string `json:"mensagem"`
}

// UserPublic dados do usuário no login (sem senha).
type UserPublic struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}
