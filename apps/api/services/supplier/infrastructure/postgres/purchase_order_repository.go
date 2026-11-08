package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Briyantama/POS-Stery/services/supplier/domain"
	"github.com/Briyantama/stery-kit/go/pgtenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PurchaseOrderRepository implements domain.PurchaseOrderRepository using PostgreSQL.
type PurchaseOrderRepository struct {
	pool *pgxpool.Pool
}

// NewPurchaseOrderRepository creates a PurchaseOrderRepository.
func NewPurchaseOrderRepository(pool *pgxpool.Pool) *PurchaseOrderRepository {
	return &PurchaseOrderRepository{pool: pool}
}

// Create inserts a purchase order and its line items in one transaction.
func (r *PurchaseOrderRepository) Create(ctx context.Context, order *domain.PurchaseOrder) error {
	return pgtenant.WithTenantTx(ctx, r.pool, order.TenantID.String(), order.StoreID.String(), func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			INSERT INTO supplier.purchase_orders
				(id, tenant_id, store_id, supplier_id, status, notes)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING created_at, updated_at
		`, order.ID, order.TenantID, order.StoreID, order.SupplierID, string(order.Status), order.Notes)

		if err := row.Scan(&order.CreatedAt, &order.UpdatedAt); err != nil {
			return fmt.Errorf("insert purchase order: %w", err)
		}

		for _, item := range order.Items {
			row := tx.QueryRow(ctx, `
				INSERT INTO supplier.purchase_order_items
					(tenant_id, purchase_order_id, product_id, quantity_ordered, quantity_received, unit_cost)
				VALUES ($1, $2, $3, $4, $5, $6)
				RETURNING id
			`, item.TenantID, item.PurchaseOrderID, item.ProductID,
				item.QuantityOrdered, item.QuantityReceived, item.UnitCost)

			if err := row.Scan(&item.ID); err != nil {
				return fmt.Errorf("insert purchase order item: %w", err)
			}
		}
		return nil
	})
}

// Get retrieves a purchase order with all its items.
func (r *PurchaseOrderRepository) Get(ctx context.Context, tenantID, storeID, orderID uuid.UUID) (*domain.PurchaseOrder, error) {
	var result *domain.PurchaseOrder

	err := pgtenant.WithTenantTx(ctx, r.pool, tenantID.String(), storeID.String(), func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT id, tenant_id, store_id, supplier_id, status, notes, created_at, updated_at
			FROM supplier.purchase_orders
			WHERE tenant_id = $1 AND store_id = $2 AND id = $3
		`, tenantID, storeID, orderID)

		order := &domain.PurchaseOrder{}
		var statusStr string
		if err := row.Scan(
			&order.ID, &order.TenantID, &order.StoreID, &order.SupplierID,
			&statusStr, &order.Notes, &order.CreatedAt, &order.UpdatedAt,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("purchase order: %w", domain.ErrNotFound)
			}
			return fmt.Errorf("get purchase order: %w", err)
		}
		order.Status = domain.PurchaseOrderStatus(statusStr)

		// Load line items.
		rows, err := tx.Query(ctx, `
			SELECT id, tenant_id, purchase_order_id, product_id,
			       quantity_ordered, quantity_received, unit_cost
			FROM supplier.purchase_order_items
			WHERE tenant_id = $1 AND purchase_order_id = $2
			ORDER BY id
		`, tenantID, orderID)
		if err != nil {
			return fmt.Errorf("list order items: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			item := &domain.PurchaseOrderItem{}
			if err := rows.Scan(
				&item.ID, &item.TenantID, &item.PurchaseOrderID, &item.ProductID,
				&item.QuantityOrdered, &item.QuantityReceived, &item.UnitCost,
			); err != nil {
				return fmt.Errorf("scan order item: %w", err)
			}
			order.Items = append(order.Items, item)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		result = order
		return nil
	})

	return result, err
}

