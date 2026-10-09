# SECURITY.md — PPO Kanban Board

Semua developer dan AI coding agent **wajib** mengikuti aturan ini.

> Konteks: proyek belajar bergaya perbankan. Dokumen ini adalah titik awal yang baik, **bukan** pengganti kebijakan keamanan dan kepatuhan bank yang sebenarnya jika suatu saat dipakai di lingkungan nyata.

## 1. Authentication

- Login email + password. Password di-hash dengan **argon2id**; jangan pernah menyimpan atau mencatat password asli.
- Kebijakan password: minimal 12 karakter, tidak boleh sama dengan email, tolak password umum.
- Lockout: 5 kali gagal berturut-turut → akun terkunci 15 menit. Hitungan direset saat login sukses.
- Pesan error login generik: "Email atau password salah."
- Password sementara dari Admin menandai `must_change_password = true`.
- Sesi: idle timeout 30 menit, durasi maksimal 8 jam. Logout dan penonaktifan user mencabut sesi.
- Cookie sesi: `HttpOnly`, `Secure` (production), `SameSite=Lax`. Token sesi disimpan ter-hash di database.
- Jangan simpan token/sesi di `localStorage` atau `sessionStorage`.
- MFA belum ada di V1 (rencana berikutnya).

## 2. Authorization (RBAC)

Aturan inti:

- **Default deny.** Endpoint tanpa aturan izin eksplisit ditolak.
- Role dan user ID diambil dari **sesi server**, tidak pernah dari body/query/header client.
- Selain role, periksa **kepemilikan/keanggotaan** (object-level): PM hanya proyek miliknya; Team Leader hanya proyek tempat dia member.
- Mengakses objek milik orang lain mengembalikan `404` (bukan `403`) untuk mencegah enumerasi, kecuali role memang boleh melihat tapi tidak boleh mengubah (maka `403`).
- Admin tidak otomatis boleh mengubah data proyek (pemisahan tugas).

### Matriks Izin (sumber kebenaran)

Legenda: `Y` = diizinkan, `R` = read-only, `Own` = hanya proyek yang dimiliki, `Asg` = hanya proyek tempat menjadi member, `-` = tidak boleh.

| Kemampuan | admin | project_manager | team_leader | audit | risk_management |
|---|---|---|---|---|---|
| Tambah user baru, kelola user, atur role, reset password | Y | - | - | - | - |
| Lihat daftar user ringkas (id, nama, role) | Y | Y | Asg | Y | Y |
| Buat proyek | - | Y | - | - | - |
| Ubah / arsipkan proyek | - | Own | - | - | - |
| Pindahkan owner proyek | Y | - | - | - | - |
| Lihat proyek | R (semua) | Own | Asg | R (semua) | R (semua) |
| Kelola member & kolom papan | - | Own | - | - | - |
| Buat / ubah task | - | Own | Asg | - | - |
| Pindah task (selain ke/dari Done) | - | Own | Asg | - | - |
| Pindah task ke/dari kolom Done | - | Own | - | - | - |
| Hapus task (soft delete) | - | Own | - | - | - |
| Komentar pada task | - | Own | Asg | - | - |
| Lihat task & komentar | R (semua) | Own | Asg | R (semua) | R (semua) |
| Ajukan risiko | - | Own | Asg | - | Y |
| Nilai, ubah level, tutup risiko | - | - | - | - | Y |
| Ubah rencana mitigasi | - | Own | - | - | Y |
| Lihat risiko | R (semua) | Own | Asg | R (semua) | Y (semua) |
| Buat / buka kembali / tutup temuan audit | - | - | - | Y | - |
| Respons temuan audit | - | Own | - | - | - |
| Lihat temuan audit | R (semua) | Own | Asg | Y (semua) | R (semua) |
| Lihat audit trail | R | - | - | Y | - |
| Ekspor audit trail (CSV) | - | - | - | Y | - |
| Dashboard | admin | PM | TL | audit | risk |

Catatan visibilitas Team Leader: `Asg` berarti Team Leader melihat **semua task di proyek tempat ia member** (termasuk task member lain), bukan hanya task miliknya. Ini berlaku juga untuk `GET /tasks` dan dashboard: query wajib dibatasi ke proyek yang diikuti pemanggil, dan dibatasi di sisi server.

Aturan alur tambahan (dipaksakan di service, bukan di UI):

