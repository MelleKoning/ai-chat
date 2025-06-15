package genaimodel

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"sync"

	// genai is the successor of the previous
	// generative-ai-go model
	"google.golang.org/genai"
)

const (
	//modelName = "gemini-2.0-flash"
	modelName = "gemini-2.5-flash-preview-05-20"
)

type theModel struct {
	systemInstruction string
	client            *genai.Client
	chatHistory       []*genai.Content
}

type ChatResult struct {
	Response   string
	ChunkCount int
}

// Action is the interface for the model
// to support the tview console application
// the callback function in the chat is to present
// intermediate results in the console
// and to allow for streaming of the response
type Action interface {
	SendSystemPrompt(func(string)) (ChatResult, error)
	ReviewFile(func(string)) (string, error)
	// ChatMessage provides a callback function for each
	// chunk of the response. Eventually will return the full
	// response as a string
	ChatMessage(string, func(string)) (ChatResult, error)
	UpdateSystemInstruction(string)
	GetHistoryLength() int

	// Chat History
	StoreChatHistory(string) error
	LoadChatHistory(string) ([]*genai.Content, error)
	GenerateChatSummary() (string, error)
	ListModels() (string, error)
}

func NewGeminiClient(ctx context.Context, apiKey string) (*genai.Client, error) {
	return genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
}

func NewModel(ctx context.Context,
	genaiClient *genai.Client,
	systemInstruction string) (Action, error) {

	return &theModel{
		systemInstruction: systemInstruction,
		client:            genaiClient,
	}, nil
}

func (m *theModel) ListModels() (string, error) {
	models, err := m.client.Models.List(context.Background(), &genai.ListModelsConfig{})
	if err != nil {
		return "nil", err
	}

	var modelNames string
	for _, model := range models.Items {
		modelNames = modelNames + ", \n" + model.Name
	}

	return modelNames, nil
}
func (m *theModel) GetHistoryLength() int {
	return len(m.chatHistory)
}
func (m *theModel) UpdateSystemInstruction(systemInstruction string) {
	m.systemInstruction = systemInstruction
}

