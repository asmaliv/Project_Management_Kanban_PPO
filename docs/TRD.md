# Technical Requirements Document (TRD) — PPO Kanban Board

> Pemilik dokumen ini: **bagaimana sistem bekerja secara teknis**. Fitur produk ada di PRD.md.

## 1. Project Overview

Aplikasi web Kanban multi-role untuk manajemen proyek, risiko, dan temuan audit. Backend REST API berbasis Go; frontend SPA React yang berkomunikasi lewat JSON API dengan sesi cookie.

## 2. Technical Goals

- Otorisasi ketat berbasis role dan kepemilikan proyek, dipaksakan di server.
- Audit trail lengkap dan tidak dapat diubah.
- Papan Kanban responsif dan konsisten (tidak kehilangan urutan task saat drag-and-drop).
- Mudah dijalankan oleh developer pemula: satu perintah untuk database lokal, satu perintah untuk backend, satu untuk frontend.

## 3. Proposed Tech Stack

> Ini usulan. Ubah hanya dengan persetujuan, lalu perbarui dokumen ini.

| Area | Pilihan | Alasan singkat |
|---|---|---|
| Bahasa backend | Go (versi stabil terbaru, minimal 1.24) | Sesuai keputusan proyek |
| HTTP router | `go-chi/chi` | Ringan, idiomatik, kompatibel `net/http` |
| Database | PostgreSQL 16+ | Transaksi, constraint, JSONB, trigger |
| Akses data | `pgx` + `sqlc` | SQL eksplisit, aman dari injection, type-safe |
| Migrasi | `golang-migrate` | Sederhana, berbasis file SQL |
| Validasi | `go-playground/validator` | Validasi struct request |
| Hash password | `argon2id` (`golang.org/x/crypto/argon2`) | Standar modern |
| Logging | `log/slog` (JSON) | Bawaan Go |
| Frontend | React + TypeScript + Vite | Ekosistem terbesar untuk AI-assisted coding |
| Styling | Tailwind CSS | Cocok dengan design tokens |
| Server state | TanStack Query | Cache dan sinkronisasi data API |
| Form | react-hook-form + zod | Validasi di client (bukan pengganti server) |
| Drag-and-drop | `dnd-kit` | Mendukung keyboard dan aksesibilitas |
| Autentikasi | Sesi di database + cookie HttpOnly | Bisa dicabut server-side, tanpa token di localStorage |
| Container lokal | Docker Compose (PostgreSQL) | Setup seragam |
| Hosting production | **TBD** | Belum diputuskan |

## 4. Functional Requirements

### 4.1 Autentikasi
- User dapat login dengan email dan password, dan logout.
- Sesi berakhir setelah idle 30 menit dan maksimal 8 jam sejak login.
- Setelah 5 kali gagal login berturut-turut, akun terkunci 15 menit.
- User dengan `must_change_password = true` hanya dapat mengakses endpoint ganti password sampai password diganti.
- Pesan gagal login tidak membedakan "email tidak ada" dan "password salah".

### 4.2 User & Role
- Admin dapat **menambah user baru** (email, nama, role, password sementara), mengubah, menonaktifkan, dan mengaktifkan user, serta mengatur role dan mereset password.
- User baru otomatis memiliki `must_change_password = true`. Password sementara hanya ditampilkan sekali dan tidak pernah disimpan atau dicatat dalam bentuk asli.
- Satu user memiliki tepat satu role.
- User nonaktif tidak dapat login dan sesi aktifnya dicabut.
- Admin tidak dapat menonaktifkan atau menurunkan role dirinya sendiri.

### 4.3 Proyek
- PM dapat membuat proyek dan otomatis menjadi owner.
- PM owner dapat mengubah, mengarsipkan proyek, serta mengelola member dan kolom.
- Hanya user berrole `team_leader` yang dapat ditambahkan sebagai member.
- Admin dapat memindahkan owner ke PM lain.
- Proyek terarsip bersifat read-only.

### 4.4 Task & Papan Kanban
- Papan menampilkan kolom berurutan dan task terurut dalam kolom.
- Task dapat dibuat, diubah, dipindah, dan dihapus (soft delete).
- Assignee harus member proyek (atau owner).
- Memindahkan task ke atau dari kolom `is_done_column` hanya boleh oleh PM owner.
- Jika kolom memiliki `wip_limit`, pemindahan yang melampaui batas ditolak (409).
- Pembaruan task memakai optimistic locking (`version`); konflik ditolak (409).
- Urutan task disimpan di server; perubahan urutan dilakukan dalam satu transaksi.

