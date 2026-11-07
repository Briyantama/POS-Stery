# FSD POS-Stery

*Functional Specification Document: POS SaaS multi-tenant untuk toko ritel (produk `POS` dalam Stery Suite): perilaku fungsional saat ini (as-is) dan arsitektur target mengikuti STACK-SPEC*

| | |
|---|---|
| **Nama Dokumen** | FSD POS-Stery (Functional Specification Document) |
| **Versi** | 1.0 (Draft) |
| **Tanggal** | 6 November 2026 |
| **Disusun oleh** | Tim Produk & Engineering |
| **Status** | Draft untuk ditinjau |
| **Dokumen Terkait** | `PRD.md` (PRD POS-Stery, bahasa Inggris); `BRD_POS.id.md` (kebutuhan bisnis BR-POS-01 s.d. BR-POS-21); `00_Analisis_Dampak_Risiko` (analisis dampak dan risiko Stery Suite); `SELA/CLAUDE.md` dan `SELA/docs/FSD_Sela.md` bagian 2 (sumber STACK-SPEC) |
| **Perubahan dari vX** | Perubahan dari vX: Dokumen pertama |

---

## 1. Pendahuluan

### 1.1 Tujuan

FSD ini menjadi acuan teknis POS-Stery: (a) mendokumentasikan perilaku fungsional yang **sudah ada di kode** beserta status dan bukti, (b) menetapkan arsitektur target yang identik dengan STACK-SPEC Stery Suite (sumber: SELA) dan keputusan D1-D12, dan (c) mendaftar selisih as-is -> to-be pada bagian 2.7 agar penyelarasan (workstream S2) dapat direncanakan. Kebutuhan bisnis ada di `BRD_POS.id.md`; kebutuhan produk ada di `PRD.md` yang tidak diubah.

### 1.2 Cakupan

Modul katalog POS: 01 AUTH, 02 PRD (katalog), 03 INV, 04 SAL, 05 SUP, 06 CUS, 07 RPT, 08 NTF (Belum), 09 AI (Belum). Ditambah lintas modul: keamanan (`FR-POS-SEC.n`) dan non-fungsional (`NFR-POS-NN`, bagian 9). Tidak ada PRD POS baru; setiap rujukan "PRD" berarti `PRD.md`.

### 1.3 Konvensi

- **Status:** Selesai, Sebagian, Belum, Pending Verification, Ditunda. **Prioritas:** M (Must), S (Should), C (Could).
- **Kebenaran = kode.** Jalur pada kolom `Bukti` relatif terhadap akar repo `POS-Stery` dan ada di disk pada tanggal dokumen. Dokumen status yang bertentangan dengan kode dicatat pada 11.3 dan tidak diulang sebagai fakta.
- **As-is vs to-be.** Bagian 4 menggambarkan perilaku saat ini (status + bukti) dan, pada "Target", perilaku yang diwajibkan stack target. Jalur kode target hanya muncul pada tabel delta 2.7, tidak pernah pada kolom `Bukti`.
- **Tanpa angka bisnis.** Harga, SLA, kuota, dan target kinerja adalah `tbd` beserta keputusan terbuka `D-POS-NN` (didefinisikan di BRD bagian 9.3). Estimasi usaha ditandai `[INFERENCE]`.
- **Rahasia.** Dokumen hanya menyebut jalur berkas yang memuat rahasia, tidak pernah nilainya.
- **Pembacaan statis.** Tidak ada build, lint, atau tes yang dijalankan untuk menyusun dokumen; temuan perilaku runtime ditandai `[INFERENCE]` atau Pending Verification.

### 1.4 Ringkasan kondisi kode (fakta terverifikasi)

| Fakta | Status | Bukti |
|---|---|---|
| Enam layanan gRPC (auth, product, inventory, sales, supplier, customer), seluruh 28 RPC punya handler | Selesai | `proto/pos/auth/v1/auth.proto` |
| Gateway REST Go `chi` dengan 14 rute bisnis, bukan Laravel | Selesai | `services/api-gateway/cmd/server/main.go` |
| Anotasi HTTP `/v1/...` pada proto tidak dipasang oleh mux apa pun (tidak ada grpc-gateway) | Selesai | `proto/pos/sales/v1/sales.proto` |
| 79 fungsi tes unit (berbasis stub); tidak ada tes integrasi Postgres | Selesai | `services/sales-service/internal/application/commands/create_sale_test.go` |
| Dua aplikasi SvelteKit SSR (admin, kasir), seluruh halaman admin hanya baca | Sebagian | `apps/admin/src/hooks.server.ts` |

---

## 2. Arsitektur dan Stack

### 2.1 Stack (STACK-SPEC, sumber SELA)

Tabel berikut adalah STACK-SPEC normatif Stery Suite; kolom "Kondisi POS saat ini" menunjukkan titik awal penyelarasan. Kolom sumber menunjuk berkas SELA (kebenaran bagi standar).

| # | Layer | Standar (target POS) | Sumber SELA | Kondisi POS saat ini |
|---|---|---|---|---|
| 1 | Layout repo | per produk: `apps/api`, `apps/web`, `deploy/`, `scripts/`, `docs/`, `Makefile`, `CLAUDE.md`, `AGENTS.md` | `SELA/CLAUDE.md` ("Layout"); `SELA/docs/FSD_Sela.md` 2.6 | monorepo `services/*` (6 layanan + gateway), `apps/admin`, `apps/cashier`, `packages/ui`, `gen/go`, `go.work`; tidak ada `AGENTS.md` |
| 2 | Bahasa | Go >= 1.25.1, satu modul per produk `github.com/<owner>/<product>` (handle pemilik `Briyantama` seperti SELA; belum diverifikasi untuk repo baru) | `SELA/apps/api/go.mod` | `go.work` 1.25.0, `services/_shared/go.mod` 1.24.0, tiap layanan punya `go.mod`; CI memakai Go 1.24 |
| 3 | Dekomposisi | modul = paket Go `services/<name>` dalam satu proses: core `Service` + repositori Postgres + adapter HTTP + adapter gRPC opsional; batas lewat interface Go dan kontrak proto agar dapat dipisah kelak | `SELA/CLAUDE.md` ("Layout"); `SELA/docs/FSD_Sela.md` 2.2 | tujuh proses terpisah yang saling memanggil lewat gRPC |
| 4 | Transport | REST `/api/v1` pada `net/http` ServeMux stdlib, amplop `{success,data,error}` lewat `kit/httpx`; gRPC (`api/proto/v1`, buf, `gen/go` ter-commit, `scripts/proto-gen.sh --check`) untuk klien mesin/internal dengan interseptor deny-by-default; tanpa tier gateway; Caddy hanya reverse proxy | `SELA/apps/api/internal/httpx`; `SELA/scripts/proto-gen.sh`; `SELA/deploy/Caddyfile` | gateway `chi` `/api/...` tanpa versi, galat `{message}` gaya Laravel; gRPC antar-layanan dengan `X-Service-Token` gagal-terbuka |
| 5 | Database | PostgreSQL 18 (`uuidv7()`), pgx/v5 SQL mentah (tanpa sqlc/ORM), goose v3.26.0 tertanam + `cmd/migrate up\|down`, `testdb.NewMigrated` | `SELA/apps/api/internal/db`; `SELA/apps/api/internal/testdb`; `SELA/apps/api/cmd/migrate` | PostgreSQL 16, golang-migrate per layanan, `gen_random_uuid()`, sqlc hanya di auth-service, pgx 5.6.0 |
| 6 | Isolasi tenant | RLS wajib (D8): `ENABLE` + `FORCE`, kebijakan `tenant_isolation` dengan `USING` dan `WITH CHECK`, `current_setting('app.tenant_id', true)::uuid` diset per transaksi oleh `kit/tenantdb.WithTenant`; POS menambah `app.store_id`; peran aplikasi NOBYPASSRLS, peran migrasi terpisah; tes lintas-tenant wajib per modul | Keputusan D8 (SELA belum punya tenant/RLS; pola dibawa dari POS/HRIS) | `ENABLE`+`FORCE` pada semua tabel bertenant, 7 dari 15 kebijakan tanpa `WITH CHECK`, tidak ada kebijakan yang memakai `app.store_id` |
| 7 | State store | Dragonfly lewat go-redis/v9: sesi (hanya hash), rate limit, kuota, antrean, skrip Lua | `SELA/apps/api/services/auth/service.go` | Redis (daftar hitam JWT, rate limit gateway), go-redis 9.6.1 |
| 8 | Object storage | RustFS (S3) lewat aws-sdk-go-v2 di balik `objstore.Store`; hanya URL presigned; pemindaian ClamAV `clamd` gagal-tertutup pada jalur unggah | `SELA/apps/api/internal/objstore`; `SELA/apps/api/services/media/scan` | tidak dipakai (POS belum menyimpan berkas; PDF struk fase 2) |
| 9 | Autentikasi | sesi opaque pada cookie HttpOnly Secure, hash di Dragonfly; tabel RBAC (roles, permissions, actions, role_actions) + `RequirePermission(code)`; tanpa JWT dan tanpa token di penyimpanan peramban | `SELA/apps/api/services/auth/service.go`; `SELA/apps/api/services/rbac` | JWT RS256 (`tid`, `sid`, `role`), cookie httpOnly berisi JWT, hanya tabel `roles` |
| 10 | Asinkron | ticker worker dalam proses + outbox Postgres + konsumen idempoten; tanpa NATS/Kafka | `SELA/apps/api/cmd/api/main.go`; `SELA/apps/api/services/event/notify.go` | NATS JetStream stream `POS_EVENTS`, publikasi best-effort, tidak ada pelanggan |
| 11 | Observabilitas | `log/slog`, `/healthz`; OpenTelemetry/Prometheus bukan baseline (keputusan D11, terbuka) | `SELA/apps/api/internal/health` | zap; `/healthz` dan `/readyz` di gateway; kode OTel/Prometheus tidak ada |
| 12 | Konfigurasi | variabel lingkungan, validasi fail-fast, rahasia tidak dicatat, `.env.example` | `SELA/apps/api/internal/config` | viper + `.env.example`; rahasia literal di `ecosystem.config.cjs` |
| 13 | PII dan audit | `kit/piicrypt` (AES-256-GCM, AAD per tujuan, blind index, rotasi kunci); `kit/audit` `audit_logs` append-only dengan trigger DB | `SELA/apps/api/internal/piicrypt`; `SELA/apps/api/internal/audit` | PII pelanggan teks biasa; tabel `auth.audit_logs` ada tetapi tak ditulis dan tanpa trigger |
| 14 | Pembayaran | `payment.Provider`, Midtrans QRIS (Xendit dihapus), webhook bertanda tangan, notifikasi lunas dicocokkan nominal | `SELA/apps/api/services/payment/provider.go`; `SELA/apps/api/services/payment/midtrans.go` | tidak ada; POS tidak menerapkan payment gateway (PRD §4). Penagihan langganan POS dilakukan platform (mode `suite`) |
| 15 | Web | SvelteKit 2 + Svelte 5 + Vite 8 + TypeScript 6, `ssr=false`, `adapter-static`, npm, Node >= 20.19; `apiFetch` + cermin amplop; i18n `{id,en}` dengan tes paritas; Vitest 4 + Testing Library; Playwright (kanal Chrome) | `SELA/apps/web/package.json`; `SELA/apps/web/src/lib/api/client.ts`; `SELA/apps/web/src/lib/i18n/messages/parity.test.ts` | dua aplikasi SSR `adapter-node` dengan BFF proxy |
| 16 | Deploy | Caddy; web statis (Vercel atau Caddy); `deploy/docker-compose.yml` untuk Dragonfly, RustFS, profil ClamAV; port standalone API 8080, gRPC 9090, web 5173 | `SELA/deploy/docker-compose.yml`; `SELA/deploy/Caddyfile` | PM2 `ecosystem.config.cjs`; compose berisi redis, nats, jaeger, prometheus (tanpa Postgres); gateway 8000 |
| 17 | Gerbang kualitas | `scripts/check.sh` (drift proto, gofmt, vet, golangci-lint, `go test -cover`, svelte-check, vitest, e2e nyata; `CHECK_STAGES`), job GitHub Actions go/web/e2e, lantai cakupan 80%, `-race` di CI | `SELA/scripts/check.sh`; `SELA/.github/workflows/ci.yml` | CI usang (lihat 11.3); tidak ada e2e |
| 18 | Proses | TDD RED/GREEN, `docs/tracking/tracker.json` + `scripts/build_tracker.py`, FSD diperbarui bersama kode, artefak `docs/aidlc/` | `SELA/docs/tracking/tracker.json`; `SELA/scripts/build_tracker.py`; `SELA/docs/aidlc` | tidak ada tracker; PRD §20 mewajibkan PRD diperbarui sebelum kode |
| 19 | Lokal | Indonesia default, Inggris didukung, IDR, WIB | `SELA/CLAUDE.md` ("Language") | UI berbahasa Inggris |

Keputusan suite yang berlaku untuk POS (selengkapnya di rencana Stery Suite): D1 komposisi (POS tetap deployable terpisah), D2 `STERY_MODE=standalone|suite`, D3 sesi opaque `stery_session` pada mode `suite`, D4 entitlement dengan snapshot dan masa tenggang, D6 outbox tanpa NATS, D7 satu database `pos` dan satu peran aplikasi `pos`, D8 RLS, D9 satu `apps/web`, D10 login kata sandi dipertahankan dengan sesi opaque, D11 observabilitas `tbd`, D12 urutan penyelarasan (POS = S2).

### 2.2 Dekomposisi modul

| Modul | As-is: layanan (proses terpisah) | To-be: paket Go dalam satu proses | Status as-is | Bukti |
|---|---|---|---|---|
| 01 AUTH | `auth-service` gRPC 8081; verifikasi JWT di gateway | paket `auth` (sesi opaque, RBAC, penentuan tenant) + port `Identity` | Sebagian | `services/auth-service/cmd/server/main.go` |
| 02 PRD | `product-service` gRPC 8082 | paket `product` | Sebagian | `services/product-service/cmd/server/main.go` |
| 03 INV | `inventory-service` gRPC 8083 | paket `inventory` | Sebagian | `services/inventory-service/cmd/server/main.go` |
| 04 SAL | `sales-service` gRPC 8084 (memanggil inventory lewat gRPC) | paket `sales` memanggil `inventory` sebagai antarmuka Go dalam satu transaksi | Sebagian | `services/sales-service/cmd/server/main.go` |
| 05 SUP | `supplier-service` gRPC 8085 (memanggil inventory lewat gRPC) | paket `supplier` | Sebagian | `services/supplier-service/cmd/server/main.go` |
| 06 CUS | `customer-service` gRPC 8086 | paket `customer` | Sebagian | `services/customer-service/cmd/server/main.go` |
| 07 RPT | `GetSalesReport` di dalam `sales-service` | paket `sales` (pelaporan) atau paket `report` | Sebagian | `services/sales-service/internal/interfaces/grpc/sales_service.go` |
| 08 NTF | tidak ada | adapter port `Notification` (hub platform pada mode `suite`; worker outbox pada `standalone`) | Belum | `services/_shared/nats/streams.go` |
| 09 AI | tidak ada | tidak ada paket (PRD fase 2/3) | Belum | `services/_shared/nats/streams.go` |
| Gateway | `api-gateway` chi 8000 | dihapus; rute REST dipasang oleh `apps/api` | Selesai | `services/api-gateway/cmd/server/main.go` |

