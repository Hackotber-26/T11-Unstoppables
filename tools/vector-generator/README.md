# Vector Generator

`generate.py` converts the fictional station and directed route CSVs into a JSON graph using the coordinate contract in `spec/vectors/vector-contract.json`. It validates node/edge references and full directed reachability; optional `--path START DESTINATION` includes a shortest-travel-time route.

Use `--check` to verify the generated JSON is byte-for-byte current. All outputs remain classified synthetic and use local meter coordinates, not GPS.