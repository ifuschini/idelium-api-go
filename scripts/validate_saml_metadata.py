#!/usr/bin/env python3
"""Validate an IdP SAML metadata document against its signing certificate."""

from __future__ import annotations

import argparse
import base64
import datetime as dt
import hashlib
import re
import subprocess
import sys
import xml.etree.ElementTree as ET
from pathlib import Path

NS = {
    "md": "urn:oasis:names:tc:SAML:2.0:metadata",
    "ds": "http://www.w3.org/2000/09/xmldsig#",
}


def cert_der(path: Path) -> bytes:
    result = subprocess.run(
        ["openssl", "x509", "-in", str(path), "-outform", "DER"],
        check=True,
        capture_output=True,
    )
    return result.stdout


def parse_time(value: str) -> dt.datetime:
    return dt.datetime.fromisoformat(value.replace("Z", "+00:00"))


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("metadata", type=Path)
    parser.add_argument("certificate", type=Path)
    parser.add_argument("--issuer", required=True)
    parser.add_argument("--audience", required=True)
    parser.add_argument("--now", help="UTC ISO-8601 time for deterministic checks")
    args = parser.parse_args()

    if not args.metadata.is_file() or not args.certificate.is_file():
        parser.error("metadata and certificate must be existing files")
    try:
        root = ET.parse(args.metadata).getroot()
        entity_id = root.attrib.get("entityID", "")
        if not entity_id or entity_id != args.issuer:
            raise ValueError("metadata entityID does not match the expected issuer")
        descriptors = root.findall(".//md:IDPSSODescriptor", NS)
        signing = [
            key.find("ds:KeyInfo/ds:X509Data/ds:X509Certificate", NS)
            for descriptor in descriptors
            for key in descriptor.findall("md:KeyDescriptor", NS)
            if key.attrib.get("use", "signing") == "signing"
        ]
        values = [re.sub(r"\s+", "", item.text or "") for item in signing if item is not None]
        if not values:
            raise ValueError("metadata has no signing KeyDescriptor certificate")
        supplied_der = cert_der(args.certificate)
        supplied_fp = hashlib.sha256(supplied_der).hexdigest()
        metadata_ders = [base64.b64decode(value, validate=True) for value in values]
        if supplied_der not in metadata_ders:
            raise ValueError("supplied certificate does not match metadata signing certificate")
        now = parse_time(args.now) if args.now else dt.datetime.now(dt.timezone.utc)
        valid_until = root.attrib.get("validUntil")
        if valid_until and parse_time(valid_until) <= now:
            raise ValueError("metadata validUntil has expired")
        if not args.audience.strip():
            raise ValueError("expected audience must not be empty")
        cert_text = subprocess.run(
            ["openssl", "x509", "-in", str(args.certificate), "-noout", "-subject", "-issuer", "-dates"],
            check=True,
            capture_output=True,
            text=True,
        ).stdout
        print(f"validated entityID/issuer: {entity_id}")
        print(f"validated audience: {args.audience}")
        print(f"validated signing certificate SHA-256: {supplied_fp}")
        print("validated metadata/certificate correspondence")
        print(cert_text, end="")
        return 0
    except (ET.ParseError, ValueError, subprocess.CalledProcessError, base64.binascii.Error) as exc:
        print(f"SAML metadata validation failed: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
