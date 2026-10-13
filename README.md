# PPO Kanban Board

Aplikasi web Project Management Kanban Board (Go + React) dengan role: Admin, Project Manager, Team Leader, Audit, dan Risk Management.

> Proyek Internal PPO

## 📊 Status Implementasi

### ✅ Phase 0: Setup & Pondasi (Selesai)
- Database, chi backend, Vite frontend, CI checklist

### ✅ Phase 1: Autentikasi & Sesi (Selesai)  
- Argon2id, cookie HttpOnly, CSRF token, login lockout, ganti password

### ✅ Phase 2: Manajemen User & RBAC (Selesai)
- Kelola pengguna, matriks izin 5 role, form tambah user

### ✅ Phase 3: Audit Trail (Selesai) ✨
- Tabel audit_logs dengan trigger append-only
- Service audit log dalam transaksi yang sama dengan perubahan data
- Catat login sukses/gagal, lockout, logout, perubahan role
- GET /audit-logs dengan filter (Audit & Admin only)
- **Tidak pernah mencatat password, hash, atau token**

### 🔄 Next: Phase 4 - Proyek, Member, dan Kolom
- CRUD proyek (PM), arsip, transfer owner (Admin)
- Kelola member dan kolom papan
- Audit log untuk semua aksi

---

## 🛠️ Tech Stack

- **Backend:** Go 1.24+ · chi router · PostgreSQL 16+ · sqlc · golang-migrate
- **Frontend:** React 18 · TypeScript · Vite · Tailwind CSS · TanStack Query
- **Security:** Argon2id · HttpOnly cookies · CSRF protection · Rate limiting
- **Audit:** Append-only audit_logs dengan trigger database

## 🤖 AI Skills Active

- **Ponytail** - Simplest working code, -53% code, -41% time
- **Graphify** - Knowledge graph, 390 nodes, 986 edges, 14 communities

## 📚 Dokumen Perencanaan

| File | Pertanyaan yang dijawab |
|---|---|
| `AGENTS.md` (root) | Bagaimana AI harus bekerja? |
| `docs/PRD.md` | Apa yang kita bangun? |
| `docs/TRD.md` | Bagaimana ini bekerja secara teknis? |
| `docs/ARCHITECTURE.md` | Bagaimana bagian-bagiannya terhubung? |
| `docs/SECURITY.md` | Bagaimana melindungi data? Siapa boleh apa? (matriks RBAC) |
| `docs/DATABASE.md` | Bagaimana data dimodelkan? |
| `docs/API.md` | Bagaimana frontend dan backend berkomunikasi? |
| `docs/CODE_STYLE.md` | Seperti apa kode di repo ini? |
| `docs/APP_FLOW.md` | Bagaimana user berpindah antar layar? |
| `docs/IMPLEMENTATION_PLAN.md` | Dalam urutan apa kita membangun? |
| `docs/TESTING.md` | Bagaimana membuktikan semuanya benar? |

## Cara Memulai dengan AI Agent

1. Salin `AGENTS.md`, `README.md`, `.gitignore`, `.env.example`, dan folder `.agents/` (berisi skill) ke root repo; salin sisanya ke folder `docs/`.
2. Commit: `docs: tambah dokumen perencanaan`.
3. Buka folder repo di Antigravity.
4. Tempel prompt pembuka di bawah.

### Prompt Pembuka

```
Sebelum menulis kode, baca AGENTS.md dan semua file di docs/.
Ringkas pemahamanmu, lalu daftarkan: kontradiksi, informasi yang hilang,
dan asumsi yang kamu buat. Jangan menulis kode dan jangan mengubah file apa pun.
Tanyakan Open Decisions di PRD.md bagian 10 sebelum kita mulai.
```

### Prompt Tiap Phase

```
Kita mengerjakan Phase <N> saja di docs/IMPLEMENTATION_PLAN.md.
Baca dokumen yang relevan, lalu berikan rencana langkah demi langkah.
Jangan mengerjakan apa pun di luar phase ini.
Sebelum coding, tuliskan asumsi dan daftar file yang akan dibuat atau diubah.
Setelah selesai, jalankan pengecekan Verify dan laporkan mana yang benar-benar
dijalankan dan mana yang belum diverifikasi.
```

## 🚀 Cara Menjalankan Lokal

### 1. Database (PostgreSQL 16)
```bash
docker compose up -d db
```

### 2. Backend (Go API Server)
```bash
cd backend
go run ./cmd/server
```
API berjalan di `http://localhost:8080`. Endpoint health check: `http://localhost:8080/healthz`.

### 3. Frontend (React + Vite SPA)
```bash
cd frontend
npm install
npm run dev
```
Aplikasi web berjalan di `http://localhost:5173`. Request `/healthz` dan `/api` otomatis diproxy ke backend.

## 🔍 Explore Architecture

Gunakan Graphify untuk memahami codebase:
```bash
# Query graph
python -m graphify query "how does authentication work?"

# Find connections
python -m graphify path "Service" "AuditLog"

# Explain component
python -m graphify explain "SessionManager"

# Open interactive visualization
start graphify-out/graph.html
```

## Alur Kerja Git

```bash
git checkout -b feat/phase-0-setup
# ...kerjakan...
git add .
git commit -m "chore: setup struktur proyek"
git push -u origin feat/phase-0-setup
# buka Pull Request, baca diff, baru merge ke main
```

