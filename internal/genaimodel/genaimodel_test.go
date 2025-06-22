package genaimodel

import (
	"context"
	"iter"
	"strconv"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"
	"google.golang.org/genai"
)

// The test proves that the model can handle slow chunk processing
// in the sense that if the UI can not catch up with the model's
// response, we can still get the final result containing all chunks
// and the implementation of the callback function will not block
// meaning that we get the finalResult as quickly as possible
// even if the UI is not able to catch up with the chunks
// You can validate the test by running it with "go test -v"
// And validate the test log output
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
			return SimulateModelChunks(10, 100*time.Millisecond)
		},
	)

	mockChatCreateServiceAPI := NewMockChatCreateServiceAPI(ctrl)
	mockChatCreateServiceAPI.EXPECT().Create(gomock.Any(), modelName, nil, gomock.Any()).
		Return(mockChatSessionAPI, nil)

		// Arrange expected calls, wire up our mock Func to the assumed client.ChatCreate field
	mockClient.EXPECT().ChatCreate().Return(mockChatCreateServiceAPI)

	// setup a slow dummy chunkReceiver
	var slowChunkReceiver string
	var slowChunkCount int
	chunkReceiver := func(s string) {
		// dummy func
		t.Logf("Received chunk %s", s)
		slowChunkReceiver += s
		time.Sleep(200 * time.Millisecond)
		slowChunkCount += 1
	}

	// Act!
	// When 50 chunks of 100ms take 5 seconds, but the UI is not able to handle the chunks that fast,
	// then the returned chatResult should still contain all chunks
	chatResult, err := model.ChatMessage("hello world", chunkReceiver)

	if err != nil {
		t.Fatalf("ChatMessage failed: %v", err)
	}

	t.Logf("slowchunks: %d slowchunkString: %s", slowChunkCount, slowChunkReceiver)

	t.Logf("chunks: %d result: %s", chatResult.ChunkCount, chatResult.Response)
}

func SimulateModelChunks(chunkCount int, chunkDuration time.Duration) func(yield func(*genai.GenerateContentResponse, error) bool) {
	return func(yield func(*genai.GenerateContentResponse, error) bool) {
		// Simulate a streaming response by yielding multiple chunks over time.
		// In a real API, these chunks would come from the actual service call.
		// Chunk 1
		for i := 0; i < chunkCount; i++ {
			chunkText := "Chunk " + strconv.Itoa(i)
			chunk := &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{Content: &genai.Content{
					Parts: []*genai.Part{
						{Text: chunkText},
					},
				},
				}},
			}
			if !yield(chunk, nil) { // Pass the chunk and a nil error.
				return // Consumer signaled to stop.
			}
			time.Sleep(chunkDuration) // Simulate network delay
		}

	}
}
