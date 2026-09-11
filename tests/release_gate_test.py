import unittest
from release_gate import successful_run, validate_release, validate_acceptance


class ReleaseGateTests(unittest.TestCase):
    def test_pending_acceptance_blocks_stable(self):
        with self.assertRaises(ValueError):
            validate_acceptance({})

    def test_requires_published_prerelease_with_assets(self):
        for release in ({}, {"prerelease": False}, {"prerelease": True, "draft": True}, {"prerelease": True, "assets": []}):
            with self.assertRaises(ValueError):
                validate_release(release, "same", "same")

    def test_requires_exact_source(self):
        release = {"prerelease": True, "assets": [{"name": "candidate"}]}
        with self.assertRaises(ValueError):
            validate_release(release, "old", "new")
        validate_release(release, "same", "same")

    def test_new_failure_cannot_use_old_success(self):
        good = dict(head_sha="sha", head_repository={"full_name": "Uqda/Core"},
                    status="completed", conclusion="success", run_number=1)
        failed = dict(good, conclusion="failure", run_number=2)
        self.assertTrue(successful_run([good], "sha", "Uqda/Core"))
        self.assertFalse(successful_run([good, failed], "sha", "Uqda/Core"))
        self.assertFalse(successful_run([good], "different", "Uqda/Core"))
        self.assertFalse(successful_run([good], "sha", "other/repo"))


if __name__ == "__main__":
    unittest.main()
