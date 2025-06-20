package genaimodel

import (
	"context"
	"iter"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"
	"google.golang.org/genai"
)

func TestChatSlowChunkProcessing(t *testing.T) {
	ctrl := gomock.NewController(t)

	// Mock the gemini client
	mockClient := NewMockGeminiClientAPI(ctrl)

	// Arrange subject under test
	model, err := NewModel(context.Background(), mockClient, "system instruction be kind")
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// setup expected return
	mockChatSessionAPI := NewMockChatSessionAPI(ctrl)
	mockChatSessionAPI.EXPECT().SendMessageStream(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, parts ...genai.Part) iter.Seq2[*genai.GenerateContentResponse, error] {
			return func(yield func(*genai.GenerateContentResponse, error) bool) {
				// Simulate a streaming response by yielding multiple chunks over time.
				// In a real API, these chunks would come from the actual service call.
				// Chunk 1
				chunk1 := &genai.GenerateContentResponse{
					Candidates: []*genai.Candidate{{Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "Hello"},
						}},
					},
					},
				}
				if !yield(chunk1, nil) { // Pass the chunk and a nil error.
					return // Consumer signaled to stop.
				}
				time.Sleep(100 * time.Millisecond) // Simulate network delay

				chunk2 := &genai.GenerateContentResponse{
					Candidates: []*genai.Candidate{{Content: &genai.Content{
						Parts: []*genai.Part{{Text: "Twice"}}},
					},
					},
				}
				if !yield(chunk2, nil) { // Pass the chunk and a nil error.
					return // Consumer signaled to stop.
				}
				time.Sleep(100 * time.Millisecond) // Simulate network delay

			}
		},
	)

	mockChatCreateServiceAPI := NewMockChatCreateServiceAPI(ctrl)
	mockChatCreateServiceAPI.EXPECT().Create(gomock.Any(), modelName, nil, gomock.Any()).
		Return(mockChatSessionAPI, nil)

		// Arrange expected calls, wire up our mock Func to the assumed client.ChatCreate field
	mockClient.EXPECT().ChatCreate().Return(mockChatCreateServiceAPI)
	// setup a dummy chunkReceiver
	chunkReceiver := func(s string) {
		// dummy func
		t.Logf("Received chunk %s", s)
	}
	chatResult, err := model.ChatMessage("hello world", chunkReceiver)

	if err != nil {
		t.Fatalf("ChatMessage failed: %v", err)
	}

	t.Logf("cunks: %d result: %s", chatResult.ChunkCount, chatResult.Response)
}
