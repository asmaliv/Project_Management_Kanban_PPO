# Panduan End-to-End Vibe Coding — PPO Kanban Board

Untuk: Junior Programmer (project pertama)
Alat: Antigravity + GitHub
Stack: Go (backend), React + TypeScript (frontend), PostgreSQL

---

## 0. Baca ini dulu

Vibe coding artinya kamu **mengarahkan** AI menulis kode. Kamu tidak mengetik semuanya sendiri, tetapi kamu tetap yang bertanggung jawab atas hasilnya. Di dunia perbankan, kode yang salah bisa berarti data bocor atau uang salah. Karena itu pola kerjamu harus disiplin:

1. **Rencana dulu, kode kemudian.** Jangan pernah minta "buatkan seluruh aplikasi".
2. **Satu phase satu waktu.** Phase berikutnya baru dimulai setelah verifikasi phase sekarang lulus.
3. **Baca diff sebelum commit.** Kalau ada yang tidak kamu pahami, minta AI menjelaskan.
4. **Bukti, bukan klaim.** "Sudah aman" dari AI belum berarti aman. Jalankan dan lihat sendiri.
5. **Dokumen adalah sumber kebenaran.** Kalau keputusan berubah, ubah dokumennya juga.
6. **Jangan pernah memasukkan data nyata** (nasabah, karyawan, bank) ke aplikasi ini.

Tidak ada tenggat di panduan ini. Lebih baik lambat tapi paham.

---

## 1. Persiapan Alat (sekali saja)

Pasang dan cek versinya:

| Alat | Cek dengan | Catatan |
|---|---|---|
| Git | `git --version` | |
| Go (versi stabil terbaru) | `go version` | Proyek butuh minimal 1.24 |
| Node.js (LTS) + npm | `node -v` dan `npm -v` | |
| Docker Desktop | `docker --version` dan `docker compose version` | Untuk PostgreSQL lokal |
| Antigravity | login dengan akun Google | |
| Klien database (opsional) | DBeaver / TablePlus | Untuk melihat isi tabel |

Pasang alat bantu Go (jalankan di terminal, mana yang gagal bisa diminta AI untuk membantu atau lihat dokumentasi resminya):

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
```

`golangci-lint` dipasang mengikuti petunjuk resmi di situs golangci-lint (cari "golangci-lint install").

Pastikan `$(go env GOPATH)/bin` ada di `PATH` supaya perintah `migrate` dan `sqlc` dikenali.

> Antigravity bisa berubah cepat. Kalau menu atau mode yang disebut di panduan ini berbeda di versimu, ikuti dokumentasi resminya. Prinsip kerjanya tetap sama.

---

## 2. Menyiapkan Repository dan Dokumen

```bash
git clone https://github.com/<username>/<nama-repo>.git
cd <nama-repo>
```

1. Ekstrak `ppo-kanban-docs.zip` ke root repo. Hasil akhirnya:
   ```
   AGENTS.md
   README.md
   .gitignore
   .env.example
   .agents/skills/ppo-endpoint-checklist/SKILL.md   (skill siap pakai)
   docs/  (12 file .md, termasuk panduan ini)
   ```
2. Pastikan `.gitignore` ada dan berisi `.env`.
3. Commit:
   ```bash
   git add .
   git commit -m "docs: tambah dokumen perencanaan"
   git push
   ```
4. Buka folder repo di Antigravity (*File → Open Folder*).

---

## 3. Siklus Kerja (ulangi di setiap phase)

```
 ┌─────────┐   ┌────────┐   ┌───────────┐   ┌────────┐   ┌────────┐   ┌────────┐
 │ 1 PLAN  │→ │ 2 REVIEW│→ │ 3 BUILD   │→ │ 4 VERIFY│→ │ 5 REVIEW│→ │ 6 COMMIT│
 │ (rencana)│  │ rencana │   │ (bertahap)│   │ (bukti) │   │ keamanan│   │ + docs  │
 └─────────┘   └────────┘   └───────────┘   └────────┘   └────────┘   └────────┘
