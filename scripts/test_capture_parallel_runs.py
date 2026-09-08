import unittest

from capture_parallel_runs import extract, load_cookie_header, resolve


class CaptureHelpersTest(unittest.TestCase):
    def test_resolve_nested_placeholders(self):
        self.assertEqual(
            resolve({"path": "/runs/{{runTokenId}}", "body": ["{{runToken}}"]}, {"runTokenId": "idrt_1", "runToken": "idrt_1.secret"}),
            {"path": "/runs/idrt_1", "body": ["idrt_1.secret"]},
        )

    def test_extract_token(self):
        self.assertEqual(extract({"token": "idrt_1.secret"}, "token"), "idrt_1.secret")

    def test_load_cookie_header_from_netscape_jar(self):
        from tempfile import NamedTemporaryFile

        with NamedTemporaryFile(mode="w+", encoding="utf-8") as jar:
            jar.write("# Netscape HTTP Cookie File\nlocalhost\tFALSE\t/\tFALSE\t0\tidelium_session\tsynthetic\n")
            jar.flush()
            self.assertEqual(load_cookie_header(jar.name), "idelium_session=synthetic")


if __name__ == "__main__":
    unittest.main()
