#!/usr/bin/env python3
# MIVA-DEPLOY: verified source upload, backup, migration, startup and health gate.
"""Deploy Miva locally on a VPS, or upload a release over verified SSH (stdlib only)."""
from __future__ import annotations
import argparse
import datetime as dt
import fcntl
import json
import os
from pathlib import Path, PurePosixPath
import re
import shlex
import subprocess
import sys
import tarfile
import tempfile
from urllib.parse import urlparse

ROOT = Path(__file__).resolve().parents[1]


def run(args, **kwargs):
    return subprocess.run([str(arg) for arg in args], check=True, **kwargs)


def env_values(path):
    values = {}
    for line in path.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        if line.startswith("export "):
            line = line[7:]
        key, sep, value = line.partition("=")
        if not sep or not re.fullmatch(r"[A-Za-z_][A-Za-z0-9_]*", key.strip()):
            raise ValueError("Format environment belum valid.")
        value = value.strip()
        if value[:1] in ('"', "'") and value[-1:] == value[:1]:
            value = value[1:-1]
        else:
            value = value.split(" #", 1)[0].rstrip()
        values[key.strip()] = value
    return values


def compose_command(env_file, project, values):
    cmd = [
        "docker",
        "compose",
        "--project-name",
        project,
        "--env-file",
        str(env_file),
        "-f",
        str(ROOT / "compose.vps.yaml"),
    ]
    if values.get("DB_NETWORK"):
        cmd += ["-f", str(ROOT / "deploy/compose.database-network.yaml")]
    return cmd


def validate_config(config):
    settings = config["services"]["backend"]["environment"]
    for key in (
        "DB_HOST",
        "DB_PORT",
        "DB_NAME",
        "DB_USER",
        "DB_PASSWORD",
        "APP_ORIGIN",
    ):
        value = str(settings.get(key, ""))
        if not value or "replace-with" in value:
            raise ValueError(f"Isi {key} di environment VPS terlebih dahulu.")
    origin = urlparse(settings["APP_ORIGIN"])
    if (
        origin.scheme not in ("http", "https")
        or not origin.netloc
        or origin.username
        or origin.password
        or origin.query
        or origin.fragment
        or origin.path not in ("", "/")
    ):
        raise ValueError(
            "APP_ORIGIN harus berupa origin domain, tanpa path atau kredensial."
        )
    if origin.scheme != "https" and origin.hostname not in (
        "localhost",
        "127.0.0.1",
        "::1",
    ):
        raise ValueError(
            "Gunakan HTTPS pada domain VPS agar mikrofon dan cookie bekerja."
        )
    if (
        origin.scheme == "https"
        and str(settings.get("COOKIE_SECURE", "")).lower() != "true"
    ):
        raise ValueError("COOKIE_SECURE harus true untuk HTTPS.")


def package_source(archive_path, excluded_paths=()):
    excluded_paths = {Path(path).resolve() for path in excluded_paths}
    allowed_dirs = {"backend", "frontend", "deploy", "scripts", "docs"}
    allowed_files = {"compose.vps.yaml", ".dockerignore", "README.md"}
    excluded = {
        "node_modules",
        "dist",
        ".git",
        "__pycache__",
        "backups",
        ".gitnexus",
        ".venv",
    }
    with tarfile.open(archive_path, "w:gz") as archive:
        for path in sorted(ROOT.rglob("*")):
            rel = path.relative_to(ROOT)
            if rel.parts[0] not in allowed_dirs and str(rel) not in allowed_files:
                continue
            if (
                any(part in excluded for part in rel.parts)
                or path.is_symlink()
                or not path.is_file()
            ):
                continue
            if path.resolve() in excluded_paths:
                continue
            if (
                path.name.startswith(".env") or path.suffix == ".env"
            ) and not path.name.endswith(".example"):
                continue
            if path.name.startswith((".deploy", ".qa-")) or path.suffix in (
                ".pem",
                ".key",
                ".p12",
                ".pfx",
                ".dump",
                ".partial",
                ".pyc",
            ):
                continue
            archive.add(path, arcname=str(rel), recursive=False)


