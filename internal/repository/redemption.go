package repository

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"searchpix/internal/db"
	"searchpix/internal/model"
)

type RedemptionRepository struct {
	db     *sql.DB
	driver string
}

func NewRedemptionRepository(database *sql.DB, driver string) *RedemptionRepository {
	return &RedemptionRepository{db: database, driver: driver}
}

func (r *RedemptionRepository) Create(customerID, productID string, pointsUsed int) (*model.Redemption, error) {
	q := `INSERT INTO redemptions (id, customer_id, product_id, points_used) VALUES ($1, $2, $3, $4) RETURNING id, customer_id, product_id, points_used, created_at`
	q = db.QueryForDriver(q, r.driver)
	id := newUUID()
	var red model.Redemption
	err := r.db.QueryRow(q, id, customerID, productID, pointsUsed).Scan(&red.ID, &red.CustomerID, &red.ProductID, &red.PointsUsed, &red.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &red, nil
}

func (r *RedemptionRepository) ListByCustomer(customerID string) ([]model.RedemptionView, error) {
	q := `SELECT r.id, r.customer_id, r.product_id, r.points_used, r.created_at, p.description as product_description, p.image_url as product_image_url
		  FROM redemptions r
		  JOIN products p ON p.id = r.product_id
		  WHERE r.customer_id = $1 ORDER BY r.created_at DESC`
	q = db.QueryForDriver(q, r.driver)
	rows, err := r.db.Query(q, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.RedemptionView
	for rows.Next() {
		var v model.RedemptionView
		var imgURL sql.NullString
		err := rows.Scan(&v.ID, &v.CustomerID, &v.ProductID, &v.PointsUsed, &v.CreatedAt, &v.ProductDescription, &imgURL)
		if err != nil {
			return nil, err
		}
		if imgURL.Valid {
			v.ProductImageURL = imgURL.String
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

// ListByTenantFiltered lista resgates do tenant com filtros (inicio, fim, nome/cpf), ordenado da data mais antiga à mais recente, paginado
func (r *RedemptionRepository) ListByTenantFiltered(tenantID, inicio, fim, q string, limit, offset int) ([]model.RedemptionListRow, int, error) {
	qNorm := strings.TrimSpace(q)
	cpfOnly := strings.ReplaceAll(qNorm, " ", "")
	allDigits := len(cpfOnly) >= 3
	for _, c := range cpfOnly {
		if c < '0' || c > '9' {
			allDigits = false
			break
		}
	}

	ph := func(i int) string {
		if r.driver == "sqlite3" {
			return "?"
		}
		return "$" + strconv.Itoa(i)
	}

	sBase := `SELECT r.id, c.name, c.cpf, c.phone, r.points_used, r.created_at, p.description FROM redemptions r JOIN customers c ON c.id = r.customer_id JOIN products p ON p.id = r.product_id WHERE c.tenant_id = ` + ph(1)
	cBase := `SELECT COUNT(*) FROM redemptions r JOIN customers c ON c.id = r.customer_id WHERE c.tenant_id = ` + ph(1)
	argsSelect := []interface{}{tenantID}
	argsCount := []interface{}{tenantID}
	pos := 2

	// Usar "YYYY-MM-DD 00:00:00" (espaço, não "T") para comparação correta no SQLite (ordem lexicográfica) e Postgres
	if inicio != "" {
		sBase += " AND r.created_at >= " + ph(pos)
		cBase += " AND r.created_at >= " + ph(pos)
		argsSelect = append(argsSelect, inicio+" 00:00:00")
		argsCount = append(argsCount, inicio+" 00:00:00")
		pos++
	}
	if fim != "" {
		// Limite superior exclusivo: dia seguinte 00:00:00 para incluir o dia inteiro de fim (ex.: 16/03 início e fim traz 16/03)
		t, err := time.Parse("2006-01-02", fim)
		if err != nil {
			t = time.Time{} // fallback evita panic; filtro pode ficar estranho
		}
		t = t.AddDate(0, 0, 1)
		fimExclusive := t.Format("2006-01-02") + " 00:00:00"
		sBase += " AND r.created_at < " + ph(pos)
		cBase += " AND r.created_at < " + ph(pos)
		argsSelect = append(argsSelect, fimExclusive)
		argsCount = append(argsCount, fimExclusive)
		pos++
	}
	if qNorm != "" {
		if allDigits {
			sBase += " AND c.cpf = " + ph(pos)
			cBase += " AND c.cpf = " + ph(pos)
			argsSelect = append(argsSelect, cpfOnly)
			argsCount = append(argsCount, cpfOnly)
		} else {
			likeVal := "%" + qNorm + "%"
			if r.driver == "sqlite3" {
				sBase += " AND (LOWER(c.name) LIKE LOWER(?) OR c.name LIKE ?)"
				cBase += " AND (LOWER(c.name) LIKE LOWER(?) OR c.name LIKE ?)"
				argsSelect = append(argsSelect, likeVal, likeVal)
				argsCount = append(argsCount, likeVal, likeVal)
			} else {
				sBase += " AND c.name ILIKE " + ph(pos)
				cBase += " AND c.name ILIKE " + ph(pos)
				argsSelect = append(argsSelect, likeVal)
				argsCount = append(argsCount, likeVal)
			}
		}
		pos++
	}

	sBase += " ORDER BY r.created_at ASC LIMIT " + ph(pos) + " OFFSET " + ph(pos+1)
	argsSelect = append(argsSelect, limit, offset)

	selectQ := db.QueryForDriver(sBase, r.driver)
	countQ := db.QueryForDriver(cBase, r.driver)

	var total int
	if err := r.db.QueryRow(countQ, argsCount...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(selectQ, argsSelect...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []model.RedemptionListRow
	for rows.Next() {
		var row model.RedemptionListRow
		var desc sql.NullString
		if err := rows.Scan(&row.ID, &row.CustomerName, &row.CPF, &row.Phone, &row.PointsUsed, &row.CreatedAt, &desc); err != nil {
			return nil, 0, err
		}
		if desc.Valid {
			row.ProductDescription = desc.String
		}
		list = append(list, row)
	}
	return list, total, rows.Err()
}
