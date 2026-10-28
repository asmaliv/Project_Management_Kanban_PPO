# Prosedur Backup & Restore — PPO Kanban Board

Dokumen ini mendefinisikan Standard Operating Procedure (SOP) untuk pencadangan (backup) dan pemulihan (restore) database PostgreSQL pada aplikasi **PPO Kanban Board**.

---

## 1. Sasaran Pemulihan (RPO & RTO)

| Metrik | Sasaran | Definisi |
|---|---|---|
| **Recovery Point Objective (RPO)** | **Maksimal 6 Jam** | Batas toleransi kehilangan data (backup snapshot dilakukan setiap 6 jam). |
| **Recovery Time Objective (RTO)** | **Maksimal 30 Menit** | Batas waktu sistem harus berhasil dipulihkan setelah kegagalan bencana. |
| **Retensi Data** | **30 Hari Lokal, 1 Tahun Offsite** | Backup lokal disimpan 30 hari; arsip bulanan dikirim ke cold storage terenkripsi. |

---

## 2. Kebijakan Integritas Audit Trail

Tabel `audit_logs` dilindungi trigger basis data yang menolak seluruh operasi `UPDATE` dan `DELETE`.
Pada saat proses restore dilakukan:
* Trigger akan secara otomatis tetap aktif dan melindungi data audit.
* Seluruh jejak audit riwayat masa lalu tidak boleh dimodifikasi selama migrasi atau pemulihan.

---

## 3. Prosedur Pencadangan (Backup)

### 3.1. Format Backup
Gunakan utilitas resmi PostgreSQL `pg_dump` dengan opsi format kustom (`-Fc`):
* Menghasilkan file terkompresi.
* Mendukung pemulihan granular (pemilihan tabel tertentu jika dibutuhkan).
* Mendukung opsi paralel `pg_restore -j`.

### 3.2. Perintah Backup Manual
Jalankan pencadangan langsung melalui container database:

```bash
# Tentukan nama file backup berdasar timestamp
BACKUP_FILE="ppo_backup_$(date +%Y%m%d_%H%M%S).dump"

# Jalankan pg_dump di dalam container db
docker compose -f docker-compose.prod.yml exec -T db \
  pg_dump -U ppo_app -d ppo_kanban_prod -Fc > "/var/backups/ppo/${BACKUP_FILE}"

# Enkripsi file backup menggunakan GPG (kunci asimetris publik perbankan)
gpg --encrypt --recipient ops-backup@bank.corp.internal "/var/backups/ppo/${BACKUP_FILE}"
rm "/var/backups/ppo/${BACKUP_FILE}"
```

### 3.3. Skrip Otomatisasi Terjadwal (Cron Job)
Simpan skrip berikut di `/opt/ppo-kanban/scripts/backup.sh` dan beri hak eksekusi (`chmod +x`):

```bash
#!/bin/bash
set -euo pipefail

BACKUP_DIR="/var/backups/ppo"
DATE=$(date +%Y%m%d_%H%M%S)
FILENAME="ppo_kanban_${DATE}.dump"
RETENTION_DAYS=30

mkdir -p "${BACKUP_DIR}"

# 1. Eksekusi pg_dump
docker compose -f /opt/ppo-kanban/docker-compose.prod.yml exec -T db \
  pg_dump -U ppo_app -d ppo_kanban_prod -Fc > "${BACKUP_DIR}/${FILENAME}"

# 2. Verifikasi ukuran file (tidak boleh 0 byte)
if [ ! -s "${BACKUP_DIR}/${FILENAME}" ]; then
  echo "ERROR: Backup file kosong!" | mail -s "PPO Backup FAILED" ops@bank.corp.internal
  exit 1
fi

# 3. Rotasi backup lokal lama (> 30 hari)
find "${BACKUP_DIR}" -name "ppo_kanban_*.dump" -mtime +${RETENTION_DAYS} -delete

echo "[${DATE}] Backup berhasil: ${FILENAME}"
```

