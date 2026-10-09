# API.md — PPO Kanban Board

## 1. Purpose

Mendokumentasikan konvensi dan endpoint API antara frontend dan backend Go.

## 2. Base Configuration

- Development: `http://localhost:8080/api/v1` (frontend memakai proxy Vite agar satu origin)
- Production: `https://<domain>/api/v1` (TBD)
- Format: JSON (`Content-Type: application/json`)
- Waktu: ISO 8601 UTC. ID: UUID string.

## 3. Authentication

- Sesi lewat cookie `HttpOnly` yang diset saat login.
- Request tulis wajib header `X-CSRF-Token` (diperoleh dari `GET /auth/me`).
- Tanpa sesi valid → `401`. Role/kepemilikan tidak cukup → `403` (atau `404` untuk objek yang tidak boleh diketahui keberadaannya).
- Izin per endpoint mengikuti matriks di **SECURITY.md**.

## 4. Endpoint Conventions

- Resource memakai kata benda jamak: `/projects`, `/tasks`, `/risks`.
- Aksi non-CRUD memakai sub-path kata kerja: `/tasks/:id/move`, `/risks/:id/assess`.
- Daftar mendukung `?page=1&page_size=20` (maks 100) dan filter spesifik.
- Pembaruan parsial memakai `PATCH`.
- Pembaruan task wajib menyertakan `version`.

## 5. Response Shapes

Sukses:

```json
{ "success": true, "data": { } }
```

Daftar:

```json
{
  "success": true,
  "data": [ ],
  "meta": { "page": 1, "page_size": 20, "total": 134 }
}
```

Error:

```json
{
  "success": false,
  "error": {
    "code": "INVALID_INPUT",
    "message": "Judul task wajib diisi.",
    "fields": { "title": "required" }
  }
}
```

## 6. Endpoints

### Sistem
| Method | Path | Akses |
|---|---|---|
| GET | `/healthz` | publik |

### Auth
| Method | Path | Akses |
|---|---|---|
| POST | `/auth/login` | publik (rate limited) |
| POST | `/auth/logout` | login |
| GET | `/auth/me` | login — mengembalikan profil, role, `csrf_token` |
| POST | `/auth/change-password` | login |

### Users
| Method | Path | Akses |
|---|---|---|
| GET | `/users` | admin (penuh); role lain: ringkas (id, nama, role) sesuai matriks |
| POST | `/users` | admin |
| GET | `/users/:id` | admin |
| PATCH | `/users/:id` | admin |
| POST | `/users/:id/reset-password` | admin |
| POST | `/users/:id/deactivate` | admin |
| POST | `/users/:id/activate` | admin |

### Projects
| Method | Path | Akses |
|---|---|---|
| GET | `/projects` | semua role (terfilter sesuai matriks) |
| POST | `/projects` | project_manager |
| GET | `/projects/:id` | sesuai matriks |
| PATCH | `/projects/:id` | PM owner |
| POST | `/projects/:id/archive` | PM owner |
| POST | `/projects/:id/transfer-owner` | admin |
| GET | `/projects/:id/members` | sesuai matriks |
| POST | `/projects/:id/members` | PM owner |
| DELETE | `/projects/:id/members/:userId` | PM owner |
| GET | `/projects/:id/board` | sesuai matriks — kolom + task |
| POST | `/projects/:id/columns` | PM owner |
| PATCH | `/columns/:id` | PM owner |
| DELETE | `/columns/:id` | PM owner (kolom Done tidak dapat dihapus; kolom berisi task ditolak) |

### Tasks
| Method | Path | Akses |
|---|---|---|
| GET | `/tasks` | sesuai matriks — daftar task lintas proyek yang boleh dilihat pemanggil; filter `project_id`, `assignee_id`, `column_id`, `overdue`, `mine`; mendukung paginasi |
| POST | `/projects/:id/tasks` | PM owner, team_leader member |
| GET | `/tasks/:id` | sesuai matriks |
| PATCH | `/tasks/:id` | PM owner, team_leader member |
| POST | `/tasks/:id/move` | PM owner; team_leader member (kecuali ke/dari Done) |
| DELETE | `/tasks/:id` | PM owner (soft delete) |
| GET | `/tasks/:id/comments` | sesuai matriks |
| POST | `/tasks/:id/comments` | PM owner, team_leader member |

