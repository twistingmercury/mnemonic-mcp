package openai

import "github.com/twistingmercury/mnemonic/internal/config"

// NewEmbeddingServiceForTest creates an EmbeddingService pointing at a custom URL.
// Exported for use in black-box tests.
func NewEmbeddingServiceForTest(cfg config.OpenAIConfig, baseURL string) EmbeddingService {
	svc := NewEmbeddingService(cfg).(*openaiEmbedding)
	svc.baseURL = baseURL
	return svc
}
