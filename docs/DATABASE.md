# DATABASE.md — PPO Kanban Board

## 1. Purpose

Menjelaskan bagaimana data dimodelkan, dihubungkan, diakses, dan diubah dengan aman.

## 2. Database Stack

- Database: **PostgreSQL 16+**
- Akses: `pgx` + **sqlc** (query ditulis di `backend/queries/*.sql`, kode hasil generate di `internal/db`)
- Migrasi: **golang-migrate** (`backend/migrations/`)
- ID: `UUID` (`gen_random_uuid()`), kecuali `audit_logs.id` (`BIGSERIAL`)
- Waktu: `TIMESTAMPTZ`, disimpan UTC

## 3. Environment

- `DATABASE_URL` wajib dari environment variable. Jangan hardcode.
- Development, staging, dan production memakai database **terpisah**.
- Aplikasi memakai user database non-superuser.

## 4. Core Models

### users
Akun aplikasi.
- `id`, `email` (unik, disimpan huruf kecil), `full_name`
- `password_hash`
- `role` — enum `user_role`: `admin`, `project_manager`, `team_leader`, `audit`, `risk_management`
- `is_active`, `must_change_password`
- `failed_login_count`, `locked_until`, `last_login_at`
- `created_at`, `updated_at`

### sessions
- `id`, `user_id` → users.id
- `token_hash` (unik) — hash dari token cookie
- `created_at`, `last_seen_at`, `expires_at`
- `ip`, `user_agent`

### projects
- `id`, `code` (unik, mis. `PRJ-0001`), `name`, `description`
- `status` — enum `project_status`: `active`, `on_hold`, `completed`, `archived`
- `owner_id` → users.id (harus berrole `project_manager`)
- `start_date`, `end_date`
- `created_by` → users.id
- `created_at`, `updated_at`, `deleted_at`

### project_members
User berrole `team_leader` yang ditugaskan ke proyek.
- `project_id` → projects.id, `user_id` → users.id (primary key gabungan)
- `added_by` → users.id, `added_at`

### board_columns
- `id`, `project_id` → projects.id
- `name`, `position` (integer, urut kiri ke kanan)
- `is_done_column` (boolean) — tepat satu per proyek
- `wip_limit` (integer, nullable)
- `created_at`

### tasks
- `id`, `project_id` → projects.id, `column_id` → board_columns.id
- `title`, `description`
- `priority` — enum `task_priority`: `low`, `medium`, `high`, `critical`
- `assignee_id` → users.id (nullable), `reporter_id` → users.id
- `due_date`, `progress_percent` (0–100)
- `position` (integer, urutan dalam kolom)
- `labels` (`text[]`)
- `version` (integer, untuk optimistic locking)
- `completed_at`, `created_at`, `updated_at`, `deleted_at`

### task_comments
- `id`, `task_id` → tasks.id, `author_id` → users.id
- `body`, `created_at`

### risks
- `id`, `code` (unik, `RSK-0001`)
- `project_id` → projects.id, `task_id` → tasks.id (nullable)
- `title`, `description`
- `category` — enum `risk_category`: `operational`, `schedule`, `technical`, `compliance`, `financial`, `security`
- `likelihood` (1–5), `impact` (1–5) — nullable sampai dinilai
- `score` (kolom generated = likelihood × impact, nullable)
- `level` — enum `risk_level`: `low`, `medium`, `high`, `critical` (diisi service dari skor)
- `status` — enum `risk_status`: `identified`, `assessed`, `mitigating`, `accepted`, `closed`
- `mitigation_plan`
- `owner_id` → users.id (nullable), `raised_by` → users.id
- `created_at`, `updated_at`, `closed_at`

### audit_findings
- `id`, `code` (unik, `FND-0001`)
- `project_id` → projects.id, `task_id` → tasks.id (nullable)
- `title`, `description`
- `severity` — enum `finding_severity`: `low`, `medium`, `high`, `critical`
- `status` — enum `finding_status`: `open`, `responded`, `closed`
- `raised_by` → users.id (berrole `audit`)
- `response`, `responded_by` → users.id, `responded_at`
- `closed_by` → users.id, `closed_at`
- `created_at`, `updated_at`

