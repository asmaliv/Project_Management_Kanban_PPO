# Architecture — PPO Kanban Board

> Pemilik dokumen ini: **bagaimana bagian-bagian sistem terhubung**.

## 1. System Overview

```
Browser (React SPA)
   │  HTTPS, cookie sesi HttpOnly, header X-CSRF-Token
   ▼
Go API Server  (chi router)
   │  middleware: request-id → logging → recover → security headers
   │              → rate limit → session auth → CSRF → RBAC
   ▼
Handler  →  Service (aturan bisnis)  →  Store (sqlc)
                                          │
                                          ▼
                                    PostgreSQL
                          (data bisnis + sessions + audit_logs)
```

V1 tidak memiliki layanan eksternal.

## 2. Tech Stack

Lihat tabel lengkap di TRD.md bagian 3. Ringkas: Go + chi + pgx/sqlc + PostgreSQL; React + TypeScript + Vite + Tailwind + TanStack Query + dnd-kit.

## 3. Project Structure

```
/
├── AGENTS.md
├── README.md
├── docker-compose.yml          # PostgreSQL lokal
├── .env.example
├── docs/                       # semua dokumen .md perencanaan
├── backend/
│   ├── cmd/server/main.go      # entry point, wiring dependency
│   ├── internal/
│   │   ├── config/             # baca env var
│   │   ├── httpapi/            # router, middleware, handler, DTO, helper respons
│   │   ├── auth/               # login, sesi, hashing, lockout
│   │   ├── rbac/               # definisi role, permission, helper otorisasi
│   │   ├── user/               # service + store user
│   │   ├── project/            # proyek, member, kolom
│   │   ├── task/               # task, komentar, pemindahan
│   │   ├── risk/               # risk register
│   │   ├── finding/            # temuan audit
│   │   ├── auditlog/           # penulisan & pembacaan audit trail
│   │   ├── dashboard/          # agregasi per role
│   │   └── db/                 # kode hasil sqlc (jangan diedit manual)
│   ├── queries/                # file .sql untuk sqlc
│   ├── migrations/             # file migrasi berurutan
│   └── sqlc.yaml
└── frontend/
    └── src/
        ├── app/                # router, layout, provider
        ├── features/           # auth, users, projects, board, risks, findings, audit, dashboard
        ├── components/         # komponen UI yang dipakai ulang
        ├── lib/                # api client, util
        └── types/              # tipe TypeScript bersama
```

Aturan dependensi backend: `httpapi → service → store → db`. Paket domain tidak boleh mengimpor `httpapi`.

## 4. Data Flow (contoh: memindahkan task)

1. User menyeret kartu; React mengirim `POST /api/v1/tasks/:id/move` dengan `to_column_id`, `position`, `version`.
2. Middleware memeriksa sesi, CSRF, dan rate limit.
3. Handler memvalidasi input.
4. Service memuat task + proyek, memeriksa **izin** (role + keanggotaan proyek) dan **aturan alur** (maker-checker, WIP limit).
5. Dalam **satu transaksi**: perbarui kolom/posisi task, susun ulang posisi, tulis `audit_logs`.
6. Respons `200` dengan data task terbaru.
7. UI memperbarui cache TanStack Query; jika gagal, kartu dikembalikan ke posisi semula dan pesan error ditampilkan.

## 5. Authentication & Session

- Login membuat baris di `sessions`; token acak disimpan **ter-hash** di database, token asli hanya di cookie `HttpOnly; Secure; SameSite=Lax`.
- CSRF: token dikirim via header `X-CSRF-Token` pada semua request tulis.
- Frontend dan API disajikan dari origin yang sama (reverse proxy / proxy dev Vite) sehingga tidak perlu CORS terbuka.

## 6. Database & Storage

- PostgreSQL adalah satu-satunya penyimpanan. Detail: DATABASE.md.
- Tidak ada penyimpanan file/objek.

## 7. External Services

Tidak ada pada V1.

## 8. Deployment

- **Lokal:** `docker compose up -d db`, backend `go run ./cmd/server`, frontend `npm run dev`.
- **Production:** TBD. Prinsip: satu binary Go + aset frontend statis di belakang reverse proxy HTTPS; database terkelola atau di-backup terjadwal; variabel lingkungan terpisah per environment.

## 9. Scalability Notes (untuk nanti, bukan V1)

- Tambahkan index sesuai pola query nyata sebelum menambah cache.
- Dashboard dapat di-cache singkat bila lambat.
- Pembersihan sesi kedaluwarsa lewat job terjadwal.
- Audit log besar dapat dipartisi per bulan.
- Notifikasi dan job latar belakang membutuhkan antrean (masuk fase setelah V1).