### 2.3 Komponen antarmuka

| Komponen | Fungsi | Status | Bukti |
|---|---|---|---|
| REST gateway (chi) | JWT -> Tenant -> Store -> `WithRole`, batas laju 300 per menit, badan 4 MB, login 20 per menit per IP | Selesai | `services/api-gateway/cmd/server/main.go` |
| Verifikator JWT + daftar hitam | verifikasi RS256 lokal dengan kunci publik; daftar hitam `jti` di Redis | Selesai | `services/api-gateway/internal/jwt/verifier.go` |
| Klien gRPC gateway | satu klien per layanan dengan `x-service-token` | Selesai | `services/api-gateway/internal/clients/base.go` |
| Admin web (SvelteKit SSR) | halaman baca: dashboard, produk, inventaris, pemasok, pelanggan, laporan | Sebagian | `apps/admin/src/routes/(app)/dashboard/+page.svelte` |
| Cashier web (SvelteKit SSR + PWA parsial) | login kasir, POS, struk | Sebagian | `apps/cashier/src/routes/(app)/pos/+page.svelte` |
| BFF proxy | `src/routes/api/[...path]/+server.ts` meneruskan ke gateway dengan token dari cookie | Selesai | `apps/admin/src/routes/api/[...path]/+server.ts` |
| Paket UI bersama | `@pos-stery/ui` | Selesai | `packages/ui` |

**Port Go (target, sama dengan Stery Suite).** Setiap produk memakai enam port dengan dua adapter (lokal untuk `standalone`, platform untuk `suite`) yang dipilih oleh `STERY_MODE`:

```go
Identity.Verify(ctx, token) (Principal, error)
Tenant.WithTenant(ctx, tenantID, scope, fn)          // scope: tenant, tenant+store
Entitlement.Allows(ctx, org, feature) (bool, error)
Notification.Enqueue(ctx, tx, msg)
Audit.Record(ctx, event)
Events.Publish(ctx, tx, envelope)
```

Pemetaan ke POS: `Tenant.WithTenant` menggantikan `WithTenantContext` (menambah `app.store_id`); `Entitlement.Allows` memeriksa fitur dan dimensi `pos.stores.max`, `pos.devices.max`; `Events.Publish` menulis baris outbox dalam transaksi bisnis.

### 2.4 Penggunaan Dragonfly

| Penggunaan | As-is (Redis) | Target (Dragonfly) | Bukti as-is |
|---|---|---|---|
| Daftar hitam akses token | kunci `jti` saat logout, dibaca gateway dan auth-service | dihapus: sesi opaque dicabut dengan menghapus hash sesi | `services/auth-service/internal/infrastructure/redisstore/token_blacklist.go` |
| Sesi | tidak ada (JWT tanpa keadaan; refresh token di Postgres) | `session:<sha256>` -> atribut (`user_id`, `tenant_id`, `store_id`, `role`) TTL `tbd` (D-POS-11); hanya hash disimpan | `services/auth-service/internal/domain/refresh_token.go` |
| Rate limit | `httprate` in-memory di gateway | penghitung Dragonfly per alamat/akun (login 20 per menit per IP dipertahankan) | `services/api-gateway/cmd/server/main.go` |
| Kunci lockout login | tidak ada | `login:fail:<email-hash>` [INFERENCE: selaras `SELA/CLAUDE.md` invariants auth] | `services/auth-service/internal/application/commands/login.go` |
| Daftar perangkat kasir aktif (`pos.devices.max`) | tidak ada | registri perangkat di Postgres + denyut jantung `device:<id>` di Dragonfly [INFERENCE: desain belum ada] | `apps/cashier/src/routes/(auth)/login/+page.svelte` |

Port standalone: Dragonfly 6379. Port profil suite: POS memakai Dragonfly 6382 (instans per produk, D7).

### 2.5 Event dan outbox

**As-is (NATS JetStream).** Stream `POS_EVENTS`, penyimpanan berkas, umur maksimum 7 hari, 1 replika. Amplop: `{event_type, tenant_id, store_id, occurred_at, payload}` (tanpa `event_id`, `actor_id`, `schema_version`). Tidak ada pelanggan (subscriber) di seluruh kode.

| Subjek | Penerbit | Konsumen | Status | Bukti |
|---|---|---|---|---|
| `Product.LowStock` | inventory (`UpdateStock`, kuantitas <= ambang; gagal terbit ditelan) | tidak ada | Sebagian | `services/inventory-service/internal/infrastructure/nats_publisher.go` |
| `Sale.Completed` | sales (best-effort setelah commit) | tidak ada (seharusnya customer/loyalitas) | Sebagian | `services/sales-service/internal/infrastructure/nats_publisher.go` |
| `Inventory.Replenished` | supplier (per item diterima) | tidak ada | Sebagian | `services/supplier-service/internal/infrastructure/nats_publisher.go` |
| `Customer.New` | customer | tidak ada | Sebagian | `services/customer-service/internal/infrastructure/nats_publisher.go` |
| `AI.OrderForecasted` | tidak ada (hanya konstanta) | tidak ada | Belum | `services/_shared/nats/streams.go` |

**Target (D6).** Peristiwa domain dalam proses + baris outbox Postgres pada transaksi bisnis; worker ticker membaca outbox dengan `FOR UPDATE SKIP LOCKED`; konsumen idempoten dengan tabel `processed_events`. Amplop lintas produk: `{event_id UUIDv7, event_type, org_id, store_id?, actor_id, occurred_at, schema_version, payload}`; pengiriman at-least-once ke platform `IntegrationService/Publish`; opt-in per organisasi lewat fitur entitlement `integration.<a>_<b>`. Pemetaan subjek: `Product.LowStock` -> `inventory.low_stock`, `Sale.Completed` -> `sales.sale.completed`, `Inventory.Replenished` -> `supplier.stock.received`, `Customer.New` -> `customer.created` (nama final `tbd` pada S2). Pelanggan dalam proses pertama: loyalitas pada `sales.sale.completed` (idempoten per `sale_id`) dan notifikasi stok rendah.

### 2.6 Struktur kode dan kontrak

**As-is.** Monorepo: `services/{auth,product,inventory,sales,supplier,customer}-service` (masing-masing `go.mod`, `cmd/server`, `internal/{domain,application,infrastructure,interfaces/grpc}`, `migrations`), `services/api-gateway`, `services/_shared` (config, database, middleware, nats, redis, observability, server, errors), `proto/pos/<svc>/v1/*.proto`, `gen/go`, `apps/{admin,cashier}`, `packages/ui`, `deploy/docker`, `scripts`, `go.work`. Arsitektur berlapis domain/application/infrastructure/interfaces per layanan (kecuali auth-service yang memakai sqlc, lainnya pgx).

| Kontrak | Lokasi | Status | Bukti |
|---|---|---|---|
| Proto 6 layanan, 28 RPC | `proto/pos/*/v1/*.proto` | Selesai | `proto/pos/customer/v1/customer.proto` |
| Kode terbangkit | `gen/go/pos/*` (ter-commit) | Selesai | `gen/go` |
| Migrasi | satu `000001_*` per layanan (auth: 000001-000003) | Sebagian | `services/auth-service/migrations/000003_add_login_tenant_resolver.up.sql` |
| Skrip validasi migrasi | hanya menghitung pasangan up/down berkas | Sebagian | `scripts/migrate-validate.sh` |

**Target.** `apps/api` (satu modul Go, `cmd/api`, `cmd/migrate`, `services/<name>`, `api/proto/v1`, `gen/go`, migrasi goose tertanam), `apps/web` (satu SPA dengan route group `(admin)` dan `(cashier)`), `deploy/`, `scripts/check.sh`, `docs/tracking/tracker.json`, `AGENTS.md`; kode bersama diimpor dari `kit` (repo `stery-suite`). Rincian jalur target ada pada 2.7.

### 2.7 Delta penyelarasan as-is -> to-be

Reuse: lapisan `domain` dan `application` dipertahankan (logika bisnis dan tes unit), `infrastructure` dan `interfaces` ditulis ulang. Estimasi usaha S/M/L [INFERENCE] (S = hari, M = minggu, L = beberapa minggu; belum divalidasi).

| as-is -> to-be | modul terdampak | reuse (domain/application kept, infrastructure/interfaces rewritten) | effort S/M/L [INFERENCE] |
|---|---|---|---|
| Gateway Go `chi` (`services/api-gateway`) -> REST `/api/v1` pada ServeMux stdlib di `apps/api/cmd/api` dengan amplop `{success,data,error}` lewat `kit/httpx` | semua | handler dipetakan ulang; middleware Tenant/Store/Role menjadi `RequirePermission` | M |
| Enam proses gRPC + saga antar-layanan -> paket Go `apps/api/services/<name>` dalam satu proses, antar modul lewat interface Go | 03, 04, 05 | domain dan perintah (`create_sale.go`, `receive_stock.go`) dipertahankan; klien gRPC antar-layanan dihapus | L |
| gRPC `X-Service-Token` gagal-terbuka (`services/_shared/middleware/grpc_interceptors.go`) -> interseptor deny-by-default dengan allowlist publik; gRPC hanya untuk klien mesin | 01 | tidak ada (ditulis ulang) | S |
| JWT RS256 (`tid`, `sid`, `role`) + refresh token -> sesi opaque cookie `stery_session`/cookie sesi POS dengan atribut `store_id`, hash di Dragonfly (D3, D10); bcrypt dipertahankan (`services/auth-service/internal/application/commands/login.go`) | 01 | domain `User`/`Store`/`Tenant`, penentuan tenant `auth.resolve_login_tenant`, rotasi/deteksi pemakaian ulang diganti kedaluwarsa sesi | M |
| `golang-migrate` -> goose v3.26.0 tertanam + `cmd/migrate`; konsolidasi migrasi per layanan menjadi satu rangkaian | semua | SQL tabel dipertahankan; tambah migrasi untuk perbaikan (CHECK stok, `WITH CHECK`, kolom baru) | M |
| sqlc (hanya `auth-service`) dan pgx 5.6.0 -> pgx/v5 SQL mentah tanpa sqlc | 01 | kueri sqlc ditulis ulang sebagai SQL mentah | S |
| zap + viper -> `log/slog` + konfigurasi env fail-fast (`kit/config`) | semua | tidak ada | S |
| Redis -> Dragonfly (go-redis/v9) | 01 | klien go-redis dipertahankan | S |
| NATS JetStream -> outbox Postgres + worker ticker + `processed_events` (D6) | 03, 04, 05, 06 | penerbit (`nats_publisher.go`) diganti `Events.Publish`; kontrak payload dipertahankan | M |
| `WithTenantContext` (`SET LOCAL`, tidak terverifikasi) -> `kit/tenantdb.WithTenant` (`set_config(..., true)`, tenant + `app.store_id`) + tes lintas-tenant | semua | tidak ada | M |
| RLS: tambah `WITH CHECK` pada 7 kebijakan; tambah kebijakan `app.store_id` pada tabel toko; peran `pos_admin`/`pos_app` -> peran migrasi terpisah + peran aplikasi NOBYPASSRLS tunggal `pos` | semua | skema dipertahankan | S |
| Penjualan: saga gRPC -> satu transaksi DB dengan kunci idempotensi, harga dari server, penurunan stok bersyarat, buku besar pergerakan stok, outbox (7.1) | 04, 03 | validasi dan perhitungan total dipertahankan | M |
| Loyalitas: tanpa pemanggil -> dipanggil dalam proses setelah commit lewat konsumen outbox, idempoten per `sale_id` | 06, 04 | `add_loyalty_points.go` dipertahankan | S |
| Penerimaan PO: non-atomik, tanpa parsial -> satu transaksi, kuantitas diterima per item, status parsial (7.2) | 05 | entitas PO dipertahankan | M |
| Money `float64` di Go -> representasi eksak (bilangan bulat satuan rupiah di Go; kolom `NUMERIC(12,2)` tetap) | 04, 05, 07 | tipe dan tes disesuaikan | S |
| Rute REST 14 -> paritas penuh dengan operasi modul (6.2), prefiks `/api/v1`, `PUT` produk dst. | semua | tidak ada | M |
| Admin + cashier SSR (`adapter-node`, BFF) -> satu `apps/web` SPA statis, route group `(admin)` dan `(cashier)`, `apiFetch`, i18n `{id,en}` | 10 | `packages/ui` dan komponen halaman dipindahkan | L |
| PWA kasir hanya cache aset -> antrean offline IndexedDB, scope service worker `/cashier/` (7.3) | 04 | tidak ada | M |
| Dashboard: data sintetis -> data nyata dengan nama produk dan bauran pembayaran | 07 | halaman dipertahankan sebagai kerangka | S |
| PM2 `ecosystem.config.cjs` + compose (redis, nats, jaeger, prometheus) -> Caddy + `deploy/docker-compose.yml` (Dragonfly, RustFS, ClamAV profil) | deploy | tidak ada | S |
| Go 1.24/1.25 dan Postgres 16 -> Go >= 1.25.1 dan PostgreSQL 18; CI -> `scripts/check.sh` + job go/web/e2e, `-race`, lantai 80% | CI | tes unit dipertahankan | M |
| Rahasia terlacak (`ecosystem.config.cjs`, `deploy/docker/init-postgres.sql`) -> `.env.example` tanpa nilai rahasia; rotasi dan pembersihan riwayat git | deploy, 01 | tidak ada | S |
| OpenTelemetry/Prometheus (hanya dependensi) -> D11 `tbd`; baseline `slog` + `/healthz` | semua | tidak ada | S |
| Adapter mode `suite` untuk enam port + entitlement `pos.stores.max`, `pos.devices.max` (S4) | 01, 05, 08 | port diperkenalkan pada S2 dengan adapter lokal | M |

**Urutan workstream S2** (tiap butir menutup satu atau lebih butir gerbang BRD 12.1): WS-1 perbaikan keamanan dan kebersihan (G5, G7, G8 sebagian; dapat dikerjakan lebih awal), WS-2 fondasi `apps/api` dan basis data (goose, PG18, tenantdb, G4, G6), WS-3 pemindahan modul 01-03 dan 05-07, WS-4 penjualan transaksional dan loyalitas (G1), WS-5 permukaan REST penuh (G2), WS-6 `apps/web` tunggal dan offline, WS-7 CI `check.sh` (G3), WS-8 keputusan PPN/struk (G8).

---

## 3. Aktor dan Model Hak Akses

### 3.1 Aktor

| Aktor | Deskripsi | Login | Bukti |
|---|---|---|---|
| Admin tenant (`admin`) | mengelola produk, stok, pemasok, pelanggan, laporan, harga | email + kata sandi | `services/auth-service/internal/domain/user.go` |
| Kasir (`cashier`) | menjual; terikat pada satu toko; login mewajibkan `store_id` | email + kata sandi + `store_id` | `services/auth-service/internal/application/commands/login.go` |
| Manajer stok (`stock_manager`) | inventaris, PO, penerimaan stok | email + kata sandi | `services/api-gateway/cmd/server/main.go` |
| Pelanggan toko | dicatat untuk loyalitas; tidak login | tidak ada | `services/customer-service/migrations/000001_create_customer_schema.up.sql` |
| Platform (mode `suite`) | menerbitkan sesi, entitlement, notifikasi, audit | layanan ke layanan (HMAC) | belum ada pada POS (Belum) |
| Operator (pemasangan) | membuat tenant, toko, pengguna pertama | belum ada jalur (Belum) | `services/auth-service/internal/infrastructure/postgres/tenant_repository.go` |