```

Tips penting:

- **Mulai chat/sesi baru untuk tiap phase.** Konteks yang panjang membuat AI mulai lupa. Karena dokumen ada di repo, AI cukup disuruh membaca ulang.
- Gunakan mode yang membuat **rencana dulu** bila Antigravity menyediakannya, dan baca rencananya sebelum menyetujui.
- Kalau Antigravity tidak otomatis membaca `AGENTS.md`, prompt di bawah sudah menyuruhnya membaca secara eksplisit.
- Kerjakan di **branch** per phase, bukan di `main`.

### 3.1 Prompt Master (salin, ganti `<N>` dan daftar dokumen)

**① PLAN**

```
Kita mengerjakan Phase <N> saja di docs/IMPLEMENTATION_PLAN.md.
Baca AGENTS.md dan dokumen berikut: <daftar dokumen relevan>.
Berikan rencana langkah demi langkah. Sebelum coding, tuliskan:
1) asumsi yang kamu buat,
2) daftar file yang akan dibuat atau diubah,
3) hal yang belum jelas atau berisiko.
Jangan menulis kode dulu. Jangan mengerjakan apa pun di luar phase ini.
```

**② BUILD** (setelah kamu membaca dan menyetujui rencana)

```
Rencana disetujui. Silakan implementasikan.
Kerjakan bertahap dan berhenti setelah tiap bagian utama supaya saya bisa
menjalankan dan memeriksanya. Ikuti docs/CODE_STYLE.md.
Jangan menambah dependency baru tanpa bertanya. Jangan ubah file di luar rencana.
```

**③ VERIFY**

```
Jalankan semua pengecekan "Verify" untuk Phase <N> di IMPLEMENTATION_PLAN.md
dan perintah di docs/TESTING.md bagian 3 yang relevan.
Pisahkan hasilnya menjadi:
(a) yang benar-benar kamu jalankan beserta hasil nyatanya,
(b) yang harus saya cek manual (beri langkah yang jelas),
(c) yang belum diverifikasi.
Jangan menandai sesuatu lulus jika tidak dijalankan.
```

**④ SECURITY REVIEW**

```
Review perubahan di phase ini seperti senior engineer bank yang skeptis.
Periksa: otorisasi di server, kebocoran data lintas role/proyek, input validation,
audit log pada setiap aksi tulis, secret atau data sensitif di log/respons,
dan SQL yang tidak terparameter. Daftarkan temuan dulu, jangan langsung memperbaiki.
```

**⑤ UPDATE DOCS**

```
Perbarui dokumen di docs/ yang berubah akibat phase ini (mis. API.md, DATABASE.md,
SECURITY.md). Tampilkan perubahannya sebelum disimpan.
```

### 3.2 Cara Membaca Diff (checklist 2 menit sebelum commit)

- [ ] Hanya file yang disebut di rencana yang berubah?
- [ ] Ada dependency baru yang tidak kamu setujui? (`go.mod` / `package.json`)
- [ ] Ada secret, password, atau token yang tertulis di kode?
- [ ] Setiap endpoint baru punya pemeriksaan login **dan** izin?
- [ ] Setiap aksi tulis menulis audit log (mulai Phase 3)?
- [ ] Ada `console.log`, kode yang dikomentari, atau TODO yang tertinggal?
- [ ] Kamu bisa menjelaskan fungsi tiap file baru dengan kata-katamu sendiri? Jika tidak, tanyakan: *"Jelaskan file X baris demi baris seolah ke junior programmer."*

---

## 4. Sesi 0 — Kickoff (sebelum Phase 0)

Tujuan: AI memahami proyek dan kamu menjawab keputusan yang masih terbuka.

```
Sebelum menulis kode, baca AGENTS.md dan semua file di docs/.
Ringkas pemahamanmu tentang aplikasi ini dengan kata-katamu sendiri.
Lalu daftarkan: kontradiksi antar dokumen, informasi yang hilang, dan asumsi yang kamu buat.
Jangan menulis kode dan jangan mengubah file apa pun.
```

Lalu buka `docs/PRD.md` bagian 10 dan jawab pertanyaan yang tersisa. Jawaban yang sudah diputuskan:

| Hal | Keputusan |
|---|---|
| Admin | Dapat **menambah user baru** untuk kelima role, mengubah, menonaktifkan, dan mereset password |
| Team Leader | Punya **dashboard tim** dan dapat melihat **seluruh task di proyek tempat ia member** |
| PM, Audit, Risk Management | Masing-masing punya dashboard sendiri |
| Reset password | Oleh Admin (belum ada email di V1) |

Yang masih perlu kamu putuskan bersama atasanmu: apakah perlu role *Team Member*, dan hosting production. Tulis jawabannya di PRD.md lalu commit.

---

## 5. Phase 0 — Project Setup

**Tujuan:** kerangka proyek berjalan di laptopmu, belum ada fitur.

**Kamu akan belajar:** struktur monorepo, Docker untuk database, file `.env`, migrasi.

**Dokumen untuk AI:** `ARCHITECTURE.md`, `TRD.md`, `CODE_STYLE.md`, `AGENTS.md`.

```bash
git checkout -b feat/phase-0-setup
```

Pakai prompt ① PLAN. Tambahan di akhir prompt:

```
Buat hanya kerangka: struktur folder sesuai ARCHITECTURE.md, server Go dengan GET /healthz,
docker-compose untuk PostgreSQL, loader konfigurasi dari environment variable,
setup migrate dan sqlc, frontend Vite + React + TypeScript + Tailwind, serta lint.
Jangan membuat fitur bisnis apa pun.
```

**Verifikasi manual:**

1. Salin `.env.example` menjadi `.env`, isi nilainya (ini hanya untuk laptopmu; **jangan di-commit**).
2. `docker compose up -d db`, lalu `docker compose ps` harus menunjukkan database *running*.
3. Dari `/backend`: `go run ./cmd/server`, lalu di terminal lain `curl http://localhost:8080/healthz` harus menjawab 200.
4. Dari `/frontend`: `npm install` lalu `npm run dev`, buka alamat yang tampil dan halaman muncul.
5. `git status` tidak boleh menampilkan `.env`.