### 4.5 Risiko
- PM, Team Leader (pada proyek mereka), dan Risk Management dapat mengajukan risiko.
- Hanya Risk Management yang dapat menilai (likelihood, impact), mengubah level, dan menutup risiko.
- Skor = likelihood × impact. Level: 1–4 Low, 5–9 Medium, 10–15 High, 16–25 Critical. Skor dan level dihitung di server.
- Risiko High/Critical tidak dapat ditutup tanpa rencana mitigasi terisi.

### 4.6 Temuan Audit
- Hanya Audit yang dapat membuat dan menutup temuan.
- Hanya PM owner proyek yang dapat merespons temuan.
- Alur status: `open` → `responded` → `closed`. Audit dapat membuka kembali (`responded` → `open`) dengan alasan.

### 4.7 Audit Trail
- Setiap aksi tulis menghasilkan satu baris `audit_logs` dalam transaksi yang sama dengan perubahan datanya.
- Tabel `audit_logs` hanya menerima `INSERT`; `UPDATE` dan `DELETE` ditolak oleh trigger database.
- Audit dan Admin dapat melihat; hanya Audit yang dapat mengekspor CSV.

### 4.8 Dashboard
- Satu endpoint dashboard mengembalikan ringkasan sesuai role pemanggil.
- Dashboard Team Leader menampilkan agregat seluruh task di proyek tempat ia member: total, terlambat, menunggu persetujuan PM, jumlah per kolom, dan beban kerja per member. Agregasi dihitung di server dan dibatasi ke proyek yang diikuti pemanggil.
- `GET /tasks` menyediakan daftar task lintas proyek dengan filter dan paginasi, dibatasi sesuai izin role (untuk halaman "Tugas Tim" dan "Tugas Saya").

## 5. Non-Functional Requirements

Angka di bawah adalah **target**, diukur di mesin development dengan data uji.

- **Performa:** memuat papan 500 task p95 < 500 ms di API; aksi pindah task p95 < 300 ms.
- **Responsif:** UI dapat dipakai mulai lebar 320px; papan scroll horizontal di layar kecil.
- **Aksesibilitas:** target WCAG 2.1 AA; semua fungsi drag-and-drop punya alternatif keyboard/menu.
- **Keamanan:** lihat SECURITY.md. Semua endpoint selain `/healthz` dan `/auth/login` butuh sesi valid.
- **Keandalan:** error terkontrol dengan format respons konsisten; tidak ada stack trace ke client.
- **Observabilitas:** log terstruktur JSON dengan `request_id`; tidak mencatat password/token.
- **Portabilitas:** seluruh konfigurasi lewat environment variable.

## 6. Integrations

V1 **tidak** memiliki integrasi eksternal. Email, SSO, dan layanan lain masuk daftar Out of Scope.

## 7. Data Requirements

- Seluruh data bisnis di PostgreSQL. Skema dan relasi: DATABASE.md.
- Tidak ada penyimpanan file.
- Data pribadi dibatasi: nama dan email kerja saja.
- Data yang dihapus bersifat soft delete (`deleted_at`) kecuali sesi kedaluwarsa.

## 8. Constraints

- Backend wajib Go.
- Secret hanya lewat environment variable.
- Otorisasi diverifikasi di server untuk setiap operasi terproteksi.
- Tidak ada aksi destruktif tanpa konfirmasi di UI.
- Tidak ada akses langsung `UPDATE`/`DELETE` ke `audit_logs`.
- Dilarang menambah dependency tanpa persetujuan.
- Tidak boleh memakai data nyata.

## 9. Definition of Done

Sebuah phase selesai jika:

- Deliverable phase berfungsi end-to-end.
- Seluruh pengecekan verifikasi phase di IMPLEMENTATION_PLAN.md lulus.
- `go vet`, `golangci-lint`, `go test ./...`, `npm run lint`, `npm run typecheck`, `npm run test` bersih.
- Test matriks RBAC untuk endpoint yang disentuh lulus untuk semua 5 role + tanpa login.
- Aksi tulis baru menghasilkan audit log.
- Tidak ada secret di repo.
- Dokumen terkait (API.md, DATABASE.md, dll) sudah diperbarui.

MVP dianggap siap jika semua phase selesai dan **TESTING.md** (termasuk Release Blockers) lulus.
