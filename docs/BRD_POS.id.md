# BRD POS-Stery

*Business Requirements Document: POS SaaS multi-tenant untuk toko ritel (produk `POS` dalam portofolio Stery Suite)*

| | |
|---|---|
| **Nama Dokumen** | BRD POS-Stery (Business Requirements Document) |
| **Versi** | 1.0 (Draft) |
| **Tanggal** | 6 November 2026 |
| **Disusun oleh** | Tim Produk & Engineering |
| **Status** | Draft untuk ditinjau |
| **Dokumen Terkait** | `PRD.md` (PRD POS-Stery, bahasa Inggris, sumber persyaratan produk); `FSD_POS.id.md` (spesifikasi fungsional, memuat matriks BR -> FR di 11.2); `00_Analisis_Dampak_Risiko` (analisis dampak dan risiko Stery Suite); `SELA/CLAUDE.md` dan `SELA/docs/FSD_Sela.md` (standar stack Stery Suite) |
| **Perubahan dari vX** | Perubahan dari vX: Dokumen pertama |

> **Catatan metode.** Dokumen ini adalah BRD pertama untuk POS-Stery. Persyaratan produk berasal dari `PRD.md` (402 baris, bahasa Inggris) yang tidak diubah; BRD ini hanya merujuk bagian-bagiannya (mis. PRD §4) dan mencatat di mana kode menyimpang dari PRD. Sumber kebenaran status adalah kode: setiap pernyataan tentang kondisi saat ini membawa status {Selesai, Sebagian, Belum, Pending Verification, Ditunda} dan jalur berkas pada kolom `Bukti` (jalur relatif terhadap akar repo `POS-Stery`). Nilai bisnis yang tidak ditemukan di repo (harga, SLA, kuota, target KPI, data kompetitor) ditulis literal `tbd` dan dicatat sebagai keputusan terbuka `D-POS-NN` (bagian 9). Tidak ada angka harga dalam dokumen ini. Tidak ada build, lint, atau tes yang dijalankan untuk menyusunnya; semua temuan berasal dari pembacaan statis kode.

---

## 1. Ringkasan Eksekutif

POS-Stery adalah platform Point of Sale (POS) SaaS multi-tenant untuk toko ritel: satu perusahaan (tenant) dapat memiliki beberapa toko, kasir menjual dari antarmuka kasir, admin mengelola produk, stok, pemasok, pelanggan, dan laporan (`PRD.md` §1-§3). Dalam portofolio Stery Suite, POS adalah SKU `POS` yang dijual mandiri, bersama salah satu produk lain, atau sebagai bagian dari paket penuh `SUITE` (lihat BRD Stery Suite).

**Kondisi saat ini (kode = kebenaran).** Backend Go terdiri dari enam layanan gRPC (auth, product, inventory, sales, supplier, customer) yang seluruh 28 RPC-nya memiliki handler nyata, ditambah gateway REST Go `chi` dengan 14 rute bisnis (Sebagian: 14 dari 28 operasi terekspos lewat REST). Ada dua aplikasi SvelteKit (admin dan kasir), keduanya SSR dengan BFF proxy. Fondasi multi-tenant (kolom `tenant_id`, RLS `ENABLE` + `FORCE` pada seluruh 15 kebijakan) sudah ada. Namun aturan bisnis inti PRD §13 belum terpenuhi sepenuhnya:

- poin loyalitas tidak pernah ditambahkan setelah penjualan (`AddLoyaltyPoints` tidak memiliki pemanggil);
- pengurangan stok dan penyimpanan penjualan bukan satu transaksi (saga gRPC dengan kompensasi best-effort) dan tidak ada kunci idempotensi;
- aturan "stok tidak boleh negatif" rentan balapan (race) karena pembaruan stok tidak memiliki penjaga di SQL;
- dashboard admin memalsukan bauran pembayaran 55/30/15 dan nama produk teratas selalu kosong;
- tidak ada PPN pada struk, tidak ada metode pembayaran tersimpan, tidak ada antrean kasir offline;
- tidak ada kode untuk membuat tenant, toko, atau pengguna pertama;
- CI usang (job Laravel, Go 1.24 versus 1.25), tidak ada tes integrasi Postgres, dan rahasia tertulis di dua berkas yang dilacak git (`ecosystem.config.cjs`, `deploy/docker/init-postgres.sql`; hanya jalur yang dicatat, bukan nilainya).

**Arah.** Keputusan pemilik (analisis `00_Analisis_Dampak_Risiko`): Stery Suite disusun dengan prinsip komposisi, bukan konsolidasi; stack, pola, dan arsitektur SELA menjadi standar seluruh suite. POS akan diselaraskan ke stack itu (modul Go dalam satu proses, REST `/api/v1` stdlib, PostgreSQL 18 dengan goose, sesi opaque, outbox Postgres, Dragonfly, RustFS, satu `apps/web` SvelteKit SPA) pada workstream S2 dan berakhir pada **gerbang stabilisasi POS** (bagian 12). Mode operasi: `STERY_MODE=standalone` (perilaku mandiri) atau `suite` (identitas, entitlement, penagihan, notifikasi, audit didelegasikan ke platform).

**Hasil bisnis yang dituju:** kasir menyelesaikan penjualan secara atomik dan dapat dipercaya; stok akurat per toko; laporan memakai data nyata; data pelanggan dilindungi; paket (jumlah toko dan perangkat) dapat dijual dan dibatasi lewat dimensi `pos.stores.max` dan `pos.devices.max` (nilai `tbd`).

