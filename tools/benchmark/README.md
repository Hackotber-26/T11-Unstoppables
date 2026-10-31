# Benchmark Runner

`run.py` measures the in-memory synthetic pooling evaluator after warmup and reports median, p95, minimum, and maximum elapsed nanoseconds. Select a dataset scenario with `--scenario`; choose run counts with `--iterations` and `--warmup`.

Results are machine-specific and are not release thresholds or production capacity promises. An optional `--output` writes a JSON report; otherwise the report is printed to stdout.