package repository

import (
	"database/sql"
	"strings"

	"searchpix/internal/db"
)

type NfceClaimRepository struct {
	db     *sql.DB
	driver string
}

func NewNfceClaimRepository(database *sql.DB, driver string) *NfceClaimRepository {
	return &NfceClaimRepository{db: database, driver: driver}
}

// Exists retorna true se a chave já foi usada neste tenant.
func (r *NfceClaimRepository) Exists(tenantID, accessKey string) (bool, error) {
	q := `SELECT 1 FROM nfce_claims WHERE tenant_id = $1 AND access_key = $2 LIMIT 1`
	q = db.QueryForDriver(q, r.driver)
	var one int
	err := r.db.QueryRow(q, tenantID, accessKey).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Insert registra uso da nota. Erro com "unique" / UNIQUE constraint = duplicata.
func (r *NfceClaimRepository) Insert(tenantID, accessKey, customerID string, valueReais float64, pointsAwarded int) error {
	q := `INSERT INTO nfce_claims (id, tenant_id, access_key, customer_id, value_reais, points_awarded) VALUES ($1, $2, $3, $4, $5, $6)`
	q = db.QueryForDriver(q, r.driver)
	id := newUUID()
	_, err := r.db.Exec(q, id, tenantID, accessKey, customerID, valueReais, pointsAwarded)
	return err
}

// IsUniqueViolation detecta conflito de chave duplicada (Postgres + SQLite).
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "unique") || strings.Contains(s, "23505") || strings.Contains(s, "constraint")
}
