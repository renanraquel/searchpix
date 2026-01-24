package model

type PixResponse struct {
	Parametros Parametros `json:"parametros"`
	Pix        []Pix      `json:"pix"`
}

type Parametros struct {
	Inicio    string    `json:"inicio"`
	Fim       string    `json:"fim"`
	Paginacao Paginacao `json:"paginacao"`
}

type Paginacao struct {
	PaginaAtual            int `json:"paginaAtual"`
	ItensPorPagina         int `json:"itensPorPagina"`
	QuantidadeDePaginas    int `json:"quantidadeDePaginas"`
	QuantidadeTotalDeItens int `json:"quantidadeTotalDeItens"`
}

type Pix struct {
	EndToEndID       string           `json:"endToEndId"`
	TxID             string           `json:"txid"`
	Valor            string           `json:"valor"`
	Horario          string           `json:"horario"`
	Pagador          *Pagador         `json:"pagador,omitempty"`
	InfoPagador      string           `json:"infoPagador,omitempty"`
	Chave            string           `json:"chave"`
	Devolucoes       []Devolucao      `json:"devolucoes,omitempty"`
	ComponentesValor ComponentesValor `json:"componentesValor"`
}

type Pagador struct {
	CPF  string `json:"cpf,omitempty"`
	CNPJ string `json:"cnpj,omitempty"`
	Nome string `json:"nome"`
}

type ComponentesValor struct {
	Original ValorOriginal `json:"original"`
}

type ValorOriginal struct {
	Valor string `json:"valor"`
}

type Devolucao struct {
	ID        string           `json:"id"`
	RtrID     string           `json:"rtrId"`
	Valor     string           `json:"valor"`
	Natureza  string           `json:"natureza"`
	Descricao string           `json:"descricao"`
	Horario   HorarioDevolucao `json:"horario"`
	Status    string           `json:"status"`
	Motivo    string           `json:"motivo"`
}

type HorarioDevolucao struct {
	Solicitacao string `json:"solicitacao"`
	Liquidacao  string `json:"liquidacao"`
}
