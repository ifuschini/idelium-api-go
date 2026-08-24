import pathlib
import re
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


class RepositoryFoundationTests(unittest.TestCase):
    def test_apache_license_is_complete_and_attributed(self) -> None:
        license_text = (ROOT / "LICENSE").read_text(encoding="utf-8")

        self.assertIn("Apache License", license_text)
        self.assertIn("Version 2.0, January 2004", license_text)
        self.assertIn("Copyright 2026 Idelium contributors", license_text)
        self.assertIn("END OF TERMS AND CONDITIONS", license_text)

    def test_repository_directives_preserve_security_and_quality_gates(self) -> None:
        directives = (ROOT / "AGENTS.md").read_text(encoding="utf-8")

        for required_rule in (
            "Use English",
            "tenant scope",
            "OpenAPI first",
            "negative cross-tenant tests",
            "Never log credentials",
            "make verify",
        ):
            self.assertIn(required_rule, directives)

    def test_makefile_exposes_required_local_verification_targets(self) -> None:
        makefile = (ROOT / "Makefile").read_text(encoding="utf-8")

        for target in (
            "build:",
            "contract-test:",
            "format-check:",
            "integration-test:",
            "smoke-targets-check:",
            "test-race:",
            "vet:",
            "verify:",
        ):
            self.assertIn(target, makefile)
        self.assertRegex(
            makefile,
            re.compile(
                r"^verify:.*format-check.*vet.*test.*test-race.*contract-test.*openapi-check.*smoke-targets-check.*build",
                re.MULTILINE,
            ),
        )

    def test_dockerfile_is_pinned_reproducible_and_non_root(self) -> None:
        dockerfile = (ROOT / "Dockerfile").read_text(encoding="utf-8")
        first_line = dockerfile.splitlines()[0]

        self.assertRegex(first_line, r"^FROM golang:[^\s]+@sha256:[0-9a-f]{64} AS builder$")
        self.assertIn("RUN go mod download && go mod verify", dockerfile)
        self.assertIn("-trimpath", dockerfile)
        self.assertIn("FROM scratch", dockerfile)
        self.assertIn("USER 65532:65532", dockerfile)
        self.assertNotRegex(dockerfile, r"(?m)^FROM [^\n]*(?::latest)(?:\s|$)")

    def test_readme_documents_scope_verification_and_license(self) -> None:
        readme = (ROOT / "README.md").read_text(encoding="utf-8")

        self.assertIn("## Current scope", readme)
        self.assertIn("## Local verification", readme)
        self.assertIn("make verify", readme)
        self.assertIn("make integration-test", readme)
        self.assertIn("Apache License 2.0", readme)
        self.assertNotIn("idelium.io", readme)


if __name__ == "__main__":
    unittest.main()
