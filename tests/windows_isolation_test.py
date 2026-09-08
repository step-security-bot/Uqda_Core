#!/usr/bin/env python3
"""Static regression checks; runtime coverage is in windows-installer.yml."""

from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parent.parent


def source(path):
    return (ROOT / path).read_text(encoding="utf-8")


class WindowsIsolationTests(unittest.TestCase):
    def test_msi_owns_new_family_and_components(self):
        msi = source("contrib/msi/build-msi.sh")
        for legacy in (
            "77757838-1a23-40a5-a720-c3b43e0260cc",
            "54a3294e-a441-4322-aefb-3bb40dd022bb",
            "c2119231-2aa3-4962-867a-9759c87beb24",
            "a916b730-974d-42a1-b687-d9d504cbb86a",
            "64a3733b-c98a-4732-85f3-20cd7da1a785",
        ):
            self.assertNotIn(legacy, msi.lower())
        self.assertIn('PKGGUID="c68fc7f4-9642-47e6-a8d5-a75e339d4318"', msi)
        self.assertEqual(len(re.findall(r'<Component Id="\w+" Guid="\*">', msi)), 4)
        for component in re.findall(r'<Component\b.*?</Component>', msi, re.S):
            self.assertEqual(len(re.findall(r'<File\b', component)), 1)
        self.assertIn("Installed OR NOT UQDA_EXISTING_SERVICE OR WIX_UPGRADE_DETECTED", msi)

    def test_no_global_driver_removal(self):
        tun = source("src/tun/tun_windows.go")
        self.assertNotIn("wintun.Uninstall", tun)
        self.assertNotIn("8f59971a-7872-4aa6-b2eb-061fc4e9d0a7", tun)
        self.assertIn("db97c42e-e485-4fbe-a30a-7d63b7409c16", tun)

    def test_config_preparation_is_privileged_and_checked(self):
        msi = source("contrib/msi/build-msi.sh")
        action = re.search(r'<CustomAction\s+Id="UpdateGenerateConfig".*?/>', msi, re.S).group()
        self.assertIn('Impersonate="no"', action)
        self.assertIn('Return="check"', action)
        self.assertIn('[SystemFolder]cmd.exe', action)
        self.assertIn('/d /c', action)
        self.assertIn('Before="StartServices"', msi)
        bat = source("contrib/msi/updateconfig.bat")
        self.assertIn('if exist "%UQDA_CONFIG_DIR%\\uqda.conf" goto validate', bat)
        self.assertIn('install-config-error.log', bat)
        self.assertIn('if errorlevel 1 goto failed_new', bat)
        self.assertIn('*S-1-5-18:(OI)(CI)F', bat)

    def test_windows_only_admin_default(self):
        self.assertIn('"tcp://localhost:19001"', source("src/config/defaults_windows.go"))


if __name__ == "__main__":
    unittest.main()
