package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	customerv1 "github.com/pos-stery/pos-stery/gen/go/pos/customer/v1"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/customer-service/internal/application/commands"
	"github.com/pos-stery/pos-stery/services/customer-service/internal/application/queries"
	"github.com/pos-stery/pos-stery/services/customer-service/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CustomerServiceServer implements customerv1.CustomerServiceServer.
type CustomerServiceServer struct {
	customerv1.UnimplementedCustomerServiceServer
	createHandler  *commands.CreateCustomerHandler
	loyaltyHandler *commands.AddLoyaltyPointsHandler
	queryHandler   *queries.CustomerQueryHandler
}

// NewCustomerServiceServer constructs the gRPC server with all handlers.
func NewCustomerServiceServer(
	create *commands.CreateCustomerHandler,
	loyalty *commands.AddLoyaltyPointsHandler,
	query *queries.CustomerQueryHandler,
) *CustomerServiceServer {
	return &CustomerServiceServer{
		createHandler:  create,
		loyaltyHandler: loyalty,
		queryHandler:   query,
	}
}

// CreateCustomer handles the CreateCustomer RPC.
func (s *CustomerServiceServer) CreateCustomer(
	ctx context.Context,
	req *customerv1.CreateCustomerRequest,
) (*customerv1.CreateCustomerResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(fmt.Errorf("invalid tenant_id: %w", sherrors.ErrInvalidArgument))
	}

	c, err := s.createHandler.Handle(ctx, commands.CreateCustomerCommand{
		TenantID: tenantID,
		Name:     req.Name,
		Phone:    req.Phone,
		Email:    req.Email,
	})
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	return &customerv1.CreateCustomerResponse{Customer: domainToProto(c, 0)}, nil
}

// GetCustomer handles the GetCustomer RPC.
func (s *CustomerServiceServer) GetCustomer(
	ctx context.Context,
	req *customerv1.GetCustomerRequest,
) (*customerv1.GetCustomerResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(fmt.Errorf("invalid tenant_id: %w", sherrors.ErrInvalidArgument))
	}
	customerID, err := uuid.Parse(req.CustomerId)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(fmt.Errorf("invalid customer_id: %w", sherrors.ErrInvalidArgument))
	}

	c, err := s.queryHandler.GetCustomer(ctx, queries.GetCustomerQuery{
		TenantID:   tenantID,
		CustomerID: customerID,
	})
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	totalPoints, err := s.queryHandler.GetTotalPoints(ctx, tenantID, customerID)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	return &customerv1.GetCustomerResponse{Customer: domainToProto(c, totalPoints)}, nil
}

// ListCustomers handles the ListCustomers RPC.
func (s *CustomerServiceServer) ListCustomers(
	ctx context.Context,
	req *customerv1.ListCustomersRequest,
) (*customerv1.ListCustomersResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(fmt.Errorf("invalid tenant_id: %w", sherrors.ErrInvalidArgument))
	}

	result, err := s.queryHandler.ListCustomers(ctx, queries.ListCustomersQuery{
		TenantID: tenantID,
		Query:    req.Query,
		Limit:    int(req.Limit),
		Offset:   int(req.Offset),
	})
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	protos := make([]*customerv1.Customer, 0, len(result.Customers))
	for _, c := range result.Customers {
		protos = append(protos, domainToProto(c, 0))
	}

	return &customerv1.ListCustomersResponse{
		Customers: protos,
		Total:     int32(result.Total),
	}, nil
}

// AddLoyaltyPoints handles the AddLoyaltyPoints RPC.
// Per PRD §13, points are only added after a successful sale — this RPC is
// called by sales-service after Sale.Completed.
func (s *CustomerServiceServer) AddLoyaltyPoints(
	ctx context.Context,
	req *customerv1.AddLoyaltyPointsRequest,
) (*customerv1.AddLoyaltyPointsResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(fmt.Errorf("invalid tenant_id: %w", sherrors.ErrInvalidArgument))
	}
	customerID, err := uuid.Parse(req.CustomerId)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(fmt.Errorf("invalid customer_id: %w", sherrors.ErrInvalidArgument))
	}
	saleID, err := uuid.Parse(req.SaleId)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(fmt.Errorf("invalid sale_id: %w", sherrors.ErrInvalidArgument))
	}

	result, err := s.loyaltyHandler.Handle(ctx, commands.AddLoyaltyPointsCommand{
		TenantID:   tenantID,
		CustomerID: customerID,
		SaleID:     saleID,
		Points:     int(req.Points),
	})
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	return &customerv1.AddLoyaltyPointsResponse{
		Customer:    domainToProto(result.Customer, result.TotalPoints),
		TotalPoints: int32(result.TotalPoints),
	}, nil
}

// domainToProto converts a domain.Customer to its protobuf representation.
func domainToProto(c *domain.Customer, loyaltyPoints int) *customerv1.Customer {
	return &customerv1.Customer{
		CustomerId:    c.ID.String(),
		TenantId:      c.TenantID.String(),
		Name:          c.Name,
		Phone:         c.Phone,
		Email:         c.Email,
		LoyaltyPoints: int32(loyaltyPoints),
		CreatedAt:     timestamppb.New(c.CreatedAt),
		UpdatedAt:     timestamppb.New(c.UpdatedAt),
	}
}
