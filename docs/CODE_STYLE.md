# CODE_STYLE.md — PPO Kanban Board

## 1. Purpose

Konvensi kode agar hasil buatan manusia maupun AI tetap mudah dibaca dan konsisten. Kode baru mengikuti pola yang sudah ada kecuali dokumen ini menyebut lain.

## 2. Stack

- Backend: Go, chi, pgx, sqlc, golang-migrate, `log/slog`
- Frontend: TypeScript (strict), React, Vite, Tailwind, TanStack Query, react-hook-form, zod, dnd-kit
- Tooling: `gofmt`/`goimports`, `golangci-lint`, ESLint, Prettier

## 3. Prinsip Umum

- Utamakan kode yang mudah dibaca daripada yang "pintar".
- Fungsi dan komponen kecil dan fokus.
- Reuse sebelum membuat baru.
- Hindari abstraksi yang belum dibutuhkan.
- Jangan menduplikasi logika yang sudah punya utilitas bersama.
- Aturan bisnis **hanya** di lapisan service (backend). UI tidak boleh menjadi satu-satunya penjaga aturan.

## 4. Go (Backend)

### Struktur & layer
- `handler` (HTTP, validasi input, mapping respons) → `service` (aturan bisnis, otorisasi, transaksi) → `store` (sqlc).
- Handler tidak mengakses database langsung. Store tidak berisi aturan bisnis.
- Dependency di-inject lewat konstruktor (`NewTaskService(store, audit)`), bukan variabel global.

### Penamaan
- Paket: huruf kecil, satu kata, tanpa underscore (`task`, `auditlog`).
- Eksport: `PascalCase`; internal: `camelCase`.
- Interface kecil, didefinisikan di sisi pemakai.
- File: `snake_case.go` (`task_service.go`, `task_handler.go`).
- Konstanta: `PascalCase` atau `camelCase`, bukan `UPPER_SNAKE`.

### Error
- Bungkus error dengan konteks: `fmt.Errorf("move task: %w", err)`.
- Definisikan error domain (`ErrForbidden`, `ErrNotFound`, `ErrVersionConflict`) dan petakan ke status HTTP di satu tempat.
- Jangan `panic` pada alur normal. Jangan mengabaikan error (`_ =`) tanpa alasan tertulis.
- Jangan membocorkan error internal ke client.

### Lain-lain
- `context.Context` selalu parameter pertama pada fungsi yang melakukan I/O.
- Transaksi dimulai di service; store menerima `Querier` agar bisa dipakai di dalam transaksi.
- Gunakan `slog` dengan field terstruktur; jangan log data sensitif.
- Query hanya lewat sqlc. Dilarang menyusun SQL dengan string concatenation.
- Test berbentuk table-driven; nama test menjelaskan perilaku (`TestMoveTask_TeamLeaderCannotMoveToDone`).

## 5. TypeScript / React (Frontend)

- Mode `strict`. Dilarang `any`; gunakan `unknown` lalu persempit.
- Komponen: fungsional, `PascalCase` (`TaskCard.tsx`). Hook: `useXxx`. Fungsi/variabel: `camelCase`. Konstanta: `UPPER_SNAKE_CASE`.
- Boolean dibaca jelas: `isLoading`, `hasAccess`, `canEdit`.
- Pisahkan UI, logika bisnis, dan pengambilan data. Logika yang dipakai ulang masuk custom hook/util.
- Pengambilan data API lewat TanStack Query di folder `features/<fitur>/api`.
- Selalu tangani state **loading, error, dan empty**.
- Props memiliki tipe eksplisit.
- Jangan menyembunyikan tombol sebagai satu-satunya pengaman izin; backend tetap menegakkan. UI memakai `canEdit`-style helper dari role hanya untuk UX.
- Jangan memakai `dangerouslySetInnerHTML`.

## 6. Struktur Frontend

```
features/<fitur>/
  api/          # hook TanStack Query
  components/   # komponen khusus fitur
  pages/        # halaman/route
  types.ts
```

Komponen yang dipakai lintas fitur masuk `src/components/`.

## 7. Formatting

- Go: `gofmt` + `goimports`; lint dengan `golangci-lint`.
- TS: Prettier + ESLint sesuai konfigurasi proyek.
- Jangan memformat ulang file yang tidak terkait perubahan.
- Rapikan import, hapus variabel dan import yang tidak terpakai.

## 8. Komentar

Tulis komentar untuk menjelaskan **MENGAPA**, bukan hal yang sudah jelas.

Bagus:
```go
// Task di kolom Done hanya boleh dipindah PM (maker-checker), lihat SECURITY.md.
```

Hindari:
```go
// tambah satu
count++
```

## 9. Commit

Format: `type: ringkasan singkat` dengan type `feat`, `fix`, `docs`, `refactor`, `test`, `chore`.
Contoh: `feat: tambah endpoint pindah task dengan aturan maker-checker`.

## 10. Sebelum Selesai

- Jalankan lint, typecheck, dan test relevan.
- Pastikan test RBAC untuk endpoint yang disentuh lulus.
- Periksa layout responsif jika UI berubah.
- Hapus log debug dan dead code.
- Perbarui dokumen terkait jika kontrak berubah.
