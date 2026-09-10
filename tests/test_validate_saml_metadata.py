import base64
import datetime as dt
import subprocess
import tempfile
import unittest
from pathlib import Path

from scripts.validate_saml_metadata import (
    cert_der,
    certificate_dates,
    entity_descriptor,
    validate_assertion,
)
import xml.etree.ElementTree as ET


ISSUER = "https://idp.example.invalid/metadata"
AUDIENCE = "https://api.idelium.org/saml/sp"


class SAMLMetadataValidatorTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.directory = Path(self.temp.name)
        self.key = self.directory / "idp.key"
        self.cert = self.directory / "idp.crt"
        subprocess.run(
            [
                "openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes",
                "-keyout", str(self.key), "-out", str(self.cert), "-days", "1",
                "-subj", "/CN=idp.example.invalid",
            ],
            check=True,
            capture_output=True,
        )

    def tearDown(self):
        self.temp.cleanup()

    def assertion(self, audience=AUDIENCE, issuer=ISSUER, expires_minutes=5):
        now = dt.datetime.now(dt.timezone.utc)
        path = self.directory / "assertion.xml"
        path.write_text(
            f'''<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion">
  <Issuer>{issuer}</Issuer>
  <Conditions NotBefore="{(now - dt.timedelta(minutes=1)).isoformat()}"
              NotOnOrAfter="{(now + dt.timedelta(minutes=expires_minutes)).isoformat()}">
    <AudienceRestriction><Audience>{audience}</Audience></AudienceRestriction>
  </Conditions>
</Assertion>''',
            encoding="utf-8",
        )
        return path, now

    def test_certificate_and_assertion_contract(self):
        encoded = base64.b64encode(cert_der(self.cert)).decode()
        root = ET.fromstring(
            f'''<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="{ISSUER}">
  <IDPSSODescriptor><KeyDescriptor use="signing"><KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#">
    <X509Data><X509Certificate>{encoded}</X509Certificate></X509Data>
  </KeyInfo></KeyDescriptor></IDPSSODescriptor>
</EntityDescriptor>'''
        )
        self.assertEqual(entity_descriptor(root, ISSUER).attrib["entityID"], ISSUER)
        not_before, not_after = certificate_dates(self.cert)
        self.assertLess(not_before, not_after)
        assertion, now = self.assertion()
        validate_assertion(assertion, ISSUER, AUDIENCE, now)

    def test_rejects_wrong_audience(self):
        assertion, now = self.assertion(audience="https://other.example.invalid/sp")
        with self.assertRaisesRegex(ValueError, "audience"):
            validate_assertion(assertion, ISSUER, AUDIENCE, now)

    def test_rejects_expired_assertion(self):
        assertion, now = self.assertion(expires_minutes=-1)
        with self.assertRaisesRegex(ValueError, "expired"):
            validate_assertion(assertion, ISSUER, AUDIENCE, now)


if __name__ == "__main__":
    unittest.main()
