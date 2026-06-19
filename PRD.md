# PRD — Multi-Tenant POS SaaS for Retail Stores

## 1. Overview

Build a multi-tenant Point of Sale (POS) SaaS platform for retail stores. The platform must support store chains with multiple stores per tenant, cashier sales flows, inventory and supplier operations, customer loyalty, daily reporting, and AI-ready capabilities for forecasting and recommendations.

The system must be designed for high concurrency, real-time inventory updates, tenant isolation, and future AI expansion.

## 2. Product Vision

Create a stable, scalable, and modular retail operating system where a tenant can manage stores, products, inventory, sales, suppliers, customers, and receipts from one platform.

The product should be:

- fast at checkout,
- safe for multi-tenant use,
- observable and testable,
- ready for AI-driven operations.

## 3. Goals

- Support multiple tenants, each with one or more stores.
- Provide admin and cashier workflows.
- Maintain accurate stock across stores.
- Record every sale and receipt.
- Support supplier ordering and restocking.
- Track customers and loyalty points.
- Generate daily and store-level reports.
- Provide AI services for forecasting, recommendations, and support.
- Enable real-time notifications for low stock and sale events.

## 4. Non-Goals

- No marketplace or multi-vendor commerce.
- No accounting/ERP replacement in phase 1.
- No complex promotions engine beyond basic discounting in the first release.
- No advanced warehouse management system in phase 1.
- No direct payment gateway implementation unless explicitly added later.

## 5. Users and Roles

### 5.1 Tenant Admin

- Manages tenant settings.
- Creates stores.
- Manages products, inventory, suppliers, pricing, and reports.
- Receives low-stock notifications.

### 5.2 Cashier

- Searches products or scans barcodes.
- Creates sales.
- Applies allowed discounts.
- Completes checkout.
- Views or prints receipt.

### 5.3 Stock Manager

- Monitors inventory.
- Creates purchase orders.
- Confirms stock replenishment.

### 5.4 Customer

- May be tracked for loyalty points.
- Receives purchase history or loyalty-based support if enabled.

## 6. Tenant Model

### 6.1 Hierarchy

Tenant hierarchy must be:

- Company / Tenant
  - Store A
  - Store B
  - Store C

### 6.2 Data Isolation

All business data must be scoped by:

- `tenant_id` mandatory
- `store_id` mandatory for store-specific records

All services must enforce tenant filtering at API, service, repository, and event levels.

## 7. Core Modules

### 7.1 Product Catalog

- Create and update products.
- Search by name, SKU, barcode.
- Store category, pricing, barcode, and product metadata.

### 7.2 Inventory Management

- Maintain stock per store.
- Add initial stock.
- Update stock on sale, receipt, or manual adjustment.
- Trigger low-stock alerts.

### 7.3 Sales / Cashier

- Create sale.
- Add line items.
- Apply discounts where allowed.
- Deduct stock.
- Save transaction history.
- Store digital receipt.

### 7.4 Supplier Management

- Manage suppliers.
- Create purchase orders.
- Receive stock into inventory.

### 7.5 Customer and Loyalty

- Create customer records.
- Track loyalty points.
- Allow points accumulation on completed sales.

### 7.6 Pricing and Discounts

- Support base price and discounted price.
- Allow controlled manual discounts.
- Keep pricing changes auditable.

### 7.7 Receipts and Invoices

- Generate digital receipt after completed sale.
- Store receipt reference and sale snapshot.
- Make receipt retrievable by sale ID.

### 7.8 Reports

- Daily sales report by store.
- Daily sales report by tenant.
- Summary of revenue, transaction count, and top products.

## 8. AI Features

### 8.1 Demand Forecasting

- Suggest reorder quantities based on past sales.
- Use historical sales per product and store.

### 8.2 Dynamic Pricing Suggestions

- Suggest price changes based on trend and stock behavior.
- Suggestions only; no automatic forced pricing in phase 1.

### 8.3 Image-to-Product Recognition

- Identify product candidates from uploaded image.
- Return product ID and name suggestions.

### 8.4 Chat Support

- Provide assistant support for product, stock, and report queries.

### 8.5 Trend Analysis

- Identify trending products and store patterns.
- Publish AI outputs as events for consumption by UI or notification service.

## 9. UX / UI Requirements

### 9.1 Admin Dashboard

- Tenant overview.
- Store management.
- Product and inventory screens.
- Supplier and purchase order screens.
- Reports and notifications.

### 9.2 Cashier Interface

- Search products.
- Barcode scanning support.
- Cart and quantity editing.
- Sale completion.
- Receipt display/print.

### 9.3 Frontend Stack

- SvelteKit for dashboards and cashier UI.
- PWA/offline mode is optional but strongly preferred for cashier resilience.

## 10. Architecture

### 10.1 Suggested Stack