### audit_logs
Append-only.
- `id` (BIGSERIAL), `occurred_at`
- `actor_id` → users.id (nullable untuk login gagal), `actor_role`
- `action` (mis. `task.move`, `user.create`, `auth.login_failed`)
- `entity_type`, `entity_id`, `project_id` (nullable)
- `before` (JSONB), `after` (JSONB) — tanpa field sensitif
- `ip`, `request_id`

## 5. Relationships

- Satu **User** (PM) memiliki banyak **Project**; satu Project memiliki satu owner.
- **Project** ↔ **User** (Team Leader) many-to-many lewat `project_members`.
- Satu **Project** memiliki banyak **board_columns**, **tasks**, **risks**, dan **audit_findings**.
- Satu **board_column** memiliki banyak **tasks**.
- Satu **Task** memiliki banyak **task_comments**; dapat dirujuk oleh banyak **risks** dan **audit_findings**.
- **audit_logs** merujuk entitas apa pun secara longgar (`entity_type` + `entity_id`, tanpa foreign key) agar tetap utuh walau entitas dihapus lunak.

## 6. Schema Rules

- Setiap record utama punya ID stabil dan `created_at`/`updated_at`.
- Gunakan foreign key untuk semua relasi (kecuali `audit_logs.entity_id`).
- Constraint unik: `users.email`, `projects.code`, `risks.code`, `audit_findings.code`, `sessions.token_hash`.
- Constraint check: `progress_percent BETWEEN 0 AND 100`; `likelihood`/`impact` BETWEEN 1 AND 5.
- Unique partial index: satu kolom `is_done_column = true` per proyek.
- Unique `(project_id, position)` pada `board_columns` (dapat ditunda dalam transaksi).
- Index yang diperlukan: `tasks(project_id, column_id, position)`, `tasks(assignee_id)`, `tasks(due_date)`, `risks(project_id, level)`, `audit_findings(project_id, status)`, `audit_logs(occurred_at)`, `audit_logs(entity_type, entity_id)`, `audit_logs(actor_id)`, `sessions(user_id)`.
- Penghapusan bisnis memakai **soft delete** (`deleted_at`). Query default harus menyaring `deleted_at IS NULL`.
- Jangan menyimpan fakta yang sama di dua tempat tanpa alasan. `score` dihitung dari likelihood × impact.

### Trigger wajib: audit_logs append-only

```sql
CREATE OR REPLACE FUNCTION audit_logs_block_mutation() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION 'audit_logs is append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_logs_no_update_delete
BEFORE UPDATE OR DELETE ON audit_logs
FOR EACH ROW EXECUTE FUNCTION audit_logs_block_mutation();
```

Pertimbangkan juga `REVOKE UPDATE, DELETE, TRUNCATE ON audit_logs` dari user aplikasi.

## 7. Migrations

Saat skema berubah:

1. Buat file migrasi baru berurutan: `backend/migrations/000001_init.up.sql` dan `.down.sql`.
2. Jangan pernah mengedit migrasi yang sudah dijalankan; buat migrasi baru.
3. Tinjau SQL hasil tulisan.
4. Uji `up` dan `down` di database lokal.
5. Jalankan `sqlc generate` jika query berubah.
6. Terapkan lewat proses deployment, bukan dengan mengubah database production secara manual.

## 8. Seed Data

- Seed hanya untuk development/test, berisi **data palsu**.
- Admin pertama dibuat dari `BOOTSTRAP_ADMIN_EMAIL` dan `BOOTSTRAP_ADMIN_PASSWORD`, wajib ganti password saat login pertama.
- Dilarang memasukkan data nyata atau secret di seed.

## 9. Production Safety

- Backup rutin dan uji restore.
- Hindari perubahan skema destruktif tanpa rencana migrasi.
- Jangan menjalankan perintah reset/drop/truncate di production.
- Gunakan transaksi untuk operasi yang harus berhasil atau gagal bersamaan (pindah task, perubahan kolom, perubahan data + audit log).
- Ikuti SECURITY.md untuk akses dan data sensitif.