---

## 2. Latar Belakang dan Pernyataan Masalah

### 2.1 Latar belakang

- Pemilik memutuskan menjual HRIS-Stery, POS-Stery, dan SELA sebagai paket (ketiganya, dua produk apa saja, atau satu produk), masing-masing dapat berjalan mandiri maupun terintegrasi.
- POS-Stery lahir sebagai monorepo microservice: gateway, enam layanan Go, dua aplikasi web, NATS JetStream, Redis, PostgreSQL 16 dengan RLS (`PRD.md` §10, `CLAUDE.md`). Repo memiliki 21 commit; commit terakhir 30 Juni 2026 pada cabang `development`.
- Stack SELA (Go modul tunggal, REST + gRPC tanpa gateway, Postgres 18, goose, Dragonfly, RustFS, SPA SvelteKit, `scripts/check.sh`) telah ditetapkan sebagai standar suite; POS harus diselaraskan, bukan ditulis ulang dari nol: lapisan domain dan aplikasi dipertahankan, infrastruktur dan antarmuka ditulis ulang (FSD 2.7).

### 2.2 Pernyataan masalah

| # | Masalah | Dampak bisnis | Status | Bukti |
|---|---|---|---|---|
| P1 | Poin loyalitas tidak pernah bertambah setelah penjualan | Program loyalitas (PRD §7.5, §13) tidak berfungsi; janji ke pelanggan tidak terpenuhi | Belum | `services/customer-service/internal/interfaces/grpc/customer_service.go` |
| P2 | Penjualan tidak atomik dan tidak idempoten: stok dikurangi lewat gRPC lalu penjualan ditulis, kompensasi hanya best-effort | Stok dan penjualan dapat tidak sinkron; klik ganda atau retry membuat penjualan ganda atau stok ganda terpotong | Sebagian | `services/sales-service/internal/application/commands/create_sale.go` |
| P3 | Pembaruan stok tanpa penjaga di SQL; tidak ada buku besar pergerakan stok | Stok negatif mungkin terjadi saat konkurensi; tidak ada jejak alasan perubahan | Sebagian | `services/inventory-service/internal/infrastructure/postgres/stock_repository.go` |
| P4 | Struk tidak memuat PPN, metode pembayaran, maupun pelanggan | Struk belum dapat dipakai sebagai bukti transaksi pajak; metode pembayaran yang dipilih kasir hilang | Belum | `services/sales-service/internal/domain/sale.go` |
| P5 | Dashboard admin memalsukan data (bauran pembayaran 55/30/15, batang sparkline sintetis, float kas 500.000 sebagai default) dan nama produk teratas kosong | Pemilik mengambil keputusan dari angka palsu | Sebagian | `apps/admin/src/routes/(app)/dashboard/+page.svelte` |
| P6 | Tidak ada jalur membuat tenant, toko, atau pengguna; tidak ada pengaturan ulang kata sandi | Pelanggan baru tidak dapat dipasang tanpa SQL manual; tidak dapat dijual sebagai SaaS | Belum | `services/auth-service/internal/infrastructure/postgres/tenant_repository.go` |
| P7 | Kasir tidak punya mode offline; service worker hanya menyimpan aset build | Toko berhenti berjualan saat internet putus (PRD §9.3 menyebut PWA/offline "sangat diutamakan") | Belum | `apps/cashier/src/service-worker.ts` |
| P8 | Rahasia tertulis di berkas yang dilacak git; CI usang; tidak ada tes integrasi Postgres | Risiko keamanan dan regresi tak terdeteksi | Sebagian | `ecosystem.config.cjs` |
| P9 | Permukaan REST hanya 14 dari 28 operasi: tidak ada `GetSale`, `ReceiveStock`, daftar PO, loyalitas, ambang stok, `GetItem`, `UpdateStock`, refresh token | Antarmuka web tidak dapat mengakses fitur yang sudah ada di backend | Sebagian | `services/api-gateway/cmd/server/main.go` |
| P10 | Selektor toko untuk admin tidak ada di UI, padahal gateway mewajibkan konteks toko untuk setiap rute terotentikasi | Menurut pembacaan statis, halaman admin tidak dapat memuat data [INFERENCE] | Belum | `services/api-gateway/internal/middleware/store.go` |
| P11 | Tidak ada pernyataan UU PDP untuk data pelanggan (nama, telepon, email) | Risiko kepatuhan belum dinilai | Pending Verification | `services/customer-service/migrations/000001_create_customer_schema.up.sql` |

---

## 3. Tujuan Bisnis

| ID tujuan | Tujuan | Ukuran keberhasilan (target `tbd`, lihat bagian 11) |
|---|---|---|
| T1 | Penjualan kasir cepat, atomik, dan dapat dipercaya | Waktu selesai penjualan; tingkat keberhasilan penjualan; nol penjualan ganda |
| T2 | Stok akurat per toko | Selisih stok sistem vs fisik; nol stok negatif |
| T3 | Laporan memakai data nyata | Nol angka sintetis di dashboard; nama produk lengkap |
| T4 | Siap dijual sebagai SaaS (tenant, toko, pengguna, paket, batas) | Waktu memasang tenant baru; dimensi `pos.stores.max` dan `pos.devices.max` ditegakkan |
| T5 | Dapat dijual mandiri maupun dalam suite | Mode `standalone` dan `suite` lolos uji yang sama |
| T6 | Aman dan patuh | Seluruh kebijakan RLS memiliki `WITH CHECK`; rahasia dirotasi dan dibersihkan; keputusan UU PDP tercatat |
| T7 | Dapat dipelihara oleh tim kecil | CI hijau dengan pola `check.sh`; satu stack, satu konvensi |

