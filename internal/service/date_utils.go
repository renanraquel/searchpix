package service

import (
	"fmt"
)

func NormalizarPeriodo(inicio, fim string) (string, string, error) {
	if inicio == "" || fim == "" {
		return "", "", fmt.Errorf("inicio e fim são obrigatórios")
	}

	inicioFinal := inicio + "T00:00:00"
	fimFinal := fim + "T23:59:59"

	return inicioFinal, fimFinal, nil
}