**Selesai jika:** semua langkah di atas berhasil dari clone baru.

**Commit:** `chore: setup kerangka backend, frontend, dan database lokal`

**Red flag:** AI membuat tabel atau halaman fitur; `.env` ikut masuk Git; AI memilih library yang tidak ada di TRD.md tanpa bertanya.

---

## 6. Phase 1 — Autentikasi & Sesi

**Tujuan:** user bisa login, ganti password, dan logout dengan aman.

**Kamu akan belajar:** hashing password (argon2id), sesi dan cookie `HttpOnly`, CSRF, lockout, rate limit.

**Dokumen untuk AI:** `SECURITY.md` (bagian 1, 3, 5), `DATABASE.md` (users, sessions), `API.md` (Auth), `APP_FLOW.md` (Login, Ganti Password).

Tambahan di prompt ① PLAN:

```
Di development memakai http://localhost, jadi cookie Secure harus bisa dimatikan lewat
COOKIE_SECURE=false di .env. Bootstrap admin pertama dibuat dari BOOTSTRAP_ADMIN_EMAIL dan
BOOTSTRAP_ADMIN_PASSWORD, dan admin itu wajib ganti password saat login pertama.
```

**Verifikasi manual:**

1. Login dengan password salah → pesan harus **sama** untuk email yang tidak ada maupun password yang salah.
2. Salah 5 kali berturut-turut → akun terkunci.
3. Login benar → buka DevTools browser (*Application → Cookies*): cookie sesi bertanda **HttpOnly**. Pastikan tidak ada token di *Local Storage*.
4. Buka tab *Network*, lihat respons login: tidak boleh ada `password` atau `password_hash`.
5. Logout, lalu tekan tombol Back dan coba buka halaman dalam → harus kembali ke Login.
6. Lihat log server: tidak boleh ada password atau token.

**Selesai jika:** semua "Verify" Phase 1 di IMPLEMENTATION_PLAN.md lulus.

**Commit:** `feat: autentikasi, sesi cookie, lockout, dan ganti password`

**Red flag:** token di `localStorage`; pesan error berbeda untuk "email tidak ada" vs "password salah"; password terlihat di log.

---

## 7. Phase 2 — Manajemen User & RBAC (Admin menambah user)

**Tujuan:** Admin dapat **menambah user baru** untuk kelima role dan mengelolanya. Role lain ditolak server.

**Kamu akan belajar:** RBAC (siapa boleh apa), middleware otorisasi, mengapa menyembunyikan menu bukan keamanan.

**Dokumen untuk AI:** `SECURITY.md` (bagian 2, matriks izin), `API.md` (Users), `APP_FLOW.md` (Pengguna), `TESTING.md` (bagian 5 dan 7).

Tambahan di prompt ① PLAN:

