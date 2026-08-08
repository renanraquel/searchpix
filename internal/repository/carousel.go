package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"searchpix/internal/db"
	"searchpix/internal/model"
)

func carouselLegacyMediaURL(id, tenantSlug string, updatedAt model.FlexTime) string {
	v := updatedAt.Unix()
	if tenantSlug != "" {
		return fmt.Sprintf("/api/public/carousel/media?id=%s&tenant=%s&v=%d", id, tenantSlug, v)
	}
	return fmt.Sprintf("/api/public/carousel/media?id=%s&v=%d", id, v)
}

func resolveCarouselMediaURL(storedURL, id, tenantSlug string, updatedAt model.FlexTime) string {
	if strings.TrimSpace(storedURL) != "" {
		return strings.TrimSpace(storedURL)
	}
	return carouselLegacyMediaURL(id, tenantSlug, updatedAt)
}

type CarouselRepository struct {
	db     *sql.DB
	driver string
}

func NewCarouselRepository(database *sql.DB, driver string) *CarouselRepository {
	return &CarouselRepository{db: database, driver: driver}
}

func (r *CarouselRepository) ListByTenant(tenantID string) ([]model.CarouselItem, error) {
	q := `SELECT id, tenant_id, media_type, title, sort_order, created_at, updated_at, COALESCE(storage_key,''), COALESCE(media_url,'')
		FROM carousel_items WHERE tenant_id = $1 ORDER BY sort_order ASC, created_at ASC`
	q = db.QueryForDriver(q, r.driver)
	rows, err := r.db.Query(q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.CarouselItem
	for rows.Next() {
		var item model.CarouselItem
		var title sql.NullString
		var storageKey, mediaURL string
		if err := rows.Scan(&item.ID, &item.TenantID, &item.MediaType, &title, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt, &storageKey, &mediaURL); err != nil {
			return nil, err
		}
		if title.Valid {
			item.Title = title.String
		}
		item.StorageKey = storageKey
		item.MediaURL = resolveCarouselMediaURL(mediaURL, item.ID, "", item.UpdatedAt)
		list = append(list, item)
	}
	return list, rows.Err()
}

func (r *CarouselRepository) ListByTenantSlug(slug string) ([]model.CarouselItem, string, error) {
	q := `SELECT ci.id, ci.tenant_id, ci.media_type, ci.title, ci.sort_order, ci.created_at, ci.updated_at,
		COALESCE(ci.storage_key,''), COALESCE(ci.media_url,''), t.slug
		FROM carousel_items ci
		JOIN tenants t ON t.id = ci.tenant_id
		WHERE t.slug = $1
		ORDER BY ci.sort_order ASC, ci.created_at ASC`
	q = db.QueryForDriver(q, r.driver)
	rows, err := r.db.Query(q, slug)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	var list []model.CarouselItem
	var tenantID string
	for rows.Next() {
		var item model.CarouselItem
		var title sql.NullString
		var storageKey, mediaURL, tenantSlug string
		if err := rows.Scan(&item.ID, &item.TenantID, &item.MediaType, &title, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt, &storageKey, &mediaURL, &tenantSlug); err != nil {
			return nil, "", err
		}
		if title.Valid {
			item.Title = title.String
		}
		tenantID = item.TenantID
		item.StorageKey = storageKey
		item.MediaURL = resolveCarouselMediaURL(mediaURL, item.ID, tenantSlug, item.UpdatedAt)
		list = append(list, item)
	}
	return list, tenantID, rows.Err()
}

func (r *CarouselRepository) GetByID(id string) (*model.CarouselItem, error) {
	q := `SELECT id, tenant_id, media_type, title, sort_order, created_at, updated_at, COALESCE(storage_key,''), COALESCE(media_url,'') FROM carousel_items WHERE id = $1`
	q = db.QueryForDriver(q, r.driver)
	var item model.CarouselItem
	var title sql.NullString
	var storageKey, mediaURL string
	err := r.db.QueryRow(q, id).Scan(&item.ID, &item.TenantID, &item.MediaType, &title, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt, &storageKey, &mediaURL)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if title.Valid {
		item.Title = title.String
	}
	item.StorageKey = storageKey
	item.MediaURL = resolveCarouselMediaURL(mediaURL, item.ID, "", item.UpdatedAt)
	return &item, nil
}

func (r *CarouselRepository) GetMediaByIDAndTenantSlug(itemID, tenantSlug string) ([]byte, string, string, string, time.Time, error) {
	q := `SELECT ci.media_data, ci.content_type, ci.media_type, COALESCE(ci.media_url,''), ci.updated_at
		FROM carousel_items ci
		JOIN tenants t ON t.id = ci.tenant_id
		WHERE ci.id = $1 AND t.slug = $2`
	q = db.QueryForDriver(q, r.driver)
	var data []byte
	var contentType sql.NullString
	var mediaType, mediaURL string
	var updatedAt model.FlexTime
	err := r.db.QueryRow(q, itemID, tenantSlug).Scan(&data, &contentType, &mediaType, &mediaURL, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, "", "", "", time.Time{}, nil
	}
	if err != nil {
		return nil, "", "", "", time.Time{}, err
	}
	ct := "application/octet-stream"
	if contentType.Valid && contentType.String != "" {
		ct = contentType.String
	}
	return data, ct, mediaType, mediaURL, updatedAt.Time, nil
}

// ListPendingR2Migration retorna itens ainda com blob local e sem URL R2.
func (r *CarouselRepository) ListPendingR2Migration() ([]CarouselBlobRow, error) {
	q := `SELECT id, tenant_id, media_type, content_type, media_data
		FROM carousel_items
		WHERE (media_url IS NULL OR media_url = '')
		  AND media_data IS NOT NULL`
	q = db.QueryForDriver(q, r.driver)
	rows, err := r.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []CarouselBlobRow
	for rows.Next() {
		var row CarouselBlobRow
		var ct sql.NullString
		if err := rows.Scan(&row.ID, &row.TenantID, &row.MediaType, &ct, &row.Data); err != nil {
			return nil, err
		}
		if ct.Valid {
			row.ContentType = ct.String
		}
		if len(row.Data) == 0 {
			continue
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

type CarouselBlobRow struct {
	ID          string
	TenantID    string
	MediaType   string
	ContentType string
	Data        []byte
}

func (r *CarouselRepository) MarkR2Migrated(id, storageKey, mediaURL string) error {
	q := `UPDATE carousel_items SET storage_key = $1, media_url = $2, media_data = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = $3`
	if r.driver == "sqlite3" {
		q = `UPDATE carousel_items SET storage_key = $1, media_url = $2, media_data = NULL, updated_at = datetime('now') WHERE id = $3`
	}
	q = db.QueryForDriver(q, r.driver)
	_, err := r.db.Exec(q, storageKey, mediaURL, id)
	return err
}

func (r *CarouselRepository) NextSortOrder(tenantID string) (int, error) {
	q := `SELECT COALESCE(MAX(sort_order), -1) + 1 FROM carousel_items WHERE tenant_id = $1`
	q = db.QueryForDriver(q, r.driver)
	var next int
	if err := r.db.QueryRow(q, tenantID).Scan(&next); err != nil {
		return 0, err
	}
	return next, nil
}

func (r *CarouselRepository) Reorder(tenantID string, itemIDs []string) error {
	for i, id := range itemIDs {
		q := `UPDATE carousel_items SET sort_order = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND tenant_id = $3`
		if r.driver == "sqlite3" {
			q = `UPDATE carousel_items SET sort_order = $1, updated_at = datetime('now') WHERE id = $2 AND tenant_id = $3`
		}
		q = db.QueryForDriver(q, r.driver)
		res, err := r.db.Exec(q, i, id, tenantID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return sql.ErrNoRows
		}
	}
	return nil
}

func (r *CarouselRepository) Create(tenantID, mediaType, title string, sortOrder int, mediaData []byte, contentType, storageKey, mediaURL string) (*model.CarouselItem, error) {
	return r.CreateWithID(newUUID(), tenantID, mediaType, title, sortOrder, mediaData, contentType, storageKey, mediaURL)
}

func (r *CarouselRepository) CreateWithID(id, tenantID, mediaType, title string, sortOrder int, mediaData []byte, contentType, storageKey, mediaURL string) (*model.CarouselItem, error) {
	if id == "" {
		id = newUUID()
	}
	q := `INSERT INTO carousel_items (id, tenant_id, media_type, title, sort_order, media_data, content_type, storage_key, media_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, tenant_id, media_type, title, sort_order, created_at, updated_at, COALESCE(storage_key,''), COALESCE(media_url,'')`
	q = db.QueryForDriver(q, r.driver)
	var item model.CarouselItem
	var titleNull sql.NullString
	var sk, mu string
	var blob interface{}
	if len(mediaData) > 0 {
		blob = mediaData
	} else {
		blob = nil
	}
	err := r.db.QueryRow(q, id, tenantID, mediaType, nullStr(title), sortOrder, blob, nullStr(contentType), nullStr(storageKey), nullStr(mediaURL)).
		Scan(&item.ID, &item.TenantID, &item.MediaType, &titleNull, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt, &sk, &mu)
	if err != nil {
		return nil, err
	}
	if titleNull.Valid {
		item.Title = titleNull.String
	}
	item.StorageKey = sk
	item.MediaURL = resolveCarouselMediaURL(mu, item.ID, "", item.UpdatedAt)
	return &item, nil
}

func (r *CarouselRepository) Update(id, title string, sortOrder int, mediaData []byte, contentType, mediaType, storageKey, mediaURL string) (*model.CarouselItem, error) {
	if len(mediaData) > 0 || storageKey != "" || mediaURL != "" {
		q := `UPDATE carousel_items SET title = $1, sort_order = $2, media_data = $3, content_type = $4, media_type = $5,
			storage_key = $6, media_url = $7, updated_at = CURRENT_TIMESTAMP WHERE id = $8`
		if r.driver == "sqlite3" {
			q = `UPDATE carousel_items SET title = $1, sort_order = $2, media_data = $3, content_type = $4, media_type = $5,
				storage_key = $6, media_url = $7, updated_at = datetime('now') WHERE id = $8`
		}
		q = db.QueryForDriver(q, r.driver)
		var blob interface{}
		if len(mediaData) > 0 {
			blob = mediaData
		} else {
			blob = nil
		}
		if _, err := r.db.Exec(q, nullStr(title), sortOrder, blob, nullStr(contentType), mediaType, nullStr(storageKey), nullStr(mediaURL), id); err != nil {
			return nil, err
		}
	} else {
		q := `UPDATE carousel_items SET title = $1, sort_order = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`
		if r.driver == "sqlite3" {
			q = `UPDATE carousel_items SET title = $1, sort_order = $2, updated_at = datetime('now') WHERE id = $3`
		}
		q = db.QueryForDriver(q, r.driver)
		if _, err := r.db.Exec(q, nullStr(title), sortOrder, id); err != nil {
			return nil, err
		}
	}
	return r.GetByID(id)
}

func (r *CarouselRepository) Delete(id string) error {
	q := `DELETE FROM carousel_items WHERE id = $1`
	q = db.QueryForDriver(q, r.driver)
	_, err := r.db.Exec(q, id)
	return err
}

func (r *CarouselRepository) GetSettings(tenantID string) (*model.CarouselSettings, error) {
	q := `SELECT tenant_id, image_duration_seconds FROM carousel_settings WHERE tenant_id = $1`
	q = db.QueryForDriver(q, r.driver)
	var s model.CarouselSettings
	err := r.db.QueryRow(q, tenantID).Scan(&s.TenantID, &s.ImageDurationSeconds)
	if err == sql.ErrNoRows {
		return &model.CarouselSettings{TenantID: tenantID, ImageDurationSeconds: 20}, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *CarouselRepository) GetSettingsByTenantSlug(slug string) (*model.CarouselSettings, error) {
	q := `SELECT cs.tenant_id, cs.image_duration_seconds
		FROM carousel_settings cs
		JOIN tenants t ON t.id = cs.tenant_id
		WHERE t.slug = $1`
	q = db.QueryForDriver(q, r.driver)
	var s model.CarouselSettings
	err := r.db.QueryRow(q, slug).Scan(&s.TenantID, &s.ImageDurationSeconds)
	if err == sql.ErrNoRows {
		return &model.CarouselSettings{ImageDurationSeconds: 20}, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *CarouselRepository) UpsertSettings(tenantID string, imageDurationSeconds int) (*model.CarouselSettings, error) {
	if imageDurationSeconds < 1 {
		imageDurationSeconds = 20
	}
	if r.driver == "postgres" {
		q := `INSERT INTO carousel_settings (tenant_id, image_duration_seconds, updated_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (tenant_id) DO UPDATE SET
				image_duration_seconds = EXCLUDED.image_duration_seconds,
				updated_at = NOW()
			RETURNING tenant_id, image_duration_seconds`
		var s model.CarouselSettings
		err := r.db.QueryRow(q, tenantID, imageDurationSeconds).Scan(&s.TenantID, &s.ImageDurationSeconds)
		if err != nil {
			return nil, err
		}
		return &s, nil
	}
	q := `INSERT INTO carousel_settings (tenant_id, image_duration_seconds, updated_at)
		VALUES (?, ?, datetime('now'))
		ON CONFLICT(tenant_id) DO UPDATE SET
			image_duration_seconds = excluded.image_duration_seconds,
			updated_at = datetime('now')`
	if _, err := r.db.Exec(q, tenantID, imageDurationSeconds); err != nil {
		return nil, err
	}
	return r.GetSettings(tenantID)
}