Catatan kode: `User.PrimaryRole()` hanya mengambil peran pertama pengguna, sehingga multi-peran per pengguna belum didukung efektif (Sebagian).

### 3.2 Matriks hak akses as-is (penegakan di gateway `WithRole`)

| Operasi (rute) | admin | cashier | stock_manager |
|---|---|---|---|
| `POST /api/logout` | ya | ya | tidak |
| `GET /api/products` | ya | ya | tidak |
| `POST /api/products`, `PUT /api/products/{id}` | ya | tidak | tidak |
| `GET /api/inventory`, `POST /api/inventory` | ya | tidak | ya |
| `POST /api/sales` | tidak | ya | tidak |
| `GET /api/sales/report` | ya | tidak | tidak |
| `GET /api/suppliers`, `POST /api/purchase-orders` | ya | tidak | ya |
| `POST /api/suppliers` | ya | tidak | tidak |
| `GET /api/customers`, `POST /api/customers` | ya | ya | tidak |

Temuan: `stock_manager` tidak dapat melihat produk (`GET /api/products`) padahal perlu memilih produk untuk PO, dan tidak dapat memanggil `POST /api/logout`; `admin` tidak dapat membuat penjualan. Bukti: `services/api-gateway/cmd/server/main.go`. Peran tersimpan pada `auth.roles` (hanya tabel peran; tidak ada tabel `permissions` meski PRD §12 mengisyaratkannya). Target: katalog izin dan `RequirePermission(code)` (FR-POS-01.9).

### 3.3 Model hak akses target

- Peran tetap `admin`, `cashier`, `stock_manager`, dipetakan ke izin (`products:write`, `sales:create`, `reports:read`, dst.; katalog final `tbd` pada S2) pada tabel `roles`, `permissions`, `actions`, `role_actions`.
- Atribut sesi: `user_id`, `tenant_id`, `store_id` (kasir wajib), `role`; admin memilih toko aktif lewat selektor (mengubah atribut sesi `store_id`, bukan header `X-Store-ID` bebas).
- Rute tingkat tenant (produk, pemasok, pelanggan) tidak memerlukan toko; rute tingkat toko (inventaris, penjualan, PO) memerlukannya.
- Mode `suite`: sesi dan izin dasar berasal dari platform (`Identity.Verify`); peran produk dipetakan dari keanggotaan organisasi (rinci `tbd`).

---

## 4. Spesifikasi Fungsional per Modul

Setiap modul: ringkasan, tabel FR (`ID | Requirement | Prioritas (M/S/C) | Status | Bukti`), lalu aturan bisnis, mesin keadaan (bila ada), dan peristiwa.

### 4.01 AUTH (identitas, tenant, toko, RBAC)

Login kata sandi bcrypt dengan penentuan tenant dari email (fungsi SECURITY DEFINER `auth.resolve_login_tenant`; email ambigu ditolak), JWT RS256 berumur 10 menit dengan klaim `tid`, `sid`, `role`, `email`; refresh token opaque 30 hari dengan hash SHA-256, rotasi dalam satu keluarga dan pencabutan keluarga saat pemakaian ulang terdeteksi. Gateway memverifikasi JWT secara lokal (kunci publik) dan memeriksa daftar hitam `jti`.

| ID | Requirement | Prioritas | Status | Bukti |
|---|---|---|---|---|
| FR-POS-01.1 | Login email + kata sandi (bcrypt); tenant ditentukan dari email lewat `auth.resolve_login_tenant`; email ambigu gagal-tertutup | M | Selesai | `services/auth-service/internal/application/commands/login.go`; `services/auth-service/migrations/000003_add_login_tenant_resolver.up.sql` |
| FR-POS-01.2 | Login kasir wajib menyertakan `store_id` yang sama dengan toko yang ditugaskan; token membawa `sid` | M | Selesai | `services/auth-service/internal/application/commands/login.go` |
| FR-POS-01.3 | Admin memilih toko aktif; gateway memvalidasi `X-Store-ID` lewat `ValidateStore`; UI selektor toko admin ada | M | Sebagian | `services/api-gateway/internal/middleware/store.go`; `apps/admin/src/lib/api.ts` |
| FR-POS-01.4 | Refresh token dengan rotasi dan pencabutan keluarga saat pemakaian ulang; tersedia lewat REST dan dipakai web | M | Sebagian | `services/auth-service/internal/application/commands/refresh.go`; `services/auth-service/internal/application/commands/refresh_test.go` |
| FR-POS-01.5 | Logout (daftar hitam `jti` + cabut refresh), logout semua perangkat, daftar sesi; tersedia untuk semua peran | S | Sebagian | `services/auth-service/internal/application/commands/logout.go`; `services/auth-service/internal/application/commands/logout_all.go`; `services/auth-service/internal/application/commands/sessions.go` |
| FR-POS-01.6 | Penegakan peran `admin`/`cashier`/`stock_manager` di API (`WithRole`) | M | Selesai | `services/api-gateway/internal/middleware/role.go`; `services/api-gateway/internal/middleware/role_test.go` |
| FR-POS-01.7 | Batas laju login 20 per menit per IP dan 300 per menit global; badan permintaan maksimum 4 MB | M | Selesai | `services/api-gateway/cmd/server/main.go` |
| FR-POS-01.8 | Pemasangan: membuat tenant, toko, pengguna pertama, dan pengaturan ulang kata sandi tanpa SQL manual | M | Belum | `services/auth-service/internal/infrastructure/postgres/tenant_repository.go` |
| FR-POS-01.9 | Katalog izin (`permissions`, `actions`, `role_actions`) dan `RequirePermission(code)` menggantikan daftar peran per rute | S | Belum | `services/auth-service/migrations/000001_create_auth_schema.up.sql` |
| FR-POS-01.10 | Sesi opaque (cookie HttpOnly Secure, hash di Dragonfly) dengan atribut `store_id` menggantikan JWT; tanpa token di penyimpanan peramban | M | Belum | `services/auth-service/internal/infrastructure/jwt/signer.go` |
| FR-POS-01.11 | Cookie sesi HttpOnly pada aplikasi web, penegakan peran di `hooks.server.ts` | S | Sebagian | `apps/admin/src/hooks.server.ts`; `apps/admin/src/lib/server/session.ts` |
| FR-POS-01.12 | Mode `STERY_MODE=standalone|suite` memilih adapter enam port; default `standalone` | M | Belum | `services/api-gateway/internal/config/config.go` |
| FR-POS-01.13 | Penegakan entitlement `pos.stores.max` (pembuatan toko) dan `pos.devices.max` (login perangkat kasir) dengan snapshot dan degradasi aman | M | Belum | `services/auth-service/internal/infrastructure/postgres/store_repository.go` |

**Aturan bisnis**

- Email login harus menentukan tepat satu tenant; jika ambigu, login ditolak tanpa petunjuk (as-is, `auth.resolve_login_tenant`).
- Kasir hanya boleh login ke toko yang ditugaskan; admin dapat memilih toko mana pun milik tenant-nya (divalidasi `ValidateStore`).
- Tabel `auth.tenants` dan `auth.roles` tidak memiliki RLS; `auth.refresh_tokens` tidak memiliki RLS dengan sengaja (dibaca sebelum konteks tenant ada). Pada target, tabel tanpa RLS wajib didaftarkan dan dibatasi (G4).
- Satu pengguna memiliki satu peran efektif (`PrimaryRole` = peran pertama).
- Mode `suite`: perilaku `standalone` tidak berubah; sesi dan entitlement berasal dari platform.

**Mesin keadaan (refresh token)**

| Dari | Peristiwa | Ke |
|---|---|---|
| aktif | refresh dipakai | dicabut, token baru dalam keluarga sama (`replaced_by`) |
| dicabut | dipakai lagi | seluruh keluarga dicabut |
| aktif | logout / logout-all | dicabut |
| aktif | lewat 30 hari | kedaluwarsa |

Target: tidak ada refresh token; sesi opaque dengan masa berlaku geser (TTL `tbd`).

**Peristiwa.** Diproduksi: tidak ada (as-is); target `auth.user.logged_in` untuk audit (`tbd`). Dikonsumsi: tidak ada.

### 4.02 PRD (katalog produk)

| ID | Requirement | Prioritas | Status | Bukti |
|---|---|---|---|---|
| FR-POS-02.1 | Membuat produk (nama, `sku` unik per tenant, `barcode`, `base_price`, `sale_price`, `unit`, kategori) | M | Selesai | `services/product-service/internal/application/commands/create_product.go`; `services/product-service/internal/application/commands/create_product_test.go` |
| FR-POS-02.2 | Mengubah produk (nama, harga, status aktif) tanpa mengubah `is_active` secara tidak sengaja | M | Sebagian | `services/product-service/internal/application/commands/update_product.go` |
| FR-POS-02.3 | Mengambil produk berdasarkan id lewat REST | S | Sebagian | `services/product-service/internal/interfaces/grpc/product_service.go` |
| FR-POS-02.4 | Mencari produk berdasarkan nama, SKU, dan barcode dengan paginasi (limit/offset diteruskan) | M | Sebagian | `services/product-service/internal/infrastructure/postgres/product_repository.go`; `services/product-service/internal/application/queries/search_products.go` |
| FR-POS-02.5 | Riwayat perubahan harga yang dapat diaudit (siapa, kapan, nilai lama dan baru) | S | Belum | `services/product-service/internal/application/commands/update_product.go` |
| FR-POS-02.6 | Manajemen kategori (buat, ubah, daftar) | S | Belum | `services/product-service/migrations/000001_create_product_schema.up.sql` |
| FR-POS-02.7 | Formulir admin untuk membuat dan mengubah produk | M | Belum | `apps/admin/src/routes/(app)/products/+page.svelte` |

**Aturan bisnis**

- `UNIQUE(tenant_id, sku)`; produk dihapus lunak (`deleted_at`).
- Pencarian as-is: `name ILIKE` atau `barcode` sama persis; SKU tidak dicari (placeholder UI "name or SKU" menyesatkan). Gateway hanya meneruskan parameter `q`.
- Handler `UpdateProduct` menetapkan `is_active` sesuai permintaan (false bila dihilangkan) sehingga pembaruan parsial menonaktifkan produk (as-is, dibaca statis).
- Harga: `base_price` dan `sale_price` opsional (`NUMERIC(12,2)`); penjualan memakai `unit_price` yang dikirim klien (diperbaiki pada FR-POS-04.4).
- Perubahan harga hanya oleh `admin`.

**Mesin keadaan.** `aktif` <-> `nonaktif` (`is_active`); `dihapus` (soft delete). Tidak ada transisi lain.

**Peristiwa.** Diproduksi: tidak ada (target `product.price_changed` untuk audit, `tbd`). Dikonsumsi: tidak ada.

### 4.03 INV (inventaris)

| ID | Requirement | Prioritas | Status | Bukti |
|---|---|---|---|---|
| FR-POS-03.1 | Menambah item stok per toko dengan kuantitas awal (`UNIQUE(tenant_id, store_id, product_id)`) | M | Selesai | `services/inventory-service/internal/application/commands/add_item.go` |
| FR-POS-03.2 | Mengubah stok secara atomik dan tidak pernah negatif, termasuk saat konkurensi (UPDATE bersyarat + `CHECK (quantity >= 0)`) | M | Sebagian | `services/inventory-service/internal/application/commands/update_stock.go`; `services/inventory-service/internal/infrastructure/postgres/stock_repository.go` |
| FR-POS-03.3 | Buku besar pergerakan stok: setiap perubahan mencatat delta, alasan, jenis dan id rujukan (penjualan, penerimaan, penyesuaian), pelaku | M | Belum | `services/inventory-service/internal/application/commands/update_stock.go` |
| FR-POS-03.4 | Mengatur ambang stok per produk per toko; daftar dan get item melaporkan `min_quantity` dan `is_low_stock` dengan benar | M | Sebagian | `services/inventory-service/internal/application/commands/set_threshold.go`; `services/inventory-service/internal/interfaces/grpc/inventory_service.go` |
| FR-POS-03.5 | Peristiwa stok rendah dipublikasikan saat kuantitas melewati ambang, dikonsumsi notifikasi | M | Sebagian | `services/inventory-service/internal/infrastructure/nats_publisher.go` |
| FR-POS-03.6 | Daftar item per toko dengan filter stok rendah; get item lewat REST | M | Sebagian | `services/inventory-service/internal/application/queries/list_items.go`; `services/inventory-service/internal/application/queries/get_item.go` |
| FR-POS-03.7 | Penyesuaian stok manual dan pengaturan ambang tersedia lewat REST dan UI admin | M | Belum | `services/api-gateway/internal/handlers/inventory.go` |

**Aturan bisnis**

- Stok tidak boleh negatif (PRD §13). As-is: `update_stock.go` membaca item (`GetItem`) lalu mengeksekusi `UPDATE ... quantity = quantity + $1` tanpa penjaga `quantity + $1 >= 0` dan tanpa `CHECK` di tabel, sehingga dua penjualan serentak dapat menghasilkan stok negatif (TOCTOU; dibaca statis, tidak diuji runtime) [INFERENCE].
- Parameter `reason` dan rujukan diabaikan oleh repositori (tidak ada buku besar).
- `AddItem`, `ListItems`, `GetItem` mengembalikan `min_quantity = 0` dan `is_low_stock = false` karena konversi proto memakai nilai tetap; hanya `UpdateStock` mengisi `LowStockAlert` dengan benar (`services/inventory-service/internal/interfaces/grpc/inventory_service.go`).
- Peristiwa stok rendah dipublikasikan bila kuantitas <= ambang; kegagalan publikasi ditelan.
- Target: pengurangan stok oleh penjualan dalam transaksi penjualan (7.1); setiap perubahan menulis baris `stock_movements`.

**Mesin keadaan.** Tidak ada status; kuantitas bertambah/berkurang dengan alasan: `RECEIPT`, `SALE`, `ADJUSTMENT` (alasan as-is dikirim tetapi dibuang).

**Peristiwa.** Diproduksi: `Product.LowStock` (target `inventory.low_stock`). Dikonsumsi: target `sales.sale.completed` tidak diperlukan (pengurangan dalam transaksi); `supplier.stock.received` dalam proses.

### 4.04 SAL (penjualan, diskon, struk)