```
Implementasikan matriks izin di SECURITY.md sebagai satu tabel permission yang dipakai
middleware dan test. Buat form "Tambah Pengguna" (email, nama lengkap, role, password sementara
yang bisa digenerate otomatis). Password sementara tampil sekali saja dan user wajib ganti saat
login pertama. Buat helper test RBAC yang menguji 6 identitas: anonim dan kelima role.
```

**Verifikasi manual:**

1. Login sebagai Admin bootstrap, ganti password.
2. Lewat **Tambah Pengguna**, buat akun uji (email palsu, misalnya `pm1@example.test`):
   - 2 × Project Manager (`pm1`, `pm2`) — dua PM dibutuhkan untuk uji Phase 4
   - 2 × Team Leader (`tl1`, `tl2`)
   - 1 × Team Leader lagi (`tl3`), nanti sebagai non-member
   - 1 × Audit, 1 × Risk Management
3. Catat password sementara masing-masing di tempat pribadi (bukan di repo).
4. Login sebagai tiap user (pakai jendela *incognito* atau browser lain untuk dua role sekaligus). Pastikan masing-masing dipaksa ganti password.
5. Sebagai non-Admin, ketik langsung alamat halaman Pengguna di browser. Harus ditolak.
6. Coba nonaktifkan akun sendiri sebagai Admin → harus ditolak. Nonaktifkan `tl3`, coba login sebagai `tl3` → gagal, lalu aktifkan kembali.
7. Jalankan `go test ./...` dan pastikan test RBAC untuk endpoint users lulus.

**Commit:** `feat: manajemen user oleh admin dan penegakan RBAC`

**Red flag:** pengecekan role hanya di frontend; role diambil dari body request; password sementara tersimpan di log atau database dalam bentuk asli.

---

## 8. Phase 3 — Audit Trail (fondasi)

**Tujuan:** setiap aksi tulis tercatat dan jejaknya tidak bisa diubah atau dihapus.

**Kamu akan belajar:** audit trail, trigger database, transaksi.

**Dokumen untuk AI:** `DATABASE.md` (audit_logs + trigger), `SECURITY.md` (bagian 7), `API.md` (Audit Trail).

Tambahan di prompt ① PLAN:

```
Penulisan audit log harus berada dalam transaksi yang sama dengan perubahan datanya.
Catat juga login sukses, login gagal, lockout, logout, dan perubahan role.
Jangan mencatat password, hash, atau token.
```

**Verifikasi manual:**

1. Tambah user baru lewat UI → buka halaman Audit Trail (sebagai Audit atau Admin) → barisnya muncul dengan actor, waktu, dan aksi yang benar.
2. Buka klien database atau jalankan:
   ```bash
   docker compose exec db psql -U <user_db> -d <nama_db> -c "UPDATE audit_logs SET action='x';"
   docker compose exec db psql -U <user_db> -d <nama_db> -c "DELETE FROM audit_logs;"
   ```
   Keduanya **harus gagal** dengan pesan append-only.
3. Login sebagai Project Manager, coba buka Audit Trail → ditolak.
4. Minta AI menunjukkan test yang membuktikan bahwa jika penulisan audit log gagal, perubahan datanya ikut dibatalkan.

**Commit:** `feat: audit trail append-only dengan trigger database`

**Red flag:** audit log ditulis di luar transaksi; ada endpoint yang bisa mengubah atau menghapus audit log.

---

## 9. Phase 4 — Proyek, Member, dan Kolom

**Tujuan:** PM membuat proyek, menyusun kolom papan, dan menambah Team Leader sebagai member.

**Kamu akan belajar:** kepemilikan data (object-level authorization), constraint database, soft delete.

**Dokumen untuk AI:** `DATABASE.md`, `SECURITY.md` (matriks), `API.md` (Projects), `APP_FLOW.md` (Daftar Proyek).

**Verifikasi manual** (pakai akun dari Phase 2):

1. `pm1` membuat proyek → kolom default (Backlog, To Do, In Progress, In Review, Done) terbentuk.
2. `pm1` menambah `tl1` dan `tl2` sebagai member. Coba menambah akun Audit sebagai member → harus ditolak.
3. Login `pm2`: proyek `pm1` **tidak boleh** terlihat, dan membuka alamat langsungnya harus menghasilkan "tidak ditemukan".
4. Login `tl1`: proyek terlihat. Login `tl3`: proyek tidak terlihat.
5. Login Audit, Risk Management, Admin: proyek terlihat tetapi tombol ubah tidak ada dan API menolak jika dipaksa.
6. Coba hapus kolom Done → ditolak. Arsipkan proyek → semua aksi tulis ditolak.

