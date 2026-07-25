package service

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"searchpix/internal/model"
)

var mesesPT = []string{
	"", "JANEIRO", "FEVEREIRO", "MARÇO", "ABRIL", "MAIO", "JUNHO",
	"JULHO", "AGOSTO", "SETEMBRO", "OUTUBRO", "NOVEMBRO", "DEZEMBRO",
}

// WeekBoundsFromDate calcula seg–dom e a quinta da semana que contém a data.
func WeekBoundsFromDate(d time.Time) (inicio, fim, reuniao time.Time) {
	d = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	// Weekday: Sunday=0 ... Saturday=6. Queremos Monday start.
	wd := int(d.Weekday())
	if wd == 0 {
		wd = 7
	}
	inicio = d.AddDate(0, 0, -(wd - 1))
	fim = inicio.AddDate(0, 0, 6)
	reuniao = inicio.AddDate(0, 0, 3) // quinta
	return inicio, fim, reuniao
}

// FormatWeekRotulo gera rótulo tipo "20-26 DE JULHO" ou "27 DE JULHO–2 DE AGOSTO".
func FormatWeekRotulo(inicio, fim time.Time) string {
	if inicio.Month() == fim.Month() && inicio.Year() == fim.Year() {
		return fmt.Sprintf("%d-%d DE %s", inicio.Day(), fim.Day(), mesesPT[int(inicio.Month())])
	}
	return fmt.Sprintf("%d DE %s–%d DE %s",
		inicio.Day(), mesesPT[int(inicio.Month())],
		fim.Day(), mesesPT[int(fim.Month())],
	)
}

// FormatWhatsAppMessage monta a mensagem no modelo combinado.
func FormatWhatsAppMessage(rotulo, titulo string, duracaoMin int, tema, nomePessoa string) string {
	linhaParte := titulo
	if duracaoMin > 0 {
		linhaParte = fmt.Sprintf("%s (%d min)", titulo, duracaoMin)
	}
	if strings.TrimSpace(tema) != "" {
		linhaParte = linhaParte + " " + strings.TrimSpace(tema)
	}
	saudacao := "Boa tarde"
	return fmt.Sprintf(
		"%s. Designação semana %s\n%s\nNome do irmão designado: %s.\nAssim que receber essa mensagem me confirme por favor. Obrigado",
		saudacao, rotulo, linhaParte, nomePessoa,
	)
}

// NomeToUpper força maiúsculas pt-BR.
func NomeToUpper(s string) string {
	return strings.Map(func(r rune) rune {
		return unicode.ToUpper(r)
	}, s)
}

// PessoaElegivelParaParte aplica regras de elegibilidade por tipo de parte.
func PessoaElegivelParaParte(p model.DesigPessoa, tipoCodigo, papel string, donoSexo string) (bool, string) {
	if !p.Ativo {
		return false, "Pessoa inativa"
	}
	switch tipoCodigo {
	case "oracao_inicial":
		if p.Tipo != "servo" && p.Tipo != "anciao" {
			return false, "Apenas servos e anciãos"
		}
		if !p.DisponivelOracaoInicial {
			return false, "Não disponível para oração inicial"
		}
		if papel != "dono" {
			return false, "Sem ajudante nesta parte"
		}
	case "oracao_final":
		if p.Tipo != "servo" && p.Tipo != "anciao" {
			return false, "Apenas servos e anciãos"
		}
		if papel != "dono" {
			return false, "Sem ajudante nesta parte"
		}
	case "presidente":
		if p.Tipo != "anciao" {
			return false, "Apenas anciãos"
		}
		if !p.QualificadoPresidente {
			return false, "Não qualificado para presidente"
		}
		if p.Capacidade == "limitado" {
			return false, "Capacidade limitada"
		}
		if papel != "dono" {
			return false, "Sem ajudante nesta parte"
		}
	case "tesouros":
		if p.Tipo != "servo" && p.Tipo != "anciao" {
			return false, "Apenas servos e anciãos"
		}
		if !p.QualificadoTesouros {
			return false, "Não qualificado para Tesouros"
		}
		if p.Capacidade == "limitado" {
			return false, "Capacidade limitada"
		}
		if papel != "dono" {
			return false, "Sem ajudante nesta parte"
		}
	case "joias":
		if p.Tipo != "servo" {
			return false, "Apenas servos ministeriais"
		}
		if papel != "dono" {
			return false, "Sem ajudante nesta parte"
		}
	case "leitura_biblia":
		if p.Tipo != "estudante" || p.Sexo != "M" {
			return false, "Apenas estudantes do sexo masculino"
		}
		if papel != "dono" {
			return false, "Sem ajudante nesta parte"
		}
	case "discurso":
		if p.Tipo != "estudante" || p.Sexo != "M" {
			return false, "Apenas estudantes do sexo masculino"
		}
		if papel != "dono" {
			return false, "Sem ajudante nesta parte"
		}
	case "iniciando_conversas", "cultivando_interesse", "explicando_crencas", "fazendo_discipulos":
		if p.Tipo != "estudante" {
			return false, "Apenas estudantes"
		}
		if papel == "ajudante" && donoSexo != "" && p.Sexo != donoSexo {
			return false, "Ajudante deve ser do mesmo sexo do dono da parte"
		}
	case "estudo_biblico":
		if p.Tipo != "anciao" {
			return false, "Apenas anciãos"
		}
		if papel != "dono" {
			return false, "Sem ajudante nesta parte"
		}
	case "vida_crista_extra":
		if p.Tipo != "servo" && p.Tipo != "anciao" {
			return false, "Apenas servos e anciãos"
		}
		if papel != "dono" {
			return false, "Sem ajudante nesta parte"
		}
	default:
		return false, "Tipo de parte desconhecido"
	}
	return true, ""
}