def local_deploy(args):
    env_file = Path(args.env_file or ROOT / "deploy/.env").resolve()
    if not env_file.is_file():
        raise ValueError(
            f"Environment belum ada: {env_file}. Salin deploy/.env.example dan isi nilainya."
        )
    env_file.chmod(0o600)
    values = env_values(env_file)
    child_env = os.environ.copy()
    # Use the selected file consistently for both Compose interpolation and env_file.
    for key in values:
        child_env.pop(key, None)
    child_env.update(MIVA_ENV_FILE=str(env_file), MIVA_RELEASE=args.release)
    cmd = compose_command(env_file, args.project_name, values)
    backups = Path(args.backups_dir or ROOT / "deploy/backups").resolve()
    lock_path = Path(args.lock_file or ROOT / "deploy/.deploy.lock").resolve()
    lock_path.parent.mkdir(parents=True, exist_ok=True)
    with lock_path.open("a") as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            raise ValueError(
                "Deploy lain sedang berjalan; tunggu hingga selesai."
            ) from None
        lock_path.chmod(0o600)
        print(f"Memeriksa konfigurasi release {args.release}…", flush=True)
        result = run(
            cmd + ["config", "--format", "json"], env=child_env, capture_output=True
        )
        validate_config(json.loads(result.stdout))
        print("Membangun image backend dan frontend…", flush=True)
        run(cmd + ["build", "backend", "frontend"], env=child_env)
        print("Mencadangkan schema english_app sebelum migrasi…", flush=True)
        backups.mkdir(parents=True, exist_ok=True, mode=0o700)
        temporary = backups / f"{args.release}.dump.partial"
        try:
            descriptor = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
            with os.fdopen(descriptor, "wb") as handle:
                run(
                    cmd + ["run", "--rm", "-T", "--no-deps", "db-backup"],
                    env=child_env,
                    stdout=handle,
                )
            if temporary.stat().st_size:
                backup = backups / f"{args.release}.dump"
                temporary.replace(backup)
                print(f"Backup tersimpan: {backup}", flush=True)
            else:
                temporary.unlink()
                print("Schema belum ada; ini pemasangan pertama.", flush=True)
        except BaseException:
            temporary.unlink(missing_ok=True)
            raise
        print(
            "Menjalankan migrasi; aplikasi lama tetap berjalan sampai langkah ini sukses…",
            flush=True,
        )
        run(cmd + ["run", "--rm", "-T", "--no-deps", "migrate"], env=child_env)
        print(
            "Menjalankan kedua container dan menunggu pemeriksaan kesehatan…",
            flush=True,
        )
        try:
            run(
                cmd
                + [
                    "up",
                    "-d",
                    "--no-deps",
                    "--wait",
                    "--wait-timeout",
                    "180",
                    "backend",
                    "frontend",
                ],
                env=child_env,
            )
        except subprocess.CalledProcessError:
            print(
                "Container baru belum sehat. Backup tetap tersedia; database tidak di-rollback otomatis. Periksa docker compose logs.",
                file=sys.stderr,
            )
            raise
        state = lock_path.parent / ".deploy-state.json"
        previous = json.loads(state.read_text()) if state.exists() else None
        state_tmp = state.with_suffix(".tmp")
        state_tmp.write_text(
            json.dumps(
                {
                    "release": args.release,
                    "previousRelease": previous.get("release") if previous else None,
                    "deployedAt": dt.datetime.now(dt.timezone.utc).isoformat(),
                },
                indent=2,
            )
            + "\n"
        )
        state_tmp.replace(state)
        print(f"Deploy {args.release} selesai; frontend dan backend sehat.", flush=True)


