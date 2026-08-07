import re
import unittest
from pathlib import Path


ROOT = Path(__file__).parents[1]
ADR_DIR = ROOT / "docs" / "adr"
REQUIRED_SECTIONS = {
    "Context",
    "Decision",
    "Alternatives considered",
    "Consequences",
    "Security and tenant isolation",
    "Compatibility",
    "Deployment and rollback",
    "Verification",
}


class ArchitectureDecisionRecordTest(unittest.TestCase):
    def records(self):
        return sorted(ADR_DIR.glob("[0-9][0-9][0-9][0-9]-*.md"))

    def test_adrs_have_unique_monotonic_numbers(self):
        records = self.records()
        numbers = [int(record.name[:4]) for record in records]
        self.assertTrue(records)
        self.assertEqual(numbers, sorted(set(numbers)))

    def test_adrs_include_governance_and_safety_sections(self):
        for record in self.records():
            with self.subTest(record=record.name):
                source = record.read_text(encoding="utf-8")
                sections = set(re.findall(r"^## (.+)$", source, re.MULTILINE))
                self.assertTrue(REQUIRED_SECTIONS <= sections)
                self.assertIsNotNone(
                    re.search(
                        r"^- Status: (Proposed|Accepted|Deprecated|Superseded by ADR-[0-9]{4})$",
                        source,
                        re.MULTILINE,
                    )
                )
                self.assertIn("dual writes", source.lower())
                self.assertIn("rollback", source.lower())
                self.assertIn("tenant", source.lower())

    def test_first_decision_accepts_strangler_and_laravel_fallback(self):
        source = (ADR_DIR / "0001-strangler-migration-model.md").read_text(
            encoding="utf-8"
        )
        self.assertIn("- Status: Accepted", source)
        self.assertIn("route-level strangler model", source)
        self.assertIn("Laravel remains the fallback", source)
        self.assertIn("Application-level dual writes are prohibited", source)


if __name__ == "__main__":
    unittest.main()
