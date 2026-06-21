package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/customer-service/internal/application/commands"
	"github.com/pos-stery/pos-stery/services/customer-service/internal/domain"
)

// ── stubs ──────────────────────────────────────────────────────────────────────

type stubFindCustomerRepo struct {
	customer *domain.Customer
	findErr  error
}

func (r *stubFindCustomerRepo) Create(_ context.Context, _ uuid.UUID, _ *domain.Customer) error {
	return nil
}

func (r *stubFindCustomerRepo) FindByID(_ context.Context, _, _ uuid.UUID) (*domain.Customer, error) {
	return r.customer, r.findErr
}

func (r *stubFindCustomerRepo) List(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]*domain.Customer, int, error) {
	return nil, 0, nil
}

type stubLoyaltyRepo struct {
	total        int
	getTotalErr  error
	addPointsErr error
}

func (r *stubLoyaltyRepo) AddPoints(_ context.Context, _ uuid.UUID, entry *domain.LoyaltyEntry) (int, error) {
	if r.addPointsErr != nil {
		return 0, r.addPointsErr
	}
	return entry.TotalPoints, nil
}

func (r *stubLoyaltyRepo) GetTotalPoints(_ context.Context, _, _ uuid.UUID) (int, error) {
	return r.total, r.getTotalErr
}

// ── helpers ────────────────────────────────────────────────────────────────────

func existingCustomer() *domain.Customer {
	return &domain.Customer{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Name:     "Loyal Customer",
		IsActive: true,
	}
}

func validLoyaltyCmd(c *domain.Customer, points int) commands.AddLoyaltyPointsCommand {
	return commands.AddLoyaltyPointsCommand{
		TenantID:   c.TenantID,
		CustomerID: c.ID,
		SaleID:     uuid.New(),
		Points:     points,
	}
}

// ── tests ──────────────────────────────────────────────────────────────────────

func TestAddLoyaltyPoints_ZeroPoints(t *testing.T) {
	c := existingCustomer()
	_, err := commands.NewAddLoyaltyPointsHandler(
		&stubFindCustomerRepo{customer: c},
		&stubLoyaltyRepo{},
	).Handle(context.Background(), validLoyaltyCmd(c, 0))

	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument for 0 points, got %v", err)
	}
}

func TestAddLoyaltyPoints_NegativePoints(t *testing.T) {
	c := existingCustomer()
	_, err := commands.NewAddLoyaltyPointsHandler(
		&stubFindCustomerRepo{customer: c},
		&stubLoyaltyRepo{},
	).Handle(context.Background(), validLoyaltyCmd(c, -10))

	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument for negative points, got %v", err)
	}
}

func TestAddLoyaltyPoints_CustomerNotFound(t *testing.T) {
	repo := &stubFindCustomerRepo{findErr: sherrors.ErrNotFound}
	cmd := commands.AddLoyaltyPointsCommand{
		TenantID:   uuid.New(),
		CustomerID: uuid.New(),
		SaleID:     uuid.New(),
		Points:     100,
	}

	_, err := commands.NewAddLoyaltyPointsHandler(repo, &stubLoyaltyRepo{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrNotFound) {
		t.Fatalf("want ErrNotFound when customer missing, got %v", err)
	}
}

func TestAddLoyaltyPoints_GetTotalError(t *testing.T) {
	c := existingCustomer()
	loyalty := &stubLoyaltyRepo{getTotalErr: errors.New("db down")}

	_, err := commands.NewAddLoyaltyPointsHandler(
		&stubFindCustomerRepo{customer: c},
		loyalty,
	).Handle(context.Background(), validLoyaltyCmd(c, 50))

	if err == nil {
		t.Fatal("want error when GetTotalPoints fails")
	}
}

func TestAddLoyaltyPoints_AddPointsRepoError(t *testing.T) {
	c := existingCustomer()
	loyalty := &stubLoyaltyRepo{total: 100, addPointsErr: errors.New("db down")}

	_, err := commands.NewAddLoyaltyPointsHandler(
		&stubFindCustomerRepo{customer: c},
		loyalty,
	).Handle(context.Background(), validLoyaltyCmd(c, 50))

	if err == nil {
		t.Fatal("want error when AddPoints repo fails")
	}
}

func TestAddLoyaltyPoints_HappyPath(t *testing.T) {
	c := existingCustomer()
	loyalty := &stubLoyaltyRepo{total: 200}

	res, err := commands.NewAddLoyaltyPointsHandler(
		&stubFindCustomerRepo{customer: c},
		loyalty,
	).Handle(context.Background(), validLoyaltyCmd(c, 75))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalPoints != 275 {
		t.Errorf("want total=275, got %d", res.TotalPoints)
	}
	if res.Customer == nil {
		t.Error("expected customer in result")
	}
}
