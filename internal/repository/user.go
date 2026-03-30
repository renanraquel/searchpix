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

func scanUserFromRow(row *sql.Row) (*model.User, error) {
	var u model.User
	var fn, cpf, ph sql.NullString
	err := row.Scan(&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.CreatedAt, &fn, &cpf, &ph)
	if err != nil {
		return nil, err
	}
	if fn.Valid {
		u.FullName = fn.String
	}
	if cpf.Valid {
		u.CPF = cpf.String
	}
	if ph.Valid {
		u.Phone = ph.String
	}
	return &u, nil
}

// GetByUsername busca usuário pelo login (username único no sistema para identificar o tenant)
func (r *UserRepository) GetByUsername(username string) (*model.User, error) {
	q := `SELECT id, tenant_id, username, password_hash, created_at, full_name, cpf, phone FROM users WHERE username = $1`
	q = db.QueryForDriver(q, r.driver)
	u, err := scanUserFromRow(r.db.QueryRow(q, username))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) GetByTenantAndUsername(tenantID, username string) (*model.User, error) {
	q := `SELECT id, tenant_id, username, password_hash, created_at, full_name, cpf, phone FROM users WHERE tenant_id = $1 AND username = $2`
	q = db.QueryForDriver(q, r.driver)
	u, err := scanUserFromRow(r.db.QueryRow(q, tenantID, username))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) Create(tenantID, username, passwordHash string) (*model.User, error) {
	return r.CreateWithProfile(tenantID, username, passwordHash, "", "", "")
}

// CreateWithProfile cria usuário; campos de perfil vazios gravam NULL no banco.
func (r *UserRepository) CreateWithProfile(tenantID, username, passwordHash, fullName, cpf, phone string) (*model.User, error) {
	q := `INSERT INTO users (id, tenant_id, username, password_hash, full_name, cpf, phone) VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, '')) RETURNING id, tenant_id, username, created_at`
	q = db.QueryForDriver(q, r.driver)
	id := newUUID()
	var u model.User
	u.PasswordHash = passwordHash
	err := r.db.QueryRow(q, id, tenantID, username, passwordHash, fullName, cpf, phone).Scan(&u.ID, &u.TenantID, &u.Username, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.FullName = fullName
	u.CPF = cpf
	u.Phone = phone
	return &u, nil
}
