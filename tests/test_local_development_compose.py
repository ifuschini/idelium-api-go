import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


class LocalDevelopmentComposeTests(unittest.TestCase):
    def test_compose_starts_go_laravel_and_seeded_mariadb(self):
        compose = (ROOT / "docker-compose.yml").read_text(encoding="utf-8")

        for required in (
            "  db:",
            "  api-go:",
            "  api-laravel:",
            "mariadb:10.6.22@sha256:",
            "./smoke/schema.sql:/docker-entrypoint-initdb.d/001-schema.sql:ro",
            "./smoke/seed.sql:/docker-entrypoint-initdb.d/002-seed.sql:ro",
            "IDELIUM_DB_PASSWORD_FILE: /run/secrets/db_password",
            "file: ./secrets/dev_db_password",
        ):
            self.assertIn(required, compose)

    def test_make_targets_and_readme_document_local_startup(self):
        makefile = (ROOT / "Makefile").read_text(encoding="utf-8")
        readme = (ROOT / "README.md").read_text(encoding="utf-8")
        gitignore = (ROOT / ".gitignore").read_text(encoding="utf-8")

        self.assertIn("dev-up:", makefile)
        self.assertIn("dev-down:", makefile)
        self.assertIn("chmod 0444 secrets/dev_db_password", makefile)
        self.assertIn("## Quick start", readme)
        self.assertIn("make dev-up", readme)
        self.assertIn("make dev-down", readme)
        self.assertIn("/secrets/dev_db_password", gitignore)


if __name__ == "__main__":
    unittest.main()
