package nfcepr

import "testing"

func TestExtractEmitterCNPJFromAccessKey_chavePR(t *testing.T) {
	t.Parallel()
	// Chave real de exemplo (PR): CNPJ 25295518000120, modelo 65
	chave := "41260325295518000120650030003502941606568820"
	got, err := ExtractEmitterCNPJFromAccessKey(chave)
	if err != nil {
		t.Fatal(err)
	}
	if got != "25295518000120" {
		t.Fatalf("CNPJ = %q", got)
	}
}

func TestExtractEmitterCNPJFromAccessKey_rejeitaModelo55(t *testing.T) {
	t.Parallel()
	good := "41260325295518000120650030003502941606568820"
	chave := good[:20] + "55" + good[22:]
	_, err := ExtractEmitterCNPJFromAccessKey(chave)
	if err == nil {
		t.Fatal("esperado erro modelo")
	}
}

func TestNormalizeCNPJ14(t *testing.T) {
	t.Parallel()
	got, err := NormalizeCNPJ14("25.295.518/0001-20")
	if err != nil || got != "25295518000120" {
		t.Fatalf("%q %v", got, err)
	}
}
