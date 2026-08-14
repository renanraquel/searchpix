package repository

import (
	"database/sql"
	"strings"

	"searchpix/internal/db"
	"searchpix/internal/model"
)

type ProductRepository struct {
	db     *sql.DB
	driver string
}

func NewProductRepository(database *sql.DB, driver string) *ProductRepository {
	return &ProductRepository{db: database, driver: driver}
}

func resolveProductImageURL(id, stored string, hasBlob bool) string {
	stored = strings.TrimSpace(stored)
	if strings.HasPrefix(stored, "http://") || strings.HasPrefix(stored, "https://") {
		return stored
	}
	if hasBlob {
		return "/api/products/image?id=" + id
	}
	return stored
}

func (r *ProductRepository) ListByTenant(tenantID string) ([]model.Product, error) {
	q := `SELECT id, tenant_id, COALESCE(image_url,''), COALESCE(image_storage_key,''),
		CASE WHEN image_data IS NOT NULL THEN 1 ELSE 0 END, description, points_required, created_at, updated_at
		FROM products WHERE tenant_id = $1 ORDER BY created_at DESC`
	q = db.QueryForDriver(q, r.driver)
	rows, err := r.db.Query(q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.Product
	for rows.Next() {
		var p model.Product
		var hasBlobInt int
		err := rows.Scan(&p.ID, &p.TenantID, &p.ImageURL, &p.StorageKey, &hasBlobInt, &p.Description, &p.PointsRequired, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		p.ImageURL = resolveProductImageURL(p.ID, p.ImageURL, hasBlobInt > 0)
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *ProductRepository) GetByID(id string) (*model.Product, error) {
	q := `SELECT id, tenant_id, COALESCE(image_url,''), COALESCE(image_storage_key,''),
		CASE WHEN image_data IS NOT NULL THEN 1 ELSE 0 END, description, points_required, created_at, updated_at
		FROM products WHERE id = $1`
	q = db.QueryForDriver(q, r.driver)
	var p model.Product
	var hasBlobInt int
	err := r.db.QueryRow(q, id).Scan(&p.ID, &p.TenantID, &p.ImageURL, &p.StorageKey, &hasBlobInt, &p.Description, &p.PointsRequired, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.ImageURL = resolveProductImageURL(p.ID, p.ImageURL, hasBlobInt > 0)
	return &p, nil
}

type ProductImageMeta struct {
	Data        []byte
	ContentType string
	PublicURL   string
	StorageKey  string
}

func (r *ProductRepository) GetImageMetaByID(productID, tenantID string) (*ProductImageMeta, error) {
	q := `SELECT image_data, image_content_type, COALESCE(image_url,''), COALESCE(image_storage_key,'')
		FROM products WHERE id = $1 AND tenant_id = $2`
	q = db.QueryForDriver(q, r.driver)
	return r.scanImageMeta(r.db.QueryRow(q, productID, tenantID))
}

func (r *ProductRepository) GetImageMetaByProductID(productID string) (*ProductImageMeta, error) {
	q := `SELECT image_data, image_content_type, COALESCE(image_url,''), COALESCE(image_storage_key,'')
		FROM products WHERE id = $1`
	q = db.QueryForDriver(q, r.driver)
	return r.scanImageMeta(r.db.QueryRow(q, productID))
}

func (r *ProductRepository) scanImageMeta(row *sql.Row) (*ProductImageMeta, error) {
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
	return &ProductImageMeta{Data: data, ContentType: ct, PublicURL: url, StorageKey: key}, nil
}

type ProductBlobRow struct {
	ID          string
	TenantID    string
	ContentType string
	Data        []byte
}

func (r *ProductRepository) ListPendingR2Migration() ([]ProductBlobRow, error) {
	q := `SELECT id, tenant_id, COALESCE(image_content_type,''), image_data
		FROM products
		WHERE image_data IS NOT NULL
		  AND (image_url IS NULL OR image_url = '' OR image_url LIKE '/api/%')`
	q = db.QueryForDriver(q, r.driver)
	rows, err := r.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ProductBlobRow
	for rows.Next() {
		var row ProductBlobRow
		if err := rows.Scan(&row.ID, &row.TenantID, &row.ContentType, &row.Data); err != nil {
			return nil, err
		}
		if len(row.Data) == 0 {
			continue
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

func (r *ProductRepository) MarkR2Migrated(id, storageKey, imageURL string) error {
	q := `UPDATE products SET image_storage_key = $1, image_url = $2, image_data = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = $3`
	if r.driver == "sqlite3" {
		q = `UPDATE products SET image_storage_key = $1, image_url = $2, image_data = NULL, updated_at = datetime('now') WHERE id = $3`
	}
	q = db.QueryForDriver(q, r.driver)
	_, err := r.db.Exec(q, storageKey, imageURL, id)
	return err
}

func (r *ProductRepository) CreateWithID(id, tenantID, imageURL, description string, pointsRequired int, imageData []byte, imageContentType, storageKey string) (*model.Product, error) {
	if id == "" {
		id = newUUID()
	}
	q := `INSERT INTO products (id, tenant_id, image_url, image_data, image_content_type, image_storage_key, description, points_required)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, tenant_id, COALESCE(image_url,''), COALESCE(image_storage_key,''), description, points_required, created_at, updated_at`
	q = db.QueryForDriver(q, r.driver)
	var p model.Product
	err := r.db.QueryRow(q, id, tenantID, nullStr(imageURL), nullBytes(imageData), nullStr(imageContentType), nullStr(storageKey), description, pointsRequired).
		Scan(&p.ID, &p.TenantID, &p.ImageURL, &p.StorageKey, &p.Description, &p.PointsRequired, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.ImageURL = resolveProductImageURL(p.ID, p.ImageURL, len(imageData) > 0)
	return &p, nil
}

func (r *ProductRepository) Create(tenantID, imageURL, description string, pointsRequired int, imageData []byte, imageContentType, storageKey string) (*model.Product, error) {
	return r.CreateWithID(newUUID(), tenantID, imageURL, description, pointsRequired, imageData, imageContentType, storageKey)
}

func (r *ProductRepository) Update(id, imageURL, description string, pointsRequired int, imageData []byte, imageContentType, storageKey string, replaceImage bool) (*model.Product, error) {
	if replaceImage {
		q := `UPDATE products SET image_url = $1, image_data = $2, image_content_type = $3, image_storage_key = $4, description = $5, points_required = $6, updated_at = CURRENT_TIMESTAMP WHERE id = $7`
		if r.driver == "sqlite3" {
			q = `UPDATE products SET image_url = $1, image_data = $2, image_content_type = $3, image_storage_key = $4, description = $5, points_required = $6, updated_at = datetime('now') WHERE id = $7`
		}
		q = db.QueryForDriver(q, r.driver)
		_, err := r.db.Exec(q, nullStr(imageURL), nullBytes(imageData), nullStr(imageContentType), nullStr(storageKey), description, pointsRequired, id)
		if err != nil {
			return nil, err
		}
	} else {
		q := `UPDATE products SET description = $1, points_required = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`
		if r.driver == "sqlite3" {
			q = `UPDATE products SET description = $1, points_required = $2, updated_at = datetime('now') WHERE id = $3`
		}
		q = db.QueryForDriver(q, r.driver)
		_, err := r.db.Exec(q, description, pointsRequired, id)
		if err != nil {
			return nil, err
		}
	}
	return r.GetByID(id)
}

func (r *ProductRepository) Delete(id string) error {
	q := `DELETE FROM products WHERE id = $1`
	q = db.QueryForDriver(q, r.driver)
	_, err := r.db.Exec(q, id)
	return err
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullBytes(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}