// ChatMessage sends a message to the model
// and returns the answer as string
// Variables:
// userPrompt: the prompt to send to the model
// onChunk: a callback function that is called for each chunk of the response
// The ChatMessage func is taking a userPrompt which is send off to
// the aimodel. Then the aimodel is going to answer in chunks. Every chunk
// received is going to be presented in the callback func " onChunk".
// It can be that the processing of that callback takes a bit of time for
//
//	the UI to render it on the UI. This is why there is a buffer
//
// created, so that if multiple chunks arrive, and the processing
// takes too much time, we can "cut-off" processing and immediately
// send back the aggregated result (final result) to the caller. At that
//
//	moment, any remaining chunks are going to be consumed
//	and not raised in the callback.
func (m *theModel) ChatMessage(userPrompt string, onChunk func(string)) (ChatResult, error) {
	// Use context for cancellation. This is the primary way to signal goroutines to stop.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is cancelled when ChatMessage returns, signaling cleanup

	// Create a buffered channel to process chunks.
	chunkChan := make(chan string, 100)

	// Use a WaitGroup to ensure the chunk processing goroutine finishes before `ChatMessage` exits.
	var wg sync.WaitGroup

	// Start the chunk processing goroutine by calling the new extracted function
	wg.Add(1)
	go func() {
		defer wg.Done()

		// This defer will drain the channel *after* the select loop exits.
		// It ensures no deadlocks if the sender closes the channel while chunks are still buffered.
		defer func() {
			for range chunkChan {
				// Keep consuming the channel until it's closed,
				// ensuring any remaining buffered chunks are processed or discarded.
			}
			log.Println("Chunk processing goroutine finished draining.")
		}()

		for {
			select {
			case chunk, ok := <-chunkChan:
				if !ok { // Channel was closed by the sender
					log.Println("Chunk processing goroutine: Chunk channel closed, exiting.")
					return // Exit the select loop, then run the draining defer
				}
				onChunk(chunk)

			case <-ctx.Done(): // Context cancelled (signal to stop)
				log.Println("Chunk processing goroutine: Context cancelled, exiting.")
				return // Exit the select loop, then run the draining defer
			}
		}
	}()
	// Add user prompt to chat history
	m.chatHistory = append(m.chatHistory, genai.NewContentFromText(userPrompt, genai.RoleUser))

	// Create chat with history
	chat, err := m.client.Chats.Create(ctx, modelName, nil, m.chatHistory)
	if err != nil {
		// If chat creation fails, immediately return and cancel context.
		cancel()
		close(chunkChan)
		wg.Wait()
		return ChatResult{}, err
	}

	// Send message to the model using streaming
	stream := chat.SendMessageStream(ctx, genai.Part{Text: userPrompt})
	var fullString strings.Builder
	var chunkCount int
	var streamErr error // to capture a streamErr if it occurs
	// Loop through the stream responses.
	// The `stream` channel itself often handles closing when the API call is done
	// or an error occurs.
	for respChunk, err := range stream { // Iterate without checking 'err' in the range clause itself
		if err != nil {
			log.Println("Error receiving stream:", err)
			streamErr = err
			break // exit the loop
		}
		// defensive check on received respChunk
		if respChunk == nil || len(respChunk.Candidates) == 0 || respChunk.Candidates[0].Content == nil || len(respChunk.
			Candidates[0].Content.Parts) == 0 {
			log.Println("Received nil or malformed chunk from stream (no explicit error reported).")
			// Decide how to handle this. You might want to treat it as an error and break,
			// or just skip this malformed chunk if acceptable.
			// For robustness, treating it as an error is safer:
			streamErr = errors.New("received malformed chunk data")
			break
		}
		part := respChunk.Candidates[0].Content.Parts[0] // Potential nil dereference if respChunk is nil!
		select {
		case chunkChan <- part.Text:
			// Send chunk to channel
		default:
			// If channel and channel buffer is full, discard
			// sending the chunk to the channelprocess - the UI
			// is simply too slow to keep up, but will eventually
			// get the final result
		}
		fullString.WriteString(part.Text)
		chunkCount++
	}

	cancel()         // Signal the goroutine to stop processing chunks
	close(chunkChan) // Close the channel
	// Wait for the chunk processing goroutine to finish its cleanup.
	wg.Wait()

	if streamErr != nil {
		log.Printf("Stream error: %v\n", streamErr)
		return ChatResult{
			Response:   fullString.String(),
			ChunkCount: chunkCount,
		}, streamErr
	}

	chatResponse := fullString.String()
	modelResponse := genai.NewContentFromText(chatResponse, genai.RoleModel)
	m.chatHistory = append(m.chatHistory, modelResponse)

	return ChatResult{chatResponse, chunkCount}, nil
}

func (m *theModel) SendSystemPrompt(onChunk func(string)) (ChatResult, error) {
	ctx := context.Background()
	// Add the prompt to the chat history to not forget about it
	m.chatHistory = append(m.chatHistory, genai.NewContentFromText(m.systemInstruction, genai.RoleModel))

	// Create chat with history
	chat, err := m.client.Chats.Create(ctx, modelName, nil, m.chatHistory)
	if err != nil {
		return ChatResult{}, err
	}

	log.Println(m.systemInstruction)
	// Send message to the model using streaming
	stream := chat.SendMessageStream(ctx, *genai.NewPartFromText(m.systemInstruction))

	// process response
	var allModelParts []*genai.Part
	var chunkCounter int
	for chunk, err := range stream {
		if err != nil {
			log.Printf("Error receiving stream: %v", err)

			fullString := buildString(allModelParts)

			return ChatResult{fullString, chunkCounter}, err
		}

		part := chunk.Candidates[0].Content.Parts[0]
		onChunk(part.Text)
		allModelParts = append(allModelParts, part)
		chunkCounter++
	}

	fullString := buildString(allModelParts)

	return ChatResult{fullString, chunkCounter}, nil
}