---

## 4. Ruang Lingkup

### 4.1 Segmen dan paket

- **Segmen sasaran** (`PRD.md` §1, §6): toko ritel dan rantai toko (satu tenant, beberapa toko). Ukuran pasar dan segmentasi lebih rinci tidak ada di repo: `tbd` (pertanyaan terbuka RQ-1 pada bagian 8).
- **SKU produk:** `POS`. **Bundel suite:** `HRS+POS`, `POS+SEL`, `SUITE` (aturan diskon bundel, siklus penagihan, masa uji: `tbd`; didefinisikan di BRD Stery Suite).
- **Dimensi batas POS** (nama tetap, nilai `tbd`): `pos.stores.max` (jumlah toko per tenant), `pos.devices.max` (jumlah perangkat kasir per tenant). Tingkat paket (nama dan nilai): `tbd` (D-POS-09).
- **Mode:** `standalone` (default; pengguna, sesi, dan paket milik POS sendiri) dan `suite` (didelegasikan ke platform). Tenant POS dalam mode `suite` terhubung ke organisasi platform lewat UUID dan kolom `external_ref`.

### 4.2 Dalam lingkup

Berasal dari `PRD.md` §3, §5, §7 dan §18 Phase 1, dengan status kode pada bagian 13:

1. Hierarki tenant dan toko, isolasi data per `tenant_id` dan `store_id` (PRD §6).
2. Autentikasi dan RBAC peran `admin`, `cashier`, `stock_manager` (PRD §5, §17).
3. Katalog produk (buat, ubah, cari) dan harga/diskon dasar (PRD §7.1, §7.6).
4. Inventaris per toko, penyesuaian, ambang stok, peringatan stok rendah (PRD §7.2).
5. Penjualan kasir, diskon terkendali, struk digital dan snapshot (PRD §7.3, §7.6, §7.7).
6. Pemasok, pesanan pembelian (PO), penerimaan stok (PRD §7.4).
7. Pelanggan dan poin loyalitas (PRD §7.5).
8. Laporan penjualan harian per toko dan per tenant (PRD §7.8).
9. Penyelarasan ke stack Stery Suite, mode `standalone`/`suite`, pembatasan paket (bagian 6 dan 12).
10. Pemasangan tenant, toko, dan pengguna (belum ada; BR-POS-21).
11. Antrean kasir offline dan pencetakan struk (keputusan terbuka D-POS-02, D-POS-03).

### 4.3 Ditunda

| Item | Alasan penundaan | Rujukan |
|---|---|---|
| Layanan notifikasi dan peringatan stok rendah ke admin (modul NTF) | Belum ada layanan; peristiwa `Product.LowStock` dipublikasikan tetapi tidak ada konsumen. Ditunda ke PRD Phase 2 dan ke hub notifikasi platform (mode `suite`) | `PRD.md` §10.2, §14.3, §18 Phase 2 |
| Peramalan permintaan AI (modul AI) | Tidak ada layanan AI di repo; PRD menempatkan peramalan dan rekomendasi pada Phase 2 (§18), harga dinamis, pengenalan citra, dan chat pada Phase 3 | `PRD.md` §8, §18 |
| Analitik lanjutan, harga dinamis, pengenalan gambar produk, chat | PRD Phase 3 | `PRD.md` §18 Phase 3 |
| Integrasi POS dengan SELA (INT-3: acara toko membuat acara SELA dan mengundang pelanggan yang opt-in) | Hipotesis suite fase S5; validasi permintaan dahulu | Rencana Stery Suite |
| Integrasi staf HRIS <-> pengguna POS (INT-1) | Hipotesis suite fase S5 | Rencana Stery Suite |
| Integrasi pemindai barcode perangkat keras | PRD Phase 2 | `PRD.md` §18 Phase 2 |

### 4.4 Di luar lingkup

Mengulang `PRD.md` §4 (Non-Goals):

- Tidak ada marketplace atau commerce multi-vendor.
- Tidak ada pengganti akuntansi/ERP pada fase 1.
- Tidak ada mesin promosi kompleks di luar diskon dasar pada rilis pertama.
- Tidak ada sistem manajemen gudang (WMS) lanjutan pada fase 1.
- Tidak ada implementasi payment gateway langsung kecuali ditambahkan secara eksplisit nanti. (Pencatatan metode pembayaran pada penjualan bukan payment gateway; lihat D-POS-04.)

---

## 5. Pemangku Kepentingan

| Peran | Deskripsi Kepentingan | Tingkat Pengaruh |
|---|---|---|
| Pemilik produk / sponsor bisnis | Keputusan paket, harga, urutan penyelarasan (S2), keputusan PPN dan struk | Tinggi |
| Admin tenant (`admin`) | Mengelola toko, produk, stok, pemasok, harga, laporan; menerima peringatan stok rendah | Tinggi |
| Kasir (`cashier`) | Mencari produk, menjual, menerapkan diskon yang diizinkan, mencetak/melihat struk; butuh kecepatan dan ketahanan offline | Tinggi |
| Manajer stok (`stock_manager`) | Memantau stok, membuat PO, mengonfirmasi penerimaan | Sedang |
| Pelanggan toko | Dilacak untuk poin loyalitas; data pribadinya dilindungi | Sedang |
| Pemasok | Penerima PO (belum ada kanal digital) | Rendah |
| Tim engineering (solo/kecil) | Menyelaraskan stack, menjaga kualitas; kapasitas WIP terbatas | Tinggi |
| Operator platform Stery Suite | Entitlement, penagihan, SSO dalam mode `suite` | Tinggi (mode `suite`) |
| Penasihat hukum / kepatuhan | Keputusan UU PDP untuk data pelanggan; keputusan PPN dan struk | Sedang (`tbd`: belum ada penunjukan) |

