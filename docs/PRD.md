# PRD — PPO Kanban Board

> Pemilik dokumen ini: **apa yang kita bangun**. Detail teknis ada di TRD.md dan ARCHITECTURE.md.

## 1. Product Overview

- **Nama:** PPO Kanban Board
- **Deskripsi satu baris:** Aplikasi web Kanban untuk mengelola alur kerja proyek, penugasan task, pelacakan progres, temuan audit, dan risiko dalam satu dashboard dengan akses berbasis role.
- **Visi:** Setiap pihak (pelaksana, manajer, audit, risiko) melihat kebenaran yang sama tentang status proyek, dengan jejak yang tidak bisa dihapus.
- **Catatan penting:** Ini proyek belajar. **Dilarang** memasukkan data nyata nasabah, karyawan, atau data internal bank.

## 2. Problem

- Status proyek tersebar di spreadsheet, chat, dan email sehingga sulit dipercaya.
- Tidak jelas siapa yang menyetujui sebuah task dinyatakan selesai.
- Risiko proyek dicatat terpisah dari pekerjaan yang terdampak.
- Auditor sulit menelusuri siapa mengubah apa dan kapan.

## 3. Goals

1. Satu papan Kanban per proyek dengan alur kerja yang jelas dan bisa dilacak.
2. Pemisahan tugas (segregation of duties): yang mengerjakan tidak sama dengan yang menyetujui selesai.
3. Risiko dan temuan audit terhubung langsung ke proyek/task terkait.
4. Setiap perubahan penting tercatat di audit trail yang tidak bisa diubah/dihapus.

## 4. Target Users (5 Role)

| Role | Identifier | Tujuan utama |
|---|---|---|
| Admin | `admin` | Mengelola akun user dan konfigurasi sistem. Tidak mengurus isi pekerjaan proyek. |
| Project Manager | `project_manager` | Membuat dan memiliki proyek, menyusun papan, menugaskan Team Leader, menyetujui task selesai, merespons temuan audit. |
| Team Leader | `team_leader` | Mengerjakan dan memperbarui task di proyek yang ditugaskan, mengajukan risiko. |
| Audit | `audit` | Memeriksa semua proyek secara read-only, membuat temuan audit, melihat dan mengekspor audit trail. |
| Risk Management | `risk_management` | Menilai, memantau, dan menutup risiko di semua proyek. |

Izin rinci per role: lihat matriks di **SECURITY.md → bagian RBAC** (satu-satunya sumber kebenaran).

## 5. Core Features (V1)

### 5.1 Autentikasi
- Login/logout dengan email + password.
- Penguncian akun setelah gagal login berulang.
- Wajib ganti password pada login pertama.

### 5.2 Manajemen User (Admin)
- **Admin menambah user baru** lewat form "Tambah Pengguna": email, nama lengkap, role (salah satu dari 5), dan password sementara. User baru wajib ganti password saat login pertama.
- Ubah data user, nonaktifkan, aktifkan, atur role, reset password.

### 5.3 Manajemen Proyek
- PM membuat proyek (kode unik, nama, tanggal, status) dan menjadi owner.
- PM menambah Team Leader sebagai member proyek.
- Admin dapat memindahkan kepemilikan proyek.

### 5.4 Papan Kanban
- Kolom default: Backlog, To Do, In Progress, In Review, Done (bisa diatur PM; Done wajib ada satu).
- Task: judul, deskripsi, prioritas, assignee, due date, progres %, label.
- Drag-and-drop antar kolom dan urutan dalam kolom (plus alternatif keyboard/menu).
- **Aturan maker-checker:** Team Leader tidak boleh memindahkan task ke/dari kolom Done. Hanya PM.
- Komentar pada task.

### 5.5 Risk Register
- Pengajuan risiko oleh PM, Team Leader, atau Risk Management.
- Penilaian oleh Risk Management: likelihood (1–5) × impact (1–5) = skor → level Low/Medium/High/Critical.
- Rencana mitigasi, status, pemilik risiko, tautan ke proyek/task.
- Ringkasan heatmap risiko.

### 5.6 Temuan Audit & Audit Trail
- Audit membuat temuan (severity, deskripsi) pada proyek/task.
- PM merespons; Audit menutup temuan.
- Audit trail otomatis untuk setiap aksi tulis; hanya bisa ditambah, tidak bisa diubah/dihapus.
- Audit dapat memfilter dan mengekspor audit trail (CSV).

### 5.7 Dashboard per Role
- Admin: jumlah user per role, akun terkunci, aktivitas terbaru.
- PM: progres proyek, task terlambat, task menunggu persetujuan, temuan terbuka.
- Team Leader (dashboard tim): ringkasan **seluruh task tim** di proyek tempat ia menjadi member, yaitu jumlah task per kolom/status, beban kerja per member, task terlambat, task yang sedang menunggu persetujuan PM, serta tab "Tugas Saya" (due soon, task dibuka ulang). Daftar lengkap seluruh task tim tersedia di halaman "Tugas Tim".
- Audit: temuan terbuka per severity, aktivitas terbaru.
- Risk Management: heatmap, risiko High/Critical, risiko tanpa mitigasi.

## 6. User Flows

Ringkasan alur utama (detail di APP_FLOW.md):

`Login → Dashboard sesuai role → Proyek → Papan Kanban → Task → (In Review) → Persetujuan PM → Done`

## 7. Requirements

- **Fungsional:** lihat TRD.md bagian 4.
- **UX:** loading, empty, error state di setiap layar; aksi destruktif wajib konfirmasi; bahasa antarmuka Indonesia.
- **Performa:** target di TRD.md bagian 5.
- **Platform:** web responsif (320px ke atas), browser modern (Chrome, Edge, Firefox, Safari versi terbaru).

## 8. Success Metrics

- 100% aksi tulis tercatat di audit trail.
- 0 kebocoran akses lintas role pada uji matriks izin.
- PM dapat membuat proyek, menambah member, dan membuat 10 task dalam kurang dari 5 menit pada uji coba pertama.
- Auditor dapat menemukan siapa mengubah sebuah task dalam kurang dari 1 menit.

## 9. Out of Scope (V1)

- Notifikasi email/push, SSO/LDAP, MFA (direncanakan setelah V1).
- Lampiran file, time tracking, Gantt chart.
- Kolaborasi real-time (WebSocket).
- Multi-tenant / multi-organisasi.
- Aplikasi mobile native.
- Integrasi dengan sistem bank apa pun.

## 10. Open Decisions (jawab sebelum Phase terkait)

1. Apakah perlu role **Team Member** di bawah Team Leader? (V1 diasumsikan tidak ada; assignee = member proyek berrole `team_leader`.)
2. Bolehkah satu proyek dimiliki lebih dari satu PM? (V1: satu owner.)
3. Apakah Audit boleh berkomentar di task, atau hanya lewat Temuan? (V1: hanya lewat Temuan.)
4. Metode reset password tanpa email: reset oleh Admin (V1). Cukupkah?
5. Hosting production: TBD.
6. **Diputuskan:** "tim" milik Team Leader = semua task di proyek tempat ia menjadi member (termasuk task milik member lain di proyek yang sama). Team Leader tidak melihat proyek di mana ia bukan member.
7. **Diputuskan:** kelima role memiliki dashboard sendiri. Project Manager melihat proyek miliknya; Audit dan Risk Management melihat seluruh proyek (read-only untuk task).

Jika dokumen bertentangan atau keputusan di atas belum dijawab, **AI agent harus bertanya** sebelum membuat asumsi besar.
