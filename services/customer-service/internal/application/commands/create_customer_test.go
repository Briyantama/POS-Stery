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

type stubCustomerRepo struct {
	createErr error
}

func (r *stubCustomerRepo) Create(_ context.Context, _ uuid.UUID, _ *domain.Customer) error {
	return r.createErr
}

func (r *stubCustomerRepo) FindByID(_ context.Context, _, _ uuid.UUID) (*domain.Customer, error) {
	return nil, nil
}

func (r *stubCustomerRepo) List(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]*domain.Customer, int, error) {
	return nil, 0, nil
}

type stubCustomerPublisher struct {
	calls int
	err   error
}

func (p *stubCustomerPublisher) PublishCustomerNew(_ context.Context, _, _, _ string) error {
	p.calls++
	return p.err
}

// ── tests ──────────────────────────────────────────────────────────────────────

func TestCreateCustomer_MissingTenantID(t *testing.T) {
	cmd := commands.CreateCustomerCommand{TenantID: uuid.Nil, Name: "Jane"}

	_, err := commands.NewCreateCustomerHandler(&stubCustomerRepo{}, &stubCustomerPublisher{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument for nil tenant_id, got %v", err)
	}
}

func TestCreateCustomer_MissingName(t *testing.T) {
	cmd := commands.CreateCustomerCommand{TenantID: uuid.New(), Name: ""}

	_, err := commands.NewCreateCustomerHandler(&stubCustomerRepo{}, &stubCustomerPublisher{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestCreateCustomer_RepoError(t *testing.T) {
	repo := &stubCustomerRepo{createErr: errors.New("db down")}
	cmd := commands.CreateCustomerCommand{TenantID: uuid.New(), Name: "Jane Doe"}

	_, err := commands.NewCreateCustomerHandler(repo, &stubCustomerPublisher{}).Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("want error when repo fails")
	}
}

func TestCreateCustomer_PublishFailureIsNonFatal(t *testing.T) {
	pub := &stubCustomerPublisher{err: errors.New("nats down")}
	cmd := commands.CreateCustomerCommand{TenantID: uuid.New(), Name: "Jane Doe", Email: "jane@example.com"}

	c, err := commands.NewCreateCustomerHandler(&stubCustomerRepo{}, pub).Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("publish failure must not propagate: %v", err)
	}
	if c == nil {
		t.Fatal("expected customer on success")
	}
}

func TestCreateCustomer_HappyPath(t *testing.T) {
	pub := &stubCustomerPublisher{}
	cmd := commands.CreateCustomerCommand{
		TenantID: uuid.New(),
		Name:     "John Smith",
		Phone:    "08111111111",
		Email:    "john@example.com",
	}

	c, err := commands.NewCreateCustomerHandler(&stubCustomerRepo{}, pub).Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID == uuid.Nil {
		t.Error("customer ID must be set")
	}
	if c.Name != cmd.Name {
		t.Errorf("want name=%q, got %q", cmd.Name, c.Name)
	}
	if !c.IsActive {
		t.Error("new customer must be active")
	}
	if pub.calls != 1 {
		t.Errorf("want 1 publish call, got %d", pub.calls)
	}
}
