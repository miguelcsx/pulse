package service

import (
	"hash/fnv"
	"math"
	"strings"
)

// Embedder generates vector embeddings for text.
type Embedder interface {
	EmbedText(text string) ([]float64, error)
	Dimensions() int
}

// HashEmbedder is a deterministic, local-first embedder.
// It is NOT semantic, but guarantees stable vectors for vector search
// without external dependencies.
type HashEmbedder struct {
	dim int
}

func NewHashEmbedder(dim int) *HashEmbedder {
	if dim <= 0 {
		dim = 384
	}
	return &HashEmbedder{dim: dim}
}

func (e *HashEmbedder) Dimensions() int {
	return e.dim
}

func (e *HashEmbedder) EmbedText(text string) ([]float64, error) {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return make([]float64, e.dim), nil
	}

	tokens := strings.Fields(normalized)
	vector := make([]float64, e.dim)

	for _, token := range tokens {
		idx, val := e.hashToken(token)
		vector[idx] += val
	}

	return l2Normalize(vector), nil
}

func (e *HashEmbedder) hashToken(token string) (int, float64) {
	h := fnv.New64a()
	_, _ = h.Write([]byte(token))
	sum := h.Sum64()

	index := int(sum % uint64(e.dim))

	// Map hash to a small magnitude in [-1, 1]
	val := float64(int64(sum>>32)%2000-1000) / 1000.0
	if val == 0 {
		val = 0.001
	}

	return index, val
}

func l2Normalize(vec []float64) []float64 {
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	if norm == 0 {
		return vec
	}
	norm = math.Sqrt(norm)
	for i, v := range vec {
		vec[i] = v / norm
	}
	return vec
}