| ID | Requirement | Prioritas | Status | Bukti |
|---|---|---|---|---|
| FR-POS-04.1 | Membuat penjualan dengan item (produk, kuantitas), memvalidasi masukan dan menghitung subtotal, diskon, total | M | Sebagian | `services/sales-service/internal/application/commands/create_sale.go`; `services/sales-service/internal/application/commands/create_sale_test.go` |
| FR-POS-04.2 | Pengurangan stok dan penyimpanan penjualan + struk dalam satu transaksi database | M | Sebagian | `services/sales-service/internal/application/commands/create_sale.go`; `services/sales-service/internal/infrastructure/grpc_clients/inventory_client.go` |
| FR-POS-04.3 | Pembuatan penjualan idempoten: ID penjualan dari klien (`client_sale_id`) unik per tenant dan toko; retry mengembalikan penjualan yang sama | M | Belum | `services/sales-service/internal/application/commands/create_sale.go` |
| FR-POS-04.4 | Harga satuan dicari di server dari katalog; diskon dibatasi wewenang peran; total tidak boleh negatif tanpa penolakan | M | Belum | `services/api-gateway/internal/handlers/sales.go`; `services/sales-service/internal/application/commands/create_sale.go` |
| FR-POS-04.5 | PPN/pajak dihitung dan disimpan pada penjualan dan struk sesuai keputusan D-POS-01 | M | Belum | `services/sales-service/internal/domain/sale.go` |
| FR-POS-04.6 | Metode pembayaran (tunai, QRIS, debit sebagai pencatatan) disimpan pada penjualan sesuai D-POS-04 | S | Belum | `apps/cashier/src/routes/(app)/pos/+page.svelte` |
| FR-POS-04.7 | Struk disimpan sebagai snapshot JSONB unik per penjualan (item, total, kasir, toko, pelanggan, pajak, metode pembayaran) | M | Sebagian | `services/sales-service/internal/infrastructure/postgres/sale_repository.go`; `services/sales-service/migrations/000001_create_sales_schema.up.sql` |
| FR-POS-04.8 | Mengambil penjualan dan struk berdasarkan id lewat REST (`GetSale`) | M | Sebagian | `services/sales-service/internal/interfaces/grpc/sales_service.go` |
| FR-POS-04.9 | Peristiwa `Sale.Completed` ditulis dalam transaksi (outbox) dan dikonsumsi loyalitas | M | Sebagian | `services/sales-service/internal/infrastructure/nats_publisher.go` |
| FR-POS-04.10 | Antarmuka kasir: cari/pindai, keranjang, penyelesaian, tampilan dan cetak struk, riwayat struk | M | Sebagian | `apps/cashier/src/routes/(app)/pos/+page.svelte`; `apps/cashier/src/routes/(app)/receipts/+page.svelte` |
| FR-POS-04.11 | Pembatalan (void) dan retur penjualan sesuai kebijakan D-POS-06 | C | Belum | `services/sales-service/internal/domain/sale.go` |
| FR-POS-04.12 | Antrean penjualan offline di perangkat kasir dengan sinkron otomatis tanpa duplikasi (7.3) | S | Belum | `apps/cashier/src/service-worker.ts` |

**Aturan bisnis**

- Penjualan harus menurunkan stok secara atomik dan membuat struk (PRD §13). As-is: urutan `validasi -> hitung -> inventory.DeductStock per item (gRPC) -> saleRepo.Create (sale + item + struk satu transaksi) -> PublishSaleCompleted (best-effort)`; bila `Create` gagal, `compensate()` memanggil `RestoreStock` per item dan hanya mencatat peringatan bila pemulihan gagal. Pengurangan stok berkomitmen terpisah dari penjualan (`services/sales-service/internal/application/commands/create_sale.go`).
- Cacat as-is: tidak ada kunci idempotensi; `UnitPrice` dan `Discount` berasal dari pemanggil; `total` dijepit pada 0; uang bertipe `float64` di Go sementara kolom `NUMERIC(12,2)`; tidak ada PPN/metode pembayaran/pelanggan pada snapshot; tidak ada panggilan loyalitas; tidak ada pemeriksaan wewenang diskon; `completed_at` dan batas hari laporan memakai zona waktu sesi DB, bukan WIB.
- Gateway: `createSaleBody` hanya memuat `customer_id`, `items[{product_id, quantity, unit_price}]`, `discount_amount`, `notes`; `payment_method` dari UI kasir dibuang (`services/api-gateway/internal/handlers/sales.go`).
- Hanya peran `cashier` yang dapat membuat penjualan (gateway).
- Struk unik per penjualan (`sales.receipts.sale_id UNIQUE`).
- Target: lihat 7.1; PPN dan metode pembayaran `tbd` (D-POS-01, D-POS-04); wewenang diskon `tbd` (D-POS-07).

**Mesin keadaan (penjualan).** As-is hanya satu nilai `status`: `COMPLETED` (default kolom; tidak ada nilai lain di kode). Target (setelah keputusan D-POS-06): `COMPLETED -> VOIDED` (admin), `COMPLETED -> RETURNED` (parsial/penuh); antrean offline menambah keadaan klien `QUEUED -> SYNCING -> SYNCED | FAILED`.

**Peristiwa.** Diproduksi: `Sale.Completed` (target `sales.sale.completed`), `Product.LowStock` secara tidak langsung. Dikonsumsi: target loyalitas (06) dan hipotesis INT-3 (platform).

### 4.05 SUP (pemasok dan PO)

| ID | Requirement | Prioritas | Status | Bukti |
|---|---|---|---|---|
| FR-POS-05.1 | Menambah pemasok (nama, kontak, telepon, email, alamat) | M | Selesai | `services/supplier-service/internal/application/commands/add_supplier.go` |
| FR-POS-05.2 | Daftar pemasok per tenant (tanpa yang dihapus lunak) | M | Selesai | `services/supplier-service/internal/interfaces/grpc/supplier_service.go` |
| FR-POS-05.3 | Membuat PO untuk satu pemasok dengan item (produk, kuantitas, biaya satuan); pemasok divalidasi, produk divalidasi milik tenant | M | Sebagian | `services/supplier-service/internal/application/commands/create_purchase_order.go` |
| FR-POS-05.4 | Daftar PO dan get PO lewat REST | S | Sebagian | `services/supplier-service/internal/interfaces/grpc/supplier_service.go` |
| FR-POS-05.5 | Menerima stok atas PO: menambah stok setiap item tepat sekali, atomik, idempoten | M | Sebagian | `services/supplier-service/internal/application/commands/receive_stock.go`; `services/supplier-service/internal/application/commands/receive_stock_test.go` |
| FR-POS-05.6 | Penerimaan parsial (`quantity_received` per item) dan pembatalan PO (`CANCELLED`) | S | Belum | `services/supplier-service/internal/application/commands/receive_stock.go` |
| FR-POS-05.7 | Peristiwa `Inventory.Replenished` setelah penerimaan, dikonsumsi notifikasi/analitik | S | Sebagian | `services/supplier-service/internal/infrastructure/nats_publisher.go` |

**Aturan bisnis**

- PO baru berstatus `PENDING`; harus memiliki minimal satu item dengan kuantitas > 0 dan `product_id` valid. As-is tidak memeriksa bahwa produk ada atau milik tenant (`create_purchase_order.go`).
- Penerimaan as-is (`receive_stock.go`): item yang tidak ada dalam permintaan dilewati; per item memanggil inventory `UpdateStock` (RECEIPT) lewat gRPC lalu menerbitkan `Inventory.Replenished` best-effort; menerima item apa pun menandai seluruh PO `RECEIVED`. Kegagalan di tengah putaran meninggalkan stok sebagian bertambah dan PO tetap `PENDING`, sehingga retry menambah ganda (tidak idempoten).
- Status PO di skema: `PENDING`, `RECEIVED`, `CANCELLED`; tidak ada RPC pembatalan.
- Kolom `quantity_received` ada tetapi tidak membentuk status parsial.
- `unit_cost` `NUMERIC(12,2)`.

**Mesin keadaan (PO).**

| Dari | Peristiwa | Ke | Status as-is |
|---|---|---|---|
| `PENDING` | terima barang | `RECEIVED` | Selesai (tanpa parsial) |
| `PENDING` | batalkan | `CANCELLED` | Belum (tanpa RPC) |
| `PENDING` | terima sebagian | `PARTIAL` (target) | Belum |
| `PARTIAL` | terima sisa | `RECEIVED` (target) | Belum |

**Peristiwa.** Diproduksi: `Inventory.Replenished` (target `supplier.stock.received`). Dikonsumsi: tidak ada.

### 4.06 CUS (pelanggan dan loyalitas)

| ID | Requirement | Prioritas | Status | Bukti |
|---|---|---|---|---|
| FR-POS-06.1 | Membuat pelanggan (nama, telepon, email) | M | Selesai | `services/customer-service/internal/application/commands/create_customer.go`; `services/customer-service/internal/application/commands/create_customer_test.go` |
| FR-POS-06.2 | Mengambil dan mendaftar pelanggan lewat REST | S | Sebagian | `services/customer-service/internal/interfaces/grpc/customer_service.go`; `services/customer-service/internal/application/queries/get_customer.go` |
| FR-POS-06.3 | Poin loyalitas ditambahkan otomatis setelah penjualan berpelanggan berhasil di-commit, tepat satu kali per `sale_id` | M | Belum | `services/customer-service/internal/application/commands/add_loyalty_points.go`; `services/customer-service/internal/interfaces/grpc/customer_service.go` |
| FR-POS-06.4 | Aturan perolehan (laju per nilai belanja) dan penukaran poin sesuai D-POS-05 | S | Belum | `services/customer-service/internal/application/commands/add_loyalty_points.go` |
| FR-POS-06.5 | Peristiwa `Customer.New` dipublikasikan setelah pelanggan dibuat | C | Sebagian | `services/customer-service/internal/infrastructure/nats_publisher.go` |
| FR-POS-06.6 | Kasir melampirkan pelanggan pada penjualan dan struk di UI | S | Belum | `apps/cashier/src/routes/(app)/pos/+page.svelte` |

**Aturan bisnis**

- Poin hanya bertambah setelah penjualan berhasil (PRD §13). **As-is tidak terpenuhi:** `AddLoyaltyPoints` (`customer_service.go`, komentar "called by sales-service after Sale.Completed") tidak memiliki pemanggil; sales-service tidak memanggilnya dan tidak ada pelanggan NATS di seluruh kode.
- Handler loyalitas membaca total lalu memasukkan baris pada transaksi terpisah (balapan); `loyalty_points.sale_id` tidak `UNIQUE` sehingga tidak idempoten (dibaca statis) [INFERENCE].
- Aturan perolehan tidak terdefinisi (pemanggil menyuplai jumlah poin); tidak ada penukaran, kedaluwarsa, atau tingkat.
- Data pelanggan (nama, telepon, email) adalah data pribadi, disimpan teks biasa (bagian 8).
- Target: konsumen outbox dalam proses menerapkan poin dalam transaksi sendiri, kunci `UNIQUE(tenant_id, sale_id)`.

**Mesin keadaan.** Pelanggan: `aktif` / `nonaktif` / `dihapus` (soft delete). Poin: buku besar tambah-saja (`points`, `total_points`).

**Peristiwa.** Diproduksi: `Customer.New` (target `customer.created`). Dikonsumsi: target `sales.sale.completed`.

### 4.07 RPT (laporan)

| ID | Requirement | Prioritas | Status | Bukti |
|---|---|---|---|---|
| FR-POS-07.1 | Laporan penjualan per rentang tanggal: pendapatan total dan jumlah transaksi, per toko dan per tenant | M | Sebagian | `services/sales-service/internal/infrastructure/postgres/sale_repository.go`; `services/sales-service/internal/interfaces/grpc/sales_service.go` |
| FR-POS-07.2 | Produk teratas dengan nama produk dan unit terjual | M | Sebagian | `services/sales-service/internal/infrastructure/postgres/sale_repository.go` |
| FR-POS-07.3 | Bauran metode pembayaran dihitung dari data penjualan nyata | S | Belum | `apps/admin/src/routes/(app)/dashboard/+page.svelte` |
| FR-POS-07.4 | Dashboard hanya menampilkan data nyata; nama field cocok dengan kontrak API | M | Sebagian | `apps/admin/src/routes/(app)/dashboard/+page.svelte`; `apps/admin/src/routes/(app)/reports/+page.svelte` |
| FR-POS-07.5 | Batas hari laporan mengikuti zona waktu WIB (atau toko, D-POS-14) | S | Belum | `services/sales-service/internal/infrastructure/postgres/sale_repository.go` |
| FR-POS-07.6 | Cakupan laporan tingkat tenant dapat dipilih admin tanpa konteks toko | S | Sebagian | `services/api-gateway/internal/handlers/sales.go` |

**Aturan bisnis**

- Semua kueri laporan menghormati cakupan tenant dan toko (PRD §13); kueri repositori mendukung cakupan toko maupun tenant, tetapi middleware `Store` gateway mewajibkan konteks toko untuk semua rute sehingga cakupan tenant tidak dapat dipakai lewat REST (dibaca statis) [INFERENCE].
- `product_name` pada produk teratas selalu `''`: SQL menulis `'' AS product_name` untuk kueri toko dan tenant tanpa join ke produk (`sale_repository.go`).
- Kontrak tidak cocok: UI dashboard mengharapkan `total_sales`, `top_products[].name`, `.quantity`; backend mengembalikan `transaction_count`, `product_id`, `product_name`, `units_sold` (`gen/go/pos/sales/v1/sales.pb.go`).
- Dashboard memalsukan data: bauran pembayaran (jatuh balik ke 55/30/15 dari `total_revenue`), batang sparkline sintetis (tujuh batang berbobot tetap), dan default float kas (`opening_float` 500.000) (`apps/admin/src/routes/(app)/dashboard/+page.svelte`, baris 44-68).
- `payment_method` tidak tersimpan sehingga bauran pembayaran tidak dapat dihitung.

**Peristiwa.** Diproduksi: tidak ada. Dikonsumsi: tidak ada (target: laporan membaca langsung dari tabel penjualan).

### 4.08 NTF (notifikasi): Belum

Tidak ada layanan notifikasi (PRD §10.2 mendaftarkan `NotificationService`, §14.3 mensyaratkan admin menerima notifikasi stok rendah). Peristiwa `Product.LowStock` diterbitkan tetapi tidak ada konsumen (Belum; `services/_shared/nats/streams.go`). Tabel `notifications` pada PRD §12 tidak ada di migrasi.

Alasan ditunda: PRD §18 menempatkan notifikasi pada Phase 2, dan mode `suite` mendelegasikan pengiriman ke hub notifikasi platform lewat port `Notification.Enqueue(ctx, tx, msg)`. Pada mode `standalone`, adapter lokal menulis baris `notifications`/outbox yang dikirim worker ticker (kanal `tbd`). Tidak ada FR POS pada dokumen ini; kebutuhan bisnis BR-POS-17 dipetakan ke FR-POS-03.5 dan modul ini. Saat dikerjakan, FR akan ditambahkan pada revisi FSD.

### 4.09 AI (peramalan permintaan): Belum

Tidak ada layanan AI (`ai-service` tidak ada); subjek `AI.OrderForecasted` hanya konstanta tanpa penerbit (`services/_shared/nats/streams.go`). `POST /api/ai/forecast-demand` pada PRD §11.2 tidak ada di gateway.

Alasan ditunda: PRD §18 menempatkan peramalan dan rekomendasi pada Phase 2 dan harga dinamis, pengenalan gambar, serta chat pada Phase 3 (koreksi atas anggapan bahwa seluruh AI = Phase 3, lihat 11.3); prasyaratnya adalah riwayat penjualan yang andal (bagian 7.1) dan buku besar stok (FR-POS-03.3). Tidak ada FR POS pada dokumen ini; BR-POS-18 ditandai Could.

### 4.SEC Keamanan lintas modul

