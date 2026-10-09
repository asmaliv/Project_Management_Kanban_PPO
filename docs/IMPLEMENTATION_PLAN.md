# Implementation Plan — PPO Kanban Board

## Project Rule

Kerjakan **satu phase dalam satu waktu**.
Jangan mulai phase berikutnya sebelum seluruh pengecekan **Verify** phase aktif lulus.
Sebelum coding tiap phase: baca dokumen terkait, tuliskan rencana, asumsi, dan daftar file yang akan dibuat/diubah.

Urutan sengaja menaruh **audit log lebih awal** (Phase 3) supaya semua fitur setelahnya otomatis tercatat.

---

## Phase 0 — Project Setup

Tasks:
- Struktur monorepo `/backend`, `/frontend`, `/docs` (sesuai ARCHITECTURE.md)
- `go mod init`, server chi minimal dengan `GET /healthz`
- `docker-compose.yml` untuk PostgreSQL lokal
- Konfigurasi env (`.env.example`, loader config)
- Setup `golang-migrate` dan `sqlc`
- Frontend Vite + React + TypeScript + Tailwind
- Lint dan format (golangci-lint, ESLint, Prettier)
- Salin dokumen ke `/docs`, `AGENTS.md` di root

Dependencies: repository GitHub sudah ada.

Deliverable:
- Backend dan frontend berjalan lokal tanpa error setup.

Verify:
- Clone baru + perintah di README → `/healthz` menjawab 200 dan halaman frontend tampil.
- `.env` tidak ikut ter-commit.

---

## Phase 1 — Autentikasi & Sesi

Tasks:
- Migrasi `users`, `sessions`
- Hash argon2id, login, logout, `GET /auth/me`, ganti password
- Cookie sesi, idle/absolute timeout, CSRF
- Lockout setelah 5 kali gagal, rate limit login
- Bootstrap admin dari env
- Halaman Login dan Ganti Password
- Penjaga route di frontend

Dependencies: Phase 0.

Deliverable:
- User dapat login, ganti password, dan logout dengan aman.

Verify:
- Kredensial salah → pesan generik.
- 5 gagal → akun terkunci.
- Tanpa login, endpoint terproteksi → 401.
- Request tulis tanpa CSRF token → ditolak.
- Password tidak muncul di log maupun respons.

---

## Phase 2 — Manajemen User & RBAC

Tasks:
- Middleware otorisasi + definisi permission dari matriks SECURITY.md
- CRUD user (Admin) termasuk **form Tambah Pengguna** untuk kelima role, nonaktifkan/aktifkan, reset password
- Halaman Pengguna
- Test matriks RBAC (helper yang dapat dipakai ulang untuk 5 role + anonim)

Dependencies: Phase 1.

Deliverable:
- Admin mengelola user; role lain ditolak.

Verify:
- Setiap endpoint users diuji untuk semua 5 role + anonim.
- Admin tidak bisa menonaktifkan/menurunkan role dirinya sendiri.
- User nonaktif tidak bisa login dan sesinya terputus.

---

## Phase 3 — Audit Trail (fondasi)

Tasks:
- Migrasi `audit_logs` + trigger append-only
- Service `auditlog` yang menulis dalam transaksi yang sama dengan perubahan data
- Catat event auth dan user (login sukses/gagal, lockout, logout, perubahan role)
- `GET /audit-logs` dengan filter, halaman Audit Trail (Audit & Admin)

Dependencies: Phase 2.

Deliverable:
- Semua aksi Phase 1–2 tercatat dan dapat ditelusuri.

Verify:
- Mencoba `UPDATE`/`DELETE` ke `audit_logs` (via SQL) gagal.
- Role selain Audit/Admin tidak dapat membaca audit trail.
- Kegagalan menulis audit log membatalkan perubahan data.

---

## Phase 4 — Proyek, Member, dan Kolom

Tasks:
- Migrasi `projects`, `project_members`, `board_columns`
- CRUD proyek (PM), arsip, transfer owner (Admin)
- Kelola member (hanya `team_leader`) dan kolom (Done wajib ada satu)
- Kolom default dibuat otomatis saat proyek dibuat
- Daftar proyek terfilter sesuai role; halaman Proyek
- Audit log untuk semua aksi

Dependencies: Phase 3.

Deliverable:
- PM membuat proyek, menyusun kolom, dan menambah Team Leader.