---

## 6. Kebutuhan Bisnis

Prioritas MoSCoW: M = Must, S = Should, C = Could. Kolom "Rujukan PRD" menunjuk bagian `PRD.md`; status implementasi per BR ada di FSD 11.2 dan 11.1.

| **ID** | **Kebutuhan Bisnis** | **Prioritas** | **Rujukan PRD** |
|---|---|---|---|
| BR-POS-01 | Kasir dapat menyelesaikan penjualan (cari/pindai produk, keranjang, diskon yang diizinkan, penyelesaian) dan sistem menyimpan riwayat serta struk. Pengurangan stok dan penyimpanan penjualan harus atomik dan tidak dapat terduplikasi oleh retry | M | §7.3, §13, §14.2 |
| BR-POS-02 | Setiap data bisnis terisolasi per tenant (wajib) dan per toko (untuk data toko); tidak ada kebocoran lintas tenant pada API, layanan, repositori, maupun peristiwa | M | §6.2, §13, §17 |
| BR-POS-03 | Akses berbasis peran: `admin` (tenant), `cashier`, `stock_manager` dengan batas wewenang yang ditegakkan di backend; kasir terikat pada toko yang ditugaskan | M | §5, §17 |
| BR-POS-04 | Admin dapat membuat dan mengubah produk, kategori, dan harga; perubahan harga dapat diaudit | M | §7.1, §7.6 |
| BR-POS-05 | Stok per toko selalu akurat: tidak boleh negatif, setiap perubahan tercatat alasan dan rujukannya, ambang stok dapat diatur dan peristiwa stok rendah dipublikasikan saat ambang terlewati | M | §7.2, §13, §14.1, §14.3 |
| BR-POS-06 | Manajer stok dapat mengelola pemasok, membuat PO, dan menerima barang sehingga stok bertambah tepat satu kali per penerimaan | M | §7.4, §14.4 |
| BR-POS-07 | Pelanggan dapat dicatat dan poin loyalitas bertambah hanya setelah penjualan berhasil, tepat sekali per penjualan; aturan perolehan dan penukaran poin ditetapkan (D-POS-05) | S | §7.5, §13, §14.5 |
| BR-POS-08 | Laporan penjualan harian per toko dan per tenant memakai data nyata (pendapatan, jumlah transaksi, produk teratas dengan nama, bauran metode pembayaran) dengan batas hari mengikuti zona waktu toko (WIB) | M | §7.8, §14.6 |
| BR-POS-09 | Perlakuan PPN/pajak ditetapkan (D-POS-01) dan struk serta laporan mencerminkannya; keputusan dicatat sebelum gerbang stabilisasi ditutup | M | §7.7 (tidak menyebut pajak: celah PRD) |
| BR-POS-10 | Struk digital dapat ditampilkan, dicetak, dan diambil ulang dari riwayat; media cetak ditetapkan (D-POS-02) | S | §7.7, §9.2 |
| BR-POS-11 | Kasir tetap dapat berjualan saat internet terputus (antrean offline, sinkron otomatis, tanpa penjualan ganda); ruang lingkup offline ditetapkan (D-POS-03) | S | §9.3, §18 Phase 2 |
| BR-POS-12 | Metode pembayaran pada penjualan (tunai, QRIS, debit sebagai pencatatan; tanpa payment gateway) ditetapkan dan tersimpan pada penjualan dan laporan (D-POS-04) | S | §4 (non-goal gateway), §7.3 |
| BR-POS-13 | Kebersihan keamanan: tidak ada rahasia di repo yang dilacak (rotasi dan pembersihan riwayat), transportasi internal diamankan, jejak audit tersedia untuk aksi sensitif | M | §17 |
| BR-POS-14 | Gerbang kualitas otomatis: CI hijau mengikuti pola `check.sh` SELA, tes integrasi Postgres (termasuk tes lintas-tenant dan RLS) berjalan, cakupan tes terukur | M | §15, §19 |
| BR-POS-15 | POS diselaraskan ke stack standar Stery Suite (STACK-SPEC) dengan perilaku fungsional dipertahankan, berakhir pada gerbang stabilisasi POS | M | Keputusan suite (D1-D12) |
| BR-POS-16 | POS dapat berjalan `standalone` atau `suite` per pelanggan (`STERY_MODE`), dengan batas paket `pos.stores.max` dan `pos.devices.max` ditegakkan lewat entitlement dan degradasi aman saat entitlement berakhir | M | Keputusan suite (D2, D4) |
| BR-POS-17 | Admin menerima notifikasi stok rendah dan peristiwa penjualan penting (modul NTF) | S | §10.2, §14.3, §18 Phase 2 |
| BR-POS-18 | Peramalan permintaan: saran jumlah pesan ulang berdasarkan riwayat penjualan per produk per toko | C | §8.1, §14.7, §18 Phase 2 |
| BR-POS-19 | Data pribadi pelanggan (nama, telepon, email) diperlakukan sesuai UU PDP: keputusan penerapan, dasar pemrosesan, retensi, dan hak subjek data ditetapkan dan tercatat (D-POS-08) | S | §17 (tidak menyebut UU PDP: celah PRD) |
| BR-POS-20 | Integrasi hipotesis dengan SELA (INT-3): acara toko dapat membuat acara SELA untuk organisasi dan mengundang pelanggan yang memberi persetujuan; hanya dikerjakan bila permintaan tervalidasi | C | Rencana Stery Suite (S5) |
| BR-POS-21 | Pemasangan pelanggan baru: tenant, toko, pengguna pertama, dan pengaturan ulang kata sandi dapat dilakukan tanpa SQL manual (mode `standalone`: konsol operator atau swa-layan, D-POS-12; mode `suite`: oleh platform) | M | §5.1 ("Creates stores"), §6.1 |

