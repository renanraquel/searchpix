package repository

import (
	"database/sql"

	"searchpix/internal/db"
	"searchpix/internal/model"
)

type PointsTransactionRepository struct {
	db     *sql.DB
	driver string
}

func NewPointsTransactionRepository(database *sql.DB, driver string) *PointsTransactionRepository {
	return &PointsTransactionRepository{db: database, driver: driver}
}

func (r *PointsTransactionRepository) Create(customerID string, amount int, kind, reference string) (*model.PointsTransaction, error) {
	q := `INSERT INTO points_transactions (id, customer_id, amount, kind, reference) VALUES ($1, $2, $3, $4, $5) RETURNING id, customer_id, amount, kind, reference, created_at`
	q = db.QueryForDriver(q, r.driver)
	id := newUUID()
	var t model.PointsTransaction
	err := r.db.QueryRow(q, id, customerID, amount, kind, reference).Scan(&t.ID, &t.CustomerID, &t.Amount, &t.Kind, &t.Reference, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *PointsTransactionRepository) ListByCustomer(customerID string) ([]model.PointsTransaction, error) {
	q := `SELECT id, customer_id, amount, kind, reference, created_at FROM points_transactions WHERE customer_id = $1 ORDER BY created_at DESC`
	q = db.QueryForDriver(q, r.driver)
	rows, err := r.db.Query(q, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.PointsTransaction
	for rows.Next() {
		var t model.PointsTransaction
		err := rows.Scan(&t.ID, &t.CustomerID, &t.Amount, &t.Kind, &t.Reference, &t.CreatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}
