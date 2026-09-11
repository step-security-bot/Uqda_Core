#!/usr/bin/env python3
"""Fail closed unless the exact stable source has successful prerelease evidence."""
import json
import os
import re
import subprocess
from pathlib import Path
from urllib.parse import quote


def api(path):
    return json.loads(subprocess.check_output(["gh", "api", path], text=True))


def validate_release(release, resolved_sha, expected_sha):
    if release.get("draft") or not release.get("prerelease"):
        raise ValueError("a published prerelease is required")
    if resolved_sha != expected_sha:
        raise ValueError("prerelease and stable source commits differ")
    if not release.get("assets"):
        raise ValueError("prerelease has no tested artifacts")


def successful_run(runs, expected_sha, repository):
    # Require the most recent run for this commit, not a superseded green run.
    eligible = [r for r in runs if r.get("head_sha") == expected_sha
                and r.get("head_repository", {}).get("full_name") == repository]
    if not eligible:
        return False
    latest = max(eligible, key=lambda r: (r.get("run_number", 0), r.get("run_attempt", 0)))
    return latest.get("status") == "completed" and latest.get("conclusion") == "success"


def validate_acceptance(checklist):
    required = ("historical_windows_migration", "windows_upgrade_rollback",
                "native_windows_x86_arm64", "native_bsd_and_supported_router_targets",
                "local_administration_security_review")
    for name in required:
        item = checklist.get(name, {})
        if item.get("passed") is not True or not item.get("evidence", "").startswith("https://github.com/"):
            raise ValueError(f"release acceptance is incomplete: {name}")


def main():
    root = Path(__file__).resolve().parent.parent
    validate_acceptance(json.loads((root / ".github/release-acceptance.json").read_text()))
    repository = os.environ["GITHUB_REPOSITORY"]
    tag = os.environ.get("TESTED_PRERELEASE", "")
    expected = os.environ["EXPECTED_SHA"]
    if not re.fullmatch(r"v\d+\.\d+\.\d+-(beta|rc)\.[1-9]\d*", tag):
        raise ValueError("provide an explicit tested beta or rc tag")
    release = api(f"repos/{repository}/releases/tags/{quote(tag, safe='')}")
    commit = api(f"repos/{repository}/commits/{quote(tag, safe='')}")
    validate_release(release, commit["sha"], expected)
    for workflow in ("ci.yml", "windows-installer.yml", "prerelease.yml"):
        result = api(f"repos/{repository}/actions/workflows/{workflow}/runs?head_sha={expected}&per_page=100")
        if not successful_run(result["workflow_runs"], expected, repository):
            raise ValueError(f"missing successful current validation: {workflow}")
    print("Exact-commit prerelease and CI gates passed")


if __name__ == "__main__":
    main()