---

## 7. Model Bisnis dan Monetisasi

Seluruh nilai komersial POS adalah `tbd`; tidak ada harga, kuota, atau SLA yang ditemukan di repo `POS-Stery` (konfirmasi: tidak ada berkas harga, penagihan, atau langganan). Dokumen ini hanya menetapkan struktur.

| Aspek | Ketetapan | Nilai |
|---|---|---|
| Unit penjualan | SKU `POS`; bundel `HRS+POS`, `POS+SEL`, `SUITE` | Aturan diskon bundel: `tbd` |
| Dimensi batas | `pos.stores.max` (toko per tenant); `pos.devices.max` (perangkat kasir per tenant) | Nilai per tingkat: `tbd` |
| Tingkat paket | Nama dan isi tingkat | `tbd` (D-POS-09) |
| Harga langganan | Per tingkat | `tbd` |
| Siklus penagihan, masa uji, denda keterlambatan | Ditentukan oleh platform (penagihan suite memakai abstraksi `payment.Provider` SELA, Midtrans QRIS) | `tbd` |
| SLA ketersediaan, dukungan | Per tingkat | `tbd` |
| Perilaku saat entitlement berakhir | Snapshot entitlement terakhir dipakai hingga masa tenggang (default 24 jam, dapat dikonfigurasi, `tbd`), lalu fitur berbayar gagal-tertutup; penjualan di kasir pada mode `standalone` tidak pernah diblokir | Masa tenggang: `tbd` |
| Pembayaran pelanggan toko (QRIS/debit di kasir) | Bukan payment gateway; hanya pencatatan (D-POS-04) | — |

Pertanyaan terbuka: harga per tingkat, apakah perangkat dibatasi per toko atau per tenant, apakah ada tingkat gratis (D-POS-09).

---

## 8. Analisis Alternatif dan Kompetitor

### 8.1 Alternatif arsitektur (ringkas; analisis penuh di `00_Analisis_Dampak_Risiko` bagian 5)

| Opsi | Deskripsi | Keputusan untuk POS |
|---|---|---|
| A. Polyrepo + repo platform | POS tetap repo dan deployable terpisah, database terpisah; platform `stery-suite` memegang identitas, entitlement, penagihan, notifikasi, audit | **Dipilih** |
| B. Monorepo | Semua produk dalam satu repo | Tidak dipilih |
| C. Satu basis kode/runtime | POS, HRIS, SELA digabung | Ditolak: melanggar penjualan mandiri dan memperbesar blast radius |

### 8.2 Alternatif internal untuk POS

| Opsi | Keterangan | Keputusan |
|---|---|---|
| Pertahankan microservice gRPC + gateway chi + NATS | Sesuai kode saat ini | Tidak dipilih: bertentangan dengan keputusan stack suite; saga lintas-layanan menyulitkan atomisitas penjualan |
| Selaraskan ke stack SELA (modul dalam satu proses, outbox Postgres) | Penjualan menjadi satu transaksi database | **Dipilih** (BR-POS-15) |

### 8.3 Kompetitor

Tidak ada data kompetitor di repo mana pun. Seluruh isi di bawah adalah `tbd`; jangan diisi tanpa riset.

| Kompetitor | Segmen | Harga | Fitur utama | Catatan |
|---|---|---|---|---|
| `tbd` | `tbd` | `tbd` | `tbd` | `tbd` |

Pertanyaan riset terbuka:

- RQ-1: Siapa segmen sasaran utama (toko tunggal, minimarket, rantai kecil) dan berapa ukurannya?
- RQ-2: Produk POS apa yang dipakai calon pelanggan hari ini dan berapa mereka membayar?
- RQ-3: Apakah pelanggan mewajibkan pencetakan struk termal dan integrasi pemindai barcode sejak rilis pertama?
- RQ-4: Seberapa sering koneksi internet toko target terputus; apakah mode offline menjadi syarat beli?
- RQ-5: Metode pembayaran apa yang wajib dicatat (tunai, QRIS, debit, dompet digital)?
- RQ-6: Apakah ada kebutuhan PPN/e-faktur yang wajib dipenuhi struk?

---

## 9. Asumsi dan Batasan

### 9.1 Asumsi

