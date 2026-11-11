# Deploy Miva ke VPS

Frontend dan backend adalah dua image/container terpisah. Frontend Nginx menyajikan Svelte dan meneruskan `/api/` serta `/healthz` ke backend Go melalui jaringan Docker. Browser memakai satu domain sehingga cookie login dan mikrofon tetap bekerja. Backend tidak membuka port publik. Tidak ada container PostgreSQL baru di konfigurasi produksi; aplikasi memakai database yang sudah ada.

## 1. Prasyarat VPS

- Linux, Python 3.9+, Docker Engine dan Docker Compose v2.24+ (mendukung `up --wait`).
- User SSH dapat menjalankan Docker dan menulis direktori aplikasi. Akses Docker setara administrator; gunakan user deploy yang memang kamu percayai.
- Domain diarahkan ke VPS dan reverse proxy HTTPS sudah aktif. Contoh Caddy untuk instalasi di host tersedia di `deploy/Caddyfile.example`; frontend secara default hanya membuka `127.0.0.1:5180`. Reverse proxy yang sudah ada harus meneruskan header Host asli. Isi `APP_ORIGIN` dengan domain publik persis, termasuk skema dan port bila ada.
- Database `toko` (atau nama database pilihanmu) sudah tersedia. Siapkan role dan schema khusus dengan administrator PostgreSQL, tanpa memberi role aplikasi akses ke tabel toko:

```sql
-- Contoh awal pada VPS baru; isi password sendiri melalui kanal admin yang aman.
CREATE ROLE english_app LOGIN PASSWORD 'GANTI_DENGAN_PASSWORD_KUAT' CONNECTION LIMIT 5;
GRANT CONNECT ON DATABASE toko TO english_app;
-- Jalankan pada database toko:
CREATE SCHEMA english_app AUTHORIZATION english_app;
```

Jika role/schema sudah ada, gunakan yang ada; jangan menjalankan CREATE ROLE ulang. Role aplikasi tidak perlu superuser, CREATEDB, atau CREATEROLE. Semua perubahan schema dibatasi ke `english_app`; tabel toko pada `public` tidak dimigrasikan. Pengaturan GRANT lain pada database milikmu tetap perlu diperiksa oleh administrator.

## 2. Konfigurasi VPS

Salin `deploy/.env.example` menjadi **file baru** `deploy/.env`, lalu isi koneksi database, domain, dan API key. Jangan menyalin konfigurasi localhost mentah-mentah ke VPS. File ini tidak masuk Git atau arsip source deployment.

```sh
cp deploy/.env.example deploy/.env
chmod 600 deploy/.env
```

Jika PostgreSQL berjalan di Docker Compose lain, cara yang disarankan adalah bergabung ke jaringan database tersebut, tanpa membuka port database ke internet:

```dotenv
DB_NETWORK=toko_default
DB_HOST=toko-postgres-1
DB_PORT=5432
```

Gunakan nama network dan hostname container yang benar di VPS-mu (`docker network ls`). Script otomatis menyertakan `deploy/compose.database-network.yaml` bila `DB_NETWORK` terisi. Jika database berjalan langsung di host, gunakan `DB_HOST=host.docker.internal` dan port yang benar; PostgreSQL perlu menerima koneksi dari jaringan bridge Docker melalui listen_addresses/pg_hba yang dibatasi. `localhost` di dalam container mengarah ke container itu sendiri. Port database yang hanya terikat ke loopback host Linux tidak otomatis bisa diakses lewat host-gateway.

Untuk HTTPS gunakan `COOKIE_SECURE=true`. Untuk pengujian localhost HTTP saja, gunakan `COOKIE_SECURE=false`. `ALLOW_REGISTRATION=false` secara bawaan: jika database masih kosong, aktifkan sementara untuk membuat akunmu, lalu nonaktifkan dan deploy ulang. `POSTGRES_TOOLS_IMAGE` harus memakai pg_dump dengan versi mayor minimal sama dengan server PostgreSQL; default `postgres:18-alpine`.

## 3. Deploy dari komputer lokal melalui SSH

