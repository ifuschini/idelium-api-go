import sys
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from scripts.check_markdown_links import check_link, destinations, find_broken_links  # noqa: E402


class MarkdownLinkTests(unittest.TestCase):
    def test_repository_markdown_links_are_valid(self):
        self.assertEqual([], find_broken_links(ROOT))

    def test_docs_index_links_to_every_document(self):
        index = ROOT / "docs" / "README.md"
        source = index.read_text(encoding="utf-8")
        indexed = {
            ((index.parent / destination.split("#", 1)[0].split("?", 1)[0]).resolve())
            for destination in destinations(source)
            if not destination.startswith(("http://", "https://", "mailto:"))
        }
        docs_root = (ROOT / "docs").resolve()
        indexed = {path for path in indexed if docs_root in path.parents}
        expected = {
            path.resolve()
            for path in (ROOT / "docs").rglob("*")
            if path.is_file() and path != index
        }
        self.assertEqual(expected, indexed)

    def test_rejects_missing_targets_and_fragments(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            source = root / "README.md"
            target = root / "target.md"
            source.write_text("", encoding="utf-8")
            target.write_text("# Existing heading\n", encoding="utf-8")

            self.assertIsNotNone(check_link(source, "missing.md", root))
            self.assertIsNotNone(check_link(source, "target.md#missing", root))
            self.assertIsNone(check_link(source, "target.md#existing-heading", root))


if __name__ == "__main__":
    unittest.main()