| # | Asumsi | Status |
|---|---|---|
| A1 | Dokumen ini hanya dokumen; tidak ada kode yang diubah. Migrasi stack adalah workstream S2 yang dispesifikasikan di FSD 2.7 | Ditetapkan |
| A2 | Aturan bisnis PRD §13 berlaku tanpa perubahan | Ditetapkan |
| A3 | Belum ada pelanggan produksi POS yang datanya harus dimigrasikan | `tbd` (tidak ada bukti ada atau tidak adanya penerapan produksi di repo; D-POS-13) |
| A4 | Satu tenant dapat memiliki banyak toko; satu kasir terikat pada satu toko | Berdasarkan kode login kasir |
| A5 | Mata uang IDR, zona waktu WIB, bahasa Indonesia default dan Inggris didukung (STACK-SPEC 19) | Ditetapkan oleh standar suite |

### 9.2 Batasan

- Tim kecil: hanya satu penyelarasan produk berjalan dalam satu waktu (D12); POS (S2) berjalan setelah gerbang SELA ditutup dan setelah ekstraksi `kit` (S1).
- Standar teknis tidak dapat dinegosiasikan per produk: STACK-SPEC (FSD 2.1).
- Tidak ada payment gateway (PRD §4).
- Dokumen tidak boleh memuat harga atau angka bisnis yang tidak ada di repo.

### 9.3 Keputusan terbuka

| ID | Keputusan | Opsi / pertanyaan | Default sementara |
|---|---|---|---|
| D-POS-01 | PPN/pajak pada struk | Tarif, inklusif atau eksklusif, per produk atau per toko, kewajiban e-faktur | `tbd` |
| D-POS-02 | Media pencetakan struk | Printer termal (ESC/POS), cetak peramban, PDF, kirim WhatsApp/email | `tbd` |
| D-POS-03 | Ruang lingkup offline kasir | Hanya antrean penjualan atau juga pencarian produk lokal; penanganan stok saat sinkron; durasi offline maksimum | `tbd` |
| D-POS-04 | Metode pembayaran | Himpunan metode, apakah QRIS hanya dicatat; tidak ada payment gateway | `tbd` |
| D-POS-05 | Aturan loyalitas | Laju perolehan per nilai belanja, penukaran, kedaluwarsa | `tbd` |
| D-POS-06 | Batal/retur penjualan | Kebijakan void dan retur, siapa berwenang | `tbd` |
| D-POS-07 | Wewenang diskon kasir | Batas persen/nominal per peran, persetujuan admin | `tbd` |
| D-POS-08 | UU PDP untuk data pelanggan | Dasar pemrosesan, retensi, hak subjek data, penunjukan penanggung jawab | `tbd` (tinjauan hukum) |
| D-POS-09 | Tingkat paket dan harga | Nilai `pos.stores.max`, `pos.devices.max`, tingkat gratis | `tbd` |
| D-POS-10 | Observabilitas | OpenTelemetry/Prometheus tidak termasuk baseline SELA (D11); modul opsional `kit` atau tidak | `tbd` |
| D-POS-11 | Domain cookie dan TTL sesi | Cookie `stery_session` (mode `suite`) dan sesi POS (mode `standalone`) | `tbd` |
| D-POS-12 | Pemasangan tenant mode `standalone` | Konsol operator, swa-layan, atau skrip | `tbd` |
| D-POS-13 | Migrasi data pelanggan POS yang sudah ada | Ada tidaknya data produksi; strategi migrasi bila ada | `tbd` |
| D-POS-14 | Batas hari laporan | Zona waktu per toko atau WIB tetap | WIB (standar STACK-SPEC 19), per toko `tbd` |

---

## 10. Risiko dan Mitigasi

| ID | Risiko | Dampak | Mitigasi |
|---|---|---|---|
| RISK-POS-01 | Penyelarasan ke stack SELA menulis ulang infrastruktur dan antarmuka POS (gateway, gRPC antar-layanan, NATS, JWT, migrasi, dua aplikasi web) | Jadwal panjang; regresi fungsional | Pertahankan domain dan aplikasi; tes karakterisasi sebelum penulisan ulang; satu penyelarasan berjalan; effort S/M/L di FSD 2.7 [INFERENCE] |
| RISK-POS-02 | Regresi RLS saat berpindah ke `kit/tenantdb` (mekanisme `SET LOCAL` saat ini belum terbukti berjalan karena tidak ada tes integrasi) | Kebocoran lintas tenant atau kegagalan semua query | Tes lintas-tenant wajib per modul; `set_config(..., true)` per transaksi; gerbang "tes integrasi Postgres berjalan" |
| RISK-POS-03 | Penjualan ganda atau stok salah karena saga tanpa idempotensi | Selisih keuangan dan stok | Penjualan satu transaksi DB dengan kunci idempotensi (FSD 7.1) |
| RISK-POS-04 | Rahasia di berkas yang dilacak git (`ecosystem.config.cjs`, `deploy/docker/init-postgres.sql`) dan interseptor token layanan gagal-terbuka | Kompromi kredensial; layanan internal terbuka | Rotasi, pembersihan riwayat git, deny-by-default (gerbang) |
| RISK-POS-05 | Konversi UI dari SSR (adapter-node, BFF) ke SPA statis dan penggabungan dua aplikasi | Regresi UI, sesi, PWA | Route group `(admin)` dan `(cashier)`; Playwright e2e |
| RISK-POS-06 | Semantik NATS JetStream (durable, 7 hari) diganti outbox Postgres | Peristiwa hilang atau ganda | Outbox `FOR UPDATE SKIP LOCKED`, konsumen idempoten, `processed_events` |
| RISK-POS-07 | Keputusan PPN/struk dan UU PDP tertunda | Gerbang tidak dapat ditutup; kepatuhan | Jadwalkan tinjauan hukum sebelum S2 selesai; D-POS-01, D-POS-08 |
| RISK-POS-08 | Kapasitas tim solo | WIP berlebih | Urutan D12; satu penyelarasan berjalan |
| RISK-POS-09 | Drift dokumen status (README, CLAUDE.md, PRD §10.1/§17, dokumen keamanan gRPC) versus kode | Keputusan berdasar klaim usang | FSD 11.3 mendaftar kontradiksi; kode = kebenaran |
| RISK-POS-10 | Harga dan tingkat paket belum ditetapkan | Tidak dapat dijual dan dibatasi | D-POS-09; dimensi tetap, nilai `tbd` |
| RISK-POS-11 | Ketergantungan pada platform sebagai titik kegagalan tunggal (mode `suite`) | Kasir tidak dapat login atau entitlement tidak dapat diperiksa | Snapshot entitlement 24 jam; mode `standalone` tidak terpengaruh; sesi kasir offline `tbd` |
| RISK-POS-12 | Kasir offline menyebabkan penjualan terlambat sinkron dengan stok atau harga basi | Stok negatif atau selisih harga | Kebijakan konflik D-POS-03; ID penjualan dari klien untuk dedup |
| RISK-POS-13 | Admin tidak dapat memakai UI karena selektor toko tidak ada dan gateway mewajibkan konteks toko pada semua rute | Fitur admin tidak dapat dipakai [INFERENCE: pembacaan statis] | Selektor toko di `apps/web (admin)` pada S2; rute tingkat tenant tidak memerlukan toko |
| RISK-POS-14 | Dashboard menampilkan angka palsu kepada pemilik | Keputusan bisnis salah | Gerbang: data sintetis dihapus |

