package service

import (
	"testing"

	"searchpix/internal/model"
)

func TestPessoaElegivel_JoiasTesourosVidaCrista(t *testing.T) {
	servo := model.DesigPessoa{Tipo: "servo", Ativo: true, QualificadoTesouros: true}
	servoSemTesouros := model.DesigPessoa{Tipo: "servo", Ativo: true, QualificadoTesouros: false}
	anciao := model.DesigPessoa{Tipo: "anciao", Ativo: true, QualificadoTesouros: true}

	ok, motivo := PessoaElegivelParaParte(servo, "joias", "dono", "")
	if !ok {
		t.Fatalf("servo em joias deveria ser elegível, got: %s", motivo)
	}
	ok, motivo = PessoaElegivelParaParte(anciao, "joias", "dono", "")
	if !ok {
		t.Fatalf("ancião em joias deveria ser elegível, got: %s", motivo)
	}

	ok, motivo = PessoaElegivelParaParte(servo, "tesouros", "dono", "")
	if !ok {
		t.Fatalf("servo qualificado em tesouros deveria ser elegível, got: %s", motivo)
	}
	ok, _ = PessoaElegivelParaParte(servoSemTesouros, "tesouros", "dono", "")
	if ok {
		t.Fatal("servo sem flag de tesouros não deveria ser elegível em tesouros")
	}
	// Joias não exige a flag de Tesouros.
	ok, motivo = PessoaElegivelParaParte(servoSemTesouros, "joias", "dono", "")
	if !ok {
		t.Fatalf("servo sem flag de tesouros ainda deve poder fazer joias, got: %s", motivo)
	}

	ok, _ = PessoaElegivelParaParte(servo, "vida_crista_extra", "dono", "")
	if ok {
		t.Fatal("servo não deveria ser elegível em Nossa Vida Cristã")
	}
	ok, motivo = PessoaElegivelParaParte(anciao, "vida_crista_extra", "dono", "")
	if !ok {
		t.Fatalf("ancião em Nossa Vida Cristã deveria ser elegível, got: %s", motivo)
	}
}