**Commit:** `feat: proyek, member, dan kolom papan`

**Red flag:** PM lain bisa melihat proyek yang bukan miliknya; filter hak akses hanya di frontend.

---

## 10. Phase 5 — Task & Papan Kanban

**Tujuan:** tim bekerja di papan Kanban dengan aturan persetujuan PM (maker-checker).

**Kamu akan belajar:** drag-and-drop, optimistic locking, transaksi untuk urutan, aturan bisnis di service.

**Dokumen untuk AI:** `DATABASE.md` (tasks), `API.md` (Tasks, termasuk `GET /tasks`), `DESIGN_SYSTEM.md` (Board Column, Task Card, Drag-and-Drop), `SECURITY.md` (aturan alur).

Tambahan di prompt ① PLAN:

```
Aturan maker-checker: Team Leader tidak boleh memindahkan task ke atau dari kolom Done, hanya PM
pemilik proyek. Dua pemindahan bersamaan harus menghasilkan satu sukses dan satu 409 (version).
Sediakan alternatif non-drag berupa menu "Pindahkan ke..." yang bisa dipakai dengan keyboard.
Buat GET /tasks (lintas proyek, filter, paginasi) yang dibatasi ke proyek yang boleh dilihat pemanggil.
```

**Verifikasi manual:**

1. `pm1` dan `tl1` membuat beberapa task, di-assign ke `tl1` dan `tl2`.
2. `tl1` menyeret task ke In Progress dan In Review → berhasil.
3. `tl1` mencoba memindahkan ke Done → **ditolak**, kartu kembali ke tempat semula, ada pesan yang jelas. Lakukan dua cara: drag dan menu "Pindahkan ke…".
4. `pm1` memindahkan ke Done → berhasil.
5. Refresh halaman: urutan dan posisi tetap.
6. Buka papan yang sama di dua jendela, ubah task yang sama di kedua jendela → yang kedua mendapat pesan konflik, bukan menimpa diam-diam.
7. Isi judul sangat panjang dan karakter aneh (`<script>`, emoji) → layout tidak rusak dan tidak ada script yang jalan.
8. Atur WIP limit 2 pada satu kolom lalu isi lebih dari 2 → ditolak.
9. Coba operasikan "Pindahkan ke…" hanya dengan keyboard (Tab dan Enter).

**Commit:** `feat: task dan papan kanban dengan aturan maker-checker`

**Red flag:** pemindahan ke Done hanya dicegah di UI; urutan task rusak setelah refresh; aksi tulis tanpa audit log.

---

## 11. Phase 6 — Risk Management

**Tujuan:** risiko diajukan, dinilai, dipantau, dan terhubung ke proyek/task.

**Kamu akan belajar:** perhitungan di server, pemisahan peran (pengaju vs penilai), agregasi heatmap.

**Dokumen untuk AI:** `DATABASE.md` (risks), `TRD.md` (4.5), `API.md` (Risks), `DESIGN_SYSTEM.md` (Risk Heatmap, warna level).

**Verifikasi manual:**

1. `tl1` mengajukan risiko pada task → status *identified*.
2. `tl1` mencoba menilai atau menutup risiko → ditolak.
3. Risk Management menilai (misal likelihood 4 × impact 4) → skor 16 dan level **Critical** muncul otomatis.
4. Coba kirim `score` atau `level` palsu lewat DevTools → harus diabaikan atau ditolak.
5. Tutup risiko Critical tanpa rencana mitigasi → ditolak.
6. `tl3` (non-member) tidak melihat risiko proyek itu. Heatmap sesuai jumlah data.

**Commit:** `feat: risk register, penilaian, dan heatmap`

**Red flag:** skor dihitung di frontend; Team Leader bisa mengubah level risiko.

---

## 12. Phase 7 — Temuan Audit & Dashboard per Role

**Tujuan:** siklus temuan audit lengkap, ekspor audit trail, dan dashboard untuk kelima role, termasuk **dashboard tim untuk Team Leader**.

**Kamu akan belajar:** alur status (state machine), agregasi data, membatasi query per pengguna.

**Dokumen untuk AI:** `DATABASE.md` (audit_findings), `API.md` (Findings, Dashboard, `GET /tasks`), `APP_FLOW.md` (Dashboard Team Leader, Tugas Tim), `SECURITY.md` (catatan visibilitas Team Leader), `DESIGN_SYSTEM.md` (Stat Card, Workload Bar).

