package repository

import (
	"database/sql"

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

func (r *ProductRepository) ListByTenant(tenantID string) ([]model.Product, error) {
	q := `SELECT id, tenant_id, image_url, image_data, description, points_required, created_at, updated_at FROM products WHERE tenant_id = $1 ORDER BY created_at DESC`
	q = db.QueryForDriver(q, r.driver)
	rows, err := r.db.Query(q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.Product
	for rows.Next() {
		var p model.Product
		var imgURL sql.NullString
		var imageData []byte
		err := rows.Scan(&p.ID, &p.TenantID, &imgURL, &imageData, &p.Description, &p.PointsRequired, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if len(imageData) > 0 {
			p.ImageURL = "/api/products/image?id=" + p.ID
		} else if imgURL.Valid {
			p.ImageURL = imgURL.String
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *ProductRepository) GetByID(id string) (*model.Product, error) {
	q := `SELECT id, tenant_id, image_url, image_data, description, points_required, created_at, updated_at FROM products WHERE id = $1`
	q = db.QueryForDriver(q, r.driver)
	var p model.Product
	var imgURL sql.NullString
	var imageData []byte
	err := r.db.QueryRow(q, id).Scan(&p.ID, &p.TenantID, &imgURL, &imageData, &p.Description, &p.PointsRequired, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(imageData) > 0 {
		p.ImageURL = "/api/products/image?id=" + p.ID
	} else if imgURL.Valid {
		p.ImageURL = imgURL.String
	}
	return &p, nil
}

// GetImageByID retorna (imageData, contentType, nil) do produto se existir e pertencer ao tenant
func (r *ProductRepository) GetImageByID(productID, tenantID string) ([]byte, string, error) {
	q := `SELECT image_data, image_content_type FROM products WHERE id = $1 AND tenant_id = $2`
	q = db.QueryForDriver(q, r.driver)
	var data []byte
	var contentType sql.NullString
	err := r.db.QueryRow(q, productID, tenantID).Scan(&data, &contentType)
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

// GetImageByProductID retorna (imageData, contentType, nil) do produto por id (sem validar tenant — para exibir em <img src> sem auth)
func (r *ProductRepository) GetImageByProductID(productID string) ([]byte, string, error) {
	q := `SELECT image_data, image_content_type FROM products WHERE id = $1`
	q = db.QueryForDriver(q, r.driver)
	var data []byte
	var contentType sql.NullString
	err := r.db.QueryRow(q, productID).Scan(&data, &contentType)
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

func (r *ProductRepository) Create(tenantID, imageURL, description string, pointsRequired int, imageData []byte, imageContentType string) (*model.Product, error) {
	q := `INSERT INTO products (id, tenant_id, image_url, image_data, image_content_type, description, points_required) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, tenant_id, image_url, description, points_required, created_at, updated_at`
	q = db.QueryForDriver(q, r.driver)
	id := newUUID()
	var p model.Product
	var imgURL sql.NullString
	err := r.db.QueryRow(q, id, tenantID, nullStr(imageURL), nullBytes(imageData), nullStr(imageContentType), description, pointsRequired).Scan(&p.ID, &p.TenantID, &imgURL, &p.Description, &p.PointsRequired, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.ImageURL = "/api/products/image?id=" + p.ID
	if len(imageData) == 0 && imgURL.Valid {
		p.ImageURL = imgURL.String
	}
	return &p, nil
}

func (r *ProductRepository) Update(id, imageURL, description string, pointsRequired int, imageData []byte, imageContentType string) (*model.Product, error) {
	if len(imageData) > 0 {
		q := `UPDATE products SET image_url = $1, image_data = $2, image_content_type = $3, description = $4, points_required = $5, updated_at = CURRENT_TIMESTAMP WHERE id = $6`
		if r.driver == "sqlite3" {
			q = `UPDATE products SET image_url = $1, image_data = $2, image_content_type = $3, description = $4, points_required = $5, updated_at = datetime('now') WHERE id = $6`
		}
		q = db.QueryForDriver(q, r.driver)
		_, err := r.db.Exec(q, nil, imageData, imageContentType, description, pointsRequired, id)
		if err != nil {
			return nil, err
		}
	} else {
		q := `UPDATE products SET image_url = $1, description = $2, points_required = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $4`
		if r.driver == "sqlite3" {
			q = `UPDATE products SET image_url = $1, description = $2, points_required = $3, updated_at = datetime('now') WHERE id = $4`
		}
		q = db.QueryForDriver(q, r.driver)
		_, err := r.db.Exec(q, nullStr(imageURL), description, pointsRequired, id)
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
