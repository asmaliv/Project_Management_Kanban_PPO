# App Flow — PPO Kanban Board

> Peta perjalanan user. Izin rinci per aksi: SECURITY.md. Endpoint: API.md.

## 1. Entry Points

- Halaman Login (`/login`) — satu-satunya halaman publik.
- Deep link ke halaman internal (mis. `/projects/:id/board`) → jika belum login, diarahkan ke Login lalu kembali ke tujuan semula.

## 2. Primary Journey (semua role)

```
Login
→ (jika must_change_password) Ganti Password
→ Dashboard sesuai role
→ menu sesuai role
```

## 3. Journey per Role

### 3.1 Admin
```
Dashboard
→ Pengguna → Buat Pengguna (email, nama, role, password sementara)
→ Edit role / Nonaktifkan / Aktifkan / Reset Password
→ Proyek (read-only) → Pindahkan Owner (jika diperlukan)
→ Audit Trail (read-only)
```

### 3.2 Project Manager
```
Dashboard
→ Proyek → Buat Proyek
→ Atur Papan (kolom, WIP limit) → Tambah Member (Team Leader)
→ Papan Kanban → Buat Task → Assign
→ Task di "In Review" → Tinjau → Setujui (pindah ke Done) / Kembalikan ke kolom sebelumnya + komentar
→ Temuan Audit proyek → Beri Respons
→ Risiko proyek → Ubah Rencana Mitigasi
→ Arsipkan Proyek (saat selesai)
```

### 3.3 Team Leader
```
Dashboard (Ringkasan Tim + tab Tugas Saya)
→ Tugas Tim (semua task di proyek yang ditugaskan; filter member, status, terlambat)
→ Proyek yang ditugaskan → Papan Kanban
→ Buat / Edit Task → Update progres
→ Seret task ke "In Progress" / "In Review" (bukan Done)
→ Komentar
→ Ajukan Risiko (terkait task/proyek)
```

### 3.4 Audit
```
Dashboard
→ Proyek (read-only) → Papan / Task / Komentar
→ Buat Temuan (proyek/task, severity)
→ Tinjau Respons PM → Tutup Temuan / Buka Kembali (dengan alasan)
→ Audit Trail → Filter → Ekspor CSV
```

### 3.5 Risk Management
```
Dashboard (heatmap)
→ Risk Register → Tinjau risiko baru (status identified)
→ Nilai (likelihood, impact) → skor & level otomatis
→ Isi / tinjau rencana mitigasi → ubah status
→ Tutup Risiko (High/Critical wajib punya mitigasi)
```

## 4. Screen Details

### Login
Input: Email, Password. Aksi: Masuk.
Error: "Email atau password salah." (generik), akun terkunci ("Akun terkunci sementara, coba lagi nanti"), terlalu banyak percobaan.
Tidak ada "Lupa password" mandiri di V1: tampilkan teks "Hubungi Administrator untuk reset password."

### Ganti Password
Input: Password lama, Password baru, Konfirmasi. Aturan password ditampilkan. Setelah sukses → Dashboard.

### Dashboard
Isi per role (lihat PRD 5.7). Setiap kartu ringkasan dapat diklik menuju daftar terfilter. Empty state jika belum ada data.

**Dashboard Team Leader** (dua tab):
- *Ringkasan Tim:* kartu angka (total task, terlambat, menunggu persetujuan PM), grafik jumlah task per kolom, beban kerja per member (jumlah task aktif), daftar task terlambat. Filter per proyek.
- *Tugas Saya:* task yang di-assign ke saya, urut due date.
- Klik angka/kartu → halaman **Tugas Tim** dengan filter sudah terisi.

### Tugas Tim (Team Leader)
Tabel semua task di proyek tempat user menjadi member: judul, proyek, kolom/status, assignee, prioritas, due date. Filter proyek, assignee, status, "terlambat", "milik saya". Klik baris → drawer detail task (aksi sesuai izin). Empty state: "Belum ada task di proyek Anda."

### Daftar Proyek
Tabel/kartu: kode, nama, status, owner, tanggal, progres. Filter status. Tombol "Buat Proyek" hanya untuk PM. Empty state: "Belum ada proyek."

### Detail / Papan Kanban
- Header: nama proyek, status, member, tombol sesuai izin.
- Kolom berurutan dengan task. Klik kartu → drawer detail task.
- Aksi pindah lewat drag-and-drop **atau** menu "Pindahkan ke…".
- Proyek terarsip: banner "Proyek diarsipkan (read-only)" dan semua aksi tulis nonaktif.

### Drawer Detail Task
Field: judul, deskripsi, prioritas, assignee, due date, progres, label, risiko terkait, komentar, riwayat aktivitas ringkas.
Aksi sesuai izin: simpan, hapus (konfirmasi), komentar.

### Pengguna (Admin)
Tabel user: nama, email, role, status. Tombol **Tambah Pengguna** membuka form: email, nama lengkap, role (5 pilihan), password sementara (boleh digenerate otomatis oleh sistem). Setelah disimpan, password sementara ditampilkan **sekali** untuk disampaikan Admin ke user; user wajib menggantinya saat login pertama. Form ubah memungkinkan perubahan nama dan role. Konfirmasi untuk nonaktifkan dan reset password (password sementara ditampilkan **sekali**).

### Risk Register
Tabel: kode, judul, proyek, kategori, skor, level, status, owner. Filter level/status/proyek. Tab Heatmap. Form ajukan risiko; form penilaian khusus Risk Management.

### Temuan Audit
Tabel: kode, judul, proyek, severity, status. Detail: deskripsi, respons PM, riwayat. Tombol aksi sesuai role dan status.

### Audit Trail
Tabel: waktu, actor, role, aksi, entitas, proyek, IP. Filter rentang waktu, actor, aksi, entitas. Detail baris menampilkan before/after. Tombol Ekspor CSV hanya untuk Audit.

## 5. Decision Points & Rules

- Pindah task ke/dari Done oleh non-PM → ditolak, kartu kembali, toast penjelasan.
- Pindah ke kolom penuh (WIP limit) → ditolak dengan pesan.
- Edit task yang sudah diubah user lain (konflik versi) → dialog "Muat ulang" dan perubahan lokal tidak ditimpa diam-diam.
- Tutup risiko High/Critical tanpa mitigasi → ditolak.
- Temuan hanya bisa ditutup Audit setelah berstatus `responded`.

## 6. Secondary Flows

### Reset Password oleh Admin
```
Pengguna → pilih user → Reset Password → konfirmasi
→ password sementara tampil sekali → user login → wajib ganti password
```

### Sesi Berakhir
```
Idle 30 menit / 8 jam → request berikutnya 401
→ UI menampilkan "Sesi berakhir" → Login → kembali ke halaman tujuan
```

### Pemindahan Owner Proyek
```
Admin → Proyek → Pindahkan Owner → pilih PM → konfirmasi → tercatat di audit trail
```

### Logout
```
Menu profil → Keluar → sesi dicabut → Login
```

## 7. Important States

Loading, Empty, Validation error, Network error, Forbidden (403), Not found (404), Version conflict (409), Rate limited (429), Session expired, Saved, Archived (read-only).

## 8. Flow Checklist untuk AI

Untuk setiap layar baru, tentukan: tujuan, konten wajib, aksi utama, aksi sekunder, state loading, empty, error, dan siapa yang boleh mengakses. Tandai langkah yang hilang sebelum membuat kode.
