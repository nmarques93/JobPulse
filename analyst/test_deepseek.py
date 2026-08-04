import os
import unittest
from unittest.mock import patch

from analyst.deepseek import enrich


class DeepSeekTests(unittest.TestCase):
    @patch.dict(os.environ, {}, clear=True)
    def test_missing_key_skips_enrichment(self):
        self.assertIsNone(enrich({}, {}, {}))


if __name__ == "__main__":
    unittest.main()
