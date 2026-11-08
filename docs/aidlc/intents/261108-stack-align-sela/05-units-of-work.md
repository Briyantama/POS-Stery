# 05 - Units of work

| Unit | Content | Gate | Depends on | Bolt |
| --- | --- | --- | --- | --- |
| U1 | Go module, copies of SELA `httpx`, `health`, `db/migrate.go`, `testdb`, `config.Load` | D1 | - | B1 |
| U2 | goose migrations, `0001_bootstrap.sql`, one file per old up/down pair | D1 | U1 | B1, B2 |
| U3 | `internal/events` publisher and consumer on Dragonfly Streams | D3 | U1 | B1 |
| U4 | `internal/authn` (and POS `internal/ratelimit`) | D2 | U1 | B1 |
| U5 | Service ports to `services/<name>` with `http.go` | D2 | U1-U4 | B2 |
| U6 | Wiring in `cmd/api` | D2 | U5 | B3 |
| U7 | Frontend toolchain and API contract readers | D4 | U5 | B4 |
| U8 | Compose, Makefile, `scripts/check.sh`, CI | D1 | U6 | B4 |
| U9 | Deletions of the old tree | D1 | U6 | B5 |
| U10 | ADRs, `CLAUDE.md`, `.claude/`, FSD stack rows | D1-D4 | U6 | B4 |

## Route table (old path to new path)

Planned routes, decided 2026-11-08 (roles the old gateway never defined were set by the orchestrator; see `audit.md`). Health: `GET /healthz`, public.

| Service | RPC | Old path | New method + path | Public | Roles | Paginated | Status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| auth | Login | `POST /api/login` | `POST /api/v1/login` | yes | - | no | 200 |
| auth | Logout | `POST /api/logout` | `POST /api/v1/logout` | no | admin, cashier | no | 200 |
| auth | Refresh | - | `POST /api/v1/auth/refresh` | yes | - | no | 200 |
| auth | LogoutAll | - | `POST /api/v1/auth/logout-all` | no | admin, cashier (sibling logout) | no | 200 |
| auth | ListSessions | - | `GET /api/v1/auth/sessions` | no | admin, cashier (sibling logout) | no | 200 |
| auth | Validate | - | none | no | no route (gateway-only) | no | 200 |
| auth | ValidateStore | - | none (in-process `Service.ValidateStore`) | - | - | - | - |
| product | SearchProducts | `GET /api/products` | `GET /api/v1/products` | no | admin, cashier | yes (limit/offset) | 200 |
| product | CreateProduct | `POST /api/products` | `POST /api/v1/products` | no | admin | no | 201 |
| product | UpdateProduct | `PUT /api/products/{id}` | `PUT /api/v1/products/{id}` | no | admin | no | 200 |
| product | GetProduct | - | `GET /api/v1/products/{product_id}` | no | admin, cashier (sibling products GET) | no | 200 |
| inventory | ListItems | `GET /api/inventory` | `GET /api/v1/inventory` | no | admin, stock_manager | yes (limit/offset) | 200 |
| inventory | AddItem | `POST /api/inventory` | `POST /api/v1/inventory` | no | admin, stock_manager | no | 201 |
| inventory | GetItem | - | `GET /api/v1/inventory/{product_id}` | no | admin, stock_manager (sibling inventory) | no | 200 |
| inventory | SetThreshold | - | `PUT /api/v1/inventory/{product_id}/threshold` | no | admin, stock_manager (sibling inventory) | no | 200 |
| inventory | UpdateStock | - | `PATCH /api/v1/inventory/{product_id}/stock` | no | admin, stock_manager (sibling inventory) | no | 200 |
| sales | CreateSale | `POST /api/sales` | `POST /api/v1/sales` | no | cashier | no | 201 |
| sales | GetSalesReport | `GET /api/sales/report` | `GET /api/v1/sales/report` | no | admin | no | 200 |
| sales | GetSale | - | `GET /api/v1/sales/{sale_id}` | no | cashier (sibling sales POST) | no | 200 |
| supplier | ListSuppliers | `GET /api/suppliers` | `GET /api/v1/suppliers` | no | admin, stock_manager | yes (limit/offset) | 200 |
| supplier | AddSupplier | `POST /api/suppliers` | `POST /api/v1/suppliers` | no | admin | no | 201 |
| supplier | CreatePurchaseOrder | `POST /api/purchase-orders` | `POST /api/v1/purchase-orders` | no | admin, stock_manager | no | 201 |
| supplier | ListPurchaseOrders | - | `GET /api/v1/purchase-orders` | no | admin, stock_manager (sibling supplier list) | yes (limit/offset) | 200 |
| supplier | ReceiveStock | - | `POST /api/v1/purchase-orders/{purchase_order_id}/receive` | no | admin, stock_manager (sibling PO POST) | no | 200 |
| customer | ListCustomers | `GET /api/customers` | `GET /api/v1/customers` | no | admin, cashier | yes (limit/offset) | 200 |
| customer | CreateCustomer | `POST /api/customers` | `POST /api/v1/customers` | no | admin, cashier | no | 201 |
| customer | GetCustomer | - | `GET /api/v1/customers/{customer_id}` | no | admin, cashier (sibling customers) | no | 200 |
| customer | AddLoyaltyPoints | - | `POST /api/v1/customers/{customer_id}/loyalty` | no | admin, cashier (sibling customers) | no | 200 |