Jadwalkan melalui crontab server (`crontab -e`):
```cron
# Backup otomatis setiap 6 jam: jam 00:00, 06:00, 12:00, 18:00
0 0,6,12,18 * * * /opt/ppo-kanban/scripts/backup.sh >> /var/log/ppo_backup.log 2>&1
```

---

## 4. Prosedur Uji Pemulihan (Restore Drill)

> [!IMPORTANT]
> **Peringatan**: Uji restore **DILARANG KERAS** dijalankan langsung ke database produksi aktif. Selalu lakukan restore drill ke database terpisah (misalnya `ppo_kanban_drill`).

### Langkah Restore Drill:
1. Siapkan database target drill di PostgreSQL:
   ```bash
   docker compose -f docker-compose.prod.yml exec -T db \
     psql -U ppo_app -d postgres -c "DROP DATABASE IF EXISTS ppo_kanban_drill;"
   
   docker compose -f docker-compose.prod.yml exec -T db \
     psql -U ppo_app -d postgres -c "CREATE DATABASE ppo_kanban_drill OWNER ppo_app;"
   ```

2. Lakukan restore dari dump file:
   ```bash
   cat "/var/backups/ppo/ppo_kanban_YYYYMMDD_HHMMSS.dump" | \
     docker compose -f docker-compose.prod.yml exec -T db \
     pg_restore -U ppo_app -d ppo_kanban_drill --clean --if-exists --no-owner
   ```

3. Verifikasi jumlah baris dan integritas tabel:
   ```bash
   docker compose -f docker-compose.prod.yml exec -T db \
     psql -U ppo_app -d ppo_kanban_drill -c "
       SELECT 'users' AS table_name, count(*) FROM users
       UNION ALL
       SELECT 'projects', count(*) FROM projects
       UNION ALL
       SELECT 'tasks', count(*) FROM tasks
       UNION ALL
       SELECT 'risks', count(*) FROM risks
       UNION ALL
       SELECT 'audit_findings', count(*) FROM audit_findings
       UNION ALL
       SELECT 'audit_logs', count(*) FROM audit_logs;
     "
   ```

4. Uji proteksi trigger audit logs pada database hasil restore:
   ```bash
   # Percobaan delete ini HARUS GAGAL dengan pesan exception trigger
   docker compose -f docker-compose.prod.yml exec -T db \
     psql -U ppo_app -d ppo_kanban_drill -c "DELETE FROM audit_logs;"
   # Ekspektasi: ERROR: UPDATE or DELETE on audit_logs is forbidden by security policy
   ```

---

## 5. Prosedur Disaster Recovery (DR) Darurat

Jika server produksi mengalami kegagalan total dan database rusak permanen:

1. **Hentikan Container Backend**:
   Cegah request tulis baru masuk saat database sedang direcover:
   ```bash
   docker compose -f docker-compose.prod.yml stop backend frontend
   ```

2. **Bersihkan Database Rusak & Buat Ulang**:
   ```bash
   docker compose -f docker-compose.prod.yml exec -T db \
     psql -U ppo_app -d postgres -c "DROP DATABASE ppo_kanban_prod WITH (FORCE);"
   
   docker compose -f docker-compose.prod.yml exec -T db \
     psql -U ppo_app -d postgres -c "CREATE DATABASE ppo_kanban_prod OWNER ppo_app;"
   ```

3. **Restore Backup Terkini**:
   ```bash
   cat "/var/backups/ppo/<backup_terakhir>.dump" | \
     docker compose -f docker-compose.prod.yml exec -T db \
     pg_restore -U ppo_app -d ppo_kanban_prod --clean --if-exists --no-owner
   ```

4. **Nyalakan Kembali Layanan**:
   ```bash
   docker compose -f docker-compose.prod.yml start backend frontend
   ```

5. **Verifikasi Operasional**:
   - Akses `/healthz` memastikan status 200.
   - Login dengan akun user dan verifikasi konsistensi data papan Kanban dan risk register.
