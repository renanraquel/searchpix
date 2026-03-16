package repository

import (
	"database/sql"

	"searchpix/internal/db"
	"searchpix/internal/model"
)

type CustomerRepository struct {
	db     *sql.DB
	driver string
}

func NewCustomerRepository(database *sql.DB, driver string) *CustomerRepository {
	return &CustomerRepository{db: database, driver: driver}
}

func (r *CustomerRepository) ListByTenant(tenantID string) ([]model.Customer, error) {
	q := `SELECT id, tenant_id, cpf, name, phone, points_balance, created_at, updated_at FROM customers WHERE tenant_id = $1 ORDER BY name`
	q = db.QueryForDriver(q, r.driver)
	rows, err := r.db.Query(q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.Customer
	for rows.Next() {
		var c model.Customer
		err := rows.Scan(&c.ID, &c.TenantID, &c.CPF, &c.Name, &c.Phone, &c.PointsBalance, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (r *CustomerRepository) GetByID(id string) (*model.Customer, error) {
	q := `SELECT id, tenant_id, cpf, name, phone, points_balance, created_at, updated_at FROM customers WHERE id = $1`
	q = db.QueryForDriver(q, r.driver)
	var c model.Customer
	err := r.db.QueryRow(q, id).Scan(&c.ID, &c.TenantID, &c.CPF, &c.Name, &c.Phone, &c.PointsBalance, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CustomerRepository) GetByTenantAndCPF(tenantID, cpf string) (*model.Customer, error) {
	cpfNorm := normalizeCPF(cpf)
	q := `SELECT id, tenant_id, cpf, name, phone, points_balance, created_at, updated_at FROM customers WHERE tenant_id = $1 AND cpf = $2`
	q = db.QueryForDriver(q, r.driver)
	var c model.Customer
	err := r.db.QueryRow(q, tenantID, cpfNorm).Scan(&c.ID, &c.TenantID, &c.CPF, &c.Name, &c.Phone, &c.PointsBalance, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CustomerRepository) Create(tenantID, cpf, name, phone string) (*model.Customer, error) {
	cpfNorm := normalizeCPF(cpf)
	q := `INSERT INTO customers (id, tenant_id, cpf, name, phone) VALUES ($1, $2, $3, $4, $5) RETURNING id, tenant_id, cpf, name, phone, points_balance, created_at, updated_at`
	q = db.QueryForDriver(q, r.driver)
	id := newUUID()
	var c model.Customer
	err := r.db.QueryRow(q, id, tenantID, cpfNorm, name, phone).Scan(&c.ID, &c.TenantID, &c.CPF, &c.Name, &c.Phone, &c.PointsBalance, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CustomerRepository) Update(id, cpf, name, phone string) (*model.Customer, error) {
	cpfNorm := normalizeCPF(cpf)
	q := `UPDATE customers SET cpf = $1, name = $2, phone = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $4`
	if r.driver == "sqlite3" {
		q = `UPDATE customers SET cpf = $1, name = $2, phone = $3, updated_at = datetime('now') WHERE id = $4`
	}
	q = db.QueryForDriver(q, r.driver)
	_, err := r.db.Exec(q, cpfNorm, name, phone, id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(id)
}

func (r *CustomerRepository) Delete(id string) error {
	q := `DELETE FROM customers WHERE id = $1`
	q = db.QueryForDriver(q, r.driver)
	_, err := r.db.Exec(q, id)
	return err
}

// AddPoints adiciona pontos ao saldo do cliente (usa transação no service)
func (r *CustomerRepository) AddPoints(id string, points int) error {
	q := `UPDATE customers SET points_balance = points_balance + $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	if r.driver == "sqlite3" {
		q = `UPDATE customers SET points_balance = points_balance + $1, updated_at = datetime('now') WHERE id = $2`
	}
	q = db.QueryForDriver(q, r.driver)
	_, err := r.db.Exec(q, points, id)
	return err
}

// SubtractPoints debita pontos do saldo
func (r *CustomerRepository) SubtractPoints(id string, points int) error {
	q := `UPDATE customers SET points_balance = points_balance - $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	if r.driver == "sqlite3" {
		q = `UPDATE customers SET points_balance = points_balance - $1, updated_at = datetime('now') WHERE id = $2`
	}
	q = db.QueryForDriver(q, r.driver)
	_, err := r.db.Exec(q, points, id)
	return err
}

func normalizeCPF(cpf string) string {
	// Remove tudo que não for dígito
	b := make([]byte, 0, len(cpf))
	for i := 0; i < len(cpf); i++ {
		if cpf[i] >= '0' && cpf[i] <= '9' {
			b = append(b, cpf[i])
		}
	}
	return string(b)
}