### Risks
| Method | Path | Akses |
|---|---|---|
| GET | `/risks` | sesuai matriks; filter `project_id`, `level`, `status` |
| POST | `/risks` | PM owner, team_leader member, risk_management |
| GET | `/risks/:id` | sesuai matriks |
| PATCH | `/risks/:id` | risk_management (semua field); PM owner (hanya `mitigation_plan`) |
| POST | `/risks/:id/assess` | risk_management |
| POST | `/risks/:id/close` | risk_management |
| GET | `/risks/summary` | sesuai matriks — data heatmap |

### Audit Findings
| Method | Path | Akses |
|---|---|---|
| GET | `/findings` | sesuai matriks; filter `project_id`, `status`, `severity` |
| POST | `/findings` | audit |
| GET | `/findings/:id` | sesuai matriks |
| POST | `/findings/:id/respond` | PM owner |
| POST | `/findings/:id/reopen` | audit |
| POST | `/findings/:id/close` | audit |

### Audit Trail
| Method | Path | Akses |
|---|---|---|
| GET | `/audit-logs` | audit, admin; filter `actor_id`, `entity_type`, `entity_id`, `project_id`, `action`, `from`, `to` |
| GET | `/audit-logs/export` | audit — CSV (rate limited) |

### Dashboard
| Method | Path | Akses |
|---|---|---|
| GET | `/dashboard` | login — isi menyesuaikan role |

Untuk `team_leader`, `data` memuat: `totals` (total, terlambat, menunggu persetujuan), `by_column` (jumlah task per nama kolom), `workload` (daftar `{user_id, full_name, active_tasks}` per member), `overdue` (daftar ringkas task terlambat), dan `my_tasks`. Semua angka hanya dihitung dari proyek tempat pemanggil menjadi member; parameter opsional `project_id` mempersempit cakupan.

## 7. Contoh Endpoint

### POST `/api/v1/tasks/:id/move`
Memindahkan task ke kolom/posisi lain.

Auth: wajib. Izin: lihat matriks (Team Leader tidak boleh ke/dari kolom Done).

Request:

```json
{
  "to_column_id": "0b6c...",
  "position": 2,
  "version": 5
}
```

Sukses — 200:

```json
{
  "success": true,
  "data": {
    "id": "7f1a...",
    "column_id": "0b6c...",
    "position": 2,
    "version": 6
  }
}
```

Dilarang (maker-checker) — 403:

```json
{
  "success": false,
  "error": {
    "code": "FORBIDDEN",
    "message": "Hanya Project Manager yang dapat memindahkan task ke atau dari kolom Done."
  }
}
```

Konflik versi — 409:

```json
{
  "success": false,
  "error": {
    "code": "VERSION_CONFLICT",
    "message": "Task sudah diubah oleh pengguna lain. Muat ulang papan."
  }
}
```

### POST `/api/v1/risks/:id/assess`

Request:

```json
{ "likelihood": 4, "impact": 4, "mitigation_plan": "Tambah reviewer cadangan." }
```

Sukses — 200: skor dan level dihitung server.

```json
{
  "success": true,
  "data": { "id": "c2d9...", "likelihood": 4, "impact": 4, "score": 16, "level": "critical", "status": "assessed" }
}
```

## 8. Status Codes

- `200` — Berhasil
- `201` — Resource dibuat
- `400` — Input tidak valid
- `401` — Perlu login / sesi tidak valid
- `403` — Login tapi tidak diizinkan
- `404` — Tidak ditemukan (atau tidak boleh diketahui)
- `409` — Konflik (versi, WIP limit, duplikat)
- `429` — Terlalu banyak request
- `500` — Error server tak terduga

## 9. Error Codes

`INVALID_INPUT`, `UNAUTHENTICATED`, `FORBIDDEN`, `NOT_FOUND`, `CONFLICT`, `VERSION_CONFLICT`, `WIP_LIMIT_EXCEEDED`, `ACCOUNT_LOCKED`, `PASSWORD_CHANGE_REQUIRED`, `RATE_LIMITED`, `INTERNAL`.

## 10. Error Rules

- Bentuk error selalu konsisten (lihat bagian 5).
- Jangan mengembalikan stack trace atau detail internal.
- Pesan harus bisa ditindaklanjuti user bila aman.
- Log server memuat `request_id` yang juga dikirim di header respons `X-Request-Id`.
- Jangan mencatat secret atau data sensitif.

## 11. Rate Limiting

- `/auth/login`: 10/menit per IP.
- Endpoint tulis: 120/menit per user.
- `/audit-logs/export`: 5/menit per user.
- Melebihi batas → `429` dengan header `Retry-After`.

## 12. Third-Party APIs

Tidak ada pada V1.

> Jika endpoint ditambah atau diubah, **perbarui dokumen ini dan test RBAC-nya** di commit yang sama.
