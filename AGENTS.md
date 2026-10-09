# Agent Instructions — PPO Kanban Board

Dokumen ini berlaku untuk semua AI coding agent (Antigravity, Claude, dll).

## Project Context

Aplikasi web Project Management Kanban Board dengan 5 role: `admin`, `project_manager`, `team_leader`, `audit`, `risk_management`.

- Backend: **Go** + chi + PostgreSQL (pgx + sqlc)
- Frontend: **React + TypeScript + Vite + Tailwind**
- Struktur monorepo: `/backend`, `/frontend`, `/docs`
- Domain: manajemen proyek di lingkungan perbankan. Prioritas utama: **keamanan, jejak audit, pemisahan tugas**.
- Proyek belajar: jangan pernah memakai data nyata.

## Before You Start

Baca file berikut sebelum mengubah kode (urutan ini):

1. `docs/PRD.md` — apa yang dibangun
2. `docs/TRD.md` — persyaratan teknis
3. `docs/ARCHITECTURE.md` — struktur sistem
4. `docs/SECURITY.md` — aturan keamanan dan **matriks RBAC**
5. `docs/DATABASE.md` dan `docs/API.md` — jika menyentuh data atau endpoint
6. `docs/CODE_STYLE.md` — gaya kode
7. `docs/DESIGN_SYSTEM.md` — jika menyentuh UI
8. `docs/APP_FLOW.md` — jika membuat layar atau alur
9. `docs/IMPLEMENTATION_PLAN.md` dan `docs/TESTING.md` — untuk tahu phase aktif dan cara verifikasi

Lalu periksa kode yang sudah ada sebelum membuat file/komponen/pola baru.

## Working Rules

- **Kerjakan satu phase dalam satu waktu** sesuai `IMPLEMENTATION_PLAN.md`. Jangan mengerjakan phase berikutnya sebelum verifikasi phase aktif lulus.
- Sebelum menulis kode: tuliskan rencana, asumsi, dan daftar file yang akan dibuat/diubah. Tunggu persetujuan untuk perubahan besar.
- Perubahan kecil dan fokus. Satu tujuan per perubahan agar mudah di-commit.
- Reuse komponen dan helper yang ada. Ikuti struktur folder yang ada.
- Jika dokumen saling bertentangan atau keputusan penting belum ada, **tanya dulu**.
- Jika keputusan produk/teknis berubah, perbarui dokumen terkait di commit yang sama.
- Setelah selesai, laporkan: apa yang dibuat, apa yang **benar-benar diverifikasi** (dan bagaimana), dan apa yang **belum** diverifikasi. Jangan menandai sesuatu "lulus" jika tidak benar-benar dijalankan.

## Security Rules (ringkas, lengkap di SECURITY.md)

- Otorisasi **selalu** diverifikasi di server. Jangan percaya role/user ID dari client.
- Default **deny**: endpoint baru tanpa aturan izin eksplisit dianggap ditolak.
- Jangan pernah melemahkan, melewati, atau menonaktifkan autentikasi/RBAC demi membuat fitur jalan.
- Jangan mengubah, menghapus, atau membuat jalur `UPDATE`/`DELETE` terhadap tabel `audit_logs`.
- Setiap aksi tulis (create/update/delete/move/approve) wajib menulis audit log.
- Jangan menaruh secret, password, token, atau data nyata di kode, dokumentasi, seed, maupun log.
- Gunakan query terparameter (sqlc). Dilarang menyusun SQL dengan string concatenation.

## Code Guidelines (ringkas, lengkap di CODE_STYLE.md)

- Go: `gofmt`, error dibungkus dengan `%w`, `context.Context` sebagai parameter pertama, tanpa `panic` di alur normal.
- Arsitektur berlapis: handler → service → store. Handler tidak boleh mengakses database langsung.
- TypeScript: mode `strict`, tanpa `any`, komponen fungsional.
- Hapus `console.log`, debug print, dan dead code sebelum selesai.

## Skills & Tools

Skill ada di `.agents/skills/`. Dokumen di `docs/` selalu menang atas skill mana pun.

- `ppo-endpoint-checklist`: wajib dijalankan setiap membuat atau mengubah endpoint, sebelum menyatakan pekerjaan selesai.
- `graphify` (jika terpasang): sebelum membuat rencana untuk phase baru, gunakan `graphify query "<pertanyaan>"` untuk memahami kode yang sudah ada, jangan membaca seluruh repo secara manual. Graph bisa usang; jika kode banyak berubah, minta saya membangun ulang dengan `/graphify .`.
- `ponytail` (jika terpasang): pilih solusi paling sederhana, tetapi jangan pernah melewati validasi input, RBAC, audit log, atau aturan di `docs/SECURITY.md`.
- Skill atau tool pihak ketiga tidak boleh menambah dependency atau mengubah struktur proyek tanpa persetujuan eksplisit.

## Commands

```bash
# Database lokal
docker compose up -d db

# Backend (jalankan dari /backend)
go run ./cmd/server
go build ./...
go vet ./...
golangci-lint run
go test ./...
govulncheck ./...

# Migrasi
migrate -path migrations -database "$DATABASE_URL" up
migrate -path migrations -database "$DATABASE_URL" down 1

# Generate kode query
sqlc generate

# Frontend (jalankan dari /frontend)
npm install
npm run dev
npm run build
npm run lint
npm run typecheck
npm run test
```

## Boundaries — jangan ubah tanpa persetujuan eksplisit

- Menambah dependency/library baru
- Mengubah skema database (harus lewat file migrasi baru, tidak pernah edit migrasi lama)
- Mengubah matriks RBAC, mekanisme sesi, atau hashing password
- Mengubah arsitektur atau struktur folder utama
- Menjalankan perintah yang destruktif (drop, reset, truncate) pada database apa pun selain database lokal development
- Menyentuh file `.env` asli
