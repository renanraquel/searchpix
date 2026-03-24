package nfcepr

import "testing"

func TestExtractValorPagarFromHTML_SEFAZPRMobileLayout(t *testing.T) {
	html := `<div id="linhaTotal" class="linhaShade">
    <label>Valor a pagar R$:</label>
    <span class="totalNumb txtMax">15,00</span>
   </div>`
	v, err := extractValorPagarFromHTML([]byte(html))
	if err != nil {
		t.Fatal(err)
	}
	if v != 15.0 {
		t.Fatalf("valor = %v, want 15.00", v)
	}
}

func TestExtractValorPagarFromHTML_Milhar(t *testing.T) {
	html := `<label>Valor a pagar R$:</label>
	<span class="totalNumb txtMax">1.234,56</span>`
	v, err := extractValorPagarFromHTML([]byte(html))
	if err != nil {
		t.Fatal(err)
	}
	if v != 1234.56 {
		t.Fatalf("valor = %v, want 1234.56", v)
	}
}
