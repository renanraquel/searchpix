package repository

import (
	"database/sql"
	"time"

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
	var fn, cpf, ph, email sql.NullString
	var emailVerified sql.NullBool
	err := row.Scan(&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.CreatedAt, &fn, &cpf, &ph, &email, &emailVerified)
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
	if email.Valid {
		u.Email = email.String
	}
	u.EmailVerified = emailVerified.Valid && emailVerified.Bool
	return &u, nil
}

// GetByUsername busca usuário pelo login (username único no sistema para identificar o tenant)
func (r *UserRepository) GetByUsername(username string) (*model.User, error) {
	q := `SELECT id, tenant_id, username, password_hash, created_at, full_name, cpf, phone, email, email_verified FROM users WHERE username = $1`
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
	q := `SELECT id, tenant_id, username, password_hash, created_at, full_name, cpf, phone, email, email_verified FROM users WHERE tenant_id = $1 AND username = $2`
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
	return r.CreateWithProfile(tenantID, username, passwordHash, "", "", "", "")
}

// CreateWithProfile cria usuário; campos de perfil vazios gravam NULL no banco.
func (r *UserRepository) CreateWithProfile(tenantID, username, passwordHash, fullName, cpf, phone, email string) (*model.User, error) {
	q := `INSERT INTO users (id, tenant_id, username, password_hash, full_name, cpf, phone, email, email_verified) VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''), $9) RETURNING id, tenant_id, username, created_at`
	q = db.QueryForDriver(q, r.driver)
	id := newUUID()
	var u model.User
	u.PasswordHash = passwordHash
	emailVerified := email == ""
	err := r.db.QueryRow(q, id, tenantID, username, passwordHash, fullName, cpf, phone, email, emailVerified).Scan(&u.ID, &u.TenantID, &u.Username, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.FullName = fullName
	u.CPF = cpf
	u.Phone = phone
	u.Email = email
	u.EmailVerified = emailVerified
	return &u, nil
}

func (r *UserRepository) GetByEmail(email string) (*model.User, error) {
	q := `SELECT id, tenant_id, username, password_hash, created_at, full_name, cpf, phone, email, email_verified FROM users WHERE email = $1`
	q = db.QueryForDriver(q, r.driver)
	u, err := scanUserFromRow(r.db.QueryRow(q, email))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) CreateEmailVerificationToken(userID, tokenHash string, expiresAt time.Time) error {
	q := `INSERT INTO user_email_verification_tokens (id, user_id, token_hash, expires_at) VALUES ($1, $2, $3, $4)`
	q = db.QueryForDriver(q, r.driver)
	_, err := r.db.Exec(q, newUUID(), userID, tokenHash, expiresAt)
	return err
}

func (r *UserRepository) ConsumeValidEmailVerificationToken(tokenHash string, now time.Time) (string, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	selectQ := db.QueryForDriver(`SELECT user_id FROM user_email_verification_tokens WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2`, r.driver)
	var userID string
	if err := tx.QueryRow(selectQ, tokenHash, now).Scan(&userID); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}

	updateTokenQ := db.QueryForDriver(`UPDATE user_email_verification_tokens SET used_at = $1 WHERE token_hash = $2`, r.driver)
	if _, err := tx.Exec(updateTokenQ, now, tokenHash); err != nil {
		return "", err
	}
	updateUserQ := db.QueryForDriver(`UPDATE users SET email_verified = TRUE WHERE id = $1`, r.driver)
	if _, err := tx.Exec(updateUserQ, userID); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	return userID, nil
}
