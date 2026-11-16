package healing_test

import (
	"testing"

	"github.com/alexandrmotologa/domdrift/internal/healing"
)

func TestShannonEntropy(t *testing.T) {
	// Simple semantic class names have low entropy
	semanticClass := "title"
	entropySemantic := healing.ComputeShannonEntropy(semanticClass)

	// A randomized hash token with diverse characters has higher entropy
	hashToken := "k8X9pL2qZ7mB1"
	entropyHash := healing.ComputeShannonEntropy(hashToken)

	if entropySemantic >= entropyHash {
		t.Errorf("expected hash entropy (%f) to be strictly greater than semantic class entropy (%f)", entropyHash, entropySemantic)
	}
}
