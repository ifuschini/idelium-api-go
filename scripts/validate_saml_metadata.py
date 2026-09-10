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
    "saml": "urn:oasis:names:tc:SAML:2.0:assertion",
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


def certificate_dates(path: Path) -> tuple[dt.datetime, dt.datetime]:
    result = subprocess.run(
        ["openssl", "x509", "-in", str(path), "-noout", "-startdate", "-enddate"],
        check=True,
        capture_output=True,
        text=True,
    )
    values = dict(line.split("=", 1) for line in result.stdout.splitlines())
    fmt = "%b %d %H:%M:%S %Y %Z"
    return (
        dt.datetime.strptime(values["notBefore"], fmt).replace(tzinfo=dt.timezone.utc),
        dt.datetime.strptime(values["notAfter"], fmt).replace(tzinfo=dt.timezone.utc),
    )


def entity_descriptor(root: ET.Element, issuer: str) -> ET.Element:
    candidates = [root] if root.tag == f"{{{NS['md']}}}EntityDescriptor" else []
    candidates.extend(root.findall(".//md:EntityDescriptor", NS))
    for candidate in candidates:
        if candidate.attrib.get("entityID") == issuer:
            return candidate
    raise ValueError("metadata entityID does not match the expected issuer")


def validate_assertion(path: Path, issuer: str, audience: str, now: dt.datetime) -> None:
    root = ET.parse(path).getroot()
    assertion_issuer = root.findtext("saml:Issuer", default="", namespaces=NS)
    if assertion_issuer != issuer:
        raise ValueError("assertion issuer does not match metadata entityID")
    audiences = {
        (item.text or "").strip()
        for item in root.findall(".//saml:AudienceRestriction/saml:Audience", NS)
    }
    if audience not in audiences:
        raise ValueError("assertion audience does not contain the expected audience")
    conditions = root.find("saml:Conditions", NS)
    if conditions is None:
        raise ValueError("assertion has no Conditions element")
    not_before = conditions.attrib.get("NotBefore")
    not_on_or_after = conditions.attrib.get("NotOnOrAfter")
    if not_before and now < parse_time(not_before):
        raise ValueError("assertion is not valid yet")
    if not not_on_or_after or now >= parse_time(not_on_or_after):
        raise ValueError("assertion has expired or lacks NotOnOrAfter")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("metadata", type=Path)
    parser.add_argument("certificate", type=Path)
    parser.add_argument("--issuer", required=True)
    parser.add_argument("--audience", required=True)
    parser.add_argument(
        "--assertion",
        required=True,
        type=Path,
        help="ephemeral SAML assertion used to validate issuer, audience, and Conditions",
    )
    parser.add_argument("--now", help="UTC ISO-8601 time for deterministic checks")
    args = parser.parse_args()

    if not args.metadata.is_file() or not args.certificate.is_file() or not args.assertion.is_file():
        parser.error("metadata, certificate, and assertion must be existing files")
    try:
        root = ET.parse(args.metadata).getroot()
        entity = entity_descriptor(root, args.issuer)
        entity_id = entity.attrib["entityID"]
        descriptors = entity.findall("md:IDPSSODescriptor", NS)
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
        valid_until = entity.attrib.get("validUntil") or root.attrib.get("validUntil")
        if valid_until and parse_time(valid_until) <= now:
            raise ValueError("metadata validUntil has expired")
        cert_not_before, cert_not_after = certificate_dates(args.certificate)
        if now < cert_not_before:
            raise ValueError("signing certificate is not valid yet")
        if now >= cert_not_after:
            raise ValueError("signing certificate has expired")
        validate_assertion(args.assertion, entity_id, args.audience, now)
        cert_text = subprocess.run(
            ["openssl", "x509", "-in", str(args.certificate), "-noout", "-subject", "-issuer", "-dates"],
            check=True,
            capture_output=True,
            text=True,
        ).stdout
        print(f"validated entityID/issuer: {entity_id}")
        print(f"validated assertion audience: {args.audience}")
        print(f"validated signing certificate SHA-256: {supplied_fp}")
        print("validated metadata/certificate correspondence")
        print(cert_text, end="")
        return 0
    except (ET.ParseError, ValueError, subprocess.CalledProcessError, base64.binascii.Error) as exc:
        print(f"SAML metadata validation failed: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