| ID | Requirement | Prioritas | Status | Bukti |
|---|---|---|---|---|
| FR-POS-SEC.1 | RLS `ENABLE` + `FORCE` pada seluruh tabel bertenant dengan kebijakan `tenant_isolation` | M | Selesai | `services/sales-service/migrations/000001_create_sales_schema.up.sql`; `services/inventory-service/migrations/000001_create_inventory_schema.up.sql` |
| FR-POS-SEC.2 | Setiap kebijakan RLS memiliki `WITH CHECK` (7 dari 15 belum) | M | Sebagian | `services/customer-service/migrations/000001_create_customer_schema.up.sql`; `services/supplier-service/migrations/000001_create_supplier_schema.up.sql` |
| FR-POS-SEC.3 | Cakupan toko ditegakkan pada RLS lewat `app.store_id` untuk tabel berkolom `store_id` | S | Belum | `services/_shared/database/tenant_context.go` |
| FR-POS-SEC.4 | Konteks tenant per transaksi bekerja pada PostgreSQL nyata dan dibuktikan tes integrasi lintas-tenant | M | Pending Verification | `services/_shared/database/tenant_context.go` |
| FR-POS-SEC.5 | Autentikasi layanan-ke-layanan deny-by-default (tanpa token = ditolak), tanpa refleksi gRPC pada produksi | M | Sebagian | `services/_shared/middleware/grpc_interceptors.go` |
| FR-POS-SEC.6 | Tidak ada rahasia pada berkas yang dilacak git; rahasia dirotasi dan dibersihkan dari riwayat | M | Belum | `ecosystem.config.cjs`; `deploy/docker/init-postgres.sql` |
| FR-POS-SEC.7 | Jejak audit append-only untuk aksi sensitif (login, perubahan harga, penyesuaian stok, void, pengguna) | S | Belum | `services/auth-service/migrations/000001_create_auth_schema.up.sql` |
| FR-POS-SEC.8 | Data pribadi pelanggan (nama, telepon, email) dilindungi sesuai keputusan UU PDP (enkripsi `kit/piicrypt` + blind index bila diputuskan) | S | Belum | `services/customer-service/migrations/000001_create_customer_schema.up.sql` |
| FR-POS-SEC.9 | Batas ukuran badan dan batas laju pada API | S | Selesai | `services/api-gateway/cmd/server/main.go` |
| FR-POS-SEC.10 | Sesi web hanya pada cookie HttpOnly; tidak ada token di `localStorage` | S | Sebagian | `apps/admin/src/lib/auth.ts`; `apps/admin/src/lib/server/session.ts` |

**Aturan bisnis**

- Seluruh akses data bisnis berada pada transaksi dengan konteks tenant; hanya mekanisme `WithTenantContext` yang ada: `SET LOCAL app.tenant_id = $1; SET LOCAL app.store_id = $2` (`services/_shared/database/tenant_context.go`). PostgreSQL tidak menerima parameter terikat pada `SET` dan menolak banyak perintah dalam satu pernyataan siap pakai, sehingga pola ini kemungkinan gagal pada koneksi nyata [INFERENCE]; tidak ada tes integrasi, jadi status Pending Verification dan target memakai `set_config(..., true)`.
- Tidak ada kebijakan RLS yang merujuk `app.store_id`; cakupan toko hanya di lapisan aplikasi (kontradiksi dengan README/CLAUDE.md, 11.3).
- Tujuh kebijakan tanpa `WITH CHECK`: `customer.loyalty_points`, `inventory.stock_thresholds`, `product.categories`, `sales.sale_items`, `sales.receipts`, `supplier.purchase_orders`, `supplier.purchase_order_items`. [INFERENCE] PostgreSQL memakai `USING` sebagai pemeriksaan untuk kebijakan `FOR ALL` bila `WITH CHECK` dihilangkan, sehingga celahnya terutama eksplisitas, bukan kebocoran terbukti.
- `auth.audit_logs` ada tetapi tidak ada kode yang menulis dan tidak ada trigger append-only.
- `server.New` mendaftarkan refleksi gRPC (as-is).

**Mesin keadaan.** Tidak berlaku.

**Peristiwa.** Diproduksi: target `audit` lewat `Audit.Record`. Dikonsumsi: tidak ada.

---

## 5. Model Data

Skema as-is: `auth`, `product`, `inventory`, `sales`, `supplier`, `customer`. Peran database as-is `pos_admin` (BYPASSRLS, dipakai migrasi) dan `pos_app` (`services/auth-service/migrations/000001_create_auth_schema.up.sql`; peran dibuat pada `deploy/docker/init-postgres.sql`). Semua primary key UUID dengan `gen_random_uuid()` (target `uuidv7()`). Kolom PII: kelas `PII` = data pribadi, `Rahasia` = kredensial, `-` = bukan PII.

### 5.1 Entitas as-is

| Skema.Tabel | Kolom kunci | RLS | Kelas PII |
|---|---|---|---|
| `auth.tenants` | `id`, `name`, `slug` (UNIQUE), `is_active`, `deleted_at` | tidak ada RLS (komentar migrasi: akses dikendalikan lapisan aplikasi lewat peran `pos_superadmin`; peran itu hanya disebut pada komentar, tidak didefinisikan di `services`, `deploy`, atau `scripts`) | - |
| `auth.stores` | `id`, `tenant_id`, `name`, `address`, `phone`, `is_active`, `deleted_at` | `ENABLE`+`FORCE`, `tenant_isolation` dengan `WITH CHECK` | alamat/telepon toko: data bisnis |
| `auth.roles` | `id`, `name` (UNIQUE; di-seed `admin`, `cashier`, `stock_manager`), `description` | tidak ada RLS | - |
| `auth.users` | `id`, `tenant_id`, `email`, `password_hash` (bcrypt), `name`, `is_active`, `deleted_at`; `UNIQUE(tenant_id, email)` | `ENABLE`+`FORCE` dengan `WITH CHECK` | email dan nama: PII; `password_hash`: Rahasia |
| `auth.user_roles` | `user_id`, `role_id`, `store_id` (NULL = semua toko, non-NULL = terikat toko, mis. kasir), `tenant_id`; unik per (pengguna, peran, tenant, toko) | `ENABLE`+`FORCE` dengan `WITH CHECK` | - |
| `auth.audit_logs` | `id`, `tenant_id`, `user_id`, `action`, `entity`, `entity_id`, `metadata JSONB`, `created_at` | tidak ada RLS; komentar menyebut "append-only" tetapi tidak ada trigger; tidak ada kode yang menulis | `metadata`: `tbd` |
| `auth.refresh_tokens` | `id`, `user_id`, `tenant_id`, `family_id`, `token_hash` (UNIQUE, SHA-256), `issued_at`, `expires_at`, `revoked_at`, `replaced_by`, `user_agent`, `ip_address` | tidak ada RLS (disengaja, pencarian lewat hash sebelum tenant diketahui) | `ip_address` dan `user_agent`: PII terkait; hash token: Rahasia |
| `product.categories` | `id`, `tenant_id`, `name` | `ENABLE`+`FORCE`, tanpa `WITH CHECK` | - |
| `product.products` | `id`, `tenant_id`, `category_id`, `name`, `sku`, `barcode`, `base_price`, `sale_price`, `unit`, `is_active`, `deleted_at`; `UNIQUE(tenant_id, sku)` | `ENABLE`+`FORCE` | - |
| `inventory.stock_items` | `tenant_id`, `store_id`, `product_id`, `quantity BIGINT`; `UNIQUE(tenant_id, store_id, product_id)` | `ENABLE`+`FORCE` | - |
| `inventory.stock_thresholds` | `tenant_id`, `store_id`, `product_id`, `min_quantity` | `ENABLE`+`FORCE`, tanpa `WITH CHECK` | - |
| `sales.sales` | `id`, `tenant_id`, `store_id`, `cashier_id`, `customer_id`, `subtotal`, `discount_amount`, `total`, `status` (default `COMPLETED`), `notes`, `completed_at` | `ENABLE`+`FORCE` | - (`customer_id` rujukan) |
| `sales.sale_items` | `sale_id`, `tenant_id`, `product_id`, `quantity INT`, `unit_price`, `discount`, `line_total` | `ENABLE`+`FORCE`, tanpa `WITH CHECK` | - |
| `sales.receipts` | `sale_id UNIQUE`, `tenant_id`, `store_id`, `snapshot JSONB`, `issued_at` | `ENABLE`+`FORCE`, tanpa `WITH CHECK` | snapshot mungkin memuat nama kasir `tbd` |
| `supplier.suppliers` | `id`, `tenant_id`, `name`, `contact`, `phone`, `email`, `address`, `deleted_at` | `ENABLE`+`FORCE` | PII bisnis (kontak pemasok) |
| `supplier.purchase_orders` | `id`, `tenant_id`, `store_id`, `supplier_id`, `status` (`PENDING`/`RECEIVED`/`CANCELLED`), `notes` | `ENABLE`+`FORCE`, tanpa `WITH CHECK` | - |
| `supplier.purchase_order_items` | `purchase_order_id`, `tenant_id`, `product_id`, `quantity_ordered`, `quantity_received`, `unit_cost` | `ENABLE`+`FORCE`, tanpa `WITH CHECK` | - |
| `customer.customers` | `id`, `tenant_id`, `name`, `phone`, `email`, `is_active`, `deleted_at` | `ENABLE`+`FORCE` | PII (nama, telepon, email) |
| `customer.loyalty_points` | `customer_id`, `tenant_id`, `points`, `total_points`, `sale_id` (tidak `UNIQUE`) | `ENABLE`+`FORCE`, tanpa `WITH CHECK` | - |

Tabel PRD §12 yang tidak ada: `notifications`, `ai_forecasts`, `permissions`. Bukti kolom: migrasi per layanan di `services/*/migrations/`; fungsi `auth.resolve_login_tenant` pada `services/auth-service/migrations/000003_add_login_tenant_resolver.up.sql`.

### 5.2 Tambahan target

| Entitas | Kolom/perubahan | Alasan |
|---|---|---|
| `sales.sales` | `client_sale_id UUID`, `UNIQUE(tenant_id, store_id, client_sale_id)`, `tax_amount`, `payment_method`, `tax_basis` (`tbd`) | idempotensi, PPN, metode pembayaran |
| `sales.sale_items` | `store_id`, `tax_amount` (`tbd`), kunci asing ke `sales.sales` dalam kebijakan `app.store_id` | RLS toko |
| `inventory.stock_items` | `CHECK (quantity >= 0)` | stok non-negatif |
| `inventory.stock_movements` | `id`, `tenant_id`, `store_id`, `product_id`, `delta`, `reason`, `ref_type`, `ref_id`, `actor_id`, `created_at` (append-only) | buku besar |
| `customer.loyalty_points` | `UNIQUE(tenant_id, sale_id)` | idempoten |
| `outbox`, `processed_events` | `event_id`, `event_type`, `payload`, `tenant_id`, `store_id`, `created_at`, `published_at`/`processed_at` | D6 |
| `audit_logs` | `kit/audit`, trigger menolak UPDATE/DELETE/TRUNCATE | FR-POS-SEC.7 |
| `auth.devices` | `id`, `tenant_id`, `store_id`, `label`, `registered_at`, `last_seen_at`, `revoked_at` | `pos.devices.max` [INFERENCE] |
| `auth.stores` | `external_ref` (suite), `org_id` (mode `suite`) | tautan organisasi platform |
| `auth.users` | `external_subject` (mode `suite`) | tautan identitas platform |
| seluruh tabel bertenant | `WITH CHECK` pada setiap kebijakan, kebijakan `app.store_id` pada tabel berkolom `store_id` | G4 |

---

## 6. Spesifikasi API

### 6.1 REST as-is (gateway chi, 14 rute bisnis)

Respons as-is tidak memakai amplop `{success,data,error}`; badan sukses adalah objek proto langsung, galat `{"message": ...}`, validasi 422 gaya Laravel `{message, errors}` (`services/api-gateway/internal/handlers/render.go`). JWT dibaca dari header `Authorization: Bearer` atau cookie `pos_token`; konteks toko dari klaim `sid` atau header `X-Store-ID`.

| Rute as-is | RPC | Peran | Status | Bukti |
|---|---|---|---|---|
| `POST /api/login` | `AuthService.Login` | publik | Selesai | `services/api-gateway/internal/handlers/auth.go` |
| `POST /api/logout` | `AuthService.Logout` | admin, cashier | Selesai | `services/api-gateway/internal/handlers/auth.go` |
| `GET /api/products` | `ProductService.SearchProducts` (hanya `q`) | admin, cashier | Sebagian | `services/api-gateway/internal/handlers/product.go` |
| `POST /api/products` | `CreateProduct` | admin | Selesai | `services/api-gateway/internal/handlers/product.go` |
| `PUT /api/products/{id}` | `UpdateProduct` | admin | Selesai | `services/api-gateway/internal/handlers/product.go` |
| `GET /api/inventory` | `ListItems` | admin, stock_manager | Selesai | `services/api-gateway/internal/handlers/inventory.go` |
| `POST /api/inventory` | `AddItem` | admin, stock_manager | Selesai | `services/api-gateway/internal/handlers/inventory.go` |
| `POST /api/sales` | `CreateSale` | cashier | Sebagian | `services/api-gateway/internal/handlers/sales.go` |
| `GET /api/sales/report` | `GetSalesReport` | admin | Sebagian | `services/api-gateway/internal/handlers/sales.go` |
| `GET /api/suppliers` | `ListSuppliers` | admin, stock_manager | Selesai | `services/api-gateway/internal/handlers/supplier.go` |
| `POST /api/suppliers` | `AddSupplier` | admin | Selesai | `services/api-gateway/internal/handlers/supplier.go` |
| `POST /api/purchase-orders` | `CreatePurchaseOrder` | admin, stock_manager | Selesai | `services/api-gateway/internal/handlers/supplier.go` |
| `GET /api/customers` | `ListCustomers` | admin, cashier | Selesai | `services/api-gateway/internal/handlers/customer.go` |
| `POST /api/customers` | `CreateCustomer` | admin, cashier | Selesai | `services/api-gateway/internal/handlers/customer.go` |
| `GET /healthz`, `GET /readyz` | - | publik | Selesai | `services/api-gateway/internal/handlers/health.go` |

Anotasi `google.api.http` pada proto mendeklarasikan rute `/v1/...` (mis. `GET /v1/sales/{sale_id}`), tetapi tidak ada grpc-gateway yang memasangnya: rute tersebut hanya ada sebagai anotasi (Selesai sebagai kontrak, Belum sebagai API hidup; `proto/pos/sales/v1/sales.proto`). PRD §11.2 mencantumkan `POST /api/ai/forecast-demand` yang tidak ada di kode.

### 6.2 Paritas REST dengan operasi modul (butir gerbang G2)

28 RPC; 14 terekspos, 14 belum. Target route = anotasi proto dengan prefiks `/api/v1` (menggantikan `/v1`).

