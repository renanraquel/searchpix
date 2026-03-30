package auth

import (
	"net/http"
	"os"
	"strings"

	"searchpix/internal/repository"
)

// defaultPixModuleSlugs slugs com acesso ao módulo PIX quando PIX_MODULE_TENANT_SLUGS não está definido.
var defaultPixModuleSlugs = []string{"ibimassas"}

// PixModuleTenantSlugs lê a lista de slugs permitidos. Env: PIX_MODULE_TENANT_SLUGS=loja1,loja2
func PixModuleTenantSlugs() []string {
	raw := strings.TrimSpace(os.Getenv("PIX_MODULE_TENANT_SLUGS"))
	if raw == "" {
		return append([]string(nil), defaultPixModuleSlugs...)
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return append([]string(nil), defaultPixModuleSlugs...)
	}
	return out
}

func slugAllowedForPixModule(slug string, allowed []string) bool {
	for _, a := range allowed {
		if a == slug {
			return true
		}
	}
	return false
}

// PixModuleGate restringe o handler a tenants cujo slug está em PIX_MODULE_TENANT_SLUGS (ou padrão ibimassas).
// Deve rodar após LoyaltyAuthMiddleware.
func PixModuleGate(tenantRepo *repository.TenantRepository, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := TenantIDFromContext(r.Context())
		if tenantID == "" {
			http.Error(w, "Não autorizado", http.StatusUnauthorized)
			return
		}
		t, err := tenantRepo.GetByID(tenantID)
		if err != nil {
			http.Error(w, "Erro ao validar acesso ao PIX", http.StatusInternalServerError)
			return
		}
		if t == nil {
			http.Error(w, "Não autorizado", http.StatusUnauthorized)
			return
		}
		if !slugAllowedForPixModule(t.Slug, PixModuleTenantSlugs()) {
			http.Error(w, "Módulo PIX não disponível para este estabelecimento", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
