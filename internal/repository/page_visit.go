package repository

import (
	"database/sql"
	"strings"

	"searchpix/internal/db"
)

type PageVisitRepository struct {
	db     *sql.DB
	driver string
}

func NewPageVisitRepository(database *sql.DB, driver string) *PageVisitRepository {
	return &PageVisitRepository{db: database, driver: driver}
}

func trimForDB(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func (r *PageVisitRepository) Create(pageKey, pagePath, queryString, tenantSlug, referrer, userAgent, ip string) error {
	q := `INSERT INTO page_visits (id, page_key, page_path, query_string, tenant_slug, referrer, user_agent, ip) VALUES ($1, $2, $3, NULLIF($4,''), NULLIF($5,''), NULLIF($6,''), NULLIF($7,''), NULLIF($8,''))`
	q = db.QueryForDriver(q, r.driver)
	_, err := r.db.Exec(
		q,
		newUUID(),
		trimForDB(pageKey, 80),
		trimForDB(pagePath, 255),
		trimForDB(queryString, 1000),
		trimForDB(tenantSlug, 120),
		trimForDB(referrer, 500),
		trimForDB(userAgent, 400),
		trimForDB(ip, 100),
	)
	return err
}
