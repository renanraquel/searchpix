package service

import (
	"testing"
	"time"
)

const sampleApostilaOutubro2026 = `
26 DE OUTUBRO–1.º DE NOVEMBRO
Nossa Vida e Ministério Cristão — Apostila do Mês — 2026
JEREMIAS 47-48
Cântico 125 e oração | Comentários iniciais (1 min)

TESOUROS DA PALAVRA DE DEUS
1. Jeová é um Juiz justo e misericordioso
(10 min)
Jeremias predisse que Jeová puniria Moabe.

2. Joias espirituais
(10 min)
Jer. 48:6 — Em que sentido os moabitas se tornariam “como um junípero no deserto”?
Sua resposta

3. Leitura da Bíblia
(4 min) Jer. 48:1-13 (th lição 11)

FAÇA SEU MELHOR NO MINISTÉRIO
4. Iniciando conversas
(3 min) DE CASA EM CASA. (lmd lição 5 ponto 4)

5. Cultivando o interesse
(4 min) TESTEMUNHO INFORMAL. Fale sobre uma das verdades.

6. Fazendo discípulos
(5 min) Incentive seu estudante a orar mais vezes.

NOSSA VIDA CRISTÃ
Cântico 158
7. Necessidades locais
(15 min)

8. Estudo bíblico de congregação
(30 min) wcg cap. 14
Comentários finais (3 min) | Cântico 54 e oração
`

func TestParseApostila_SampleWeek(t *testing.T) {
	res := ParseApostila(sampleApostilaOutubro2026, time.Time{})
	if res.DataReferencia.Format("2006-01-02") != "2026-10-26" {
		t.Fatalf("data = %s, want 2026-10-26", res.DataReferencia.Format("2006-01-02"))
	}
	want := []struct {
		codigo string
		dur    int
	}{
		{"tesouros", 10},
		{"joias", 10},
		{"leitura_biblia", 4},
		{"iniciando_conversas", 3},
		{"cultivando_interesse", 4},
		{"fazendo_discipulos", 5},
		{"vida_crista_extra", 15},
		{"estudo_biblico", 30},
	}
	if len(res.Partes) != len(want) {
		t.Fatalf("partes=%d %+v", len(res.Partes), res.Partes)
	}
	for i, w := range want {
		p := res.Partes[i]
		if p.Codigo != w.codigo || p.DuracaoMin != w.dur {
			t.Errorf("parte %d = %s %dmin, want %s %dmin (tema=%q)", i, p.Codigo, p.DuracaoMin, w.codigo, w.dur, p.Tema)
		}
	}
	if res.Partes[0].Tema == "" {
		t.Error("tesouros deveria ter o tema do discurso")
	}
	if res.Partes[3].Tema == "" {
		t.Error("iniciando conversas deveria ter o tema da apostila")
	}
}

func TestParseApostila_SameMonthRange(t *testing.T) {
	res := ParseApostila("20-26 DE JULHO\n1. Joias espirituais\n(10 min)\n", time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if res.DataReferencia.Format("2006-01-02") != "2026-07-20" {
		t.Fatalf("data = %s", res.DataReferencia.Format("2006-01-02"))
	}
}
