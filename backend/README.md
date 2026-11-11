# Backend Miva

Backend memakai Go dengan pola **handler → service → repository**. Struktur data bersama berada di `model`. Pembuatan dependency dilakukan sekali di `app/bootstrap.go`.

```text
main.go                          Titik masuk: serve atau migrate
internal/
  app/bootstrap.go               Hubungkan database, repository, service, provider, HTTP
  config/config.go               Baca konfigurasi dari environment
  handler/                       HTTP, JSON/FormData, cookie, origin, status respons
    server.go                    Router dan pemeriksaan sesi
    auth.go                      Login, daftar, logout
    lessons.go                   Materi, draft, submit, heartbeat, riwayat
    studies.go                   Persiapan sesi dan status pekerjaan
    words.go / chat.go           Kamus dan percakapan
    reading.go / audio.go        Rekaman, pemeriksaan bacaan, sumber suara
    dashboard.go                 Dashboard, pengaturan, kalender
  service/                       Aturan dan urutan proses aplikasi
    services.go                  Penyusunan service
    auth.go                      Password, sesi, pengaturan akun
    lessons.go                   Validasi jawaban, penilaian, idempotensi submit
    studies.go                   Persiapan materi di latar belakang
    reviews.go                   Rangkuman tutor tanpa mengubah skor
    progress.go                  Hitung progres mingguan
    words.go / chat.go           Lookup dan percakapan sesuai goal pengguna
    reading.go                   Pilih referensi bacaan dan pemeriksa suara
    reading_alignment.go         Pencocokan kata, bukan skor fonetik
    audio.go                     Teks suara dari materi milik pengguna
  model/                         Struktur data, validasi, scoring, kalender
  repository/
    contracts.go                 Interface penyimpanan yang dipakai service
    postgres/                    SQL, transaksi, pemetaan baris database
      users.go / words.go        Akun, sesi, kosakata
      lessons.go                 Ambil dan simpan materi beserta kosakata
      lesson_progress.go         Draft, waktu aktif, hasil, riwayat
      messages.go / jobs.go       Percakapan dan persiapan materi
      readings.go / limits.go     Hasil bacaan, pembatasan permintaan
  provider/
    ai/                          HTTP API key, prompt, format respons model
    speech/                      Pemeriksa suara Azure opsional
  database/                      Koneksi PostgreSQL dan mesin migrasi
    migrations/                  SQL bernomor; jangan ubah file yang sudah diterapkan
  platform/                      Error aplikasi, identitas, utilitas JSON
  integration/                   Skenario pengujian lintas lapisan
```

## Cara membaca satu fitur

Contoh menyimpan jawaban:

1. `handler/server.go` memastikan origin dan sesi pengguna valid.
2. `handler/lessons.go` membaca JSON untuk `PATCH /api/lesson/draft`.
3. `service/lessons.go` → `Draft()` memeriksa ID, kepemilikan materi, status selesai, dan pilihan jawaban.
4. `repository/contracts.go` → `Lessons.SaveDraft()` adalah kontrak penyimpanan.
5. `repository/postgres/lesson_progress.go` menjalankan SQL yang dibatasi ID pengguna.
6. Handler mengirim respons JSON. Service dan repository tidak menulis respons HTTP.

Untuk penilaian, ikuti `Submit()` pada file service yang sama. `model.Grade()` menghitung skor dari kunci yang tersimpan. Penjelasan tutor ditambahkan melalui `ReviewService`; skor tetap berdasarkan jawaban soal. Pengulangan submit mengembalikan hasil yang sudah tersimpan.

## Batas tanggung jawab

- **Handler:** membaca input, memanggil service, menyusun respons. Tidak menjalankan SQL, memeriksa password, atau menghubungi model secara langsung.
- **Service:** menentukan apa yang harus dilakukan dan urutannya. Hanya bergantung pada interface repository dan provider; tidak bergantung pada pgx atau `net/http`.
- **Repository:** menyimpan dan mengambil data. Menangani transaksi, pembatasan kepemilikan pada query, dan konversi baris menjadi model. Tidak menghitung nilai atau membuat prompt.
- **Model:** bentuk data dan aturan validasinya. Tidak mengetahui koneksi database maupun HTTP.
- **Provider:** implementasi koneksi layanan luar. Interface `AI` dan `Speech` dimiliki service sehingga pengujian dapat memakai provider simulasi.
- **App:** merangkai dependency dan menjalankan server; seluruh SQL aplikasi berada di repository, sedangkan migrasi berada di database.

Interface repository dibuat per kebutuhan fitur, bukan satu repository generik dengan operasi CRUD bebas. `service.New()` menghubungkannya secara eksplisit. Tes `internal/app/architecture_test.go` menjaga batas import antar lapisan.

## Menambah fitur

Tentukan model input/hasil, tambahkan aturan di service, tambahkan method repository bila perlu menyimpan data, implementasikan SQL di `postgres`, lalu tambahkan handler dan rutenya. Tambahkan migrasi baru untuk perubahan schema. Jangan mengubah checksum migrasi lama.

## Menjalankan dan memeriksa

Dari root proyek:

```sh
docker compose up -d --build
./scripts/test.sh
```

Dengan Go 1.26, dari folder `backend`:

```sh
go run .
go run . migrate
go test ./...
go vet ./...
```

Tes database memerlukan `RUN_DB_TESTS=1` dan konfigurasi akun khusus aplikasi. `scripts/test.sh` sudah menyiapkannya melalui Docker. Tes provider memakai server simulasi, tidak menghabiskan saldo API.

Konfigurasi dan perintah deployment tetap mengikuti [panduan VPS](../docs/deploy-vps.md).
