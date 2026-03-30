package validation

import (
	"regexp"
	"strconv"
)

var nonDigit = regexp.MustCompile(`\D`)

// OnlyDigitsCPF extrai só dígitos do CPF.
func OnlyDigitsCPF(s string) string {
	return nonDigit.ReplaceAllString(s, "")
}

// IsValidCPF valida CPF brasileiro (11 dígitos e verificadores).
func IsValidCPF(s string) bool {
	d := OnlyDigitsCPF(s)
	if len(d) != 11 {
		return false
	}
	if d == "" {
		return false
	}
	first := d[0]
	for i := 1; i < 11; i++ {
		if d[i] != first {
			goto check
		}
	}
	return false
check:
	var sum int
	for i := 0; i < 9; i++ {
		n, _ := strconv.Atoi(string(d[i]))
		sum += n * (10 - i)
	}
	rest := (sum * 10) % 11
	if rest == 10 {
		rest = 0
	}
	v9, _ := strconv.Atoi(string(d[9]))
	if rest != v9 {
		return false
	}
	sum = 0
	for i := 0; i < 10; i++ {
		n, _ := strconv.Atoi(string(d[i]))
		sum += n * (11 - i)
	}
	rest = (sum * 10) % 11
	if rest == 10 {
		rest = 0
	}
	v10, _ := strconv.Atoi(string(d[10]))
	return rest == v10
}
