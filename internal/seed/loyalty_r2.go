package seed

import (
	"log"

	"searchpix/internal/r2"
	"searchpix/internal/repository"
)

// MigrateLoyaltyImagesToR2 envia fotos de produto e fundo do tenant para o R2.
// Itens já com URL http(s) são ignorados — seguro em todo boot.
func MigrateLoyaltyImagesToR2(productRepo *repository.ProductRepository, tenantRepo *repository.TenantRepository, client *r2.Client) {
	if client == nil {
		return
	}
	migrateProductsToR2(productRepo, client)
	migrateTenantBackgroundsToR2(tenantRepo, client)
}

func migrateProductsToR2(repo *repository.ProductRepository, client *r2.Client) {
	if repo == nil {
		return
	}
	pending, err := repo.ListPendingR2Migration()
	if err != nil {
		log.Printf("Product R2 migrate: erro ao listar pendentes: %v", err)
		return
	}
	if len(pending) == 0 {
		log.Printf("Product R2 migrate: nada pendente")
		return
	}
	log.Printf("Product R2 migrate: %d item(ns) para enviar", len(pending))
	ok := 0
	for _, row := range pending {
		key := r2.ProductObjectKey(row.TenantID, row.ID, row.ContentType)
		_, publicURL, err := client.UploadWithTimeout(key, row.Data, row.ContentType)
		if err != nil {
			log.Printf("Product R2 migrate: falha no item %s: %v", row.ID, err)
			continue
		}
		if err := repo.MarkR2Migrated(row.ID, key, publicURL); err != nil {
			log.Printf("Product R2 migrate: falha ao atualizar item %s: %v", row.ID, err)
			continue
		}
		ok++
	}
	log.Printf("Product R2 migrate: concluído (%d/%d)", ok, len(pending))
}

func migrateTenantBackgroundsToR2(repo *repository.TenantRepository, client *r2.Client) {
	if repo == nil {
		return
	}
	pending, err := repo.ListPendingBackgroundR2Migration()
	if err != nil {
		log.Printf("Tenant background R2 migrate: erro ao listar pendentes: %v", err)
		return
	}
	if len(pending) == 0 {
		log.Printf("Tenant background R2 migrate: nada pendente")
		return
	}
	log.Printf("Tenant background R2 migrate: %d item(ns) para enviar", len(pending))
	ok := 0
	for _, row := range pending {
		key := r2.TenantBackgroundKey(row.ID, row.ContentType)
		_, publicURL, err := client.UploadWithTimeout(key, row.Data, row.ContentType)
		if err != nil {
			log.Printf("Tenant background R2 migrate: falha no tenant %s: %v", row.ID, err)
			continue
		}
		if err := repo.MarkBackgroundR2Migrated(row.ID, key, publicURL); err != nil {
			log.Printf("Tenant background R2 migrate: falha ao atualizar tenant %s: %v", row.ID, err)
			continue
		}
		ok++
	}
	log.Printf("Tenant background R2 migrate: concluído (%d/%d)", ok, len(pending))
}
