import importlib.util
import unittest
from pathlib import Path


SCRIPT = Path(__file__).resolve().parents[1] / "experiments" / "pooling_sensitivity.py"
SPEC = importlib.util.spec_from_file_location("pooling_sensitivity", SCRIPT)
EXPERIMENT = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(EXPERIMENT)


class PoolingSensitivityTests(unittest.TestCase):
    def test_analysis_is_deterministic(self):
        self.assertEqual(EXPERIMENT.analyze_all(), EXPERIMENT.analyze_all())

    def test_checked_in_exports_match_current_run(self):
        EXPERIMENT.check_exports(EXPERIMENT.analyze_all())

    def test_party_seats_obey_pool_capacity(self):
        requests = [
            {"station_id": "s1", "destination_corridor_id": "c1", "requested_offset_seconds": "0", "party_size": "2", "request_id": "a"},
            {"station_id": "s1", "destination_corridor_id": "c1", "requested_offset_seconds": "5", "party_size": "2", "request_id": "b"},
        ]
        groups = EXPERIMENT.form_groups(requests, capacity=3, max_wait=60)
        self.assertEqual([len(group) for group in groups], [1, 1])

    def test_accessible_demand_requires_compatible_driver(self):
        requests = [{
            "station_id": "s1", "destination_corridor_id": "c1", "requested_offset_seconds": "0",
            "party_size": "1", "request_id": "a", "accessibility_required": "true",
        }]
        result = EXPERIMENT.evaluate("scenario", requests, [], capacity=4, max_wait=60)
        self.assertEqual(result["requests_unserved"], 1)
        self.assertEqual(result["accessible_requests_unserved"], 1)


if __name__ == "__main__":
    unittest.main()