Tambahan di prompt ① PLAN:

```
Dashboard Team Leader harus menampilkan seluruh task di proyek tempat dia menjadi member
(termasuk task member lain): total, terlambat, menunggu persetujuan PM, jumlah per kolom,
dan beban kerja per member, plus tab "Tugas Saya". Sediakan halaman "Tugas Tim" dengan filter.
Seluruh agregasi dihitung di server dan dibatasi ke proyek yang diikuti pemanggil.
Angka dashboard harus sama dengan isi papan.
```

**Verifikasi manual — temuan audit:**

1. Audit membuat temuan di proyek `pm1`. `pm1` merespons. Audit menutup.
2. Coba tutup temuan yang masih *open* (belum direspons) → ditolak.
3. `pm2` tidak melihat temuan proyek `pm1`.

**Verifikasi manual — dashboard:**

1. `tl1` membuka Dashboard → Ringkasan Tim memuat task milik `tl1` **dan** `tl2` di proyek yang sama.
2. Bandingkan angkanya dengan papan: total, jumlah per kolom, task terlambat harus sama.
3. Buat satu task terlambat (due date kemarin) → muncul di daftar terlambat dan angkanya naik.
4. Klik angka "terlambat" → pindah ke Tugas Tim dengan filter terisi.
5. `tl3` (non-member) → dashboard kosong atau tidak memuat task proyek itu sama sekali.
6. Login tiap role lain dan pastikan dashboard-nya hanya berisi data yang boleh mereka lihat:
   - Admin: jumlah user per role, akun terkunci, aktivitas terbaru
   - PM: progres proyek miliknya, task terlambat, task menunggu persetujuan, temuan terbuka
   - Audit: temuan terbuka per severity, aktivitas terbaru
   - Risk Management: heatmap, risiko High/Critical, risiko tanpa mitigasi
7. Audit mengekspor CSV audit trail → buka file, pastikan tidak ada password, hash, atau token.

**Commit:** `feat: temuan audit, ekspor audit trail, dan dashboard per role`

**Red flag:** dashboard Team Leader menghitung task dari proyek yang bukan miliknya; filter hanya di frontend; jumlah di dashboard berbeda dengan papan.

---

## 13. Phase 8 — Production Readiness

**Tujuan:** MVP siap rilis dan terbukti lolos pengecekan.

**Dokumen untuk AI:** seluruh `docs/`, terutama `SECURITY.md` dan `TESTING.md`.

Langkah:

1. Pasang header keamanan dan rate limit menyeluruh (prompt ① PLAN dengan fokus bagian 5 SECURITY.md).
2. Jalankan:
   ```bash
   cd backend && go vet ./... && golangci-lint run && go test ./... -race && govulncheck ./...
   cd ../frontend && npm run lint && npm run typecheck && npm run test && npm audit
   ```
3. Buat Dockerfile non-root dan dokumentasi deployment serta backup/restore.
4. Kerjakan **seluruh `docs/TESTING.md`** dengan prompt ③ VERIFY. Uji juga di ponsel sungguhan.
5. Jalankan prompt ④ SECURITY REVIEW untuk seluruh proyek.
6. Pastikan tidak ada *Release Blocker* (TESTING.md bagian 16).

**Skrip demo akhir** (jika harus mendemokan ke atasan):

```
Admin menambah PM dan Team Leader → PM membuat proyek dan menambah Team Leader
→ Team Leader membuat task dan memindahkannya sampai In Review
→ Team Leader mencoba ke Done (ditolak) → PM menyetujui
→ Team Leader membuka dashboard tim → mengajukan risiko → Risk Management menilai
→ Audit membuat temuan → PM merespons → Audit menutup
→ Audit menunjukkan jejak lengkap di Audit Trail dan mengekspor CSV
```

**Commit:** `chore: hardening dan kesiapan production`

---

## 14. Alur Git per Phase

```bash
git checkout main && git pull
git checkout -b feat/phase-<N>-<nama-singkat>
# ... siklus PLAN → BUILD → VERIFY → REVIEW ...
git add <file-yang-dimaksud>        # hindari git add . bila kamu belum cek git status
git status                           # pastikan tidak ada .env atau file aneh
git commit -m "feat: <ringkasan>"
git push -u origin feat/phase-<N>-<nama-singkat>
```

