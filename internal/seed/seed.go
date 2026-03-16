package seed

import (
	"log"

	"searchpix/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// Dados do primeiro tenant e usuário (seed inicial)
const (
	SeedTenantName = "Ibimassas"
	SeedTenantSlug = "ibimassas"
	SeedUsername   = "ibimassas"
	SeedPassword   = "ibimassas2026@"
)

// Run cria o primeiro tenant e usuário se o banco estiver vazio (local e produção)
func Run(tenantRepo *repository.TenantRepository, userRepo *repository.UserRepository) {
	list, err := tenantRepo.List()
	if err != nil {
		log.Printf("Seed: aviso ao listar tenants: %v", err)
		return
	}
	if len(list) > 0 {
		return
	}

	tenant, err := tenantRepo.Create(SeedTenantName, SeedTenantSlug)
	if err != nil {
		log.Printf("Seed: erro ao criar tenant: %v", err)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(SeedPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Seed: erro ao gerar hash da senha: %v", err)
		return
	}

	_, err = userRepo.Create(tenant.ID, SeedUsername, string(hash))
	if err != nil {
		log.Printf("Seed: erro ao criar usuário: %v", err)
		return
	}

	log.Printf("Seed: tenant %q e usuário %q criados. Use para login: tenant_slug=%s, user=%s", SeedTenantName, SeedUsername, SeedTenantSlug, SeedUsername)
}
