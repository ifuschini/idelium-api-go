import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


class SmokeSchemaTests(unittest.TestCase):
    def test_audit_schema_supports_parallel_run_lifecycle_events(self):
        schema = (ROOT / "smoke" / "schema.sql").read_text(encoding="utf-8")
        for column in ("actorTenantId", "activeTenantId", "idProject", "action", "targetType", "targetId", "result", "sourceIp", "metadata"):
            self.assertIn(f"{column} ", schema)


if __name__ == "__main__":
    unittest.main()