| Modul | Operasi (RPC) | REST as-is | Rute target `/api/v1` | Gerbang G2 |
|---|---|---|---|---|
| 01 | `Login` | `POST /api/login` | `POST /api/v1/auth/login` | terpenuhi (pindah rute) |
| 01 | `Logout` | `POST /api/logout` | `POST /api/v1/auth/logout` | terpenuhi (pindah rute) |
| 01 | `Refresh` | tidak ada | dihapus (sesi opaque, FR-POS-01.10) | digantikan |
| 01 | `LogoutAll` | tidak ada | `POST /api/v1/auth/logout-all` | belum |
| 01 | `ListSessions` | tidak ada | `GET /api/v1/auth/sessions` | belum |
| 01 | `Validate` | tidak ada (internal) | internal (`Identity.Verify`) | n/a |
| 01 | `ValidateStore` | tidak ada (dipakai gateway) | internal (atribut sesi `store_id`) | n/a |
| 01 | provisioning tenant/toko/pengguna | tidak ada | `POST /api/v1/stores`, `POST /api/v1/users` dst. (`tbd`, D-POS-12) | belum |
| 02 | `CreateProduct`, `UpdateProduct`, `SearchProducts` | ada (sebagian parameter) | `POST/PUT/GET /api/v1/products` | pindah rute |
| 02 | `GetProduct` | tidak ada | `GET /api/v1/products/{product_id}` | belum |
| 03 | `AddItem`, `ListItems` | ada | `POST/GET /api/v1/inventory` | pindah rute |
| 03 | `GetItem` | tidak ada | `GET /api/v1/inventory/{product_id}` | belum |
| 03 | `UpdateStock` | tidak ada | `PATCH /api/v1/inventory/{product_id}/stock` | belum |
| 03 | `SetThreshold` | tidak ada | `PUT /api/v1/inventory/{product_id}/threshold` | belum |
| 04 | `CreateSale` | ada (field terbatas) | `POST /api/v1/sales` | pindah rute |
| 04 | `GetSale` | tidak ada | `GET /api/v1/sales/{sale_id}` | belum |
| 07 | `GetSalesReport` | ada | `GET /api/v1/sales/report` | pindah rute |
| 05 | `AddSupplier`, `ListSuppliers`, `CreatePurchaseOrder` | ada | `POST/GET /api/v1/suppliers`, `POST /api/v1/purchase-orders` | pindah rute |
| 05 | `ListPurchaseOrders` | tidak ada | `GET /api/v1/purchase-orders` | belum |
| 05 | `ReceiveStock` | tidak ada | `POST /api/v1/purchase-orders/{purchase_order_id}/receive` | belum |
| 06 | `CreateCustomer`, `ListCustomers` | ada | `POST/GET /api/v1/customers` | pindah rute |
| 06 | `GetCustomer` | tidak ada | `GET /api/v1/customers/{customer_id}` | belum |
| 06 | `AddLoyaltyPoints` | tidak ada | tidak terekspos: dipanggil dalam proses (FR-POS-06.3); tampilan poin `GET /api/v1/customers/{customer_id}` | digantikan |

### 6.3 Kontrak REST target

- Prefiks `/api/v1` pada ServeMux stdlib; amplop `{success, data, error}` (`kit/httpx`); galat internal berpesan generik tetap; pembatasan dengan `RequirePermission(code)`.
- Sesi lewat cookie HttpOnly; tanpa CORS (web dan API satu origin lewat reverse proxy, seperti SELA).
- `POST /api/v1/sales` menerima `client_sale_id` (header `Idempotency-Key` atau badan; bentuk final `tbd`), `customer_id?`, `items[{product_id, quantity, discount?}]`, `payment_method` (`tbd`), `notes?`; `unit_price` TIDAK diterima dari klien. Respons: penjualan + struk.
- Paginasi `limit`/`offset` konsisten pada semua daftar (as-is hanya `q` yang diteruskan pada pencarian produk).
- Rute tingkat toko mensyaratkan atribut sesi `store_id`.

### 6.4 Kontrak gRPC

| Layanan | Berkas proto | RPC | Status | Bukti |
|---|---|---|---|---|
| `pos.auth.v1.AuthService` | `proto/pos/auth/v1/auth.proto` | 7: `Login`, `Validate`, `Logout`, `Refresh`, `LogoutAll`, `ListSessions`, `ValidateStore` | Selesai | `services/auth-service/internal/interfaces/grpc/auth_service.go` |
| `pos.product.v1.ProductService` | `proto/pos/product/v1/product.proto` | 4 | Selesai | `services/product-service/internal/interfaces/grpc/product_service.go` |
| `pos.inventory.v1.InventoryService` | `proto/pos/inventory/v1/inventory.proto` | 5: `AddItem`, `UpdateStock`, `ListItems`, `GetItem`, `SetThreshold` | Selesai | `services/inventory-service/internal/interfaces/grpc/inventory_service.go` |
| `pos.sales.v1.SalesService` | `proto/pos/sales/v1/sales.proto` | 3 | Selesai | `services/sales-service/internal/interfaces/grpc/sales_service.go` |
| `pos.supplier.v1.SupplierService` | `proto/pos/supplier/v1/supplier.proto` | 5 | Selesai | `services/supplier-service/internal/interfaces/grpc/supplier_service.go` |
| `pos.customer.v1.CustomerService` | `proto/pos/customer/v1/customer.proto` | 4 | Selesai | `services/customer-service/internal/interfaces/grpc/customer_service.go` |

Catatan: `DeductStock` dan `RestoreStock` bukan RPC, melainkan metode pembungkus klien pada sales-service yang memanggil `InventoryService.UpdateStock` dengan `quantity_delta` negatif atau positif (`services/sales-service/internal/infrastructure/grpc_clients/inventory_client.go`). Target: gRPC hanya untuk klien mesin (`api/proto/v1`, `buf`, `gen/go` ter-commit, `scripts/proto-gen.sh --check`) dengan interseptor deny-by-default; antar-modul memakai interface Go.

---

## 7. Alur Proses Kritis

### 7.1 Penjualan sebagai satu transaksi database (menggantikan saga gRPC)

**As-is (saga).**

1. Gateway memverifikasi JWT, menentukan `tenant_id`, `store_id`, `cashier_id`, lalu memanggil `SalesService.CreateSale` (peran `cashier`).
2. `create_sale.go` memvalidasi masukan dan menghitung subtotal, diskon, total (total dijepit ke 0) dari harga dan diskon yang dikirim klien.
3. Untuk setiap item, `inventory.DeductStock` dipanggil lewat gRPC; kegagalan di tengah memicu `compensate()` (`RestoreStock`, hanya peringatan log bila gagal).
4. `saleRepo.Create` menulis penjualan, item, dan struk dalam satu transaksi.
5. `PublishSaleCompleted` ke NATS, best-effort.
6. Loyalitas tidak dipanggil. Tidak ada kunci idempotensi.

Kegagalan yang mungkin: proses mati antara langkah 3 dan 4 (stok berkurang tanpa penjualan); retry setelah timeout (penjualan ganda atau stok terpotong ganda); `RestoreStock` gagal (stok hilang).

**Target.**

1. Klien mengirim `POST /api/v1/sales` dengan `client_sale_id` (UUID dibuat di perangkat), item (tanpa harga), `customer_id?`, `payment_method?`.
2. `kit/tenantdb.WithTenant(ctx, tenantID, scope tenant+store, fn)` membuka transaksi, mengatur `app.tenant_id` dan `app.store_id` lewat `set_config(..., true)`.
3. Dalam transaksi: `INSERT ... ON CONFLICT (tenant_id, store_id, client_sale_id) DO NOTHING`; bila konflik, kembalikan penjualan yang sudah ada (idempoten).
4. Harga satuan dibaca dari `product.products`; diskon divalidasi terhadap wewenang peran (D-POS-07); PPN dihitung (D-POS-01); total dihitung dengan representasi uang eksak.
5. Untuk setiap item: `UPDATE inventory.stock_items SET quantity = quantity - $q WHERE ... AND quantity - $q >= 0`; nol baris = stok kurang -> transaksi dibatalkan dengan galat 409; tulis `stock_movements` (`reason = SALE`, `ref_id = sale_id`).
6. Tulis `sales.sales`, `sales.sale_items`, `sales.receipts` (snapshot).
7. Tulis baris outbox `sales.sale.completed` (dan `inventory.low_stock` bila melewati ambang) pada transaksi yang sama; `COMMIT`.
8. Worker outbox dalam proses memberikan `sales.sale.completed` ke konsumen loyalitas (idempoten per `sale_id`, transaksi sendiri, `UNIQUE(tenant_id, sale_id)`); kegagalan loyalitas tidak menggagalkan penjualan dan diulang.

Titik gagal: transaksi tunggal menghilangkan keadaan setengah jadi; retry aman lewat `client_sale_id`. Aturan bisnis: PRD §13 (stok non-negatif, atomik, struk wajib, poin hanya setelah penjualan). Pengujian wajib: dua penjualan serentak pada stok 1, retry dengan `client_sale_id` sama, lintas-tenant.

### 7.2 Penerimaan PO

**As-is** (`receive_stock.go`): ambil PO (`PENDING`); untuk tiap item dalam permintaan (item di luar permintaan dilewati) panggil inventory `UpdateStock` (RECEIPT) lewat gRPC, lalu publikasi `Inventory.Replenished`; tandai seluruh PO `RECEIVED`. Gagal di tengah -> stok sebagian bertambah dan PO tetap `PENDING`; retry menambah ganda.

**Target.**

1. `POST /api/v1/purchase-orders/{id}/receive` dengan daftar `{product_id, quantity_received}` dan `Idempotency-Key`.
2. Dalam satu transaksi tenant+store: kunci PO (`SELECT ... FOR UPDATE`); validasi status bukan `RECEIVED`/`CANCELLED`; setiap kuantitas <= `quantity_ordered - quantity_received`.
3. Per item: tambah `quantity_received`; tambah stok (`stock_items` upsert) dan tulis `stock_movements` (`reason = RECEIPT`, `ref_id = po_id`).
4. Status PO: semua item lengkap -> `RECEIVED`, sebagian -> `PARTIAL`.
5. Tulis outbox `supplier.stock.received`; `COMMIT`.

Hanya peran `admin` dan `stock_manager` (izin `purchase_orders:receive`, katalog `tbd`).

### 7.3 Antrean kasir offline

Status as-is: Belum. Service worker (`apps/cashier/src/service-worker.ts`) hanya menyimpan aset build (stale-while-revalidate, melewati non-GET dan `/api/`); `manifest.json` memiliki `start_url` `/pos` dan satu ikon; tidak ada antrean penjualan.

**Target** (mengikuti pola antrean unggah tamu SELA: IndexedDB `queue`, mesin keadaan pengunggah, pemicu sinkron, kebijakan cache `policy.ts`; `SELA/apps/web/src/lib/pwa/policy.ts`):

1. Service worker dengan scope `/cashier/` (`D9`), aset aplikasi dan katalog produk terakhir (hanya baca) di-cache.
2. Saat offline, kasir menyelesaikan penjualan ke antrean IndexedDB dengan `client_sale_id` UUID; struk sementara ditandai "belum tersinkron".
3. Saat jaringan kembali, pengunggah mengirim antrean berurutan lewat `POST /api/v1/sales` dengan `client_sale_id` (idempoten, 7.1).
4. Status per penjualan antrean: `QUEUED -> SYNCING -> SYNCED | FAILED` (penolakan 409 stok kurang atau 422 harga: tampil di daftar "perlu tindakan" kasir/admin).
5. Kebijakan konflik: stok kurang saat sinkron -> penjualan tetap sah (barang sudah terjual) dan stok ditandai selisih untuk admin; atau ditolak (keputusan D-POS-03, belum ditetapkan).
6. Batas durasi offline, kapasitas antrean, dan pencatatan perangkat (`pos.devices.max`): `tbd`.
7. Autentikasi offline: sesi kasir yang sudah ada tetap berlaku sampai kedaluwarsa; login baru memerlukan jaringan (`tbd`, D-POS-03).

---

## 8. Keamanan, Privasi dan UU PDP

### 8.1 Model ancaman dan kontrol

| Ancaman | Kontrol as-is | Status | Bukti | Target |
|---|---|---|---|---|
| Kebocoran lintas tenant | RLS `ENABLE`+`FORCE`, filter `tenant_id` pada kueri; 7 kebijakan tanpa `WITH CHECK`; tidak ada tes integrasi | Sebagian | `services/_shared/database/tenant_context.go` | RLS lengkap + `kit/tenantdb` + tes lintas-tenant (G4, G6) |
| Pemalsuan layanan internal | `x-service-token` HMAC (jendela 30 detik) diverifikasi hanya bila ada; kosong/`changeme`/tanpa header diterima | Sebagian | `services/_shared/middleware/grpc_interceptors.go` | deny-by-default; refleksi gRPC dimatikan di produksi |
| Pencurian kredensial dari repo | rahasia literal pada berkas terlacak | Belum | `ecosystem.config.cjs` | rotasi, bersihkan riwayat, `.env.example` (G5) |
| Pencurian sesi | JWT 10 menit di cookie httpOnly; daftar hitam `jti`; refresh 30 hari | Sebagian | `services/auth-service/internal/infrastructure/jwt/signer.go` | sesi opaque, hash di Dragonfly |
| Brute force login | 20 per menit per IP; tidak ada kunci akun | Sebagian | `services/api-gateway/cmd/server/main.go` | batas laju per akun dan alamat + lockout |
| Eskalasi peran | `WithRole` per rute; tanpa katalog izin | Sebagian | `services/api-gateway/internal/middleware/role.go` | `RequirePermission` |
| Penipuan harga/diskon oleh klien | harga dan diskon dari klien diterima | Belum | `services/api-gateway/internal/handlers/sales.go` | harga dari server, batas wewenang |
| Penjualan ganda | tidak ada kunci idempotensi | Belum | `services/sales-service/internal/application/commands/create_sale.go` | `client_sale_id` |
| Penyangkalan aksi | `auth.audit_logs` tidak ditulis | Belum | `services/auth-service/migrations/000001_create_auth_schema.up.sql` | `kit/audit` append-only |

Transportasi gRPC antar-layanan sengaja plaintext (`docs/security/grpc-transport-security.md`); pada target tidak ada gRPC antar-layanan (satu proses) sehingga risiko ini hilang untuk jalur internal, dan gRPC mesin memakai TLS di Caddy `tbd`.

### 8.2 Data pribadi dan UU PDP

| Data | Lokasi | Kelas | Status perlindungan | Bukti |
|---|---|---|---|---|
| Nama, telepon, email pelanggan | `customer.customers` | PII | teks biasa, tanpa enkripsi kolom | `services/customer-service/migrations/000001_create_customer_schema.up.sql` |
| Email pengguna (staf) | `auth.users` | PII | teks biasa | `services/auth-service/migrations/000001_create_auth_schema.up.sql` |
| Kata sandi | `auth.users.password_hash` | Rahasia | bcrypt | `services/auth-service/internal/application/commands/login.go` |
| Kontak pemasok (nama, telepon, email) | `supplier.suppliers` | PII bisnis | teks biasa | `services/supplier-service/migrations/000001_create_supplier_schema.up.sql` |
| Riwayat belanja pelanggan | `sales.sales.customer_id` | PII terkait | pseudonim per id | `services/sales-service/migrations/000001_create_sales_schema.up.sql` |

- Tidak ada pernyataan UU PDP di repo `POS-Stery` (pencarian menyeluruh; tidak ada). Penerapan UU No. 27/2022 pada data pelanggan toko, dasar pemrosesan (persetujuan/kontrak), retensi, hak subjek data (akses, koreksi, hapus), notifikasi insiden, dan penunjukan penanggung jawab adalah keputusan terbuka D-POS-08 dan membutuhkan tinjauan hukum sebelum S2 ditutup. Dokumen ini tidak menyatakan kepatuhan.
- Target teknis yang disiapkan: kolom PII dienkripsi dengan `kit/piicrypt` (AES-256-GCM, AAD per tujuan) disertai blind index untuk pencarian telepon/email bila diputuskan; penghapusan pelanggan memakai anonimisasi (riwayat penjualan dan poin dipertahankan tanpa PII); tabel audit append-only.
- Struk bersifat dokumen transaksi; apakah snapshot boleh memuat data pelanggan ditentukan oleh keputusan D-POS-01 dan D-POS-08 (lihat BRD 9.3).

### 8.3 Kebersihan rahasia dan rantai pasok