- Laravel gateway for auth and HTTP API.
- Go microservices for domain logic.
- gRPC between gateway and services.
- NATS for event-driven communication.
- Postgres for transactional storage.
- Redis for cache/session/ephemeral needs.
- Elasticsearch optional for search.
- MinIO optional for object storage.
- SvelteKit frontend.

### 10.2 Services

- AuthService
- ProductService
- InventoryService
- SalesService
- SupplierService
- CustomerService
- NotificationService
- AIService

### 10.3 Event Bus

NATS subjects:

- `Product.LowStock`
- `Sale.Completed`
- `Inventory.Replenished`
- `Customer.New`
- `AI.OrderForecasted`

## 11. API Contracts

### 11.1 gRPC Services

- AuthService: Login, Validate
- ProductService: CreateProduct, SearchProducts, UpdateProduct
- InventoryService: AddItem, UpdateStock, ListItems, GetItem
- SalesService: CreateSale, GetSalesReport
- SupplierService: AddSupplier, CreatePurchaseOrder, ReceiveStock
- CustomerService: CreateCustomer, GetCustomer, AddLoyaltyPoints
- NotificationService: event consumption / push notifications
- AIService: ForecastDemand, RecognizeProduct, ChatSupport

### 11.2 REST Gateway

- `POST /api/login`
- `POST /api/logout`
- `GET /api/products`
- `POST /api/products`
- `GET /api/inventory`
- `POST /api/inventory`
- `POST /api/sales`
- `GET /api/sales/report`
- `GET /api/suppliers`
- `POST /api/suppliers`
- `POST /api/purchase-orders`
- `POST /api/ai/forecast-demand`

## 12. Data Model

### 12.1 Required Tables

- tenants
- stores
- users
- roles / permissions
- products
- categories
- inventory
- suppliers
- purchase_orders
- purchase_order_items
- customers
- loyalty_points
- sales
- sale_items
- receipts
- notifications
- ai_forecasts
- audit_logs

### 12.2 Required Fields

At minimum, core records must include:

- `tenant_id`
- `store_id` where applicable
- created/updated timestamps
- soft delete where appropriate

## 13. Business Rules

- Stock cannot go negative.
- Sale completion must deduct inventory atomically.
- Low stock event must publish when threshold is crossed.
- A sale must create a receipt record.
- Customer points must be updated only after successful sale.
- Every service must reject cross-tenant access.
- All report queries must respect tenant and store scoping.

## 14. Acceptance Criteria

### 14.1 Product and Inventory

- Admin can add products.
- Admin can define initial stock per store.
- Inventory is updated correctly after product creation and stock actions.

### 14.2 Sales Flow

- Cashier can create a sale.
- Cashier can add items and complete transaction.
- Inventory decreases after sale.
- Digital receipt is stored.

### 14.3 Low Stock Alert

- When stock drops below threshold, a `Product.LowStock` event is published.
- Admin receives a notification.

### 14.4 Supplier and Replenishment

- Supplier records can be created.
- Purchase orders can be created.
- Stock can be received and inventory updated.

### 14.5 Customers and Loyalty

- Customer records can be created.
- Loyalty points can be added.

### 14.6 Reports

- Daily sales report can be generated by day and store.

### 14.7 AI

- AI service can suggest reorder quantities from past sales.

## 15. Testing Strategy

- Unit tests for service logic and repositories.
- Integration tests for sale-to-inventory event flow.
- E2E tests for admin and cashier workflows.
- Contract tests for gRPC endpoints.
- Smoke tests after build/deploy.

## 16. Observability and Operations

- Structured logs.
- Metrics for sales, stock changes, event publishing, and API latency.
- Tracing across gateway and services.
- Alerts for service failures and queue/event lag.

## 17. Security Requirements

- JWT / Sanctum-based authentication at gateway.
- RBAC for admin and cashier roles.
- Tenant and store claims must be validated in every request.
- Service-to-service auth must be enforced.
- No data leakage across tenants.

## 18. Delivery Phases

### Phase 1

- Monorepo scaffold
- Tenant/store hierarchy
- Auth gateway
- Product, inventory, sales, customer, supplier services
- Core REST + gRPC contracts
- Basic SvelteKit admin and cashier UI
- NATS events
- DB schema

### Phase 2

- Notifications and reporting expansion
- Offline/PWA improvements
- Barcode scanner integration
- AI forecasting and product recommendation

### Phase 3

- Advanced analytics
- Dynamic pricing suggestions
- Image recognition
- Chat support
- Production hardening and scaling

## 19. Definition of Done

A feature is done only when:

- the contract is defined,
- code is implemented,
- tests are added,
- tenant isolation is enforced,
- and the behavior is documented.

## 20. Spec-Driven Delivery Rules

- Do not implement features that are not written in this PRD.
- If something is missing, update this PRD before coding.
- Keep all work traceable to a PRD section.
- Prefer small, reviewable increments.
- Do not invent new workflows, tables, or services unless the PRD is updated first.
