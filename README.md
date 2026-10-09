# PPO Kanban Board

Aplikasi web Project Management Kanban Board (Go + React) dengan role: Admin, Project Manager, Team Leader, Audit, dan Risk Management.

> Proyek belajar. Jangan memasukkan data nyata nasabah/karyawan/bank.

## Dokumen Perencanaan

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
| `docs/DESIGN_SYSTEM.md` | Seperti apa tampilannya? |
| `docs/APP_FLOW.md` | Bagaimana user berpindah antar layar? |
| `docs/IMPLEMENTATION_PLAN.md` | Dalam urutan apa kita membangun? |
| `docs/TESTING.md` | Bagaimana membuktikan semuanya benar? |
| `docs/GUIDE_VIBE_CODING.md` | Langkah demi langkah membangun semuanya (untuk junior) |

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

## Alur Kerja Git

```bash
git checkout -b feat/phase-0-setup
# ...kerjakan...
git add .
git commit -m "chore: setup struktur proyek"
git push -u origin feat/phase-0-setup
# buka Pull Request, baca diff, baru merge ke main
```