- Berkas terlacak yang memuat rahasia (hanya jalur): `ecosystem.config.cjs` (token layanan, kata sandi Redis, NATS, Postgres) dan `deploy/docker/init-postgres.sql` (kata sandi peran `pos_admin` dan `pos_app`). `scripts/migrate-helper.sh` memiliki kata sandi admin bawaan lemah. `docs/security/grpc-transport-security.md` menyebut nilai bawaan `changeme`.
- Tidak dilacak git tetapi ada di disk kerja: `.env`, `keys/private.pem`, `keys/public.pem`, `server.exe`, `apps/*/build/` (diabaikan `.gitignore`; koreksi atas klaim baseline, lihat 11.3). Kunci JWT pribadi dipakai pada pengembangan lokal; pada target JWT dihapus.
- Setelah rotasi, riwayat git perlu dibersihkan dan semua kredensial lama dianggap bocor (G5).
- Dependensi Go berbeda versi antar layanan (grpc, pgx, nats.go, jwt/v5); target satu `go.mod`.

---

## 9. Kebutuhan Non-Fungsional Teknis

Seluruh angka kinerja dan kapasitas adalah `tbd` (tidak ada di repo, termasuk PRD yang hanya menyebut "high concurrency" tanpa angka); baseline hanya bila terhitung dari kode.

| ID | Kebutuhan | Kategori | Target | Status as-is | Bukti |
|---|---|---|---|---|---|
| NFR-POS-01 | Waktu selesai penjualan (p95) per toko | Kinerja | `tbd` | Tidak diukur | `services/sales-service/internal/application/commands/create_sale.go` |
| NFR-POS-02 | Jumlah kasir serentak per toko dan toko per tenant | Kapasitas | `tbd` (selaras `pos.stores.max`, `pos.devices.max`) | Tidak diukur | `services/api-gateway/cmd/server/main.go` |
| NFR-POS-03 | Ketersediaan layanan API | Ketersediaan | `tbd` (SLA per tingkat) | Tidak ada pemantauan | `services/api-gateway/internal/handlers/health.go` |
| NFR-POS-04 | Penjualan atomik dan idempoten | Integritas | tanpa penjualan ganda, tanpa stok negatif | Tidak terpenuhi | `services/sales-service/internal/application/commands/create_sale.go` |
| NFR-POS-05 | Isolasi tenant dan toko dibuktikan tes lintas-tenant per modul | Keamanan | 100% modul | Belum ada tes integrasi | `services/_shared/database/tenant_context.go` |
| NFR-POS-06 | Cakupan tes minimal 80% (Go dan web), `-race` di CI | Kualitas | 80% | 79 fungsi tes unit berbasis stub, CI usang | `.github/workflows/ci.yml` |
| NFR-POS-07 | Seluruh migrasi naik -> turun -> naik pada DB baru di CI | Kualitas | wajib | Skrip hanya menghitung pasangan berkas | `scripts/migrate-validate.sh` |
| NFR-POS-08 | Konfigurasi lewat env fail-fast, rahasia tidak dicatat, `.env.example` tanpa nilai rahasia | Operasional | wajib | Sebagian | `services/_shared/config` |
| NFR-POS-09 | Logging terstruktur `slog` JSON dan `/healthz`; metrik/tracing = D11 `tbd` | Observabilitas | `slog` + `/healthz` | zap; tidak ada metrik/tracing sebenarnya | `services/_shared/observability` |
| NFR-POS-10 | Ketahanan kasir offline: PWA scope `/cashier/`, antrean IndexedDB | Ketahanan | durasi offline `tbd` | Belum | `apps/cashier/src/service-worker.ts` |
| NFR-POS-11 | Cadangan dan pemulihan Postgres | Operasional | RPO/RTO `tbd` | Tidak ada | `deploy/docker/docker-compose.yml` |
| NFR-POS-12 | Retensi data transaksi dan data pribadi | Kepatuhan | `tbd` (D-POS-08) | Tidak ada kebijakan | `services/customer-service/migrations/000001_create_customer_schema.up.sql` |
| NFR-POS-13 | Lokalisasi: Indonesia default, Inggris didukung, IDR, WIB; batas hari WIB | Kegunaan | wajib | UI Inggris; waktu DB sesi | `apps/cashier/src/routes/(app)/pos/+page.svelte` |
| NFR-POS-14 | Aksesibilitas dan kegunaan layar kasir (target sentuh, kontras, keyboard/pemindai) | Kegunaan | `tbd` | Tidak dinilai | `apps/cashier/src/routes/(app)/pos/+page.svelte` |
| NFR-POS-15 | Satu stack dan satu konvensi (STACK-SPEC) dengan perilaku `standalone` tidak berubah di mode `suite` | Maintainability | wajib | Menyimpang | `go.work` |
| NFR-POS-16 | Degradasi aman saat platform tidak tersedia: snapshot entitlement 24 jam (dapat dikonfigurasi, `tbd`), gagal-tertutup untuk fitur berbayar, penjualan `standalone` tidak pernah diblokir | Ketahanan | 24 jam (`tbd`) | Belum ada mode `suite` | `services/api-gateway/internal/config/config.go` |

---

## 10. Struktur Layar

### 10.1 Layar as-is

| Aplikasi | Layar | Fungsi | Status | Bukti |
|---|---|---|---|---|
| Admin | `(auth)/login` | login admin/stok | Selesai | `apps/admin/src/routes/(auth)/login/+page.svelte` |
| Admin | `(app)/dashboard` | ringkasan: pendapatan, jumlah transaksi, produk teratas; bauran pembayaran dan sparkline palsu; nama field tidak cocok | Sebagian | `apps/admin/src/routes/(app)/dashboard/+page.svelte` |
| Admin | `(app)/products` | daftar produk (hanya baca) | Sebagian | `apps/admin/src/routes/(app)/products/+page.svelte` |
| Admin | `(app)/inventory` | daftar inventaris (hanya baca) | Sebagian | `apps/admin/src/routes/(app)/inventory/+page.svelte` |
| Admin | `(app)/suppliers` | daftar pemasok (hanya baca) | Sebagian | `apps/admin/src/routes/(app)/suppliers/+page.svelte` |
| Admin | `(app)/customers` | daftar pelanggan (hanya baca) | Sebagian | `apps/admin/src/routes/(app)/customers/+page.svelte` |
| Admin | `(app)/reports` | laporan penjualan | Sebagian | `apps/admin/src/routes/(app)/reports/+page.svelte` |
| Kasir | `(auth)/login` | login kasir; meminta Store ID (UUID) diketik | Sebagian | `apps/cashier/src/routes/(auth)/login/+page.svelte` |
| Kasir | `(app)/pos` | cari produk, keranjang, tender (Tunai/QRIS/Debit hanya UI), struk | Sebagian | `apps/cashier/src/routes/(app)/pos/+page.svelte` |
| Kasir | `(app)/receipts` | tempat penampung: teks "Receipt history will appear here. PDF export available in Phase 2." | Belum | `apps/cashier/src/routes/(app)/receipts/+page.svelte` |
| Kedua | BFF `api/[...path]`, `auth/session` | proxy ke gateway, cookie sesi | Selesai | `apps/cashier/src/routes/auth/session/+server.ts` |
| Kasir | Service worker + manifest | cache aset; tanpa antrean offline; `start_url` `/pos` | Sebagian | `apps/cashier/static/manifest.json` |

Temuan UI (dibaca statis, `[INFERENCE]` untuk dampak runtime):

- Tidak ada selektor toko admin dan tidak ada halaman admin yang mengirim `X-Store-ID`, sementara gateway mewajibkannya (403 "Store context required.") untuk peran admin: halaman admin kemungkinan tidak dapat memuat data (`services/api-gateway/internal/middleware/store.go`).
- Tidak ada alur refresh token di web maupun gateway: JWT berumur 10 menit sementara cookie berumur 30 hari.
- Helper `getToken`/`setSession` berbasis `localStorage` ada tetapi `setSession` tidak pernah dipanggil (kode mati).
- Fallback URL BFF `http://localhost:8080` berbeda dari port bawaan gateway 8000.
- Tampilan struk di kasir mengharapkan `{sale_id, total_amount, items_count}`, gateway mengembalikan `{sale: {...}, receipt: {...}}` sehingga struk menampilkan "—" dan 0 [INFERENCE].
- Placeholder pencarian menyebut SKU yang tidak dicari.

### 10.2 Struktur target (D9)

Satu `apps/web` (SvelteKit SPA, `ssr=false`, `adapter-static`):

| Route group | Halaman | Peran |
|---|---|---|
| `(auth)` | login (email + kata sandi; kasir memilih toko dari daftar yang ditugaskan, bukan mengetik UUID) | semua |
| `(admin)` | dashboard, produk (buat/ubah/kategori/riwayat harga), inventaris (penyesuaian, ambang, pergerakan), pemasok, PO (buat/terima/batal), pelanggan, laporan, pengaturan toko dan pengguna, selektor toko | admin, stock_manager |
| `(cashier)` | POS, riwayat dan cetak struk, antrean offline ("perlu tindakan") | cashier |

Scope PWA `/cashier/`; `apiFetch` dengan `credentials: 'include'`; i18n `{id,en}` dengan tes paritas; komponen dari `packages/ui` dipindahkan.

---

## 11. Matriks Ketertelusuran dan Status

### 11.1 Status per FR

| ID | Prioritas | Status | Bukti |
|---|---|---|---|
| FR-POS-01.1 | M | Selesai | `services/auth-service/internal/application/commands/login.go` |
| FR-POS-01.2 | M | Selesai | `services/auth-service/internal/application/commands/login.go` |
| FR-POS-01.3 | M | Sebagian | `services/api-gateway/internal/middleware/store.go` |
| FR-POS-01.4 | M | Sebagian | `services/auth-service/internal/application/commands/refresh.go` |
| FR-POS-01.5 | S | Sebagian | `services/auth-service/internal/application/commands/logout.go` |
| FR-POS-01.6 | M | Selesai | `services/api-gateway/internal/middleware/role.go` |
| FR-POS-01.7 | M | Selesai | `services/api-gateway/cmd/server/main.go` |
| FR-POS-01.8 | M | Belum | `services/auth-service/internal/infrastructure/postgres/tenant_repository.go` |
| FR-POS-01.9 | S | Belum | `services/auth-service/migrations/000001_create_auth_schema.up.sql` |
| FR-POS-01.10 | M | Belum | `services/auth-service/internal/infrastructure/jwt/signer.go` |
| FR-POS-01.11 | S | Sebagian | `apps/admin/src/hooks.server.ts` |
| FR-POS-01.12 | M | Belum | `services/api-gateway/internal/config/config.go` |
| FR-POS-01.13 | M | Belum | `services/auth-service/internal/infrastructure/postgres/store_repository.go` |
| FR-POS-02.1 | M | Selesai | `services/product-service/internal/application/commands/create_product.go` |
| FR-POS-02.2 | M | Sebagian | `services/product-service/internal/application/commands/update_product.go` |
| FR-POS-02.3 | S | Sebagian | `services/product-service/internal/interfaces/grpc/product_service.go` |
| FR-POS-02.4 | M | Sebagian | `services/product-service/internal/infrastructure/postgres/product_repository.go` |
| FR-POS-02.5 | S | Belum | `services/product-service/internal/application/commands/update_product.go` |
| FR-POS-02.6 | S | Belum | `services/product-service/migrations/000001_create_product_schema.up.sql` |
| FR-POS-02.7 | M | Belum | `apps/admin/src/routes/(app)/products/+page.svelte` |
| FR-POS-03.1 | M | Selesai | `services/inventory-service/internal/application/commands/add_item.go` |
| FR-POS-03.2 | M | Sebagian | `services/inventory-service/internal/infrastructure/postgres/stock_repository.go` |
| FR-POS-03.3 | M | Belum | `services/inventory-service/internal/application/commands/update_stock.go` |
| FR-POS-03.4 | M | Sebagian | `services/inventory-service/internal/application/commands/set_threshold.go` |
| FR-POS-03.5 | M | Sebagian | `services/inventory-service/internal/infrastructure/nats_publisher.go` |
| FR-POS-03.6 | M | Sebagian | `services/inventory-service/internal/application/queries/list_items.go` |
| FR-POS-03.7 | M | Belum | `services/api-gateway/internal/handlers/inventory.go` |
| FR-POS-04.1 | M | Sebagian | `services/sales-service/internal/application/commands/create_sale.go` |
| FR-POS-04.2 | M | Sebagian | `services/sales-service/internal/application/commands/create_sale.go` |
| FR-POS-04.3 | M | Belum | `services/sales-service/internal/application/commands/create_sale.go` |
| FR-POS-04.4 | M | Belum | `services/api-gateway/internal/handlers/sales.go` |
| FR-POS-04.5 | M | Belum | `services/sales-service/internal/domain/sale.go` |
| FR-POS-04.6 | S | Belum | `apps/cashier/src/routes/(app)/pos/+page.svelte` |
| FR-POS-04.7 | M | Sebagian | `services/sales-service/internal/infrastructure/postgres/sale_repository.go` |
| FR-POS-04.8 | M | Sebagian | `services/sales-service/internal/interfaces/grpc/sales_service.go` |
| FR-POS-04.9 | M | Sebagian | `services/sales-service/internal/infrastructure/nats_publisher.go` |
| FR-POS-04.10 | M | Sebagian | `apps/cashier/src/routes/(app)/pos/+page.svelte` |
| FR-POS-04.11 | C | Belum | `services/sales-service/internal/domain/sale.go` |
| FR-POS-04.12 | S | Belum | `apps/cashier/src/service-worker.ts` |
| FR-POS-05.1 | M | Selesai | `services/supplier-service/internal/application/commands/add_supplier.go` |
| FR-POS-05.2 | M | Selesai | `services/supplier-service/internal/interfaces/grpc/supplier_service.go` |
| FR-POS-05.3 | M | Sebagian | `services/supplier-service/internal/application/commands/create_purchase_order.go` |
| FR-POS-05.4 | S | Sebagian | `services/supplier-service/internal/interfaces/grpc/supplier_service.go` |
| FR-POS-05.5 | M | Sebagian | `services/supplier-service/internal/application/commands/receive_stock.go` |
| FR-POS-05.6 | S | Belum | `services/supplier-service/internal/application/commands/receive_stock.go` |
| FR-POS-05.7 | S | Sebagian | `services/supplier-service/internal/infrastructure/nats_publisher.go` |
| FR-POS-06.1 | M | Selesai | `services/customer-service/internal/application/commands/create_customer.go` |
| FR-POS-06.2 | S | Sebagian | `services/customer-service/internal/interfaces/grpc/customer_service.go` |
| FR-POS-06.3 | M | Belum | `services/customer-service/internal/application/commands/add_loyalty_points.go` |
| FR-POS-06.4 | S | Belum | `services/customer-service/internal/application/commands/add_loyalty_points.go` |
| FR-POS-06.5 | C | Sebagian | `services/customer-service/internal/infrastructure/nats_publisher.go` |
| FR-POS-06.6 | S | Belum | `apps/cashier/src/routes/(app)/pos/+page.svelte` |
| FR-POS-07.1 | M | Sebagian | `services/sales-service/internal/infrastructure/postgres/sale_repository.go` |
| FR-POS-07.2 | M | Sebagian | `services/sales-service/internal/infrastructure/postgres/sale_repository.go` |
| FR-POS-07.3 | S | Belum | `apps/admin/src/routes/(app)/dashboard/+page.svelte` |
| FR-POS-07.4 | M | Sebagian | `apps/admin/src/routes/(app)/dashboard/+page.svelte` |
| FR-POS-07.5 | S | Belum | `services/sales-service/internal/infrastructure/postgres/sale_repository.go` |
| FR-POS-07.6 | S | Sebagian | `services/api-gateway/internal/handlers/sales.go` |
| FR-POS-SEC.1 | M | Selesai | `services/sales-service/migrations/000001_create_sales_schema.up.sql` |
| FR-POS-SEC.2 | M | Sebagian | `services/customer-service/migrations/000001_create_customer_schema.up.sql` |
| FR-POS-SEC.3 | S | Belum | `services/_shared/database/tenant_context.go` |
| FR-POS-SEC.4 | M | Pending Verification | `services/_shared/database/tenant_context.go` |
| FR-POS-SEC.5 | M | Sebagian | `services/_shared/middleware/grpc_interceptors.go` |
| FR-POS-SEC.6 | M | Belum | `ecosystem.config.cjs` |
| FR-POS-SEC.7 | S | Belum | `services/auth-service/migrations/000001_create_auth_schema.up.sql` |
| FR-POS-SEC.8 | S | Belum | `services/customer-service/migrations/000001_create_customer_schema.up.sql` |
| FR-POS-SEC.9 | S | Selesai | `services/api-gateway/cmd/server/main.go` |
| FR-POS-SEC.10 | S | Sebagian | `apps/admin/src/lib/server/session.ts` |

