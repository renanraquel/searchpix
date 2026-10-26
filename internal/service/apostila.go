package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ApostilaParteDraft parte extraída da apostila (ainda sem IDs do banco).
type ApostilaParteDraft struct {
	Codigo     string
	Titulo     string
	Tema       string
	DuracaoMin int
}

// ApostilaParseResult resultado do parser de uma página da apostila.
type ApostilaParseResult struct {
	DataReferencia time.Time
	RotuloFonte    string
	Partes         []ApostilaParteDraft
	Avisos         []string
}

var (
	reNumbered        = regexp.MustCompile(`(?i)^\s*(\d{1,2})\.\s+(.+?)\s*$`)
	reDuration        = regexp.MustCompile(`(?i)\((\d+)\s*min`)
	reYear            = regexp.MustCompile(`\b(20\d{2})\b`)
	reRangeSameMonth  = regexp.MustCompile(`(?i)(\d{1,2})(?:\.º|º|o)?\s*[-–—]\s*(\d{1,2})(?:\.º|º|o)?\s+DE\s+([A-ZÇÃÉÍÓÚÂÊÔ]+)\b`)
	reRangeCrossMonth = regexp.MustCompile(`(?i)(\d{1,2})(?:\.º|º|o)?\s+DE\s+([A-ZÇÃÉÍÓÚÂÊÔ]+)\s*[-–—]\s*(\d{1,2})(?:\.º|º|o)?\s+DE\s+([A-ZÇÃÉÍÓÚÂÊÔ]+)\b`)
)

var mesesNome = map[string]time.Month{
	"JANEIRO": 1, "FEVEREIRO": 2, "MARCO": 3, "MARÇO": 3, "ABRIL": 4,
	"MAIO": 5, "JUNHO": 6, "JULHO": 7, "AGOSTO": 8, "SETEMBRO": 9,
	"OUTUBRO": 10, "NOVEMBRO": 11, "DEZEMBRO": 12,
}

type apostilaTipoMatch struct {
	codigo string
	keys   []string
}

var apostilaTipoMatches = []apostilaTipoMatch{
	{codigo: "joias", keys: []string{"joias espirituais"}},
	{codigo: "leitura_biblia", keys: []string{"leitura da biblia"}},
	{codigo: "iniciando_conversas", keys: []string{"iniciando conversas"}},
	{codigo: "cultivando_interesse", keys: []string{"cultivando o interesse"}},
	{codigo: "explicando_crencas", keys: []string{"explicando suas crencas"}},
	{codigo: "fazendo_discipulos", keys: []string{"fazendo discipulos"}},
	{codigo: "discurso", keys: []string{"discurso"}},
	{codigo: "estudo_biblico", keys: []string{"estudo biblico de congregacao", "estudo biblico"}},
	{codigo: "vida_crista_extra", keys: []string{"necessidades locais"}},
	{codigo: "tesouros", keys: []string{"tesouros da palavra de deus"}},
}

