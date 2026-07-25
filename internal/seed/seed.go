package seed

import (
	"log"

	"searchpix/internal/model"
	"searchpix/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// Dados do primeiro tenant e usuário (seed inicial)
const (
	SeedTenantName = "Ibimassas"
	SeedTenantSlug = "ibimassas"
	SeedUsername   = "ibimassas"
	SeedPassword   = "ibimassas2026@"

	SeedAdminUsername = "ADMIN"
	SeedAdminPassword = "Pomarola6770@"
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

// RunAdminAndDesignacao garante usuário ADMIN e catálogo de tipos de parte.
func RunAdminAndDesignacao(tenantRepo *repository.TenantRepository, userRepo *repository.UserRepository, desigRepo *repository.DesignacaoRepository) {
	if err := desigRepo.SeedTiposParte(); err != nil {
		log.Printf("Seed designação: erro ao criar tipos de parte: %v", err)
	}

	exist, err := userRepo.GetByUsername(SeedAdminUsername)
	if err != nil {
		log.Printf("Seed ADMIN: erro ao buscar usuário: %v", err)
		return
	}
	if exist != nil {
		return
	}

	tenants, err := tenantRepo.List()
	if err != nil || len(tenants) == 0 {
		log.Printf("Seed ADMIN: nenhum tenant disponível para vincular o ADMIN")
		return
	}
	tenantID := tenants[0].ID
	for _, t := range tenants {
		if t.Slug == SeedTenantSlug {
			tenantID = t.ID
			break
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(SeedAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Seed ADMIN: erro ao gerar hash: %v", err)
		return
	}
	_, err = userRepo.CreateWithProfileAndRole(tenantID, SeedAdminUsername, string(hash), "", "", "", "", model.RoleAdmin)
	if err != nil {
		log.Printf("Seed ADMIN: erro ao criar usuário: %v", err)
		return
	}
	log.Printf("Seed ADMIN: usuário %q criado com acesso às telas de designações", SeedAdminUsername)
}