Modul 08 NTF dan 09 AI tidak memiliki FR (Belum, lihat 4.08 dan 4.09).

Ringkasan hitung status FR: 68 FR (13 + 7 + 7 + 12 + 7 + 6 + 6 + 10). Selesai 11, Sebagian 30, Belum 26, Pending Verification 1 (hitungan dari tabel di atas; bukan persentase kesiapan).

### 11.2 Matriks BR -> FR

Seluruh BR didefinisikan di `BRD_POS.id.md` bagian 6. Kolom NFR menunjuk kebutuhan non-fungsional pada bagian 9.

| BR | FR terkait | NFR terkait | Catatan |
|---|---|---|---|
| BR-POS-01 | FR-POS-04.1, FR-POS-04.2, FR-POS-04.3, FR-POS-04.4, FR-POS-04.7, FR-POS-04.8, FR-POS-04.10 | NFR-POS-01, NFR-POS-04 | Alur 7.1 |
| BR-POS-02 | FR-POS-SEC.1, FR-POS-SEC.2, FR-POS-SEC.3, FR-POS-SEC.4 | NFR-POS-05 | Gerbang G4, G6 |
| BR-POS-03 | FR-POS-01.2, FR-POS-01.3, FR-POS-01.6, FR-POS-01.9 | - | Matriks 3.2 |
| BR-POS-04 | FR-POS-02.1, FR-POS-02.2, FR-POS-02.3, FR-POS-02.4, FR-POS-02.5, FR-POS-02.6, FR-POS-02.7 | - | - |
| BR-POS-05 | FR-POS-03.1, FR-POS-03.2, FR-POS-03.3, FR-POS-03.4, FR-POS-03.5, FR-POS-03.6, FR-POS-03.7 | NFR-POS-04 | - |
| BR-POS-06 | FR-POS-05.1, FR-POS-05.2, FR-POS-05.3, FR-POS-05.4, FR-POS-05.5, FR-POS-05.6, FR-POS-05.7 | - | Alur 7.2 |
| BR-POS-07 | FR-POS-06.1, FR-POS-06.2, FR-POS-06.3, FR-POS-06.4, FR-POS-06.6 | - | Gerbang G1 |
| BR-POS-08 | FR-POS-07.1, FR-POS-07.2, FR-POS-07.3, FR-POS-07.4, FR-POS-07.5, FR-POS-07.6 | NFR-POS-13 | Gerbang G7 |
| BR-POS-09 | FR-POS-04.5 | - | D-POS-01; gerbang G8 |
| BR-POS-10 | FR-POS-04.7, FR-POS-04.8, FR-POS-04.10 | - | D-POS-02; gerbang G8 |
| BR-POS-11 | FR-POS-04.12 | NFR-POS-10 | Alur 7.3; D-POS-03 |
| BR-POS-12 | FR-POS-04.6, FR-POS-07.3 | - | D-POS-04 |
| BR-POS-13 | FR-POS-SEC.5, FR-POS-SEC.6, FR-POS-SEC.7, FR-POS-SEC.10, FR-POS-01.5, FR-POS-01.7 | NFR-POS-08 | Gerbang G5 |
| BR-POS-14 | FR-POS-SEC.4 | NFR-POS-06, NFR-POS-07 | Gerbang G3, G6 |
| BR-POS-15 | FR-POS-01.10, FR-POS-01.11 | NFR-POS-09, NFR-POS-15 | Delta 2.7 |
| BR-POS-16 | FR-POS-01.12, FR-POS-01.13 | NFR-POS-16 | Mode `suite` |
| BR-POS-17 | FR-POS-03.5 | - | Modul 4.08 tanpa FR (Belum) |
| BR-POS-18 | (tidak ada FR POS) | - | Modul 4.09 tanpa FR (Belum, PRD fase 2/3) |
| BR-POS-19 | FR-POS-SEC.8 | NFR-POS-12 | D-POS-08 |
| BR-POS-20 | (tidak ada FR POS) | - | Hipotesis INT-3, dispesifikasikan pada FSD Stery Suite 4.07; Ditunda |
| BR-POS-21 | FR-POS-01.8 | - | D-POS-12 |

### 11.3 Kontradiksi dokumen status vs kode

**A. Kontradiksi dokumen terhadap kode** (kode = kebenaran; dokumen tidak diubah oleh pekerjaan ini).

| # | Dokumen | Klaim | Kode menunjukkan | Bukti |
|---|---|---|---|---|
| K1 | `PRD.md` §10.1 | "Laravel gateway for auth and HTTP API" | Gateway Go `chi`; tidak ada direktori Laravel | `services/api-gateway/cmd/server/main.go` |
| K2 | `PRD.md` §17 | autentikasi "JWT / Sanctum" di gateway | JWT RS256; tidak ada Sanctum | `services/auth-service/internal/infrastructure/jwt/signer.go` |
| K3 | `PRD.md` §11.1 | `AuthService: Login, Validate` | 7 RPC (`Logout`, `Refresh`, `LogoutAll`, `ListSessions`, `ValidateStore` ditambahkan) | `proto/pos/auth/v1/auth.proto` |
| K4 | `PRD.md` §11.2 | `POST /api/ai/forecast-demand` ada | rute tidak ada; kode menambah rute pelanggan dan `PUT /api/products/{id}` | `services/api-gateway/cmd/server/main.go` |
| K5 | `PRD.md` §13 | "Customer points updated only after successful sale" | tidak ditegakkan: `AddLoyaltyPoints` tidak memiliki pemanggil | `services/customer-service/internal/interfaces/grpc/customer_service.go` |
| K6 | `PRD.md` §12.1 | tabel `notifications`, `ai_forecasts`, `permissions` wajib | tidak ada pada migrasi | `services/auth-service/migrations/000001_create_auth_schema.up.sql` |
| K7 | `PRD.md` §7.1 | pencarian produk berdasarkan nama, SKU, barcode | hanya nama (`ILIKE`) dan barcode persis | `services/product-service/internal/infrastructure/postgres/product_repository.go` |
| K8 | `PRD.md` §14.3 | admin menerima notifikasi stok rendah | tidak ada konsumen peristiwa | `services/inventory-service/internal/infrastructure/nats_publisher.go` |
| K9 | `README.md` dan `CLAUDE.md` | empat lapis isolasi tenant, termasuk RLS cakupan toko | tidak ada kebijakan RLS yang memakai `app.store_id`; cakupan toko hanya lapisan aplikasi | `services/_shared/database/tenant_context.go` |
| K10 | `docs/security/grpc-transport-security.md` | pod tanpa rahasia tidak dapat memanggil layanan backend mana pun | interseptor menerima permintaan tanpa header token, tanpa metadata, atau dengan rahasia kosong/`changeme` (gagal-terbuka) | `services/_shared/middleware/grpc_interceptors.go` |
| K11 | komentar interseptor | rantai "recover -> otel -> metrics -> logging -> service-auth -> tenant-enforce" | hanya service-auth dan logging; tidak ada penegakan tenant pada gRPC | `services/_shared/middleware/grpc_interceptors.go` |
| K12 | `README.md` | "structured logging via OpenTelemetry" | logging memakai zap; kode OpenTelemetry/Prometheus tidak ada (hanya `go.sum`/default viper); `METRICS_PORT` tidak dibaca kode | `services/_shared/observability` |
| K13 | `CLAUDE.md` | `make dev` menjalankan Postgres 16 | `deploy/docker/docker-compose.yml` berisi redis, nats, jaeger, prometheus tanpa Postgres | `deploy/docker/docker-compose.yml` |
| K14 | `CLAUDE.md` dan `README.md` | tenant "di-seed lewat console command" | tidak ada kode atau seed yang membuat tenant, toko, atau pengguna | `services/auth-service/internal/infrastructure/postgres/tenant_repository.go` |
| K15 | `CLAUDE.md` | bagian aplikasi SvelteKit menyebut "Laravel gateway" di `PUBLIC_GATEWAY_URL`, diagramnya sendiri Go chi | Go chi | `CLAUDE.md` |
| K16 | `README.md` / `CLAUDE.md` | Go 1.22+ / pnpm | `go.work` 1.25.0, `Dockerfile.base` 1.25; ada `package-lock.json` (npm) | `go.work` |
| K17 | `.github/workflows/ci.yml` | job "Laravel gateway tests" (php 8.3, composer, Pest, `apps/api-gateway`) | direktori `apps/api-gateway` tidak ada; gateway adalah Go | `.github/workflows/ci.yml` |
| K18 | `.github/workflows/ci.yml` | Go `'1.24'`; pemicu `main`/`develop`; `npm --filter @pos-stery/admin check` | Go 1.25 (`Dockerfile.base`, `go.work`); cabang aktif `development`; `--filter` bergaya pnpm di bawah npm; tanpa `package.json` akar; tanpa e2e | `.github/workflows/ci.yml` |
| K19 | `scripts/preflight.sh` | mensyaratkan PHP/Composer/pnpm dan Go 1.24+ | tumpukan tidak lagi memakai PHP/pnpm | `scripts/preflight.sh` |
| K20 | komentar `AddLoyaltyPoints` | "called by sales-service after Sale.Completed" | tidak ada pemanggil | `services/customer-service/internal/interfaces/grpc/customer_service.go` |
| K21 | `PRD.md` §18 vs rencana suite | rencana menyebut peramalan AI sebagai Phase 3 | `PRD.md` §18 menempatkan peramalan dan rekomendasi pada Phase 2; harga dinamis, citra, chat pada Phase 3 | `PRD.md` |
| K22 | UI dashboard admin | menampilkan bauran pembayaran 55/30/15 dan sparkline sebagai data | angka disintesis dari `total_revenue`; backend tidak menyimpan metode pembayaran | `apps/admin/src/routes/(app)/dashboard/+page.svelte` |
| K23 | UI kasir | `payment_method` Tunai/QRIS/Debit dipilih | gateway membuang `payment_method`; tidak ada kolom | `services/api-gateway/internal/handlers/sales.go` |

**B. Koreksi atas klaim baseline yang dibaca ulang dari kode** (klaim dari pemindai [S] yang tidak sepenuhnya benar).

| # | Klaim baseline [S] | Kode menunjukkan | Bukti |
|---|---|---|---|
| C1 | `scripts/migrate-validate.sh` tidak ada | berkas ada; hanya memeriksa pasangan up/down untuk enam layanan | `scripts/migrate-validate.sh` |
| C2 | `keys/`, `.env`, `server.exe` dilacak git | tidak dilacak (diabaikan `.gitignore`), tetapi ada di disk kerja; yang dilacak hanya `ecosystem.config.cjs` dan `deploy/docker/init-postgres.sql` | `.gitignore` |
| C3 | empat kebijakan RLS tanpa `WITH CHECK` | tujuh: `customer.loyalty_points`, `inventory.stock_thresholds`, `product.categories`, `sales.sale_items`, `sales.receipts`, `supplier.purchase_orders`, `supplier.purchase_order_items` | `services/supplier-service/migrations/000001_create_supplier_schema.up.sql` |
| C4 | OpenTelemetry/Prometheus ada di POS | hanya dependensi tidak langsung dan default konfigurasi; tidak ada kode | `services/_shared/observability` |
| C5 | interseptor token layanan melindungi gRPC | gagal-terbuka (lihat K10) | `services/_shared/middleware/grpc_interceptors.go` |
| C6 | sqlc dipakai di POS | hanya `auth-service`; layanan lain pgx mentah (direktori `sqlc` di sales-service kosong) | `services/auth-service/sqlc.yaml` |
| C7 | tes integrasi Postgres "tidak pernah dijalankan" | tidak ada sama sekali (target Makefile `test-integration` hanya mengatur variabel env) | `Makefile` |
| C8 | `extractIPAddr` mengembalikan `0.0.0.0` (auth) | klaim milik HRIS, tidak berlaku untuk POS | `services/auth-service/internal/application/commands/login.go` |

**C. Belum diverifikasi runtime** (Pending Verification, [INFERENCE] dari pembacaan statis): `SET LOCAL app.tenant_id = $1` dengan parameter terikat dan dua perintah (FR-POS-SEC.4); admin tanpa selektor toko menerima 403; struk kasir menampilkan nilai kosong; JWT kedaluwarsa tanpa refresh; balapan stok dan loyalitas.

---

## 12. Catatan Penutup

- **Status dokumen.** Draft pertama; terbit bersama `BRD_POS.id.md`. Pembaruan: setiap perubahan kode POS wajib memperbarui bagian 4, 5, 6, dan 11 (kebiasaan FSD SELA, STACK-SPEC 18); versi dinaikkan dan kolom "Perubahan dari vX" diisi.
- **Keputusan terbuka yang memblokir** (didefinisikan di BRD 9.3): D-POS-01 (PPN) dan D-POS-02 (struk) menutup gerbang G8; D-POS-03 (offline) menentukan alur 7.3; D-POS-04 (metode pembayaran) menentukan skema penjualan dan laporan; D-POS-05 (loyalitas) menentukan FR-POS-06.4; D-POS-08 (UU PDP) menentukan FR-POS-SEC.8; D-POS-09 (paket) menentukan batas `pos.stores.max` dan `pos.devices.max`; D-POS-10 (observabilitas); D-POS-11 (cookie/TTL sesi); D-POS-12 (pemasangan tenant); D-POS-13 (migrasi data).
- **Hal yang tidak dijalankan.** Tidak ada build, lint, atau tes; angka (28 RPC, 14 rute, 79 fungsi tes, 15 kebijakan RLS, 7 tanpa `WITH CHECK`) berasal dari pembacaan dan penghitungan statis.
- **Nilai bisnis.** Harga, SLA, kuota, target KPI, data kompetitor, dan angka kinerja seluruhnya `tbd`; tidak ada angka `Rp` dalam dokumen ini.
- **Terjemahan.** Versi Inggris `FSD_POS.en.md` dibuat oleh lajur terjemahan dengan struktur identik; versi Indonesia ini adalah sumber kebenaran.
