# Miva — Go + Svelte + PostgreSQL

Untuk membaca alur implementasi, mulai dari [peta kode dan arsitektur](docs/architecture.md). Konvensi serta perintah format tersedia di [CONTRIBUTING.md](CONTRIBUTING.md).

Aplikasi belajar Inggris dengan backend **Go**, frontend **Svelte JavaScript**, dan PostgreSQL lokal **toko** pada port **55432**.

Mode utama sekarang **API key**, dengan konfigurasi awal VPS Indonesia. Tidak perlu login atau langganan ChatGPT Plus. Akses AI memakai saldo dari penyedia API yang dipilih; pembaca suara perangkat tetap tersedia tanpa API audio.

## Pasang API key

Edit **backend/.env**. Base URL dan model awal sudah disiapkan:

```dotenv
AI_BASE_URL=https://member.vpsindonesia.co.id/api/v1
AI_API_KEY=isi_api_key_vps_indonesia_di_sini
AI_MODEL=sonnet-5
AI_API_FORMAT=chat_completions
AI_JSON_MODE=prompt
```

`sonnet-5` mengikuti ID pada katalog yang diberikan. Nama model harus sama persis dengan katalog penyedia; ubah AI_MODEL jika ingin menggunakan model lain. Harga, saldo, akses model, dan kompatibilitas nyata tetap mengikuti penyedia.

API key hanya disimpan pada backend/.env, tidak di frontend atau Git. Jangan memakai file env database sebagai berkas unggahan.

Dari folder proyek, terapkan key yang baru:

```sh
docker compose up -d --force-recreate
```

Lalu buka **http://localhost:5180**, masuk ke akun aplikasi, dan pilih **Atur belajar hari ini**. Panel koneksi sengaja tidak ditampilkan di aplikasi; konfigurasi penyedia dilakukan melalui environment backend.

Tidak ada API key yang disertakan atau ditebak. Selama AI_API_KEY kosong, fitur pembuat materi, pencarian AI, dan chat belum aktif. Kamus awal, suara perangkat, login, goal, dan kalender tetap bisa digunakan.

## Jalankan / hentikan

Docker Desktop perlu aktif. Untuk build pertama atau sesudah perubahan kode:

```sh
BUILDX_CONFIG=/tmp/english-app-buildx docker compose up -d --build
```

Untuk menghentikan aplikasi:

```sh
docker compose down
```

Ini hanya menghentikan Miva; tidak menghentikan database toko.

## Format API dan keluaran soal

Backend mendukung `AI_API_FORMAT=chat_completions` untuk API kompatibel OpenAI:

- POST ke `AI_BASE_URL + /chat/completions`.
- Authorization: Bearer dari AI_API_KEY.
- Model dari AI_MODEL, messages system/user/assistant, stream:false, max_tokens.
- Tidak mengirim parameter khusus Responses API ke Chat Completions.
- Tidak mengikuti redirect endpoint API untuk mencegah key diteruskan ke tujuan lain.
- Menerima jawaban teks dan teks dalam content parts. Jawaban yang terpotong, ditolak, kosong, atau tidak valid tidak disimpan sebagai materi selesai.

`AI_JSON_MODE=prompt` adalah default agar tidak bergantung pada dukungan response_format penyedia. Instruksi dan JSON schema dikirim sebagai prompt, lalu hasil divalidasi di backend sebelum disimpan. Code fence JSON sederhana ditoleransi; JSON rusak dan materi tidak lengkap ditolak tanpa perbaikan otomatis berbayar.

Jika penyedia mendukungnya secara resmi, bisa pilih:

```dotenv
AI_JSON_MODE=json_object
# atau
AI_JSON_MODE=json_schema
```

Mode json_schema mengirim response_format.json_schema dengan strict:true. Tidak ada retry otomatis ke mode lain agar kegagalan tidak menimbulkan tagihan tambahan tanpa terlihat.

Untuk penyedia yang memang mendukung Responses API, ada pilihan eksplisit `AI_API_FORMAT=responses`. Jangan mengubahnya hanya karena provider menyebut “OpenAI-compatible”; pastikan endpoint yang didukung.

Koneksi model hanya menggunakan API key, Base URL, model, dan format protokol yang dikonfigurasi. Implementasi OAuth/ChatGPT Plus dan audio Realtime lama sudah dihapus; backend tidak mengambil token langganan atau mengirim key ke endpoint audio lain.

## Penghematan

