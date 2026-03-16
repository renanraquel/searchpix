package repository

import (
	"database/sql"

	"searchpix/internal/db"
	"searchpix/internal/model"
)

type TenantRepository struct {
	db     *sql.DB
	driver string
}

func NewTenantRepository(database *sql.DB, driver string) *TenantRepository {
	return &TenantRepository{db: database, driver: driver}
}

func (r *TenantRepository) List() ([]model.Tenant, error) {
	q := `SELECT id, name, slug, created_at FROM tenants ORDER BY name`
	q = db.QueryForDriver(q, r.driver)
	rows, err := r.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.Tenant
	for rows.Next() {
		var t model.Tenant
		err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (r *TenantRepository) GetByID(id string) (*model.Tenant, error) {
	q := `SELECT id, name, slug, created_at FROM tenants WHERE id = $1`
	q = db.QueryForDriver(q, r.driver)
	var t model.Tenant
	err := r.db.QueryRow(q, id).Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TenantRepository) GetBySlug(slug string) (*model.Tenant, error) {
	q := `SELECT id, name, slug, created_at FROM tenants WHERE slug = $1`
	q = db.QueryForDriver(q, r.driver)
	var t model.Tenant
	err := r.db.QueryRow(q, slug).Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TenantRepository) Create(name, slug string) (*model.Tenant, error) {
	q := `INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3) RETURNING id, name, slug, created_at`
	q = db.QueryForDriver(q, r.driver)
	id := newUUID()
	var t model.Tenant
	err := r.db.QueryRow(q, id, name, slug).Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// UpdateBackground atualiza a imagem de fundo do tenant
func (r *TenantRepository) UpdateBackground(tenantID string, data []byte, contentType string) error {
	q := `UPDATE tenants SET background_image_data = $1, background_image_content_type = $2 WHERE id = $3`
	q = db.QueryForDriver(q, r.driver)
	_, err := r.db.Exec(q, data, contentType, tenantID)
	return err
}

// GetBackgroundBySlug retorna a imagem de fundo para o tenant identificado pelo slug
func (r *TenantRepository) GetBackgroundBySlug(slug string) ([]byte, string, error) {
	q := `SELECT background_image_data, background_image_content_type FROM tenants WHERE slug = $1`
	q = db.QueryForDriver(q, r.driver)
	var data []byte
	var contentType sql.NullString
	err := r.db.QueryRow(q, slug).Scan(&data, &contentType)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 {
		return nil, "", nil
	}
	ct := "image/jpeg"
	if contentType.Valid && contentType.String != "" {
		ct = contentType.String
	}
	return data, ct, nil
}
