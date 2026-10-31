package vectors

import (
	"encoding/json"
	"testing"
)

type graph struct {
	Classification        string `json:"classification"`
	CoordinateReference   string `json:"coordinateReference"`
	GeographicCoordinates bool   `json:"geographicCoordinates"`
	Nodes                 []struct {
		ID string `json:"id"`
	} `json:"nodes"`
	Edges []struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"edges"`
	ExampleRoute struct {
		Nodes         []string `json:"nodes"`
		TravelSeconds int      `json:"travelSeconds"`
	} `json:"exampleRoute"`
}

func TestGeneratedVectorFixtureContract(t *testing.T) {
	contents := []byte(`{
		"classification":"synthetic",
		"coordinateReference":"synthetic-local-cartesian",
		"geographicCoordinates":false,
		"nodes":[{"id":"ST-A1"},{"id":"ST-A2"},{"id":"ST-A3"}],
		"edges":[{"from":"ST-A1","to":"ST-A2"},{"from":"ST-A2","to":"ST-A3"}],
		"exampleRoute":{"nodes":["ST-A1","ST-A2","ST-A3"],"travelSeconds":450}
	}`)
	var result graph
	if err := json.Unmarshal(contents, &result); err != nil {
		t.Fatal(err)
	}
	if result.Classification != "synthetic" || result.CoordinateReference != "synthetic-local-cartesian" || result.GeographicCoordinates {
		t.Fatalf("vector fixture has incorrect provenance: %+v", result)
	}
	if len(result.Nodes) != 3 || len(result.Edges) != 2 || len(result.ExampleRoute.Nodes) != 3 || result.ExampleRoute.TravelSeconds != 450 {
		t.Fatalf("unexpected generated vector contents: nodes=%d edges=%d route=%+v", len(result.Nodes), len(result.Edges), result.ExampleRoute)
	}
	nodeIDs := make(map[string]bool, len(result.Nodes))
	for _, node := range result.Nodes {
		nodeIDs[node.ID] = true
	}
	for _, edge := range result.Edges {
		if !nodeIDs[edge.From] || !nodeIDs[edge.To] {
			t.Fatalf("vector edge has unknown endpoint: %+v", edge)
		}
	}
}