- Sesi baru hanya dibuat saat pengguna meminta. Refresh atau permintaan ulang dengan ID yang sama tidak membuat materi baru. Maksimal satu persiapan aktif per akun.
- Materi, kosakata, jawaban, dan percakapan tersimpan di PostgreSQL.
- Tutor default menjawab kurang dari 90 kata, dengan batas keluaran 450 token. Penjelasan lebih panjang tetap dibatasi token.
- Hanya 10 pesan terakhir dikirim sebagai konteks percakapan.
- Materi dibuat per bagian: batch 5 kosakata, satu tense/bacaan/listening per permintaan, dan batch 10 soal. Batas keluaran per permintaan 1.500–7.500 token; jumlah permintaan bertambah mengikuti porsi materi. Entri kamus dibatasi 1.000 token.
- Skor pilihan ganda dihitung lokal di server. Rangkuman pribadi sesudah submit memakai satu panggilan model. Membaca progres, ekspor kalender, dan suara perangkat tidak memanggil model.
- AI_DAILY_LIMIT=120 membatasi jumlah panggilan per akun per hari. Ini bukan batas pengeluaran rupiah; jumlah token dan tarif provider menentukan biaya nyata.
- Tes koneksi dibatasi tiga kali per akun per 10 menit.

Harga atau saldo provider tidak dipalsukan menjadi estimasi pemakaian aktual di UI. Aplikasi belum memiliki penghitung pengeluaran rupiah atau akses ke saldo provider. Pantau saldo dari dashboard penyedia.

## Fitur

- Registrasi/login aplikasi dengan password bcrypt dan cookie sesi HTTP-only.
- Goal mingguan: menit, jumlah pelajaran, CEFR A1–B2, tenses, hari dan jam WIB.
- Materi harian sesuai pilihan: kosakata, 12 pilihan tenses, reading, maksimal lima listening, maksimal 30 soal dan pembahasan.
- Jawaban sementara otomatis tersimpan; grading dan submit idempoten dilakukan di backend.
- Kamus Inggris/Indonesia dengan arti, IPA, Verb 1–2–3, contoh dan terjemahan. Dua belas verb awal tersedia tanpa key.
- Tutor bilingual untuk role-play, percakapan dan koreksi ringan.
- Suara perangkat untuk kata, setiap bentuk verb, contoh kalimat, listening, dan balasan tutor. Pilih suara Inggris, kecepatan, ulangi/jeda/hentikan. Kualitas serta ketersediaan bergantung browser dan suara perangkat; sebagian suara dapat memakai layanan vendor.
- Dikte bahasa Inggris/Indonesia jika SpeechRecognition didukung browser. Pada chat, hasil ucapan digabung dengan draft lalu otomatis dikirim setelah selesai bicara. Pilihan bahasa dan kontrol playback tersedia melalui tombol pengaturan di sebelah mikrofon. Jika tidak didukung, gunakan teks. Reading menyediakan mode kecocokan kata, dengan opsi layanan pemeriksaan rekaman sebagaimana dijelaskan di bawah.
- Pengingat kalender .ics berulang untuk Google/Apple/Outlook. Impor dan aktifkan notifikasi di kalender. Saat mengganti jadwal, perbarui acara lama agar tidak ganda.
- Statistik berdasarkan pelajaran yang selesai. Waktu halaman pelajaran aktif dicatat lewat heartbeat server dan masuk statistik setelah submit.

Transkrip listening tidak ditampilkan sebelum submit, tetapi pembaca suara perangkat membutuhkan teks di klien dan teks tersebut bisa diperiksa dengan developer tools. Kunci jawaban tetap disimpan dan dinilai di server.

## Database dan resource

Memakai database **toko**, role dan schema khusus **english_app**, bukan PostgreSQL baru. Tidak mengubah tabel toko pada schema public.

Role bukan superuser, tidak boleh membuat database/role, batas lima koneksi. Pool backend maksimal tiga koneksi. Hak SELECT role ini terhadap tabel toko pada schema public telah diperiksa dan tidak tersedia saat penyiapan.

Tabel khusus dikelola oleh migrasi bernomor di backend/internal/database/migrations, dengan riwayat dan checksum tersimpan pada schema aplikasi. Data aplikasi dipisahkan per akun; tes integrasi memverifikasi akses lintas akun ditolak.

Container app: 256 MB RAM, 0,5 CPU, GOMEMLIMIT 192 MiB, non-root, filesystem read-only dan tmpfs 16 MB. Listening memakai suara browser dan tidak menambah cache audio di PostgreSQL.

Cadangkan schema english_app bersama database yang sudah ada. Jangan menghapus volume PostgreSQL toko untuk mereset aplikasi ini.

## Struktur

```text
backend/
  main.go                 Titik masuk serve / migrate
  internal/
    app/                  Penyusunan dependency dan startup
    config/               Konfigurasi environment
    handler/              HTTP, sesi, cookie dan respons
    service/              Aturan dan proses belajar
    model/                Data, validasi, scoring dan kalender
    repository/           Interface penyimpanan
      postgres/           SQL dan transaksi per fitur
    provider/             Adapter AI dan pemeriksaan suara
    database/migrations/  Migrasi SQL bernomor, checksum tetap
    platform/             Error, ID dan utilitas JSON
    integration/          Pengujian lintas lapisan
frontend/src/
  pages/                  Halaman Svelte per fitur
  components/             Komponen tampilan
  styles/                 Fondasi dan tema responsif
compose.yaml              Docker lokal
compose.vps.yaml          Backend dan frontend terpisah untuk VPS
scripts/test.sh           Pengujian Go, PostgreSQL dan frontend
```

