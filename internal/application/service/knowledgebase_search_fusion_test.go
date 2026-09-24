package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestRequestRRFWeightsReorderExactMatchWithoutChangingWorkspace(t *testing.T) {
	tenant := &types.RetrievalConfig{RRFVectorWeight: 0.9, RRFKeywordWeight: 0.1}
	vectorWeight, keywordWeight := 0.3, 0.7
	override := types.RRFWeightOverride{
		RRFVectorWeight: &vectorWeight, RRFKeywordWeight: &keywordWeight,
	}

	vector, keyword := rankedRetrievalCandidates()
	baseline := fuseWithRRF(context.Background(), vector, keyword,
		effectiveRRFConfig(tenant, types.RRFWeightOverride{}))
	if baseline[0].ChunkID != "semantic-0" {
		t.Fatalf("workspace weights unexpectedly favored %q", baseline[0].ChunkID)
	}

	vector, keyword = rankedRetrievalCandidates()
	results := fuseWithRRF(context.Background(), vector, keyword,
		effectiveRRFConfig(tenant, override))
	if results[0].ChunkID != "exact-student-id" {
		t.Fatalf("request keyword weight did not promote exact match: first=%q", results[0].ChunkID)
	}
	if tenant.RRFVectorWeight != 0.9 || tenant.RRFKeywordWeight != 0.1 {
		t.Fatalf("request changed workspace config: %+v", tenant)
	}
}

func rankedRetrievalCandidates() ([]*types.IndexWithScore, []*types.IndexWithScore) {
	const exactChunkID = "exact-student-id"
	vector := make([]*types.IndexWithScore, 70)
	for i := range vector {
		vector[i] = &types.IndexWithScore{
			ChunkID: fmt.Sprintf("semantic-%d", i),
			Score:   float64(len(vector) - i),
		}
	}
	vector[len(vector)-1].ChunkID = exactChunkID
	keyword := []*types.IndexWithScore{{ChunkID: exactChunkID, Score: 1}}
	return vector, keyword
}
