# Testing Guide — PPO Kanban Board

## 1. Testing Goal

Memastikan perjalanan utama berfungsi, gagal dengan aman, **tidak ada kebocoran akses antar role**, dan audit trail tidak dapat dimanipulasi.

## 2. Jenis Pengecekan

| Jenis | Arti |
|---|---|
| Happy path | Fitur berjalan saat semua input benar |
| Validation | Input kosong/salah ditolak dengan pesan jelas |
| Error handling | Kegagalan server/jaringan ditangani |
| Permissions | Role/pengguna yang salah **tidak** bisa mengakses |
| Responsive | Berfungsi di layar kecil dan besar |
| Accessibility | Bisa dipakai dengan keyboard dan pembaca layar |
| Regression | Perubahan baru tidak merusak fitur lama |

## 3. Perintah

```bash
# backend (dari /backend)
go vet ./...
golangci-lint run
go test ./... -race
govulncheck ./...

# frontend (dari /frontend)
npm run lint
npm run typecheck
npm run test
npm audit
```

Pisahkan **pengecekan otomatis** dari **pengecekan manual**. Jangan menandai apa pun "lulus" jika tidak benar-benar dijalankan.

## 4. Critical User Journey

```
Admin buat user PM dan Team Leader
→ PM login, ganti password, buat proyek
→ PM tambah Team Leader sebagai member
→ Team Leader buat task, pindah ke In Progress → In Review
→ PM menyetujui (pindah ke Done)
→ Team Leader ajukan risiko → Risk Management menilai
→ Audit buat temuan → PM merespons → Audit menutup
→ Audit melihat seluruh jejak di Audit Trail dan mengekspor CSV
```

Rilis **tidak boleh** keluar jika perjalanan ini rusak.

## 5. Matriks RBAC (otomatis, wajib)

Untuk **setiap endpoint** di API.md, tulis test untuk 6 identitas:
`anonim`, `admin`, `project_manager`, `team_leader`, `audit`, `risk_management`.
Tambahkan variasi kepemilikan: PM pemilik vs PM lain; Team Leader member vs bukan member.

Hasil yang diharapkan harus cocok dengan matriks di SECURITY.md. Test dibuat table-driven dari satu tabel izin agar perubahan matriks langsung memunculkan test gagal.

- [ ] Anonim → `401` di semua endpoint terproteksi
- [ ] Role tanpa izin → `403` (atau `404` untuk objek yang disembunyikan)
- [ ] PM B tidak dapat membaca/mengubah proyek PM A
- [ ] Team Leader non-member tidak dapat membaca proyek/task/risiko
- [ ] Role dan user ID dari body/header client diabaikan

## 6. Authentication Tests

### Login
- [ ] Kredensial benar berhasil
- [ ] Kredensial salah → pesan generik
- [ ] Email tidak terdaftar → pesan sama dengan password salah
- [ ] 5 gagal berturut-turut → akun terkunci 15 menit
- [ ] User nonaktif tidak bisa login
- [ ] Rate limit login memicu `429`

### Sesi
- [ ] Idle 30 menit → sesi tidak berlaku
- [ ] Logout mencabut sesi (cookie lama ditolak)
- [ ] Menonaktifkan user memutus sesi aktifnya
- [ ] Request tulis tanpa/salah CSRF token ditolak
- [ ] `must_change_password` memblokir endpoint lain sampai password diganti
- [ ] Password tidak muncul di respons, log, maupun audit log

## 7. User Management Tests
- [ ] Admin dapat menambah user baru dengan setiap dari 5 role; role lain ditolak
- [ ] User baru dapat login dengan password sementara, lalu **wajib** ganti password
- [ ] Admin dapat mengubah/nonaktifkan user
- [ ] Email duplikat ditangani (`409`)
- [ ] Admin tidak dapat menonaktifkan/menurunkan role dirinya sendiri
- [ ] Reset password menampilkan password sementara hanya sekali

## 8. Project Tests
- [ ] PM membuat proyek, otomatis jadi owner, kolom default terbentuk
- [ ] Hanya `team_leader` yang bisa ditambah sebagai member
- [ ] Kolom Done tidak dapat dihapus; kolom berisi task tidak dapat dihapus
- [ ] Proyek terarsip menolak semua aksi tulis
- [ ] Transfer owner hanya oleh Admin dan tercatat

