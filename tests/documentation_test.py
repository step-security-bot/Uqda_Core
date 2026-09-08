#!/usr/bin/env python3
"""Repository documentation consistency checks."""

from __future__ import annotations

import re
import subprocess
from pathlib import Path
from urllib.parse import unquote, urlsplit

ROOT = Path(__file__).resolve().parent.parent
SELF = Path(__file__).resolve()

RETIRED_REFERENCES = (
    "uqda-" + "gateway",
    "uqda-" + "latency",
    "docs/" + "HOME_GATEWAY.md",
    "docs/" + "CAFE_GATEWAY.md",
    "docs/" + "PERFORMANCE.md",
    "contrib/" + "gateway",
    "contrib/" + "performance",
)

MARKDOWN_LINK = re.compile(r"!?(?:\[[^\]]*\])\(([^)]+)\)")

NETWORK_GUIDE = ROOT / "docs" / "NETWORK_GUIDE_AR.md"
PLATFORM_FACTS = {
    "src/config/defaults_linux.go": (
        "unix:///var/run/uqda.sock",
        "/etc/uqda.conf",
        "65535",
    ),
    "src/config/defaults_darwin.go": (
        "unix:///var/run/uqda.sock",
        "/etc/uqda.conf",
        "65535",
    ),
    "src/config/defaults_windows.go": (
        "tcp://localhost:19001",
        "65535",
    ),
    "src/config/defaults_freebsd.go": (
        "/usr/local/etc/uqda.conf",
        "/dev/tun0",
        "32767",
    ),
    "src/config/defaults_openbsd.go": (
        "/etc/uqda.conf",
        "tun0",
        "16384",
    ),
}


def tracked_files() -> list[Path]:
    output = subprocess.check_output(
        ["git", "ls-files", "-z"], cwd=ROOT
    ).decode("utf-8")
    return [ROOT / name for name in output.rstrip("\0").split("\0") if name]


def check_retired_references(files: list[Path]) -> list[str]:
    errors: list[str] = []
    for path in files:
        if path == SELF or not path.is_file():
            continue
        try:
            text = path.read_text(encoding="utf-8")
        except UnicodeDecodeError:
            continue
        for reference in RETIRED_REFERENCES:
            if reference.lower() in text.lower():
                errors.append(
                    f"{path.relative_to(ROOT)} still references retired content: {reference}"
                )
    return errors


def check_markdown_links(files: list[Path]) -> list[str]:
    errors: list[str] = []
    for path in files:
        if path.suffix.lower() != ".md" or not path.is_file():
            continue
        text = path.read_text(encoding="utf-8")
        for match in MARKDOWN_LINK.finditer(text):
            raw = match.group(1).strip().strip("<>")
            if not raw or raw.startswith("#"):
                continue
            parsed = urlsplit(raw)
            if parsed.scheme or parsed.netloc or raw.startswith("/"):
                continue
            target_text = unquote(parsed.path)
            if not target_text:
                continue
            target = (path.parent / target_text).resolve()
            try:
                target.relative_to(ROOT)
            except ValueError:
                errors.append(
                    f"{path.relative_to(ROOT)} links outside the repository: {raw}"
                )
                continue
            if not target.exists():
                errors.append(
                    f"{path.relative_to(ROOT)} has a missing local link: {raw}"
                )
    return errors


def check_network_guide_platform_facts() -> list[str]:
    """Keep documented paths, endpoints, interface names and MTUs in sync."""
    errors: list[str] = []
    guide = NETWORK_GUIDE.read_text(encoding="utf-8")
    for source_name, facts in PLATFORM_FACTS.items():
        source = (ROOT / source_name).read_text(encoding="utf-8")
        for fact in facts:
            if fact not in source:
                errors.append(
                    f"{source_name} no longer contains documented platform fact: {fact}"
                )
            if fact not in guide:
                errors.append(
                    f"{NETWORK_GUIDE.relative_to(ROOT)} is missing platform fact: {fact}"
                )
    windows_doc_facts = (
        r"%ProgramData%\UQDA\uqda.conf",
        r"%ProgramData%\UQDA\uqda.log",
    )
    for fact in windows_doc_facts:
        if fact not in guide:
            errors.append(
                f"{NETWORK_GUIDE.relative_to(ROOT)} is missing Windows fact: {fact}"
            )
    return errors


def main() -> int:
    files = tracked_files()
    errors = check_retired_references(files)
    errors.extend(check_markdown_links(files))
    errors.extend(check_network_guide_platform_facts())
    if errors:
        for error in errors:
            print(f"documentation error: {error}")
        return 1
    print("documentation checks passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
