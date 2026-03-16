package model

type Tenant struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Slug      string   `json:"slug"`
	// URL pública da imagem de fundo (montada na API, não vem do banco)
	BackgroundImageURL string   `json:"background_image_url,omitempty"`
	CreatedAt FlexTime `json:"created_at"`
}

// User usuário do sistema (por tenant)
type User struct {
	ID           string   `json:"id"`
	TenantID     string   `json:"tenant_id"`
	Username     string   `json:"username"`
	PasswordHash string   `json:"-"`
	CreatedAt   FlexTime `json:"created_at"`
}

// Product produto resgatável por pontos
type Product struct {
	ID             string   `json:"id"`
	TenantID       string   `json:"tenant_id"`
	ImageURL       string   `json:"image_url"`
	Description    string   `json:"description"`
	PointsRequired int      `json:"points_required"`
	CreatedAt      FlexTime `json:"created_at"`
	UpdatedAt      FlexTime `json:"updated_at"`
}

// Customer cliente da padaria
type Customer struct {
	ID            string   `json:"id"`
	TenantID      string   `json:"tenant_id"`
	CPF           string   `json:"cpf"`
	Name          string   `json:"name"`
	Phone         string   `json:"phone"`
	PointsBalance int      `json:"points_balance"`
	CreatedAt     FlexTime `json:"created_at"`
	UpdatedAt     FlexTime `json:"updated_at"`
}

// PointsTransaction lançamento de pontos (ganho ou resgate)
type PointsTransaction struct {
	ID         string   `json:"id"`
	CustomerID string   `json:"customer_id"`
	Amount     int      `json:"amount"` // positivo = ganho, negativo = resgate
	Kind       string   `json:"kind"`  // "earn" | "redeem"
	Reference  string   `json:"reference"`
	CreatedAt  FlexTime `json:"created_at"`
}

// Redemption resgate de produto
type Redemption struct {
	ID         string   `json:"id"`
	CustomerID string   `json:"customer_id"`
	ProductID  string   `json:"product_id"`
	PointsUsed int      `json:"points_used"`
	CreatedAt  FlexTime `json:"created_at"`
}

// RedemptionView para listagem com dados do produto
type RedemptionView struct {
	Redemption
	ProductDescription string `json:"product_description"`
	ProductImageURL    string `json:"product_image_url,omitempty"`
}

// RedemptionListRow para consulta de resgates (nome, cpf, telefone, pontos, data)
type RedemptionListRow struct {
	ID         string   `json:"id"`
	CustomerName string `json:"customer_name"`
	CPF        string   `json:"cpf"`
	Phone      string   `json:"phone"`
	PointsUsed int      `json:"points_used"`
	CreatedAt  FlexTime `json:"created_at"`
	ProductDescription string `json:"product_description,omitempty"`
}