## 9. Kanban Tests
- [ ] Buat/ubah/hapus (soft) task
- [ ] Assignee harus member proyek
- [ ] Pindah task antar kolom dan urutan dalam kolom tersimpan setelah refresh
- [ ] Team Leader **tidak** bisa pindah ke/dari Done; PM bisa
- [ ] WIP limit menolak dengan `WIP_LIMIT_EXCEEDED`
- [ ] Dua user memindah task bersamaan → satu `409 VERSION_CONFLICT`, urutan tetap konsisten
- [ ] Edit dengan `version` lama ditolak
- [ ] Teks sangat panjang dan karakter khusus tidak merusak layout/keamanan
- [ ] Alternatif keyboard "Pindahkan ke…" berfungsi
- [ ] Team Leader melihat semua task di proyek tempat ia member (termasuk milik member lain) dan **tidak** melihat task proyek lain
- [ ] `GET /tasks` menghormati filter, paginasi, dan batas akses tiap role
- [ ] Dashboard Team Leader: angka total/terlambat/menunggu persetujuan/per kolom sama dengan isi papan; beban kerja per member benar
- [ ] Dashboard Team Leader tidak menghitung task dari proyek non-member

## 10. Risk Tests
- [ ] PM, Team Leader (member), Risk Management dapat mengajukan
- [ ] Hanya Risk Management yang dapat menilai/menutup
- [ ] Skor = likelihood × impact dan level sesuai ambang (1–4, 5–9, 10–15, 16–25)
- [ ] Client tidak dapat mengirim `score`/`level` sendiri
- [ ] Tutup High/Critical tanpa mitigasi ditolak
- [ ] Heatmap sesuai data

## 11. Audit Finding Tests
- [ ] Hanya Audit yang membuat, membuka kembali, dan menutup
- [ ] Hanya PM owner yang merespons
- [ ] Status tidak dapat dilompati (`open → closed` langsung ditolak)
- [ ] Temuan hanya terlihat oleh role/proyek yang berhak

## 12. Audit Trail Tests
- [ ] Setiap aksi tulis menghasilkan baris audit log
- [ ] `UPDATE`/`DELETE` langsung ke `audit_logs` gagal (trigger)
- [ ] Jika penulisan audit log gagal, perubahan data ikut dibatalkan
- [ ] Hanya Audit dan Admin dapat membaca; hanya Audit dapat ekspor
- [ ] Filter (waktu, actor, entitas, aksi) memberi hasil benar
- [ ] CSV dan log tidak memuat password, hash, atau token

## 13. Responsive Checks

Uji minimal: mobile kecil (320px), mobile besar, tablet, laptop/desktop.

- [ ] Tidak ada overflow horizontal di luar area papan/tabel
- [ ] Tombol tetap mudah disentuh
- [ ] Form tetap terbaca
- [ ] Navigasi tetap dapat dipakai
- [ ] Modal/drawer muat di layar

## 14. Accessibility Checks
- [ ] Semua input punya label
- [ ] Semua kontrol dapat dijangkau keyboard; fokus terlihat
- [ ] Error tidak disampaikan hanya lewat warna
- [ ] Urutan heading logis
- [ ] Kontras memenuhi WCAG AA
- [ ] Perubahan dinamis (toast, hasil pemindahan) diumumkan ke pembaca layar

## 15. Security / Privacy Checks
- [ ] Tidak ada secret di kode klien atau repository
- [ ] Autentikasi dan otorisasi diverifikasi di server untuk setiap aksi
- [ ] Input divalidasi di server; field tak dikenal ditolak
- [ ] Tidak ada stack trace di respons error
- [ ] Header keamanan terpasang (CSP, HSTS, nosniff, dll)
- [ ] `govulncheck` dan `npm audit` tidak menyisakan temuan kritis
- [ ] Tidak ada SQL hasil string concatenation (tinjau kode)

## 16. Release Blockers

Jangan rilis jika:
- Login/logout rusak
- Ada kebocoran data atau akses lintas role/proyek
- Team Leader dapat memindahkan task ke/dari Done
- Audit log bisa diubah/dihapus, atau aksi tulis tidak tercatat
- Skor/level risiko bisa dimanipulasi dari client
- Penyimpanan kehilangan perubahan user
- Secret terekspos
- Alur kritis tidak dapat dipakai di mobile

## 17. Format Hasil Uji

Untuk setiap kegagalan catat:
- Test:
- Expected:
- Actual:
- Device/browser:
- Steps to reproduce:
- Screenshot/log:
- Severity:
- Status:

## 18. Cara Menguji Jika Tidak Teknis

1. Pakai aplikasi persis seperti pengguna baru.
2. Lalu sengaja lakukan hal yang salah: kosongkan field, isi data aneh, tekan back, refresh, matikan jaringan, klik dua kali.
3. Gunakan **akun terpisah untuk tiap role** dan coba melakukan hal yang tidak boleh.
4. Uji di ponsel sungguhan dan browser desktop.
5. Tuliskan hasil yang diharapkan dan yang terjadi.
6. Jangan hanya tanya AI "apakah ini aman?" — minta ia menunjukkan apa yang diuji dan apa yang belum diverifikasi.
