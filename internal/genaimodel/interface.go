package genaimodel

import (
	"context"
	"io"

	"iter"

	"google.golang.org/genai"
)

// ModelMetadataServiceAPI abstracts genai.Models (for List, Get, etc.)
// This is for model metadata, NOT content generation.
type ModelMetadataServiceAPI interface {
	List(ctx context.Context, cfg *genai.ListModelsConfig) (genai.Page[genai.Model], error)
	// Add other *genai.Models methods if needed, e.g., Get()
}

// ChatSessionAPI abstracts *genai.Chat (the actual chat session after creation)
type ChatSessionAPI interface {
	// SendMessageStream is the method on the *genai.Chat instance
	SendMessageStream(ctx context.Context, parts ...genai.Part) iter.Seq2[*genai.GenerateContentResponse, error]
	// Add other *genai.Chat methods if used, e.g., History()
}

// ChatCreateServiceAPI abstracts *genai.Chats (the factory for creating chat sessions)
type ChatCreateServiceAPI interface {
	// Create method on *genai.Chats returns a *genai.Chat (which implements ChatSessionAPI)
	// IMPORTANT: Match genai.Chats.Create's exact signature.
	Create(ctx context.Context, model string, config *genai.GenerateContentConfig, history []*genai.Content) (ChatSessionAPI, error)
}

// ModelServiceAPI abstracts *genai.GenerativeModelService
// This is for direct model interaction (non-chat based generation, embeddings, etc.)
type ModelServiceAPI interface {
	GenerateContentStream(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) iter.Seq2[*genai.GenerateContentResponse, error]
}

// FileServiceAPI abstracts *genai.Files
type FileServiceAPI interface {
	Upload(ctx context.Context, r io.Reader, config *genai.UploadFileConfig) (*genai.File, error)
	// Add other *genai.Files methods if needed, e.g., Get, Delete
}

// GeminiClientAPI is the top-level interface for our wrapper around *genai.Client.
// All its methods return the specific sub-service interfaces.
type GeminiClientAPI interface {
	GetModelsService() ModelMetadataServiceAPI
	GetChatCreateService() ChatCreateServiceAPI            // Corrected name: factory for creating chats
	GetGenerativeModelService(name string) ModelServiceAPI // This one takes a model name
	GetFileService() FileServiceAPI
	// Close() error // Add if you need to close the client.
}

// chatSessionWrapper implements ChatSessionAPI for *genai.Chat.
type chatSessionWrapper struct {
	chat *genai.Chat
}

func (w *chatSessionWrapper) SendMessageStream(ctx context.Context, parts ...genai.Part) iter.Seq2[*genai.
	GenerateContentResponse, error] {
	return w.chat.SendMessageStream(ctx, parts...)
}

// Add other *genai.Chat methods here, wrapping them.

// chatCreateServiceWrapper implements ChatCreateServiceAPI for *genai.Chats.
type chatCreateServiceWrapper struct {
	chats *genai.Chats
}

func (w *chatCreateServiceWrapper) Create(ctx context.Context, model string, config *genai.GenerateContentConfig, history []*genai.Content) (ChatSessionAPI, error) {
	concreteChat, err := w.chats.Create(ctx, model, config, history)
	if err != nil {
		return nil, err
	}
	return &chatSessionWrapper{chat: concreteChat}, nil
}

type modelMetadataServiceWrapper struct {
	models *genai.Models
}

func (w *modelMetadataServiceWrapper) List(ctx context.Context, cfg *genai.ListModelsConfig) (genai.Page[genai.
	Model], error) {
	return w.models.List(ctx, cfg)
}

type modelServiceWrapper struct {
	genModel *genai.Models
}

func (w *modelServiceWrapper) GenerateContentStream(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) iter.Seq2[*genai.GenerateContentResponse, error] {
	return w.genModel.GenerateContentStream(ctx, model, contents, config)
}

// fileServiceWrapper implements FileServiceAPI for *genai.Files.
type fileServiceWrapper struct {
	files *genai.Files
}

func (w *fileServiceWrapper) Upload(ctx context.Context, r io.Reader, config *genai.UploadFileConfig) (*genai.File,
	error) {
	return w.files.Upload(ctx, r, config)
}

// Add other *genai.Files methods here, wrapping them.

// genaiClientWrapper is the top-level wrapper for *genai.Client, implementing GeminiClientAPI.
type genaiClientWrapper struct {
	client *genai.Client
}

func (w *genaiClientWrapper) GetModelsService() ModelMetadataServiceAPI {
	return &modelMetadataServiceWrapper{models: w.client.Models}
}

func (w *genaiClientWrapper) GetChatCreateService() ChatCreateServiceAPI {
	return &chatCreateServiceWrapper{chats: w.client.Chats}
}

func (w *genaiClientWrapper) GetGenerativeModelService(name string) ModelServiceAPI {
	return &modelServiceWrapper{genModel: w.client.Models}
}

func (w *genaiClientWrapper) GetFileService() FileServiceAPI {
	return &fileServiceWrapper{files: w.client.Files}
}
