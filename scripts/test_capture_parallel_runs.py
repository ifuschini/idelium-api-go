import unittest

from capture_parallel_runs import extract, resolve


class CaptureHelpersTest(unittest.TestCase):
    def test_resolve_nested_placeholders(self):
        self.assertEqual(
            resolve({"path": "/runs/{{runTokenId}}", "body": ["{{runToken}}"]}, {"runTokenId": "idrt_1", "runToken": "idrt_1.secret"}),
            {"path": "/runs/idrt_1", "body": ["idrt_1.secret"]},
        )

    def test_extract_token(self):
        self.assertEqual(extract({"token": "idrt_1.secret"}, "token"), "idrt_1.secret")


if __name__ == "__main__":
    unittest.main()