// ReviewFile revies the "gitdiff.txt" file
func (m *theModel) ReviewFile(onChunk func(string)) (string, error) {
	filePart, fileUri := m.addAFile(context.Background(), m.client)
	log.Printf("fileUri is %s", fileUri)

	// Start with chatHistory
	genaiContents := append([]*genai.Content{}, m.chatHistory...)

	// we first create a Part for file,
	// later we add an additional part
	// to this slice to add the Command below
	parts := []*genai.Part{
		filePart,
	}

	fileContent := genai.NewContentFromParts(parts, genai.RoleUser)

	// Include fileContent
	genaiContents = append(genaiContents, fileContent)

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(m.systemInstruction, genai.RoleModel),
	}

	commandText := `* Do not include the provided diff output in the response.

		The file {fileUri} contains the git diff output to be reviewed.

		AI OUTPUT:`
	commandText = strings.Replace(commandText, "{fileUri}", fileUri, 1)

	// add command as additional part
	// to the last item in the genaiContents
	genaiCommandPart := &genai.Part{Text: commandText}
	lastContentPart := len(genaiContents) - 1
	genaiContents[lastContentPart].Parts = append(genaiContents[lastContentPart].Parts, genaiCommandPart)

	// add the command text to the file contents
	//genaiCommandText := genai.Text(commandText)
	//genaiContents = append(genaiContents, genaiCommandText...)
	stream := m.client.Models.GenerateContentStream(
		context.Background(),
		modelName,
		genaiContents,
		config,
	)

	var allModelParts []*genai.Part

	for chunk, err := range stream {
		if err != nil {
			return "", err

		}
		part := chunk.Candidates[0].Content.Parts[0]
		onChunk(part.Text) // raise callback func
		allModelParts = append(allModelParts, part)

	}

	fullString := buildString(allModelParts)

	// Combine all parts into a single part and add to chat history
	modelResponse := genai.NewContentFromText(fullString, genai.RoleModel)
	m.chatHistory = append(m.chatHistory, modelResponse)

	// fileio.WriteMarkdown(fullString, "codereview.md")

	return fullString, nil
}

// uploads a file to gemini
func (m *theModel) addAFile(ctx context.Context, client *genai.Client) (*genai.Part, string) {
	// during the chat, we can continuously update the below file by providing
	// a different diff. For example to get a diff for a golang repository,
	// we can issue the following command:
	// git diff -U10 7c904..dcfc69 -- . ':!vendor' > gitdiff.txt
	// the hashes are examples from two consecutive git hashes found when
	// simply doing a "git log" statement. Put the oldest hash first so that added
	// lines get a + and removed lines get a -, or you get it backwards.
	// note that the "-- . `:! vendor` part is to ignore the vendor file, as we are
	// only interested in actual updates of changes.
	fileContents, err := os.Open("./gitdiff.txt")
	if err != nil {
		panic(err)
	}
	upFile, err := client.Files.Upload(ctx, fileContents, &genai.UploadFileConfig{
		MIMEType: "text/plain",
	})
	if err != nil {
		panic(err)
	}

	return genai.NewPartFromURI(upFile.URI, upFile.MIMEType), upFile.URI
}

func buildString(resp []*genai.Part) string {
	var build strings.Builder
	for _, p := range resp {

		build.WriteString(p.Text)
	}

	return build.String()
}

func (m *theModel) GenerateChatSummary() (string, error) {
	ctx := context.Background()

	chat, err := m.client.Chats.Create(ctx, modelName, nil, m.chatHistory)
	if err != nil {
		return "", err
	}

	// Craft the prompt for the AI model
	prompt := `Summarize the chat history in approximately 10-15 keywords, suitable for use in
  a filename.  Do not include punctuation or special characters.
  Only respond with the summary for the filename`

	// Send the message to the model
	prt := &genai.Part{Text: prompt}
	resp, err := chat.Send(ctx, prt) // Use SendMessage instead of SendMessageStream
	if err != nil {
		return "", err
	}

	summary := resp.Candidates[0].Content.Parts[0].Text

	return string(summary), nil
}
