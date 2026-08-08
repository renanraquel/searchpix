package seed

import (
	"log"

	"searchpix/internal/r2"
	"searchpix/internal/repository"
)

// MigrateCarouselToR2 envia blobs antigos do Postgres/SQLite para o R2 uma vez.
// Itens já com media_url são ignorados — seguro rodar em todo boot.
func MigrateCarouselToR2(repo *repository.CarouselRepository, client *r2.Client) {
	if client == nil || repo == nil {
		return
	}
	pending, err := repo.ListPendingR2Migration()
	if err != nil {
		log.Printf("Carousel R2 migrate: erro ao listar pendentes: %v", err)
		return
	}
	if len(pending) == 0 {
		log.Printf("Carousel R2 migrate: nada pendente")
		return
	}
	log.Printf("Carousel R2 migrate: %d item(ns) para enviar", len(pending))
	ok := 0
	for _, row := range pending {
		key := r2.CarouselObjectKey(row.TenantID, row.ID, row.ContentType)
		_, publicURL, err := client.UploadWithTimeout(key, row.Data, row.ContentType)
		if err != nil {
			log.Printf("Carousel R2 migrate: falha no item %s: %v", row.ID, err)
			continue
		}
		if err := repo.MarkR2Migrated(row.ID, key, publicURL); err != nil {
			log.Printf("Carousel R2 migrate: falha ao atualizar item %s: %v", row.ID, err)
			continue
		}
		ok++
	}
	log.Printf("Carousel R2 migrate: concluído (%d/%d)", ok, len(pending))
}
