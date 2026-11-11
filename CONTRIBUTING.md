# Membaca dan mengubah kode Miva

Mulai dari [peta arsitektur](docs/architecture.md) untuk memahami alur, identitas modul, dan lokasi fitur.

## Konvensi

- Komponen halaman memakai `PascalCasePage.svelte`; komponen yang dipakai ulang ada di `components/`.
- JavaScript memakai `camelCase` untuk state dan fungsi; Go mengikuti `gofmt` dan penamaan ekspor Go; Python memakai `snake_case` dan Black.
- Gunakan nama yang menerangkan peran. Komentar menjelaskan urutan, alasan atau batas penting, bukan mengulang tiap baris.
- Identitas `MIVA-*` diletakkan pada batas modul, bukan ditambahkan ke setiap variabel atau response.
- Pertahankan nama field JSON, URL API, cookie sesi dan ID SQL ketika merapikan kode. Perubahan kontrak harus disengaja dan diuji.
- SQL migrasi yang sudah diterapkan tidak boleh diubah, termasuk whitespace dan newline.
- Ikuti analisis dampak di `AGENTS.md` sebelum memindahkan atau mengubah fungsi.

## Menyiapkan alat pengembangan

```sh
npm --prefix frontend ci
python3 -m venv .venv
. .venv/bin/activate
python3 -m pip install -r requirements-dev.txt
```

## Format

```sh
make format
make format-check
```

Prettier menangani Svelte, JavaScript, CSS, JSON, YAML dan Markdown; Black menangani script Python; `scripts/format-go.sh` memakai gofmt host atau Docker Go 1.26 bila gofmt belum ada. Konfigurasi editor ada di `.editorconfig`. Secret env, dependency, build, petunjuk agen dan migrasi SQL tidak ikut diformat oleh Prettier.

## Pemeriksaan perilaku

```sh
./scripts/test.sh
python3 -m unittest discover -s scripts/tests -v
./scripts/test-deploy.sh
```

Tes terakhir memakai container/database sementara dan tidak memakai saldo API. Tes database aplikasi memakai role/schema khusus dari `backend/.env`; jangan mengarahkannya ke database yang tidak diizinkan. Jangan mengunggah file env atau backup ke repository.

Saat menambah fitur, mulai dari halaman → callback App → `api.js` → routing server → handler fitur → domain → penyimpanan. Pertahankan pemisahan itu dan perbarui peta kode bila alur berubah.