Verifikasi fingerprint server melalui kanal VPS yang dipercaya, lalu pastikan host sudah ada di `~/.ssh/known_hosts`. Script memakai StrictHostKeyChecking dan tidak menerima host key secara otomatis. Instalasi Docker/Python di VPS tidak dilakukan diam-diam oleh script.

```sh
python3 scripts/deploy.py \
  --host user@alamat-vps \
  --remote-dir /opt/miva \
  --env-file deploy/.env
```

Opsional: `--identity ~/.ssh/id_ed25519`, `--ssh-port 22`, dan `--project-name miva`. Perintah di atas mengirim **file env yang kamu pilih** ke `/opt/miva/shared/.env` melalui SSH dengan izin 600, terpisah dari source. Untuk deploy selanjutnya, hilangkan `--env-file` agar memakai env VPS yang sudah ada:

```sh
python3 scripts/deploy.py --host user@alamat-vps --remote-dir /opt/miva
```

Source diunggah ke `/opt/miva/releases/<release-id>`. File `.env`, node_modules, hasil build lokal, backup, dan private key tidak masuk arsip. Backup berada di `/opt/miva/shared/backups`. Direktori release dan backup lama sengaja tidak dihapus otomatis; kelola retensinya sesuai kapasitas disk.

## 4. Menjalankan langsung di VPS

Dari root source proyek, dengan env VPS yang sudah diisi:

```sh
python3 scripts/deploy.py --env-file deploy/.env
```

Alurnya: kunci deploy → validasi konfigurasi → build dua image → backup schema aplikasi → migrasi → jalankan backend/frontend → tunggu keduanya sehat → catat release berhasil. Script hanya memakai Python standard library, Docker, dan SSH; tidak membutuhkan pip/Paramiko.

Backup memakai pg_dump custom-format khusus schema `english_app`, bukan seluruh `toko`. Jika schema belum ada, pemasangan pertama tidak memiliki data untuk dicadangkan. Jika backup atau migrasi gagal, script berhenti sebelum mengganti container aplikasi. Kegagalan startup setelah migrasi tidak otomatis mengembalikan database; backup dan image release sebelumnya tetap tersedia. Ada jeda singkat saat container direcreate; ini bukan rolling deployment tanpa downtime. Jangan menjalankan deploy ketika pengguna sedang menyiapkan sesi belajar panjang, karena restart menandai job yang belum selesai sebagai terhenti.

## 5. Deploy otomatis dari GitHub

Workflow `.github/workflows/deploy-vps.yml` berjalan saat push ke `main` atau dijalankan manual. Ia menguji script Python, unit test Go tanpa database produksi, dan build frontend terlebih dahulu. Job deploy memakai environment GitHub **production**.

Siapkan env VPS terlebih dahulu pada `/opt/miva/shared/.env` lewat deploy pertama di atas. Kemudian isi environment secrets:

| Secret                | Isi                                                                                                     |
| --------------------- | ------------------------------------------------------------------------------------------------------- |
| `VPS_SSH_KEY`         | Private key khusus user deploy                                                                          |
| `VPS_SSH_KNOWN_HOSTS` | Entri known_hosts server yang fingerprint-nya sudah diverifikasi; port nonstandar memakai `[host]:port` |

Isi environment variables:

| Variable         | Isi                           |
| ---------------- | ----------------------------- |
| `VPS_HOST`       | IP/hostname VPS               |
| `VPS_USER`       | User SSH                      |
| `VPS_SSH_PORT`   | Opsional, default 22          |
| `VPS_DEPLOY_DIR` | Opsional, default `/opt/miva` |

API key dan password database tidak perlu menjadi GitHub secrets; workflow memakai env yang tetap berada di VPS. Jika repository menggunakan branch lain, ubah filter `branches`. Aktifkan required reviewers di environment production bila menginginkan persetujuan sebelum deploy. Belum ada workflow yang dipasang ke akun GitHub atau VPS oleh pembuatan file ini; ia aktif setelah source didorong ke repository dan konfigurasi diisi.

