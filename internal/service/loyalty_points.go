package service

import (
	"errors"
	"fmt"
	"math"

	"searchpix/internal/model"
	"searchpix/internal/repository"
)

var (
	ErrCustomerNotFound = errors.New("cliente não encontrado")
	ErrProductNotFound  = errors.New("produto não encontrado")
	ErrInsufficientPoints = errors.New("pontos insuficientes")
)

type LoyaltyPointsService struct {
	customerRepo   *repository.CustomerRepository
	productRepo    *repository.ProductRepository
	pointsRepo     *repository.PointsTransactionRepository
	redemptionRepo *repository.RedemptionRepository
}

func NewLoyaltyPointsService(
	customerRepo *repository.CustomerRepository,
	productRepo *repository.ProductRepository,
	pointsRepo *repository.PointsTransactionRepository,
	redemptionRepo *repository.RedemptionRepository,
) *LoyaltyPointsService {
	return &LoyaltyPointsService{
		customerRepo:   customerRepo,
		productRepo:    productRepo,
		pointsRepo:     pointsRepo,
		redemptionRepo: redemptionRepo,
	}
}

// EarnPoints adiciona pontos pelo valor em R$ (1 ponto a cada R$5, sempre arredondando para cima). tenantID para garantir escopo.
func (s *LoyaltyPointsService) EarnPoints(tenantID, cpf string, valueReais float64) (points int, err error) {
	ref := fmt.Sprintf("Compra R$ %.2f", valueReais)
	return s.EarnPointsWithReference(tenantID, cpf, valueReais, ref)
}

// EarnPointsWithReference mesma regra de pontuação, com texto livre no histórico (ex.: NFC-e).
func (s *LoyaltyPointsService) EarnPointsWithReference(tenantID, cpf string, valueReais float64, reference string) (points int, err error) {
	customer, err := s.customerRepo.GetByTenantAndCPF(tenantID, cpf)
	if err != nil {
		return 0, err
	}
	if customer == nil {
		return 0, ErrCustomerNotFound
	}
	points = int(math.Ceil(valueReais / 5.0))
	if points <= 0 {
		return 0, fmt.Errorf("valor deve ser maior que zero")
	}
	if err := s.customerRepo.AddPoints(customer.ID, points); err != nil {
		return 0, err
	}
	_, err = s.pointsRepo.Create(customer.ID, points, "earn", reference)
	return points, err
}

// Redeem cria o resgate e debita os pontos (usa transação no handler via repo)
func (s *LoyaltyPointsService) Redeem(tenantID, customerID, productID string) (*model.Redemption, error) {
	customer, err := s.customerRepo.GetByID(customerID)
	if err != nil || customer == nil || customer.TenantID != tenantID {
		return nil, ErrCustomerNotFound
	}
	product, err := s.productRepo.GetByID(productID)
	if err != nil || product == nil || product.TenantID != tenantID {
		return nil, ErrProductNotFound
	}
	if customer.PointsBalance < product.PointsRequired {
		return nil, ErrInsufficientPoints
	}
	if err := s.customerRepo.SubtractPoints(customerID, product.PointsRequired); err != nil {
		return nil, err
	}
	_, err = s.pointsRepo.Create(customerID, -product.PointsRequired, "redeem", "Resgate: "+product.Description)
	if err != nil {
		// compensar saldo em caso de falha após debitar
		_ = s.customerRepo.AddPoints(customerID, product.PointsRequired)
		return nil, err
	}
	return s.redemptionRepo.Create(customerID, productID, product.PointsRequired)
}
