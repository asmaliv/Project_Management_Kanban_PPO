# Design System — PPO Kanban Board

## 1. Direction

Profesional, bersih, tenang, dan dapat dipercaya — cocok untuk lingkungan perbankan. Padat informasi tetapi tidak ramai. Warna dipakai untuk makna (status, prioritas, risiko), bukan dekorasi. Bahasa antarmuka: Indonesia.

## 2. Color Tokens

Definisikan sebagai CSS variables / Tailwind theme.

```
--background:      #F8FAFC
--surface:         #FFFFFF
--surface-muted:   #F1F5F9
--border:          #E2E8F0
--text:            #0F172A
--text-muted:      #64748B
--primary:         #2563EB
--primary-hover:   #1D4ED8
--primary-soft:    #DBEAFE
--success:         #16A34A
--warning:         #D97706
--danger:          #DC2626
--info:            #0891B2
--focus-ring:      #2563EB
```

Dark mode: tidak termasuk V1.

### Warna semantik

Prioritas task: `low` abu, `medium` biru, `high` oranye (`#EA580C`), `critical` merah.

Level risiko: `low` hijau, `medium` kuning/amber, `high` oranye, `critical` merah.

Status temuan: `open` merah, `responded` amber, `closed` hijau.

**Aturan:** warna tidak boleh menjadi satu-satunya pembawa makna. Selalu sertakan label teks dan/atau ikon.

## 3. Typography

Font: **Inter** (fallback: system-ui, sans-serif).

| Gaya | Ukuran / Line height | Berat |
|---|---|---|
| H1 | 28px / 36px | Bold |
| H2 | 22px / 30px | Semibold |
| H3 | 18px / 26px | Semibold |
| Body | 14px / 20px | Regular |
| Body besar | 16px / 24px | Regular |
| Caption | 12px / 16px | Medium |

Angka pada tabel dan skor memakai `tabular-nums`.

## 4. Spacing

Skala 4px: `4, 8, 12, 16, 24, 32, 48`. Padding kartu 12–16px. Jarak antar kolom papan 16px.

## 5. Radius & Shadows

- Radius: input/tombol 6px, kartu/modal 8px, badge penuh (pill).
- Shadow: kartu `0 1px 2px rgba(15,23,42,.06)`; modal/dropdown `0 8px 24px rgba(15,23,42,.12)`; kartu saat di-drag sedikit lebih tinggi.
- Border 1px `--border` lebih diutamakan daripada shadow tebal.

## 6. Layout

- **Sidebar** kiri (navigasi sesuai role) + **topbar** (nama proyek/halaman, profil, logout).
- Menu hanya menampilkan item yang diizinkan untuk role (hanya untuk UX, bukan keamanan).
- Konten maksimal lebar 1440px kecuali papan Kanban yang boleh melebar dan scroll horizontal.

Menu per role:

| Role | Menu |
|---|---|
| admin | Dashboard, Pengguna, Proyek (read-only), Audit Trail |
| project_manager | Dashboard, Proyek, Risiko (proyek saya), Temuan |
| team_leader | Dashboard (Tim), Tugas Tim, Tugas Saya, Proyek, Risiko (ajukan) |
| audit | Dashboard, Proyek (read-only), Temuan, Audit Trail |
| risk_management | Dashboard, Risk Register, Proyek (read-only) |

## 7. Components

Reuse komponen yang ada sebelum membuat baru.

- **Button:** primary, secondary, ghost, destructive. Ukuran sm/md. State: hover, focus, active, disabled, loading.
- **Input / Select / Textarea:** label di atas, teks bantuan, pesan error di bawah, state focus/disabled/error.
- **Badge:** prioritas, level risiko, status, role. Bentuk pill, teks + warna.
- **Avatar:** inisial nama, ukuran 24/32.
- **Board Column:** judul, jumlah task, indikator WIP limit (mis. `3/5`), tombol tambah task (hanya jika diizinkan), area drop.
- **Task Card:** judul (maks 2 baris), badge prioritas, avatar assignee, due date (merah + ikon jika lewat tenggat), indikator risiko terkait, progres tipis.
- **Modal / Drawer:** detail task di drawer kanan; konfirmasi aksi destruktif di modal kecil.
- **Table:** header lengket, baris hover, paginasi, filter di atas (Audit Trail, Risiko, Temuan).
- **Risk Heatmap:** grid 5×5 likelihood × impact dengan jumlah risiko per sel dan label teks.
- **Stat Card:** angka besar (tabular-nums) + label + ikon; dapat diklik menuju daftar terfilter. Dipakai di semua dashboard.
- **Workload Bar:** batang horizontal jumlah task aktif per member, selalu disertai angka teks (bukan warna saja).
- **Chart (task per kolom):** batang sederhana dengan label dan angka; sediakan tabel alternatif untuk pembaca layar.
- **Toast:** sukses/error, otomatis hilang 5 detik, dapat ditutup.
- **Empty state:** ikon sederhana + satu kalimat + aksi utama (jika diizinkan).
- **Confirm dialog:** wajib untuk hapus task, arsip proyek, nonaktifkan user, tutup temuan/risiko.

## 8. States (wajib di setiap layar)

Hover, focus (cincin 2px `--focus-ring`), active, disabled, **loading** (skeleton untuk papan/tabel, spinner pada tombol), **empty**, **error** (pesan + tombol coba lagi), **forbidden** (halaman "Anda tidak memiliki akses").

## 9. Drag-and-Drop

- Kartu yang sedang di-drag: shadow lebih besar, sedikit transparan; area drop disorot garis putus-putus.
- Penolakan server (mis. maker-checker, WIP limit): kartu kembali ke posisi semula + toast error yang menjelaskan alasan.
- **Wajib ada alternatif non-drag:** menu "Pindahkan ke…" di kartu dan dapat dioperasikan dengan keyboard.
- Kolom Done menampilkan ikon gembok untuk role yang tidak boleh memindah ke sana.

## 10. Responsive Rules

- Mobile: `< 640px` — sidebar menjadi menu hamburger; papan scroll horizontal dengan snap per kolom; tabel menjadi daftar kartu atau scroll horizontal.
- Tablet: `640–1024px` — sidebar dapat diciutkan.
- Desktop: `> 1024px` — sidebar penuh.
- Tidak boleh ada overflow horizontal pada halaman selain area papan dan tabel yang disengaja.
- Target sentuh minimal 40×40px.

## 11. Accessibility

- Kontras teks minimal WCAG AA (4.5:1 untuk teks biasa).
- Semua kontrol dapat dijangkau dan dioperasikan lewat keyboard; fokus selalu terlihat.
- Gunakan elemen semantik (`button`, `nav`, `main`, `table`, heading berurutan).
- Input punya `label`; error dihubungkan dengan `aria-describedby`.
- Gambar/ikon bermakna punya teks alternatif; ikon dekoratif `aria-hidden`.
- Perubahan dinamis (toast, hasil drag) diumumkan lewat `aria-live`.
- Hormati `prefers-reduced-motion`.