// List returns a paginated list of purchase orders, optionally filtered by status.
func (r *PurchaseOrderRepository) List(
	ctx context.Context,
	tenantID, storeID uuid.UUID,
	status string,
	limit, offset int,
) ([]*domain.PurchaseOrder, int, error) {
	var orders []*domain.PurchaseOrder
	var total int

	err := pgtenant.WithTenantTx(ctx, r.pool, tenantID.String(), storeID.String(), func(tx pgx.Tx) error {
		baseWhere := "WHERE tenant_id = $1 AND store_id = $2"
		args := []any{tenantID, storeID}

		if status != "" {
			baseWhere += fmt.Sprintf(" AND status = $%d", len(args)+1)
			args = append(args, status)
		}

		countSQL := "SELECT COUNT(*) FROM supplier.purchase_orders " + baseWhere
		if err := tx.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
			return fmt.Errorf("count purchase orders: %w", err)
		}

		if limit <= 0 || limit > 200 {
			limit = 50
		}

		selectSQL := `SELECT id, tenant_id, store_id, supplier_id, status, notes, created_at, updated_at
			FROM supplier.purchase_orders ` + baseWhere +
			fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
		args = append(args, limit, offset)

		rows, err := tx.Query(ctx, selectSQL, args...)
		if err != nil {
			return fmt.Errorf("list purchase orders: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			order := &domain.PurchaseOrder{}
			var statusStr string
			if err := rows.Scan(
				&order.ID, &order.TenantID, &order.StoreID, &order.SupplierID,
				&statusStr, &order.Notes, &order.CreatedAt, &order.UpdatedAt,
			); err != nil {
				return fmt.Errorf("scan purchase order: %w", err)
			}
			order.Status = domain.PurchaseOrderStatus(statusStr)
			orders = append(orders, order)
		}
		return rows.Err()
	})

	return orders, total, err
}

// ReceiveOrder books a goods receipt in one transaction. The order row is locked FOR UPDATE and
// must be PENDING, so concurrent or repeated receipts of the same order cannot both apply.
// Stock is written to inventory.stock_items in the same transaction (same tenant/store RLS
// context) so the receipt and the stock increment commit or roll back together.
func (r *PurchaseOrderRepository) ReceiveOrder(
	ctx context.Context,
	tenantID, storeID, orderID uuid.UUID,
	lines []domain.ReceivedLine,
) (*domain.PurchaseOrder, []domain.ReceivedLine, error) {
	var applied []domain.ReceivedLine

	err := pgtenant.WithTenantTx(ctx, r.pool, tenantID.String(), storeID.String(), func(tx pgx.Tx) error {
		var status string
		if err := tx.QueryRow(ctx, `
			SELECT status FROM supplier.purchase_orders
			WHERE tenant_id = $1 AND store_id = $2 AND id = $3
			FOR UPDATE
		`, tenantID, storeID, orderID).Scan(&status); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("purchase order: %w", domain.ErrNotFound)
			}
			return fmt.Errorf("lock purchase order: %w", err)
		}
		if domain.PurchaseOrderStatus(status) != domain.StatusPending {
			return fmt.Errorf("order is %s, only PENDING orders can be received: %w", status, domain.ErrConflict)
		}

		for _, line := range lines {
			tag, err := tx.Exec(ctx, `
				UPDATE supplier.purchase_order_items
				SET quantity_received = quantity_received + $1
				WHERE tenant_id = $2 AND purchase_order_id = $3 AND product_id = $4
			`, line.Quantity, tenantID, orderID, line.ProductID)
			if err != nil {
				return fmt.Errorf("update received quantity for product %s: %w", line.ProductID, err)
			}
			if tag.RowsAffected() == 0 {
				return fmt.Errorf("product %s is not on this order: %w", line.ProductID, domain.ErrInvalidArgument)
			}

			tag, err = tx.Exec(ctx, `
				UPDATE inventory.stock_items
				SET quantity = quantity + $1, updated_at = NOW()
				WHERE tenant_id = $2 AND store_id = $3 AND product_id = $4
			`, int64(line.Quantity), tenantID, storeID, line.ProductID)
			if err != nil {
				return fmt.Errorf("increase stock for product %s: %w", line.ProductID, err)
			}
			if tag.RowsAffected() == 0 {
				return fmt.Errorf("inventory item for product %s: %w", line.ProductID, domain.ErrNotFound)
			}
			applied = append(applied, line)
		}

		if _, err := tx.Exec(ctx, `
			UPDATE supplier.purchase_orders
			SET status = $1, updated_at = NOW()
			WHERE tenant_id = $2 AND id = $3
		`, string(domain.StatusReceived), tenantID, orderID); err != nil {
			return fmt.Errorf("update order status: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	order, err := r.Get(ctx, tenantID, storeID, orderID)
	if err != nil {
		return nil, nil, err
	}
	return order, applied, nil
}
