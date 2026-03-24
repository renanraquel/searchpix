package nfcepr

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Na SEFAZ-PR o rótulo e o valor ficam em nós separados, ex.:
// <label>Valor a pagar R$:</label><span class="totalNumb txtMax">15,00</span>
var valorPagarRegex = regexp.MustCompile(`(?is)Valor\s+a\s+pagar\s*R\$\s*:?\s*(?:<[^>]+>|\s)+([\d]{1,3}(?:\.\d{3})*,\d{2}|\d+,\d{2})`)

// Reserva: label fechado imediatamente antes do span do total.
var valorPagarRegexLabeledSpan = regexp.MustCompile(
	`(?is)<label>\s*Valor\s+a\s+pagar\s*R\$\s*:?\s*</label>\s*<span[^>]*>\s*([\d]{1,3}(?:\.\d{3})*,\d{2}|\d+,\d{2})\s*</span>`)

// ParseBrazilianMoney converte "15,00" ou "1.234,56" para float64.
func ParseBrazilianMoney(s string) (float64, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")
	return strconv.ParseFloat(s, 64)
}

// IsPRNFCeConsultaURL indica se a URL parece ser consulta NFC-e da SEFAZ-PR (V1).
func IsPRNFCeConsultaURL(pageURL string) bool {
	u := strings.ToLower(strings.TrimSpace(pageURL))
	return strings.Contains(u, "fazenda.pr.gov.br") && strings.Contains(u, "nfce")
}

// FetchValorPagarFromConsultaURL faz GET na página pública e extrai "Valor a pagar R$:".
func FetchValorPagarFromConsultaURL(pageURL string) (float64, error) {
	client := &http.Client{Timeout: 18 * time.Second}
	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "SearchPix/1.0 (NFC-e consulta pública)")
	res, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("consulta retornou HTTP %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return 0, err
	}
	return extractValorPagarFromHTML(body)
}

func extractValorPagarFromHTML(body []byte) (float64, error) {
	for _, re := range []*regexp.Regexp{valorPagarRegex, valorPagarRegexLabeledSpan} {
		m := re.FindSubmatch(body)
		if len(m) < 2 {
			continue
		}
		v, err := ParseBrazilianMoney(string(m[1]))
		if err != nil || v <= 0 {
			continue
		}
		return v, nil
	}
	return 0, fmt.Errorf("valor total não encontrado na página")
}