- Task hanya boleh masuk/keluar kolom Done oleh PM owner (maker-checker).
- Hanya Audit yang menutup temuan; hanya Risk Management yang menutup risiko.
- Proyek terarsip read-only untuk semua role.

Perubahan pada matriks ini wajib disetujui eksplisit dan diikuti perubahan test RBAC.

## 3. Secrets & Environment Variables

- Semua secret lewat environment variable. Jangan hardcode.
- Jangan commit `.env`. Simpan `.env.example` berisi **nama variabel saja**.
- Kredensial development dan production harus berbeda.
- Jangan menaruh secret di dokumentasi, seed, log, atau kode frontend.

Variabel yang dipakai:

```
APP_ENV=
APP_PORT=
DATABASE_URL=
SESSION_SECRET=
COOKIE_SECURE=
BOOTSTRAP_ADMIN_EMAIL=
BOOTSTRAP_ADMIN_PASSWORD=
```

`BOOTSTRAP_ADMIN_*` hanya dipakai sekali untuk membuat admin pertama di development; admin tersebut wajib ganti password saat login pertama.

## 4. Input Validation

- Validasi semua input dari client **di server**, walaupun UI sudah memvalidasi.
- Tolak field tak dikenal (`DisallowUnknownFields`) dan tipe data salah.
- Batasi ukuran body request (mis. 1 MB) dan panjang string.
- Validasi enum (role, prioritas, status) terhadap daftar tetap.
- Untuk output di React, jangan memakai `dangerouslySetInnerHTML`. Teks komentar/deskripsi diperlakukan sebagai teks biasa.

## 5. API Security

- Semua endpoint kecuali `GET /healthz` dan `POST /api/v1/auth/login` butuh sesi valid.
- Request tulis (`POST/PATCH/PUT/DELETE`) wajib header `X-CSRF-Token` yang valid.
- Rate limiting: login 10 percobaan/menit per IP; endpoint tulis umum 120/menit per user; ekspor 5/menit per user.
- HTTPS wajib di production; aktifkan HSTS.
- Header keamanan: `Content-Security-Policy` ketat, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`, `X-Frame-Options: DENY`.
- Tidak ada CORS terbuka. Frontend dan API satu origin.
- Jangan mengembalikan stack trace atau pesan error internal ke client.

## 6. Data Protection

- Simpan hanya data yang dibutuhkan (nama, email kerja, role).
- Query hanya lewat sqlc/parameter. **Dilarang** menyusun SQL dengan string concatenation.
- User database aplikasi dibatasi hak aksesnya; tidak memakai superuser.
- Backup database terjadwal di production.
- Jangan menaruh data nyata di environment development.

## 7. Audit Trail (kontrol kunci)

- Semua aksi tulis wajib menghasilkan baris `audit_logs` dalam **transaksi yang sama**.
- Isi minimum: waktu, actor, role actor, aksi, tipe & id entitas, sebelum/sesudah (tanpa field sensitif), IP, `request_id`.
- Tabel `audit_logs` dilindungi trigger agar `UPDATE`/`DELETE` ditolak. Jangan membuat jalur yang menghindarinya.
- Jangan menuliskan password, hash, token, atau secret ke audit log maupun log aplikasi.
- Login sukses, login gagal, lockout, logout, dan perubahan role dicatat.

## 8. Error Handling & Logging

Pesan error ke user tidak boleh memuat: kredensial database, secret, stack trace, path file internal, atau informasi akun sensitif.

Log server memuat konteks cukup untuk debug (`request_id`, user id, rute, status) tanpa data sensitif.

## 9. Dependency & Infrastructure

- Jalankan `govulncheck ./...` dan `npm audit` sebelum rilis.
- Perbarui dependency secara berkala; kunci versi lewat `go.sum` dan `package-lock.json`.
- Image Docker memakai base image resmi dan berjalan sebagai non-root.

## 10. AI Agent Rules

AI coding agent **tidak boleh**:

- Menemukan atau menaruh kredensial production, atau mengarang secret.
- Menonaktifkan autentikasi atau melewati RBAC demi membuat fitur berjalan.
- Mengubah matriks izin, mekanisme sesi, hashing, atau trigger audit tanpa persetujuan.
- Membuat endpoint tanpa pemeriksaan autentikasi dan otorisasi.
- Mengekspos secret ke kode client.
- Membuat jalur `UPDATE`/`DELETE` pada `audit_logs`.

Jika permintaan bertentangan dengan dokumen ini, **hentikan dan tanyakan**.