Lalu buka **Pull Request** di GitHub, baca tab *Files changed* sendiri sekali lagi, baru merge ke `main`. Commit kecil dan sering lebih baik daripada satu commit raksasa.

---

## 15. Debugging dengan AI

Saat ada error, jangan hanya menulis "error, tolong perbaiki". Pakai format ini:

```
Masalah: <apa yang terjadi, satu kalimat>
Yang diharapkan: <apa yang seharusnya terjadi>
Langkah mengulang: <1, 2, 3>
Pesan error / log: <tempel, TANPA password/token/secret>
Yang baru saya ubah: <ringkas>
Jelaskan dulu penyebab yang paling mungkin dan buktikan sebelum mengubah kode.
```

Aturan:

- Jangan menempelkan secret, isi `.env`, atau data nyata ke AI.
- Jika AI mencoba "memperbaiki" dengan melemahkan keamanan (mematikan pengecekan, melonggarkan izin), **tolak**. Itu bukan perbaikan.
- Jika setelah 3 percobaan belum beres, mulai sesi baru, jelaskan masalah dari awal, atau tanya senior.

### Masalah umum

| Gejala | Kemungkinan penyebab |
|---|---|
| Login berhasil tapi langsung dianggap belum login | `COOKIE_SECURE=true` di http lokal, atau frontend dan API beda origin (cek proxy Vite) |
| Request tulis ditolak padahal sudah login | Header `X-CSRF-Token` tidak dikirim |
| `migrate` bilang *dirty* | Migrasi gagal separuh; minta AI memperbaiki dengan migrasi baru, jangan edit database manual |
| `port already in use` | Proses lama masih jalan; ganti port atau hentikan prosesnya |
| `sqlc generate` gagal | SQL query salah atau skema belum termigrasi |
| Database tidak bisa dihubungi | Container belum jalan (`docker compose ps`), atau `DATABASE_URL` salah |

---

## 16. Checklist Keamanan Harian

Sebelum setiap commit:

- [ ] Tidak ada secret di kode, dokumen, atau log
- [ ] Setiap endpoint baru punya cek login **dan** cek izin di server
- [ ] Input divalidasi di server
- [ ] Query tidak disusun dengan string concatenation
- [ ] Aksi tulis menghasilkan audit log
- [ ] Pesan error tidak membocorkan detail internal
- [ ] Test RBAC untuk endpoint yang disentuh lulus

---

## 17. Refleksi Singkat Tiap Phase (5 menit)

Tulis jawaban di catatan pribadimu (atau PR description):

1. Satu hal baru yang saya pelajari.
2. Satu bagian kode yang awalnya tidak saya pahami dan sekarang saya pahami.
3. Satu hal yang AI lakukan salah dan bagaimana saya menangkapnya.
4. Dokumen mana yang saya perbarui?

Kebiasaan ini yang membedakan "pengguna AI" dari engineer yang makin hari makin kuat.

---

## 18. Glosarium

| Istilah | Arti sederhana |
|---|---|
| RBAC | Hak akses berdasarkan role (siapa boleh melakukan apa) |
| Maker-checker | Yang mengerjakan tidak boleh sekaligus yang menyetujui |
| Audit trail | Catatan siapa melakukan apa dan kapan, tidak boleh diubah |
| Sesi / cookie HttpOnly | Tanda login yang disimpan browser dan tidak bisa dibaca JavaScript |
| CSRF | Serangan yang memanfaatkan sesi login korban; dicegah dengan token khusus |
| Hashing | Mengubah password jadi kode satu arah agar password asli tidak tersimpan |
| Migrasi | File SQL berurutan yang mengubah struktur database |
| sqlc | Alat yang mengubah file SQL jadi kode Go yang aman dari SQL injection |
| Optimistic locking | Mencegah dua orang saling menimpa perubahan lewat nomor versi |
| Soft delete | Data ditandai terhapus, tidak benar-benar dihilangkan |
| Maker-checker pada Done | Hanya PM yang boleh menyetujui task selesai |

---

## 19. Ingat

Dokumen dan AI adalah alat bantumu. Kamu tetap harus mereview kode, menjalankan aplikasi, dan memverifikasi hal-hal yang sensitif terhadap keamanan. Jika ragu pada keputusan keamanan atau akses data, **berhenti dan tanyakan ke senior** sebelum melanjutkan.
