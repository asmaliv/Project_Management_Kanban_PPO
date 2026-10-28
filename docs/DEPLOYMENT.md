# Panduan Deployment Produksi — PPO Kanban Board

Dokumen ini menjelaskan prosedur instalasi, konfigurasi, dan pengoperasian aplikasi **PPO Kanban Board** di lingkungan produksi (production environment) perbankan yang aman dan terisolasi.

---

## 1. Arsitektur Produksi

```
[ Klien / Browser ]
        │ HTTPS (TLS 1.3 / Port 443)
        ▼
[ Reverse Proxy / Cloudflare / Nginx Ingress ]
        │ HTTP (Internal Bridge Network)
        ▼
[ Frontend Container (Nginx Unprivileged:8080) ]
        │ Static assets (/index.html, /assets/*)
        │ Proxy API requests (/api/*, /healthz)
        ▼
[ Backend Container (Go Binary Non-Root:8080) ]
        │ Database Connection Pool (pgxpool)
        ▼
[ Database Container / Managed PostgreSQL 16 ]
```

### Prinsip Keamanan Utama
1. **Zero Public DB Exposure**: Database PostgreSQL hanya dapat diakses melalui internal bridge network container, port `5432` tidak dibuka ke publik.
2. **Non-Root Execution**: Backend berjalan dengan UID `10001:10001` (`appuser`), dan Frontend berjalan dengan user `nginx`.
3. **Defense-in-Depth Headers**: CSP ketat, HSTS, X-Content-Type-Options, Referrer-Policy, dan Permissions-Policy ditegakkan di layer web server dan layer backend Go.
4. **Append-Only Audit Logs**: Trigger database melindungi integritas tabel `audit_logs`.

---

## 2. Prasyarat Server

* **Sistem Operasi**: Linux (Ubuntu 22.04 LTS / 24.04 LTS, Debian 12, atau RHEL 9).
* **Hardware Minimum**: 2 vCPU, 4 GB RAM, 40 GB SSD.
* **Perangkat Lunak**:
  * Docker Engine versi >= 24.0
  * Docker Compose versi >= 2.20
  * Git
  * OpenSSL (untuk membuat secret acak)
* **Jaringan & Domain**:
  * Domain/subdomain dengan DNS A record yang mengarah ke IP publik server.
  * Port 80 dan 443 terbuka pada firewall (UFW/Security Group).

---

## 3. Langkah Instalasi Awal

### 3.1. Clone Repository
```bash
git clone https://github.com/asmaliv/Project_Management_Kanban_PPO.git /opt/ppo-kanban
cd /opt/ppo-kanban
```

### 3.2. Buat File Konfigurasi Produksi
Salin template konfigurasi:
```bash
cp .env.production.example .env.production
chmod 600 .env.production
```

Generate secret acak berkeamanan tinggi:
```bash
# Generate database password (32 karakter acak)
openssl rand -hex 16

# Generate session secret (32 bytes base64)
openssl rand -base64 32

# Generate initial admin temporary password
openssl rand -base64 16
```

Edit `.env.production` dengan nilai yang dihasilkan:
```bash
nano .env.production
```
Pastikan variabel berikut terisi dengan benar:
- `POSTGRES_DB=ppo_kanban_prod`
- `POSTGRES_USER=ppo_app`
- `POSTGRES_PASSWORD=<password_database_dari_openssl>`
- `SESSION_SECRET=<session_secret_dari_openssl>`
- `COOKIE_SECURE=true`
- `BOOTSTRAP_ADMIN_EMAIL=admin@bank.corp.internal`
- `BOOTSTRAP_ADMIN_PASSWORD=<admin_password_dari_openssl>`

### 3.3. Menjalankan Database & Migrasi Skema
Jalankan service database terlebih dahulu:
```bash
docker compose -f docker-compose.prod.yml up -d db
```

Tunggu hingga PostgreSQL siap (healthcheck healthy):
```bash
docker compose -f docker-compose.prod.yml ps
```

Jalankan migrasi database menggunakan `golang-migrate`:
```bash
# Menggunakan migrate tool lokal atau container migrate resmi
docker run --rm -v /opt/ppo-kanban/backend/migrations:/migrations --network ppo-kanban_ppo_net \
  migrate/migrate:v4.18.2 \
  -path=/migrations \
  -database "postgres://ppo_app:<password_database>@db:5432/ppo_kanban_prod?sslmode=disable" \
  up
```

### 3.4. Build dan Jalankan Seluruh Container
```bash
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
```

Periksa status container:
```bash
docker compose -f docker-compose.prod.yml ps
```
Semua container (`db`, `backend`, `frontend`) wajib berstatus `healthy`.

---

## 4. Verifikasi Pasca Deployment

1. **Uji Healthcheck**:
   ```bash
   curl -I http://localhost/healthz
   # Respon yang diharapkan: HTTP/1.1 200 OK
   ```

2. **Uji Security Headers**:
   ```bash
   curl -I http://localhost/
   # Verifikasi adanya: Content-Security-Policy, X-Frame-Options: DENY, X-Content-Type-Options: nosniff
   ```

3. **Login Pertama Kali**:
   - Buka aplikasi di browser.
   - Login dengan email dan password `BOOTSTRAP_ADMIN_EMAIL` & `BOOTSTRAP_ADMIN_PASSWORD`.
   - Sistem akan mewajibkan penggantian password administrator pada login pertama (`must_change_password=true`).
   - Ganti password dengan password baru yang memenuhi kebijakan keamanan perbankan (minimal 12 karakter, huruf besar, kecil, angka, simbol).

---

## 5. Prosedur Update & Pemeliharaan (Rolling Update)

Ketika ada versi rilis baru dari git:

```bash
cd /opt/ppo-kanban

# 1. Tarik perubahan terbaru
git fetch origin main
git checkout main
git pull origin main

# 2. Jalankan migrasi database baru jika ada
docker run --rm -v /opt/ppo-kanban/backend/migrations:/migrations --network ppo-kanban_ppo_net \
  migrate/migrate:v4.18.2 \
  -path=/migrations \
  -database "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@db:5432/${POSTGRES_DB}?sslmode=disable" \
  up

# 3. Build image baru dan restart container tanpa downtime lama
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build backend frontend
```

---

## 6. Monitoring & Troubleshooting

* **Melihat log backend**:
  ```bash
  docker compose -f docker-compose.prod.yml logs -f backend
  ```
* **Melihat log web server frontend**:
  ```bash
  docker compose -f docker-compose.prod.yml logs -f frontend
  ```
* **Melihat log database**:
  ```bash
  docker compose -f docker-compose.prod.yml logs -f db
  ```
* **Rotasi Log Docker**: Konfigurasi logging driver Docker di `/etc/docker/daemon.json` agar ukuran log tidak memenuhi disk:
  ```json
  {
    "log-driver": "json-file",
    "log-opts": {
      "max-size": "50m",
      "max-file": "5"
    }
  }
  ```