---

## 11. KPI

Definisi bersifat tetap; seluruh target `tbd` (pemilik menetapkan; D-POS-09 untuk yang bersifat komersial). Baseline hanya dicantumkan bila dapat dihitung dari kode.

| KPI | Definisi | Baseline saat ini | Target |
|---|---|---|---|
| Waktu selesai penjualan | Median waktu dari tombol "Bayar" hingga struk tampil, per toko | Tidak diukur | `tbd` |
| Tingkat keberhasilan penjualan | Penjualan selesai / percobaan penjualan, per hari | Tidak diukur | `tbd` |
| Penjualan ganda | Jumlah penjualan dengan kunci idempotensi sama yang tersimpan lebih dari sekali | Tidak ada kunci idempotensi | 0 (nilai target `tbd` bila pemilik mengubah) |
| Akurasi stok | Selisih stok sistem vs hitung fisik, per toko per bulan | Tidak diukur | `tbd` |
| Stok negatif | Jumlah baris stok dengan kuantitas < 0 | Tidak dijaga di SQL | 0 (`tbd`) |
| Konsistensi loyalitas | Penjualan berpelanggan yang menghasilkan tepat satu entri poin / seluruh penjualan berpelanggan | 0 pemanggil | `tbd` |
| Keberhasilan sinkron offline | Penjualan antrean yang tersinkron tanpa duplikasi / seluruh penjualan antrean | Tidak ada antrean | `tbd` |
| Cakupan WITH CHECK RLS | Kebijakan dengan `WITH CHECK` / seluruh kebijakan tenant | 8 dari 15 (analisis statis) | 15 dari 15 |
| Kesehatan CI | Persentase proses CI hijau pada cabang utama | CI usang | `tbd` |
| Cakupan tes | Cakupan pernyataan Go dan web | Hanya tes unit berbasis stub (79 fungsi tes) | Lantai 80% (standar STACK-SPEC 17) |
| Waktu memasang tenant | Waktu dari kontrak hingga tenant, toko, dan pengguna pertama aktif | Tidak ada jalur | `tbd` |
| Adopsi paket | Tenant aktif per tingkat; pemakaian terhadap `pos.stores.max` dan `pos.devices.max` | Tidak ada pelacakan | `tbd` |

---

## 12. Linimasa dan Fase

Tanggal kalender: `tbd` (pemilik menetapkan; tidak ada tanggal yang dapat diturunkan dari repo). Urutan mengikuti keputusan D12 Stery Suite.

| Fase | Isi | Prasyarat | Keluaran |
|---|---|---|---|
| S0 | Dokumen (dokumen ini, FSD POS, analisis, BRD/PRD/FSD suite) dan penutupan gerbang SELA | — | Dokumen disetujui |
| S1 | Ekstraksi `kit` dari SELA dan platform IAM/ENT; POS belum disentuh | S0 | `kit` tersedia |
| S2 | **Penyelarasan POS** ke STACK-SPEC (workstream di FSD 2.7) | S1 | Gerbang stabilisasi POS (di bawah) |
| S3 | Penyelarasan HRIS (tidak menyentuh POS) | S2 | — |
| S4 | Adapter mode `suite` pada POS (identitas, entitlement, notifikasi, audit, peristiwa), penagihan dan konsol | S3 | POS dapat berjalan `suite` |
| S5 | Integrasi lintas produk (INT-3 hipotesis) dan produk baru | S4 | Hanya bila permintaan tervalidasi |

### 12.1 Gerbang stabilisasi POS (kriteria keluar S2)

Delapan butir, seluruhnya wajib. Butir yang tidak bergantung pada stack baru (rotasi rahasia, penghapusan data palsu, pencatatan keputusan) dapat dikerjakan lebih awal pada stack saat ini [INFERENCE].

