import importlib.util
import unittest
from pathlib import Path


SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "build.py"
SPEC = importlib.util.spec_from_file_location("dataset_builder", SCRIPT)
BUILDER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(BUILDER)


class DatasetBuilderTests(unittest.TestCase):
    def test_build_output_is_deterministic(self):
        self.assertEqual(BUILDER.generate_outputs(), BUILDER.generate_outputs())

    def test_scenario_rejects_bad_discount(self):
        scenario = BUILDER.load_scenarios()[0]
        scenario["pool_discount_bps"] = 10001
        with self.assertRaises(ValueError):
            BUILDER.validate_scenario(scenario)

    def test_checked_in_fixtures_match_sources(self):
        counts = BUILDER.validate_generated()
        self.assertEqual(counts["scenarios"], 5)
        self.assertGreater(counts["demand_requests"], 0)


if __name__ == "__main__":
    unittest.main()