package repository

import (
	"database/sql"

	"searchpix/internal/db"
)

type NfceEmitterRepository struct {
	db     *sql.DB
	driver string
}

func NewNfceEmitterRepository(database *sql.DB, driver string) *NfceEmitterRepository {
	return &NfceEmitterRepository{db: database, driver: driver}
}

func (r *NfceEmitterRepository) ListByTenant(tenantID string) ([]string, error) {
	q := `SELECT cnpj FROM tenant_nfce_emitters WHERE tenant_id = $1 ORDER BY cnpj`
	q = db.QueryForDriver(q, r.driver)
	rows, err := r.db.Query(q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]string, 0, 4)
	for rows.Next() {
		var cnpj string
		if err := rows.Scan(&cnpj); err != nil {
			return nil, err
		}
		list = append(list, cnpj)
	}
	return list, rows.Err()
}

func (r *NfceEmitterRepository) Add(tenantID, cnpj string) error {
	q := `INSERT INTO tenant_nfce_emitters (id, tenant_id, cnpj) VALUES ($1, $2, $3)`
	q = db.QueryForDriver(q, r.driver)
	_, err := r.db.Exec(q, newUUID(), tenantID, cnpj)
	return err
}

func (r *NfceEmitterRepository) Remove(tenantID, cnpj string) error {
	q := `DELETE FROM tenant_nfce_emitters WHERE tenant_id = $1 AND cnpj = $2`
	q = db.QueryForDriver(q, r.driver)
	_, err := r.db.Exec(q, tenantID, cnpj)
	return err
}