| # | Butir gerbang | BR terkait | Cara verifikasi |
|---|---|---|---|
| G1 | Loyalitas tersambung dalam proses setelah commit penjualan, idempoten per penjualan | BR-POS-07 | Tes integrasi: penjualan berpelanggan menghasilkan tepat satu entri poin, bahkan saat diulang |
| G2 | Paritas permukaan REST dengan operasi modul (setiap operasi bisnis punya rute `/api/v1`) | BR-POS-01, BR-POS-10 | Tabel paritas FSD 6 tanpa baris "Belum" |
| G3 | CI hijau dengan pola `check.sh` SELA (proto, gofmt, vet, golangci-lint, tes dengan cakupan, svelte-check, vitest, e2e nyata; `-race` di CI) | BR-POS-14 | Proses CI hijau pada commit gerbang |
| G4 | RLS `WITH CHECK` pada setiap kebijakan | BR-POS-02 | Tes migrasi mendaftar kebijakan tanpa `WITH CHECK` = 0 |
| G5 | Rahasia yang dilacak git dirotasi dan dibersihkan dari riwayat | BR-POS-13 | Pemindaian rahasia pada riwayat bersih; kredensial lama tidak berlaku |
| G6 | Tes integrasi Postgres berjalan (termasuk tes lintas-tenant) | BR-POS-14, BR-POS-02 | Tes menyala di CI tanpa skip |
| G7 | Data dashboard palsu dihapus | BR-POS-08 | Tidak ada konstanta sintetis pada halaman dashboard; nama produk lengkap |
| G8 | Keputusan PPN dan struk dicatat | BR-POS-09, BR-POS-10 | D-POS-01 dan D-POS-02 berstatus diputuskan di catatan keputusan |

---

## 13. Status Implementasi Saat Ini

Ringkasan per modul katalog POS (rincian per FR di FSD 11.1). Status: Selesai, Sebagian, Belum, Pending Verification, Ditunda. Persentase kesiapan tidak dinyatakan kecuali ditandai `[INFERENCE]`.

| Modul | Cakupan | Status | Bukti |
|---|---|---|---|
| 01 AUTH | Login kata sandi bcrypt dengan penentuan tenant, JWT RS256 10 menit, refresh token rotasi dengan deteksi pemakaian ulang, logout, RBAC tiga peran di gateway. Belum: pembuatan tenant/toko/pengguna, selektor toko admin, refresh di REST dan web | Sebagian | `services/auth-service/internal/application/commands/login.go` |
| 02 PRD (katalog) | Buat, ubah, ambil, cari produk. Belum: pencarian SKU, kategori (tabel ada, tanpa RPC), riwayat harga, formulir UI | Sebagian | `services/product-service/internal/infrastructure/postgres/product_repository.go` |
| 03 INV (inventaris) | Tambah item, ubah stok, daftar, ambang, peristiwa stok rendah. Belum: penjaga stok non-negatif, buku besar pergerakan, rute REST untuk ubah stok/ambang/get item | Sebagian | `services/inventory-service/internal/application/commands/update_stock.go` |
| 04 SAL (penjualan) | Penjualan dengan item, diskon item, total, struk snapshot, peristiwa `Sale.Completed`. Belum: atomisitas, idempotensi, PPN, metode pembayaran, struk di UI, riwayat struk | Sebagian | `services/sales-service/internal/application/commands/create_sale.go` |
| 05 SUP (pemasok/PO) | Tambah/daftar pemasok, buat PO, terima stok. Belum: penerimaan parsial dan atomik, pembatalan PO, rute REST untuk terima dan daftar PO | Sebagian | `services/supplier-service/internal/application/commands/receive_stock.go` |
| 06 CUS (pelanggan/loyalitas) | Buat, ambil, daftar pelanggan; RPC `AddLoyaltyPoints`. Belum: pemanggil poin setelah penjualan, aturan perolehan dan penukaran | Sebagian | `services/customer-service/internal/application/commands/add_loyalty_points.go` |
| 07 RPT (laporan) | Laporan penjualan (pendapatan, jumlah transaksi, produk teratas). Belum: nama produk (selalu kosong), bauran pembayaran, batas hari WIB; dashboard memalsukan data | Sebagian | `services/sales-service/internal/infrastructure/postgres/sale_repository.go` |
| 08 NTF (notifikasi) | Tidak ada layanan; peristiwa stok rendah dipublikasikan tanpa konsumen | Belum | `services/_shared/nats/streams.go` |
| 09 AI (peramalan) | Tidak ada layanan AI; subjek `AI.OrderForecasted` hanya konstanta tanpa penerbit | Belum | `services/_shared/nats/streams.go` |
| Lintas modul: keamanan | RLS `ENABLE` + `FORCE` pada seluruh tabel bertenant; 7 dari 15 kebijakan tanpa `WITH CHECK`; interseptor token layanan gagal-terbuka; rahasia di dua berkas terlacak | Sebagian | `services/_shared/middleware/grpc_interceptors.go` |
| Lintas modul: kualitas | 79 fungsi tes unit (berbasis stub), tidak ada tes integrasi; CI usang | Sebagian | `.github/workflows/ci.yml` |

---

## 14. Persetujuan

| Nama | Jabatan | Tanda Tangan | Tanggal |
|---|---|---|---|
| | Product Owner | | |
| | Business Sponsor | | |
| | Engineering Lead | | |
