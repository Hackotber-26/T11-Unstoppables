import importlib.util
import unittest
from pathlib import Path


SCRIPT = Path(__file__).resolve().parents[1] / "validate.py"
SPEC = importlib.util.spec_from_file_location("spec_validator", SCRIPT)
VALIDATOR = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(VALIDATOR)


class SpecValidationTests(unittest.TestCase):
    def test_repository_specs_are_consistent(self):
        self.assertEqual(VALIDATOR.validate(), [])


if __name__ == "__main__":
    unittest.main()