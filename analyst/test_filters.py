import unittest

from analyst.filters import score_posting


class FilterTests(unittest.TestCase):
    def test_matching_remote_role_is_reviewable(self):
        profile = {
            "role_types": {"product_backend": ["backend engineer"]},
            "seniority": ["senior"], "remote_allowed": True,
            "remote_terms": ["remote"], "strong_skills": ["go", "python"],
            "minimum_score": 5,
        }
        result = score_posting("Senior Backend Engineer", "Remote", "Build services in Go and Python", profile)
        self.assertEqual(result.recommendation, "review")
        self.assertEqual(result.matched, ["go", "python"])

    def test_unrelated_role_is_skipped(self):
        self.assertEqual(score_posting("Designer", "New York", "Create visual systems", {}).recommendation, "skip")

    def test_exclusions_override_good_match(self):
        profile = {
            "role_types": {"product_backend": ["backend engineer"]},
            "seniority": ["senior"], "remote_allowed": True,
            "remote_terms": ["remote"], "strong_skills": ["python"],
            "excluded_keywords": ["machine learning"],
        }
        result = score_posting("Senior Backend Engineer", "Remote", "Machine learning and Python", profile)
        self.assertEqual(result.recommendation, "skip")


if __name__ == "__main__":
    unittest.main()
