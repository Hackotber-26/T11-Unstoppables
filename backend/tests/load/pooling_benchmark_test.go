package load

import (
	"fmt"
	"testing"
	"time"

	"t11-unstoppables/internal/pooling"
)

func BenchmarkPoolFormation1000Requests(b *testing.B) {
	requests := make([]pooling.Request, 1000)
	start := time.Unix(1_800_000_000, 0)
	for index := range requests {
		requests[index] = pooling.Request{
			ID: fmt.Sprintf("r-%04d", index), OriginID: fmt.Sprintf("station-%d", index%8),
			Destination: fmt.Sprintf("corridor-%d", index%4), RequestedAt: start.Add(time.Duration(index%300) * time.Second),
			PartySize: 1 + index%2, MaxWaitSeconds: 300, MaxDetourMeters: 2500,
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := pooling.Form(requests, 4, false); err != nil {
			b.Fatal(err)
		}
	}
}
