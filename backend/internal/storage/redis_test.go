package storage

import "testing"

func TestParseRedisOptionsKeepsCredentialsAndDatabase(t *testing.T) {
	options, err := parseRedisOptions("redis://:test-password@cache:6379/2")
	if err != nil {
		t.Fatal(err)
	}
	if options.Addr != "cache:6379" || options.Password != "test-password" || options.DB != 2 {
		t.Fatalf("Redis URL fields were not retained: %+v", options)
	}
}
