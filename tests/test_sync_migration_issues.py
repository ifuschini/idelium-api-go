"""Tests for migration issue materialization."""

from __future__ import annotations

import importlib.util
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
MODULE_PATH = ROOT / "scripts" / "sync_migration_issues.py"
SPEC = importlib.util.spec_from_file_location("sync_migration_issues", MODULE_PATH)
assert SPEC and SPEC.loader
MODULE = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class MigrationIssuePlanTests(unittest.TestCase):
    def setUp(self) -> None:
        self.waves = MODULE.parse_backlog(ROOT / "docs" / "migration" / "epics.md")

    def test_materializes_complete_three_level_plan(self) -> None:
        self.assertEqual(11, len(self.waves))
        self.assertEqual(60, sum(len(wave.tracks) for wave in self.waves))
        self.assertEqual(107, sum(len(wave.tickets) for wave in self.waves))
        self.assertFalse(
            [
                (wave.number, track.name)
                for wave in self.waves
                for track in wave.tracks
                if not track.tickets
            ]
        )

    def test_routes_representative_tickets_to_expected_tracks(self) -> None:
        expected = {
            "Migrate browser-version reads": "Platform catalog reads",
            "Add CLI smoke tests that can target Laravel or Go by route owner": "Web and CLI smoke targeting",
            "Migrate project reads and writes": "Projects",
            "Migrate run-token issue, validation, and revocation": "Run tokens",
            "Verify empty installs using Go migrations": "Empty install and upgrade verification",
        }
        actual = {
            ticket: track.name
            for wave in self.waves
            for track in wave.tracks
            for ticket in track.tickets
        }
        for ticket, track in expected.items():
            self.assertEqual(track, actual[ticket])


if __name__ == "__main__":
    unittest.main()
