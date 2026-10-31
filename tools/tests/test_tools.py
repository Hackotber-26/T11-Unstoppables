import importlib.util
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]


def load_module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


SCENARIO = load_module("scenario_builder", ROOT / "tools" / "scenario-builder" / "build.py")
VECTORS = load_module("vector_generator", ROOT / "tools" / "vector-generator" / "generate.py")
BENCHMARK = load_module("benchmark_runner", ROOT / "tools" / "benchmark" / "run.py")
MIGRATIONS = load_module("migration_linter", ROOT / "tools" / "migration-tools" / "check.py")


class ToolsTests(unittest.TestCase):
    def test_scenario_derivation_is_valid_and_deterministic(self):
        template = ROOT / "datasets" / "scenarios" / "weekday_peak.json"
        first = SCENARIO.build_scenario(template, "tool_test", 73001, 42)
        second = SCENARIO.build_scenario(template, "tool_test", 73001, 42)
        self.assertEqual(first, second)
        self.assertEqual(first["rider_count"], 42)

    def test_vector_graph_has_shortest_path(self):
        graph = VECTORS.build_graph()
        route = VECTORS.shortest_path(graph, "ST-A1", "ST-A4")
        self.assertEqual(route["nodes"], ["ST-A1", "ST-A2", "ST-A3", "ST-A4"])
        self.assertEqual(route["travelSeconds"], 705)

    def test_benchmark_statistics_are_order_independent(self):
        self.assertEqual(BENCHMARK.percentile([50, 10, 30, 20, 40], 0.95), 50)

    def test_migration_linter_flags_destructive_sql(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "000001_drop_table.sql"
            path.write_text("DROP TABLE users;\n", encoding="utf-8")
            self.assertTrue(any("destructive SQL" in error for error in MIGRATIONS.check(Path(directory))))

    def test_repository_migrations_pass_static_lint(self):
        self.assertEqual(MIGRATIONS.check(ROOT / "backend" / "migrations"), [])


if __name__ == "__main__":
    unittest.main()