func foldPT(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch r {
		case 'Á', 'À', 'Ã', 'Â':
			r = 'A'
		case 'É', 'Ê':
			r = 'E'
		case 'Í':
			r = 'I'
		case 'Ó', 'Ô', 'Õ':
			r = 'O'
		case 'Ú', 'Ü':
			r = 'U'
		case 'Ç':
			r = 'C'
		}
		if unicode.IsSpace(r) {
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(r)
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func monthFromName(name string) time.Month {
	key := foldPT(name)
	if m, ok := mesesNome[key]; ok {
		return m
	}
	if m, ok := mesesNome[strings.ReplaceAll(key, "Ç", "C")]; ok {
		return m
	}
	return 0
}

func detectYear(text string, fallback time.Time) int {
	best := 0
	for _, m := range reYear.FindAllStringSubmatch(text, -1) {
		y, _ := strconv.Atoi(m[1])
		if y >= 2020 && y <= 2100 && y > best {
			best = y
		}
	}
	if best > 0 {
		return best
	}
	if fallback.IsZero() {
		return time.Now().UTC().Year()
	}
	return fallback.Year()
}

func parseApostilaDate(text string, fallback time.Time) (time.Time, string, []string) {
	var avisos []string
	year := detectYear(text, fallback)
	head := text
	if len(head) > 2500 {
		head = head[:2500]
	}
	norm := strings.ToUpper(head)
	norm = strings.NewReplacer("–", "-", "—", "-", "−", "-").Replace(norm)

	if m := reRangeCrossMonth.FindStringSubmatch(norm); len(m) == 5 {
		d1, _ := strconv.Atoi(m[1])
		d2, _ := strconv.Atoi(m[3])
		mo1 := monthFromName(m[2])
		mo2 := monthFromName(m[4])
		if mo1 != 0 && mo2 != 0 && d1 > 0 && d2 > 0 {
			inicio := time.Date(year, mo1, d1, 0, 0, 0, 0, time.UTC)
			return inicio, strings.TrimSpace(m[0]), avisos
		}
	}
	if m := reRangeSameMonth.FindStringSubmatch(norm); len(m) == 4 {
		d1, _ := strconv.Atoi(m[1])
		d2, _ := strconv.Atoi(m[2])
		mo := monthFromName(m[3])
		if mo != 0 && d1 > 0 && d2 > 0 {
			inicio := time.Date(year, mo, d1, 0, 0, 0, 0, time.UTC)
			return inicio, strings.TrimSpace(m[0]), avisos
		}
	}
	if !fallback.IsZero() {
		avisos = append(avisos, "Não achei a data no texto; usei a data informada no formulário.")
		return fallback, "", avisos
	}
	avisos = append(avisos, "Não achei a data da semana no texto. Informe uma data no campo acima.")
	return time.Time{}, "", avisos
}

func currentSection(line string) string {
	f := foldPT(line)
	switch {
	case strings.Contains(f, "TESOUROS DA PALAVRA"):
		return "tesouros"
	case strings.Contains(f, "FACA SEU MELHOR"):
		return "ministerio"
	case strings.Contains(f, "NOSSA VIDA CRISTA"):
		return "vida"
	default:
		return ""
	}
}

func skipLine(line string) bool {
	f := foldPT(line)
	if f == "" || f == "SUA RESPOSTA" {
		return true
	}
	if strings.HasPrefix(f, "CANTICO") || strings.HasPrefix(f, "PARA MEDITAR") {
		return true
	}
	if strings.Contains(f, "COMENTARIOS INICIAIS") || strings.Contains(f, "COMENTARIOS FINAIS") {
		return true
	}
	if strings.Contains(f, "NENHUM VIDEO") || strings.Contains(f, "BIBLIOTECA ON-LINE") {
		return true
	}
	return false
}

func classifyApostilaTitle(title, section string) (codigo, tituloPadrao, temaDoTitulo string, ok bool) {
	f := foldPT(title)
	for _, m := range apostilaTipoMatches {
		for _, key := range m.keys {
			if f == foldPT(key) || strings.HasPrefix(f, foldPT(key)+" ") {
				rest := strings.TrimSpace(title)
				// Se o título é só o nome do tipo, tema vem do bloco.
				if foldPT(rest) == foldPT(key) {
					return m.codigo, "", "", true
				}
				return m.codigo, "", strings.TrimSpace(rest), true
			}
		}
	}
	switch section {
	case "tesouros":
		return "tesouros", "", strings.TrimSpace(title), true
	case "vida":
		return "vida_crista_extra", strings.TrimSpace(title), "", true
	default:
		return "", "", "", false
	}
}

func extractDurationAndTema(block []string) (int, string) {
	joined := strings.Join(block, " ")
	dur := 0
	if m := reDuration.FindStringSubmatch(joined); len(m) == 2 {
		dur, _ = strconv.Atoi(m[1])
	}
	var temaParts []string
	for _, line := range block {
		if skipLine(line) {
			continue
		}
		clean := reDuration.ReplaceAllString(line, " ")
		clean = strings.TrimSpace(clean)
		if clean == "" {
			continue
		}
		f := foldPT(clean)
		if currentSection(clean) != "" {
			continue
		}
		if strings.HasPrefix(f, "JEREMIAS ") || strings.HasPrefix(f, "ISA ") {
			continue
		}
		temaParts = append(temaParts, clean)
	}
	tema := strings.TrimSpace(strings.Join(temaParts, " "))
	tema = strings.Join(strings.Fields(tema), " ")
	if len([]rune(tema)) > 280 {
		tema = string([]rune(tema)[:280])
	}
	return dur, tema
}

func tipoNomePadrao(codigo string) string {
	switch codigo {
	case "tesouros":
		return "Tesouros da Palavra de Deus"
	case "joias":
		return "Joias espirituais"
	case "leitura_biblia":
		return "Leitura da Bíblia"
	case "iniciando_conversas":
		return "Iniciando conversas"
	case "cultivando_interesse":
		return "Cultivando o interesse"
	case "explicando_crencas":
		return "Explicando suas crenças"
	case "fazendo_discipulos":
		return "Fazendo discípulos"
	case "discurso":
		return "Discurso"
	case "vida_crista_extra":
		return "Parte de Nossa Vida Cristã"
	case "estudo_biblico":
		return "Estudo bíblico de congregação"
	default:
		return codigo
	}
}

// ParseApostila extrai data e partes numeradas de uma página colada da apostila.
func ParseApostila(texto string, dataFallback time.Time) ApostilaParseResult {
	texto = strings.ReplaceAll(texto, "\r\n", "\n")
	texto = strings.ReplaceAll(texto, "\u00a0", " ")
	res := ApostilaParseResult{}
	ref, rotuloFonte, avisos := parseApostilaDate(texto, dataFallback)
	res.DataReferencia = ref
	res.RotuloFonte = rotuloFonte
	res.Avisos = avisos
	if strings.TrimSpace(texto) == "" {
		res.Avisos = append(res.Avisos, "Cole o texto da página da apostila.")
		return res
	}

	lines := strings.Split(texto, "\n")
	section := ""
	type item struct {
		title   string
		section string
		block   []string
	}
	var items []item
	var cur *item
	flush := func() {
		if cur != nil {
			items = append(items, *cur)
			cur = nil
		}
	}
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if s := currentSection(line); s != "" {
			section = s
			continue
		}
		if m := reNumbered.FindStringSubmatch(line); len(m) == 3 {
			flush()
			cur = &item{title: strings.TrimSpace(m[2]), section: section}
			continue
		}
		if cur != nil && line != "" {
			cur.block = append(cur.block, line)
		}
	}
	flush()

	for _, it := range items {
		codigo, tituloAlt, temaTitulo, ok := classifyApostilaTitle(it.title, it.section)
		if !ok {
			res.Avisos = append(res.Avisos, fmt.Sprintf("Ignorei a parte “%s” (tipo não reconhecido).", it.title))
			continue
		}
		dur, temaBloco := extractDurationAndTema(it.block)
		titulo := tipoNomePadrao(codigo)
		if tituloAlt != "" {
			titulo = tituloAlt
		}
		tema := ""
		switch codigo {
		case "tesouros":
			tema = temaTitulo
		case "joias":
			tema = ""
		case "vida_crista_extra":
			if tituloAlt != "" {
				titulo = tituloAlt
			} else if strings.TrimSpace(it.title) != "" {
				titulo = strings.TrimSpace(it.title)
			}
			tema = temaBloco
		default:
			tema = temaTitulo
			if tema == "" {
				tema = temaBloco
			}
		}
		res.Partes = append(res.Partes, ApostilaParteDraft{
			Codigo:     codigo,
			Titulo:     titulo,
			Tema:       tema,
			DuracaoMin: dur,
		})
	}
	if len(res.Partes) == 0 {
		res.Avisos = append(res.Avisos, "Não encontrei partes numeradas (1. Tesouros, 4. Iniciando conversas…). Confira se colou o programa da semana.")
	}
	return res
}
