package nfcepr

import (
	"errors"
	"net/url"
	"strings"
)

// ErrNoAccessKey indica que o texto colado não contém chave NFC-e (44 dígitos) reconhecível.
var ErrNoAccessKey = errors.New("não foi possível identificar a chave de acesso da NFC-e (44 dígitos)")

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
