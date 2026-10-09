# Git Workflow Rules

## 1. Branching
- Jangan pernah commit atau push langsung ke `main`.
- Satu fitur/task = satu branch: `feat/<nama-singkat>`, `fix/<nama-singkat>`, `docs/<nama-singkat>`.
- Buat branch dengan: `git checkout -b <nama-branch>`.
- Merge ke `main` dilakukan manual oleh user, bukan agent.

## 2. Commit Kecil dan Logis
- Satu commit = satu perubahan logis. Jangan gabungkan perubahan yang tidak berkaitan.
- Commit setiap selesai satu sub-task, bukan menumpuk di akhir.

## 3. Review Sebelum Commit (wajib)
Sebelum setiap commit:
1. `git status` untuk melihat file yang berubah.
2. `git add <file-spesifik>` (hindari `git add -A` dan `git commit -a`).
3. `git diff --cached` untuk review final.
4. Pastikan tidak ada file rahasia (`.env`, key, token, credential).
5. `git commit -m "<pesan>"`.

## 4. Format Pesan Commit
Format:

    <type>(<scope opsional>): <subject>

    <body opsional>

    <footer opsional>

Type yang diizinkan:
- `feat`: fitur baru
- `fix`: perbaikan bug
- `refactor`: ubah struktur kode tanpa ubah perilaku
- `perf`: peningkatan performa
- `style`: format/whitespace, tanpa ubah makna kode
- `test`: tambah/perbaiki test
- `docs`: perubahan dokumentasi saja
- `build`: build tool, CI, dependency, versi
- `ops`: infrastruktur, deployment, backup
- `chore`: lain-lain (misal `.gitignore`)

Aturan subject:
- Bahasa Inggris, imperative present tense ("add", bukan "added").
- Huruf pertama kecil.
- Tanpa titik di akhir.
- Singkat dan jelas.
- Jangan pakai nomor issue sebagai scope.
- Jika belum selesai, tambahkan `(WIP)` di akhir subject.

Contoh:
- `feat(kanban): add drag and drop for task cards`
- `fix: add missing parameter to service call`
- `docs: update PRD with phase 0 scope`
- `chore: add node_modules to gitignore`

## 5. Push
- Setelah commit, jalankan: `git push origin <nama-branch>` (gunakan `-u` untuk push pertama).
- Push dilakukan setiap selesai satu task.

## 6. Larangan Keras
- Jangan `git push --force` atau `--force-with-lease`.
- Jangan `git rebase` pada commit yang sudah di-push.
- Jangan `git reset --hard` tanpa persetujuan user.
- Jangan hapus branch remote yang belum di-merge.
- Jangan commit file `.env`, credential, atau data sensitif.

## 7. Laporan
Setelah selesai, laporkan: nama branch, daftar commit (hash + pesan), dan status push.