def remote_deploy(args):
    if not re.fullmatch(r"[A-Za-z0-9_.@:\[\]-]+", args.host) or args.host.startswith(
        "-"
    ):
        raise ValueError("Alamat SSH belum valid.")
    remote = PurePosixPath(args.remote_dir)
    if not remote.is_absolute() or ".." in remote.parts or str(remote) == "/":
        raise ValueError(
            "--remote-dir harus direktori absolut khusus aplikasi, misalnya /opt/miva."
        )
    ssh = [
        "ssh",
        "-o",
        "BatchMode=yes",
        "-o",
        "StrictHostKeyChecking=yes",
        "-p",
        str(args.ssh_port),
    ]
    if args.identity:
        ssh += ["-i", str(Path(args.identity).expanduser())]
    ssh += [args.host]
    release_dir = str(remote / "releases" / args.release)
    shared = remote / "shared"
    remote_env = str(shared / ".env")
    run(ssh + ["mkdir -p " + shlex.join([release_dir, str(shared)])])
    if args.env_file:
        env_file = Path(args.env_file).resolve()
        if not env_file.is_file():
            raise ValueError("File environment VPS tidak ditemukan.")
        print(
            "Mengirim environment pilihanmu melalui SSH ke shared/.env (izin 600)…",
            flush=True,
        )
        with env_file.open("rb") as handle:
            run(
                ssh
                + [
                    f'set -eu; umask 077; cat > {shlex.quote(remote_env + ".tmp")}; mv {shlex.quote(remote_env + ".tmp")} {shlex.quote(remote_env)}'
                ],
                stdin=handle,
            )
    run(ssh + ["test -f " + shlex.quote(remote_env)])
    print(
        f"Mengirim release {args.release}; file rahasia dan backup tidak masuk arsip…",
        flush=True,
    )
    with tempfile.TemporaryDirectory(prefix="miva-deploy-") as temp:
        archive = Path(temp) / "release.tar.gz"
        package_source(archive, [args.env_file] if args.env_file else [])
        with archive.open("rb") as handle:
            run(ssh + ["tar -xzf - -C " + shlex.quote(release_dir)], stdin=handle)
    command = [
        "python3",
        release_dir + "/scripts/deploy.py",
        "--env-file",
        remote_env,
        "--release",
        args.release,
        "--project-name",
        args.project_name,
        "--backups-dir",
        str(shared / "backups"),
        "--lock-file",
        str(shared / "deploy.lock"),
    ]
    run(ssh + [shlex.join(command)])
    print(f"Release aktif: {release_dir}", flush=True)


def parse_args(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--host", help="user@alamat-vps; kosongkan untuk menjalankan langsung di VPS"
    )
    parser.add_argument("--remote-dir", default="/opt/miva")
    parser.add_argument("--ssh-port", type=int, default=22)
    parser.add_argument(
        "--identity", help="Path private key SSH (opsional; dapat memakai SSH agent)"
    )
    parser.add_argument(
        "--env-file",
        help="Environment VPS; saat remote hanya file ini yang dikirim secara terpisah",
    )
    parser.add_argument("--backups-dir")
    parser.add_argument("--lock-file")
    parser.add_argument("--project-name", default="miva")
    parser.add_argument(
        "--release",
        default=dt.datetime.now(dt.timezone.utc).strftime("%Y%m%dT%H%M%S%fZ"),
    )
    args = parser.parse_args(argv)
    if not re.fullmatch(r"[a-z0-9][a-z0-9_-]{0,62}", args.project_name):
        parser.error(
            "--project-name harus memakai huruf kecil, angka, underscore atau tanda minus"
        )
    if not re.fullmatch(r"[A-Za-z0-9_][A-Za-z0-9_.-]{0,100}", args.release):
        parser.error("--release belum valid untuk tag image")
    if not 1 <= args.ssh_port <= 65535:
        parser.error("--ssh-port harus 1–65535")
    return args


def main():
    try:
        args = parse_args()
        remote_deploy(args) if args.host else local_deploy(args)
    except subprocess.CalledProcessError:
        print(
            "Deploy dihentikan karena satu langkah gagal. Jangan menghapus volume database.",
            file=sys.stderr,
        )
        return 1
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"Deploy dihentikan: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
