#!/usr/bin/env python3
"""Patch the Pi runtime config with cabinet identity and API auth settings."""

from __future__ import annotations

import argparse
import ipaddress
import json
import os
from pathlib import Path
from urllib.parse import urlparse


def csv_values(raw: str) -> list[str]:
    return [item.strip() for item in raw.split(",") if item.strip()]


def require_text(name: str, value: str) -> str:
    value = value.strip()
    if not value:
        raise SystemExit(f"{name} is required")
    return value


def validate_profile(args: argparse.Namespace) -> None:
    parsed = urlparse(args.wire_host_url)
    if parsed.scheme not in {"http", "https"} or not parsed.netloc:
        raise SystemExit("wire_host_url must be a valid http/https URL")

    if not args.listener_dns_name and not args.listener_ip:
        raise SystemExit("listener_dns_name or listener_ip is required")

    for ip_value in csv_values(args.required_san_ips):
        try:
            ipaddress.ip_address(ip_value)
        except ValueError as exc:
            raise SystemExit(f"required_san_ips contains invalid IP {ip_value!r}") from exc

    if args.listener_ip:
        try:
            ipaddress.ip_address(args.listener_ip)
        except ValueError as exc:
            raise SystemExit(f"listener_ip is invalid: {args.listener_ip!r}") from exc


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config", required=True, help="source JSON config path")
    parser.add_argument("--output", required=True, help="patched JSON output path")
    parser.add_argument("--api-token-env", default="G2S_API_TOKEN", help="environment variable containing API token")
    parser.add_argument("--wire-host-url", required=True)
    parser.add_argument("--listener-dns-name", default="")
    parser.add_argument("--listener-ip", default="")
    parser.add_argument("--required-san-dns", default="")
    parser.add_argument("--required-san-ips", default="")
    parser.add_argument("--host-id", required=True)
    parser.add_argument("--first-test-egm-ids", required=True)
    args = parser.parse_args()

    token = require_text(args.api_token_env, os.environ.get(args.api_token_env, ""))
    validate_profile(args)

    with Path(args.config).open("r", encoding="utf-8") as handle:
        config = json.load(handle)

    config.setdefault("api", {})["auth_token"] = token
    config["cabinet_profile"] = {
        "wire_host_url": require_text("wire_host_url", args.wire_host_url),
        "listener_dns_name": args.listener_dns_name.strip(),
        "listener_ip": args.listener_ip.strip(),
        "required_san_dns": csv_values(args.required_san_dns),
        "required_san_ips": csv_values(args.required_san_ips),
        "host_id": require_text("host_id", args.host_id),
        "first_test_egm_ids": csv_values(args.first_test_egm_ids),
    }

    if not config["cabinet_profile"]["first_test_egm_ids"]:
        raise SystemExit("first_test_egm_ids must contain at least one EGM ID")
    if not config["cabinet_profile"]["required_san_dns"] and not config["cabinet_profile"]["required_san_ips"]:
        raise SystemExit("required_san_dns or required_san_ips must contain at least one value")

    output = Path(args.output)
    with output.open("w", encoding="utf-8") as handle:
        json.dump(config, handle, indent=2)
        handle.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
