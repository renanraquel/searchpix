package repository

import (
	"database/sql"

	"searchpix/internal/db"
	"searchpix/internal/model"
)

type UserRepository struct {
	db     *sql.DB
	driver string
}

func NewUserRepository(database *sql.DB, driver string) *UserRepository {
	return &UserRepository{db: database, driver: driver}
}

// GetByUsername busca usuário pelo login (username único no sistema para identificar o tenant)
func (r *UserRepository) GetByUsername(username string) (*model.User, error) {
	q := `SELECT id, tenant_id, username, password_hash, created_at FROM users WHERE username = $1`
	q = db.QueryForDriver(q, r.driver)
	var u model.User
	err := r.db.QueryRow(q, username).Scan(&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByTenantAndUsername(tenantID, username string) (*model.User, error) {
	q := `SELECT id, tenant_id, username, password_hash, created_at FROM users WHERE tenant_id = $1 AND username = $2`
	q = db.QueryForDriver(q, r.driver)
	var u model.User
	err := r.db.QueryRow(q, tenantID, username).Scan(&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Create(tenantID, username, passwordHash string) (*model.User, error) {
	q := `INSERT INTO users (id, tenant_id, username, password_hash) VALUES ($1, $2, $3, $4) RETURNING id, tenant_id, username, created_at`
	q = db.QueryForDriver(q, r.driver)
	id := newUUID()
	var u model.User
	u.PasswordHash = passwordHash
	err := r.db.QueryRow(q, id, tenantID, username, passwordHash).Scan(&u.ID, &u.TenantID, &u.Username, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
