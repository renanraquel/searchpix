package nfcepr

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ErrNoAccessKey indica que o texto colado não contém chave NFC-e (44 dígitos) reconhecível.
var ErrNoAccessKey = errors.New("não foi possível identificar a chave de acesso da NFC-e (44 dígitos)")

// ErrNotNFCeModel indica chave que não é de NFC-e (modelo 65).
var ErrNotNFCeModel = errors.New("a chave não corresponde a uma NFC-e (modelo 65)")

// NormalizeCNPJ14 retorna somente os 14 dígitos do CNPJ ou erro se o tamanho for inválido.
func NormalizeCNPJ14(s string) (string, error) {
	d := onlyDigits(s)
	if len(d) != 14 {
		return "", errors.New("CNPJ deve conter 14 dígitos")
	}
	return d, nil
}

// ExtractEmitterCNPJFromAccessKey retorna o CNPJ do emitente codificado na chave (pos. 7–20),
// conforme layout da chave de acesso da NF-e/NFC-e. Exige modelo 65 (NFC-e).
func ExtractEmitterCNPJFromAccessKey(chave string) (string, error) {
	if len(chave) != 44 {
		return "", fmt.Errorf("chave de acesso inválida: esperado 44 dígitos, obtido %d", len(chave))
	}
	for i := 0; i < len(chave); i++ {
		if chave[i] < '0' || chave[i] > '9' {
			return "", errors.New("chave de acesso inválida: deve conter apenas dígitos")
		}
	}
	if chave[20:22] != "65" {
		return "", ErrNotNFCeModel
	}
	return chave[6:20], nil
}

// ExtractAccessKeyFromPayload aceita URL completa do QR (ex.: PR), query p= ou só os 44 dígitos.
func ExtractAccessKeyFromPayload(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", ErrNoAccessKey
	}
	d := onlyDigits(s)
	if len(d) == 44 {
		return d, nil
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", ErrNoAccessKey
	}
	p := u.Query().Get("p")
	if p == "" {
		return "", ErrNoAccessKey
	}
	p, _ = url.QueryUnescape(p)
	parts := strings.Split(p, "|")
	if len(parts) == 0 {
		return "", ErrNoAccessKey
	}
	chave := onlyDigits(parts[0])
	if len(chave) != 44 {
		return "", ErrNoAccessKey
	}
	return chave, nil
}

func onlyDigits(s string) string {
	b := make([]byte, 0, 44)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			b = append(b, c)
		}
	}
	return string(b)
}