## 6. Menambah migrasi berikutnya

Buat file SQL baru seperti `backend/internal/database/migrations/0003_nama_perubahan.sql`. Nama harus 4 digit, underscore, deskripsi huruf kecil/angka/underscore, lalu `.sql`. File ikut ditanam ke binary Go saat build. Migrasi berjalan terurut dan dicatat bersama checksum SHA-256 di `english_app.schema_migrations`.

- Jangan mengedit file yang sudah pernah diterapkan. Tambahkan nomor berikutnya.
- Semua migrasi tertunda dan riwayatnya di-commit dalam satu transaksi PostgreSQL; bila satu SQL gagal, perubahan batch itu dibatalkan.
- Kunci transaksi advisory mencegah dua migrator berjalan bersamaan. Riwayat yang berubah, urutan salah, atau database yang lebih baru ditolak.
- Migrasi harus kompatibel dengan aplikasi versi lama yang masih berjalan saat proses ini berlangsung. Pilih penambahan kolom/tabel terlebih dahulu; lakukan penghapusan/rename dalam tahap terpisah setelah semua kode sudah beralih.
- Jangan memakai `CREATE INDEX CONCURRENTLY`, `VACUUM`, atau transaksi `BEGIN/COMMIT` sendiri di file ini; perintah yang tidak boleh berjalan di transaksi memerlukan prosedur admin terpisah.
- Schema lama aplikasi diadopsi melalui dua migrasi awal yang idempoten; akun dan progres yang ada tidak dihapus.
- Produksi memakai `MIGRATE_ON_START=false`: proses serve memeriksa versi schema, tanpa menjalankan DDL. Service migrasi atau script deploy menerapkannya terlebih dahulu. Konfigurasi lokal lama tetap mendukung migrasi otomatis saat startup.

Sumber schema aktif hanya file bernomor di `backend/internal/database/migrations/`. Snapshot SQL lama dan implementasi koneksi langganan sudah dihapus. Jangan mengubah file migrasi yang pernah diterapkan.

## Operasi dan pemulihan

Dari source release di VPS, gunakan env bersama dan project yang sama:

```sh
export MIVA_ENV_FILE=/opt/miva/shared/.env
# Isi ID dari /opt/miva/shared/.deploy-state.json:
export MIVA_RELEASE=ID_RELEASE_AKTIF
docker compose -p miva --env-file "$MIVA_ENV_FILE" -f compose.vps.yaml ps
docker compose -p miva --env-file "$MIVA_ENV_FILE" -f compose.vps.yaml logs --tail 100 backend frontend
```

Jika DB_NETWORK terisi, tambahkan `-f deploy/compose.database-network.yaml` pada perintah manual. Jangan menjalankan `docker compose config` ke log publik karena hasilnya dapat berisi secret.

Untuk rollback aplikasi, gunakan tag image release sebelumnya dari `.deploy-state.json`, lalu `up -d --no-deps --wait backend frontend`. Backend menolak schema yang lebih baru dari binary; hanya lakukan rollback bila daftar migrasi versi itu sama dan perubahan kompatibel. Bila schema perlu diperbaiki, prioritaskan migrasi koreksi baru. Jangan memaksa menghapus migration history atau melakukan restore database otomatis. Pemulihan backup harus dilakukan operator setelah menghentikan penulisan dan memverifikasi data mana yang akan diganti. Jangan menghapus volume database toko.

Pengujian:

```sh
python3 -m unittest discover -s scripts/tests -v
./scripts/test.sh
./scripts/test-deploy.sh
```

Smoke test terakhir memakai PostgreSQL/container dan akun sementara sendiri, menjalankan dua deploy, mengecek backup, login API melalui frontend, dan membersihkan container test. Ia tidak memakai database atau saldo API produksi.

Referensi: [Compose startup order](https://docs.docker.com/compose/how-tos/startup-order/), [Compose up --wait](https://docs.docker.com/reference/cli/docker/compose/up/), [Nginx upstream DNS](https://nginx.org/en/docs/http/ngx_http_upstream_module.html).
