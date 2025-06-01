package tviewview

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/MelleKoning/ai-chat/internal/genaimodel"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (tv *tviewApp) storeChatHistory() {
	filename, _ := tv.GenerateChatHistoryFilename()
	err := tv.aimodel.StoreChatHistory(filename)
	if err != nil {
		log.Printf("Error storing chat history: %v", err)
		tv.progressView.SetText(fmt.Sprintf("Error storing chat history: %v", err))
	} else {
		log.Printf("Chat history stored to: %s", filename)
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Chat history stored to:\n%s", filename)).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				//tv.app.QueueUpdateDraw(func() {
				tv.app.SetRoot(tv.flex, true) // Set focus back to the main layout
				//})
			})
		tv.progressView.SetText(filename)
		tv.app.SetRoot(modal, false)

	}
}

// loadChatHistory loads chat history from a file
// and runs from the async routine selected from the
// dropdown, so updates are done via QueueUpdateDraw
func (tv *tviewApp) loadChatHistory(filename string) {
	contentList, err := tv.aimodel.LoadChatHistory(filename)
	if err != nil {
		log.Printf("Error loading chat history: %v", err)
		tv.app.QueueUpdate(func() {
			tv.progressView.SetText(fmt.Sprintf("Error loading chat history: %v", err))
		})

		return
	}
	log.Printf("Chat history loaded from: %s", filename)
	tv.app.QueueUpdate(func() {
		tv.progressView.SetText(fmt.Sprintf("Chat history loaded from %s", filename))
		// Update the outputView with the loaded chat history
		// (You'll need to iterate through the chatHistory and format the output)
		tv.outputView.Clear()
	})

	for _, content := range contentList {
		// Format the output based on the content's role (user or model)
		if content.Role == "user" {
			tv.app.QueueUpdate(func() {
				tv.progress.appendUserCommandToOutput(content.Parts[0].Text)
			})

		} else {
			tv.app.QueueUpdate(func() {
				tv.progress.handleFinalModelResult(genaimodel.ChatResult{
					Response:   content.Parts[0].Text,
					ChunkCount: 1,
				}, nil)
			})
		}
	}
}

func getChatHistoryFiles() ([]string, error) {
	// 1. Get the user's configuration directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	// 2. Construct the history directory path
	historyDir := filepath.Join(configDir, "ai-chat", "history")

	// 3. Read the files from the directory
	var files []string
	err = filepath.WalkDir(historyDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, d.Name())
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func (tv *tviewApp) createChatHistorySelectionModal(selectedFileChan chan string) {
	chatHistoryFiles, err := getChatHistoryFiles()
	if err != nil {
		log.Printf("Error getting chat history files: %v", err)
		tv.progressView.SetText(fmt.Sprintf("Error getting chat history files: %v", err))
		selectedFileChan <- "" // Signal error by sending an empty string
		return
	}

	// Create a list to display the chat history files
	fileList := tview.NewList()
	for _, file := range chatHistoryFiles {
		fileList.AddItem(file, "", 0, nil)
	}
	fileList.SetBorder(true).SetTitle("Select Chat History File (ESC to exit)")

	// Create a flex layout for the modal
	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(fileList, 0, 1, true)

	// Create a function to close the modal and reset the UI
	closeModal := func() {
		tv.app.SetInputCapture(nil)   // undo the override of the TAB and ESC key
		tv.app.SetRoot(tv.flex, true) // Restore original layout
	}

	// Handle file selection
	fileList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		selectedFileChan <- chatHistoryFiles[index] // Send selected file
		log.Printf("Selected chat history file: %s", chatHistoryFiles[index])
		closeModal() // Signal modal is closed
	})

	// Set input capture to handle TAB and ESCAPE
	tv.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTAB:
			if tv.app.GetFocus() == fileList {
				tv.app.SetFocus(nil) // Focus on buttons if we add buttons
			} else {
				tv.app.SetFocus(fileList)
			}
			return nil // Consume the event
		case tcell.KeyEscape:
			closeModal()           // Signal modal is closed
			selectedFileChan <- "" // Send empty string to signal cancellation
			return nil             // Consume the event
		}
		return event // Pass other events through
	})

	// Set the modal as the root of the application
	tv.app.SetRoot(modal, true)
}

func (tv *tviewApp) SelectChatHistoryFile() {
	// Create the channel
	selectedFileChan := make(chan string)

	// Start the goroutine to await the response *before* creating the modal
	go func() {
		defer close(selectedFileChan)

		selectedFile := <-selectedFileChan // Receive the file

		if selectedFile == "" {
			log.Println("Chat history selection cancelled or failed.")
			return
		}

		tv.loadChatHistory(selectedFile)
		tv.app.Draw()
	}()

	// Open the chat history selection modal, passing in the channel
	tv.createChatHistorySelectionModal(selectedFileChan)
}

// In your tviewApp:
func (tv *tviewApp) GenerateChatHistoryFilename() (string, error) {
	var summary string
	summary, err := tv.aimodel.GenerateChatSummary()
	if err != nil {
		log.Printf("Error generating chat summary: %v", err)

		// use default filename - it will be sanitized further
		summary = "chat_summary"
	}
	filename := fmt.Sprintf("chat_%s_%s.json", time.Now().Format("20060102150405"), summary)

	// Sanitize and format the summary for use as a filename
	sanitizedSummary := sanitizeFilename(filename) // Implement sanitizeFilename

	return sanitizedSummary, nil
}

func sanitizeFilename(filename string) string {
	// Remove spaces and convert to lowercase
	filename = strings.ToLower(strings.TrimSpace(filename))

	reg := regexp.MustCompile("[^a-zA-Z0-9._-]+") // Allow alphanumeric, dot, underscore, and dash
	filename = reg.ReplaceAllString(strings.ReplaceAll(filename, " ", "_"), "")
	return filename
}