Alur backend: **handler → service → repository → PostgreSQL**. Panduan membaca kode dan contoh alur draft/submit ada di [backend/README.md](backend/README.md), dengan peta fitur di [docs/architecture.md](docs/architecture.md).

## Pengembangan dan pengujian

Butuh Node 22.13+ dan Go 1.26 untuk menjalankan tanpa Docker.

```sh
npm --prefix frontend ci
# Terminal backend, dari backend/:
APP_ORIGIN=http://localhost:5174 go run .
# Terminal frontend, dari root:
npm --prefix frontend run dev
```

Buka localhost:5174. Vite mem-proxy /api ke backend port 8080.

```sh
./scripts/test.sh
```

Pengujian mencakup database, isolasi akun, sesi, CSRF, penyimpanan jawaban, grading, submit idempoten, format Chat Completions, generasi JSON, lookup verb, pemangkasan riwayat, batas keluaran, saldo habis, API key invalid, jawaban terpotong, dan larangan meneruskan key melalui redirect. Tes menggunakan provider simulasi, bukan panggilan berbayar ke VPS Indonesia.

Akses VPS Indonesia langsung belum dapat diverifikasi sampai API key yang valid dipasang. Keberhasilan tes simulasi bukan jaminan kualitas, identitas model, atau ketersediaan layanan pihak ketiga.

## Miva: sesi belajar pilihanmu

Dari beranda, pilih **Atur belajar hari ini**. Satu sesi dapat berisi 0–30 kosakata, maksimal 3 dari 12 tenses, 0–5 reading, dan 0–5 listening. Pilih minimal satu materi. Jumlah latihan mengikuti pilihan awal, maksimal 30 soal, dengan minimal satu soal untuk setiap bahan reading/listening, tense yang dipilih, dan bagian kosakata. Pilihan tersimpan per akun di perangkat. Materi dibuat bertahap di server dan persiapannya tetap berjalan ketika halaman ditutup; jangan restart backend selama persiapan. Permintaan ulang dengan ID sesi yang sama tidak membuat materi berbayar dua kali.

Goal akun menjadi acuan setiap permintaan materi, contoh, latihan dan rangkuman. Kosakata sesi baru memiliki minimal tiga contoh, dan setiap tense memiliki enam contoh, aturan pemakaian, pola kalimat, serta kesalahan umum. Listening memiliki pertanyaan langsung di bawah tiap bahan. Jawaban disimpan otomatis dan dapat diteruskan dari riwayat. Setelah submit, server menghitung skor dari kunci yang dibuat bersama materi; koreksi setiap soal dan rangkuman pribadi ditampilkan. Jika layanan rangkuman sedang gagal, skor tetap tersimpan dan rangkuman bisa dicoba lagi. Riwayat menampilkan 50 sesi terbaru.

### Latihan membaca dengan suara

Secara default, gunakan pengenal ucapan browser (Chrome/Edge yang mendukung SpeechRecognition). Skor ini adalah **kecocokan kata**, bukan penilaian fonetik. Hasil pengenalan ucapan dapat berbeda dari ucapan sebenarnya. Pemutar contoh menggunakan suara perangkat, tersedia tanpa panggilan audio berbayar; kecepatan dan pilihan suara ada di bagian **Pengaturan suara** yang tertutup secara bawaan.

Untuk penilaian akustik yang sesungguhnya, backend mendukung Azure Speech Pronunciation Assessment melalui `AZURE_SPEECH_KEY` dan `AZURE_SPEECH_ENDPOINT` di `backend/.env`. Isi resource endpoint HTTPS atau regional Speech endpoint, lalu build/recreate container. Layanan ini terpisah dari key model teks dan bisa memiliki biaya sendiri. Ketika tersedia, reading merekam maksimal 30 detik, mengirim WAV mono 16 kHz ke pemeriksa suara, dan menampilkan skor akurasi, kelancaran, kelengkapan serta kata yang perlu dilatih. Rekaman tidak disimpan oleh Miva; hanya hasil penilaian tersimpan per akun. Detail protokol: https://learn.microsoft.com/en-us/azure/ai-services/speech-service/rest-speech-to-text-short

Perubahan jumlah materi berlaku untuk sesi baru. Sesi lama tetap dapat dilanjutkan tanpa dihapus atau dibuat ulang. Untuk membangun sekaligus menjalankan pembaruan:

```sh
docker compose up -d --build
```

## Deploy VPS dengan container terpisah

Gunakan `compose.vps.yaml` untuk backend Go dan frontend Svelte/Nginx yang terpisah. `compose.yaml` lama tetap tersedia untuk penggunaan lokal. Script Python `scripts/deploy.py` mendukung deploy langsung di VPS atau unggah lewat SSH, backup schema sebelum migrasi, penerapan migrasi bernomor otomatis, dan pemeriksaan kesehatan. Workflow push ke `main` tersedia pada `.github/workflows/deploy-vps.yml` setelah secrets/variables VPS diisi.

Panduan lengkap dan contoh konfigurasi: [docs/deploy-vps.md](docs/deploy-vps.md).
