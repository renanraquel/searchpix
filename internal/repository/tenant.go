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

func scanTenantFull(row interface{ Scan(dest ...interface{}) error }) (*model.Tenant, error) {
	var t model.Tenant
	var nfceCNPJ sql.NullString
	err := row.Scan(&t.ID, &t.Name, &t.Slug, &nfceCNPJ, &t.BackgroundImageURL, &t.BackgroundStorageKey, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	if nfceCNPJ.Valid && nfceCNPJ.String != "" {
		t.NfceEmitterCNPJ = nfceCNPJ.String
	}
	return &t, nil
}

func (r *TenantRepository) GetByID(id string) (*model.Tenant, error) {
	q := `SELECT id, name, slug, nfce_emitter_cnpj, COALESCE(background_image_url,''), COALESCE(background_storage_key,''), created_at FROM tenants WHERE id = $1`
	q = db.QueryForDriver(q, r.driver)
	t, err := scanTenantFull(r.db.QueryRow(q, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TenantRepository) GetBySlug(slug string) (*model.Tenant, error) {
	q := `SELECT id, name, slug, nfce_emitter_cnpj, COALESCE(background_image_url,''), COALESCE(background_storage_key,''), created_at FROM tenants WHERE slug = $1`
	q = db.QueryForDriver(q, r.driver)
	t, err := scanTenantFull(r.db.QueryRow(q, slug))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return t, nil
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

// DeleteByID remove o estabelecimento (e usuários em cascata). Uso: rollback em signup público.
func (r *TenantRepository) DeleteByID(id string) error {
	q := `DELETE FROM tenants WHERE id = $1`
	q = db.QueryForDriver(q, r.driver)
	_, err := r.db.Exec(q, id)
	return err
}

// SetNfceEmitterCNPJ grava o CNPJ do emitente (14 dígitos) usado para validar NFC-e na pontuação pública.
func (r *TenantRepository) SetNfceEmitterCNPJ(tenantID, cnpj14 string) error {
	q := `UPDATE tenants SET nfce_emitter_cnpj = $1 WHERE id = $2`
	q = db.QueryForDriver(q, r.driver)
	_, err := r.db.Exec(q, cnpj14, tenantID)
	return err
}

// UpdateBackground atualiza a imagem de fundo do tenant (armazenamento local).
func (r *TenantRepository) UpdateBackground(tenantID string, data []byte, contentType string) error {
	q := `UPDATE tenants SET background_image_data = $1, background_image_content_type = $2 WHERE id = $3`
	q = db.QueryForDriver(q, r.driver)
	_, err := r.db.Exec(q, data, contentType, tenantID)
	return err
}

func (r *TenantRepository) UpdateBackgroundR2(tenantID, storageKey, publicURL, contentType string) error {
	q := `UPDATE tenants SET background_storage_key = $1, background_image_url = $2, background_image_content_type = $3, background_image_data = NULL WHERE id = $4`
	q = db.QueryForDriver(q, r.driver)
	_, err := r.db.Exec(q, storageKey, publicURL, contentType, tenantID)
	return err
}

type TenantBackgroundMeta struct {
	Data        []byte
	ContentType string
	PublicURL   string
	StorageKey  string
}

func (r *TenantRepository) GetBackgroundMetaBySlug(slug string) (*TenantBackgroundMeta, error) {
	q := `SELECT background_image_data, background_image_content_type, COALESCE(background_image_url,''), COALESCE(background_storage_key,'') FROM tenants WHERE slug = $1`
	q = db.QueryForDriver(q, r.driver)
	return r.scanBackgroundMeta(r.db.QueryRow(q, slug))
}

func (r *TenantRepository) scanBackgroundMeta(row *sql.Row) (*TenantBackgroundMeta, error) {
	var data []byte
	var contentType sql.NullString
	var url, key string
	err := row.Scan(&data, &contentType, &url, &key)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ct := "image/jpeg"
	if contentType.Valid && contentType.String != "" {
		ct = contentType.String
	}
	return &TenantBackgroundMeta{Data: data, ContentType: ct, PublicURL: url, StorageKey: key}, nil
}

type TenantBackgroundBlobRow struct {
	ID          string
	ContentType string
	Data        []byte
}

func (r *TenantRepository) ListPendingBackgroundR2Migration() ([]TenantBackgroundBlobRow, error) {
	q := `SELECT id, COALESCE(background_image_content_type,''), background_image_data
		FROM tenants
		WHERE background_image_data IS NOT NULL
		  AND (background_image_url IS NULL OR background_image_url = '')`
	q = db.QueryForDriver(q, r.driver)
	rows, err := r.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []TenantBackgroundBlobRow
	for rows.Next() {
		var row TenantBackgroundBlobRow
		if err := rows.Scan(&row.ID, &row.ContentType, &row.Data); err != nil {
			return nil, err
		}
		if len(row.Data) == 0 {
			continue
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

func (r *TenantRepository) MarkBackgroundR2Migrated(id, storageKey, publicURL string) error {
	q := `UPDATE tenants SET background_storage_key = $1, background_image_url = $2, background_image_data = NULL WHERE id = $3`
	q = db.QueryForDriver(q, r.driver)
	_, err := r.db.Exec(q, storageKey, publicURL, id)
	return err
}

// GetBackgroundBySlug retorna a imagem de fundo para o tenant identificado pelo slug
func (r *TenantRepository) GetBackgroundBySlug(slug string) ([]byte, string, error) {
	meta, err := r.GetBackgroundMetaBySlug(slug)
	if err != nil || meta == nil {
		return nil, "", err
	}
	if len(meta.Data) == 0 {
		return nil, "", nil
	}
	return meta.Data, meta.ContentType, nil
}