Verify:
- PM A tidak dapat melihat/mengubah proyek PM B (404).
- Team Leader hanya melihat proyek tempat dia member.
- Audit/Risk/Admin melihat semua proyek tetapi tidak dapat mengubah.
- Kolom Done tidak dapat dihapus; proyek terarsip read-only.

---

## Phase 5 — Task & Papan Kanban

Tasks:
- Migrasi `tasks`, `task_comments`
- CRUD task, optimistic locking (`version`), soft delete
- Endpoint `move` dengan transaksi penyusunan posisi, WIP limit, aturan maker-checker
- Endpoint `GET /tasks` (lintas proyek, filter, paginasi) yang dibatasi sesuai izin role
- Komentar task
- UI papan: kolom, kartu, drawer detail, drag-and-drop + menu "Pindahkan ke…"
- Audit log untuk semua aksi

Dependencies: Phase 4.

Deliverable:
- Tim dapat bekerja di papan Kanban dengan aturan persetujuan PM.

Verify:
- Team Leader tidak bisa memindahkan ke/dari Done (API dan UI).
- Pemindahan bersamaan oleh dua user → satu berhasil, satu 409; urutan tetap konsisten.
- Refresh tidak menghilangkan perubahan.
- WIP limit ditegakkan.
- Alternatif keyboard untuk memindah task berfungsi.

---

## Phase 6 — Risk Management

Tasks:
- Migrasi `risks`
- Ajukan risiko (PM, Team Leader, Risk Management); nilai/tutup (Risk Management saja)
- Skor dan level dihitung server; aturan mitigasi untuk High/Critical
- Tautan risiko ke proyek/task; indikator risiko di kartu task
- Halaman Risk Register + heatmap dan `GET /risks/summary`
- Audit log

Dependencies: Phase 5.

Deliverable:
- Risiko tercatat, dinilai, dipantau, dan terhubung ke pekerjaan.

Verify:
- Skor dan level tidak bisa dimanipulasi dari client.
- Hanya Risk Management yang bisa menilai/menutup.
- High/Critical tanpa mitigasi tidak dapat ditutup.
- Team Leader hanya melihat risiko proyek tempat dia member.

---

## Phase 7 — Temuan Audit & Dashboard per Role

Tasks:
- Migrasi `audit_findings`
- Buat/respons/buka kembali/tutup temuan sesuai alur status
- Halaman Temuan Audit
- Ekspor audit trail CSV (Audit saja, rate limited)
- `GET /dashboard` dan UI dashboard untuk kelima role
- Dashboard Team Leader: Ringkasan Tim (total, terlambat, menunggu persetujuan, task per kolom, beban kerja per member) + tab Tugas Saya
- Halaman "Tugas Tim" untuk Team Leader (memakai `GET /tasks`)

Dependencies: Phase 6.

Deliverable:
- Siklus temuan audit lengkap dan dashboard bermakna per role.

Verify:
- Hanya Audit yang membuat/menutup; hanya PM owner yang merespons.
- Alur status tidak bisa dilompati.
- Dashboard tiap role hanya memuat data yang boleh dilihat role tersebut.
- Team Leader melihat task milik member lain di proyek yang sama, tetapi tidak melihat satu pun task di proyek yang bukan miliknya.
- Angka di dashboard Team Leader sama dengan jumlah task yang tampak di papan.
- CSV tidak memuat field sensitif.

---

## Phase 8 — Production Readiness

Tasks:
- Header keamanan (CSP, HSTS, dll) dan rate limit menyeluruh
- Review keamanan (SECURITY.md), `govulncheck`, `npm audit`
- Uji responsif dan aksesibilitas
- Dockerfile (non-root), konfigurasi environment production
- Dokumentasi deployment dan prosedur backup/restore
- Jalankan seluruh **TESTING.md**

Dependencies: Phase 7.

Deliverable:
- MVP siap rilis.

Verify:
- Seluruh checklist TESTING.md lulus; tidak ada Release Blocker.

---

## Out of Scope untuk V1

- Notifikasi email/push
- SSO/LDAP dan MFA
- Lampiran file
- Time tracking dan Gantt
- Kolaborasi real-time (WebSocket)
- Multi-tenant
- Aplikasi mobile native
- Integrasi sistem bank

## Cara Bekerja dengan Plan Ini

1. Berikan AI hanya phase aktif + dokumen relevan.
2. Minta daftar file yang akan diubah sebelum mengubahnya.
3. Setelah selesai, jalankan pengecekan Verify.
4. Perbaiki kegagalan sebelum lanjut.
5. Perbarui dokumen jika scope atau keputusan berubah.
