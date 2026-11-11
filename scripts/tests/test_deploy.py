import importlib.util
import json
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location(
    "deploy", Path(__file__).resolve().parents[1] / "deploy.py"
)
deploy = importlib.util.module_from_spec(spec)
spec.loader.exec_module(deploy)

SETTINGS = {
    "DB_HOST": "db",
    "DB_PORT": "5432",
    "DB_NAME": "test",
    "DB_USER": "test",
    "DB_PASSWORD": "test-only-password",
    "APP_ORIGIN": "https://miva.example.com",
    "COOKIE_SECURE": "true",
}


class DeployTests(unittest.TestCase):
    def test_production_origin_and_cookie(self):
        deploy.validate_config({"services": {"backend": {"environment": SETTINGS}}})
        for changes in (
            {"APP_ORIGIN": "http://public.example.com"},
            {"COOKIE_SECURE": "false"},
            {"DB_PASSWORD": "replace-with-password"},
        ):
            with self.assertRaises(ValueError):
                deploy.validate_config(
                    {"services": {"backend": {"environment": {**SETTINGS, **changes}}}}
                )

    def exercise(self, failure=None):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            (root / "deploy").mkdir()
            env = root / "deploy/.env"
            env.write_text("\n".join(f"{k}={v}" for k, v in SETTINGS.items()))
            args = deploy.parse_args(
                ["--env-file", str(env), "--release", "test-release"]
            )
            commands = []

            def fake_run(command, **kwargs):
                commands.append(command)
                if "config" in command:
                    return subprocess.CompletedProcess(
                        command,
                        0,
                        json.dumps(
                            {"services": {"backend": {"environment": SETTINGS}}}
                        ).encode(),
                    )
                phase = (
                    command[-1]
                    if command[-1] in ("migrate", "db-backup")
                    else ("up" if "up" in command else "build")
                )
                if phase == failure:
                    raise subprocess.CalledProcessError(1, command)
                if phase == "db-backup":
                    kwargs["stdout"].write(b"PGDMP-test-backup")
                return subprocess.CompletedProcess(command, 0)

            with patch.object(deploy, "ROOT", root), patch.object(
                deploy, "run", fake_run
            ):
                if failure:
                    with self.assertRaises(subprocess.CalledProcessError):
                        deploy.local_deploy(args)
                else:
                    deploy.local_deploy(args)
            state = root / "deploy/.deploy-state.json"
            self.assertEqual(state.exists(), failure is None)
            if failure != "db-backup":
                self.assertTrue((root / "deploy/backups/test-release.dump").is_file())
            return commands

    def test_order_and_health_gate(self):
        commands = self.exercise()
        backup = next(i for i, c in enumerate(commands) if c[-1] == "db-backup")
        migration = next(i for i, c in enumerate(commands) if c[-1] == "migrate")
        start = next(i for i, c in enumerate(commands) if "up" in c)
        self.assertLess(backup, migration)
        self.assertLess(migration, start)
        self.assertIn("--wait", commands[start])

    def test_backup_failure_never_migrates(self):
        commands = self.exercise("db-backup")
        self.assertFalse(any("migrate" == c[-1] or "up" in c for c in commands))

    def test_migration_failure_never_replaces_app(self):
        self.assertFalse(any("up" in c for c in self.exercise("migrate")))

    def test_unhealthy_release_is_not_marked_successful(self):
        self.exercise("up")

    def test_archive_excludes_secrets_backups_and_symlinks(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            for file in (
                "backend/main.go",
                "backend/.env",
                "deploy/.env",
                "deploy/.env.example",
                "deploy/production.env",
                "deploy/credentials.custom",
                "deploy/backups/old.dump",
                "frontend/node_modules/pkg/index.js",
                "frontend/src/App.svelte",
                "deploy/.deploy-state.json",
                ".git/config",
            ):
                path = root / file
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text("fixture")
            (root / "backend/secret-link.go").symlink_to(root / "backend/.env")
            archive = root / "test.tar.gz"
            with patch.object(deploy, "ROOT", root):
                deploy.package_source(archive, [root / "deploy/credentials.custom"])
            with tarfile.open(archive) as handle:
                self.assertEqual(
                    set(handle.getnames()),
                    {
                        "backend/main.go",
                        "deploy/.env.example",
                        "frontend/src/App.svelte",
                    },
                )


if __name__ == "__main__":
    unittest.main()
