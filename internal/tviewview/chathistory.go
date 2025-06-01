package tviewview

import (
	"errors"
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

var ErrNoChatHistoryFiles = errors.New("no chat history files found")

const (
	fileSelectionModalPageName = "fileSelectionModal"
	confirmationPageName       = "confirmation"
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
	// Start progress *before* loading from disk to measure total time.
	tv.app.QueueUpdate(func() {
		tv.progress.startProgress()
		tv.outputView.Clear() // Clear the output view *before* starting the load
	})
	contentList, err := tv.aimodel.LoadChatHistory(filename)
	if err != nil {
		log.Printf("Error loading chat history: %v", err)
		tv.app.QueueUpdate(func() {
			tv.progressView.SetText(fmt.Sprintf("Error loading chat history: %v", err))
		})

		return
	}
	log.Printf("Chat history loaded from: %s", filename)

	// Update the outputView with the loaded chat history
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

	tv.app.QueueUpdateDraw(func() {
		tv.app.SetRoot(tv.flex, true)
	})
}

func getChatHistoryFolder() string {
	// 1. Get the user's configuration directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Printf("Error getting user config directory: %v", err)
		return "unkowndir"
	}

	return filepath.Join(configDir, "ai-chat", "history")
}
func getChatHistoryFiles() ([]string, error) {

	// 3. Read the files from the directory
	var files []string
	err := filepath.WalkDir(getChatHistoryFolder(), func(path string, d fs.DirEntry, err error) error {
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

func (tv *tviewApp) refreshFileList(fileList *tview.List) ([]string, error) {
	files, err := getChatHistoryFiles()
	if err != nil {
		return nil, fmt.Errorf("error getting chat history files: %w", err)
	}

	if len(files) == 0 {
		return nil, ErrNoChatHistoryFiles
	}

	fileList.Clear()
	for _, file := range files {
		fileList.AddItem(file, "", 0, nil)
	}
	return files, nil // Return the slice of file names
}

func (tv *tviewApp) createChatHistorySelectionModal(selectedFileChan chan string) {
	fileList := tview.NewList()
	chatHistoryFiles, err := tv.refreshFileList(fileList) // Capture the returned slice
	if err != nil {
		log.Printf("Error refreshing file list: %v", err)
		tv.progressView.SetText(fmt.Sprintf("Error refreshing chat history files: %v", err))
		selectedFileChan <- ""
		return
	}

	fileList.SetBorder(true).SetTitle("Select Chat History File (ESC to exit)")
	// Create a function to close the modal and reset the UI
	closeModal := func() {
		tv.app.SetInputCapture(nil)                     // undo the override of the TAB and ESC key
		tv.pages.RemovePage(fileSelectionModalPageName) // Remove the modal from pages
		tv.pages.RemovePage(confirmationPageName)       // Remove confirmation page if it exist
		tv.app.SetRoot(tv.flex, true)                   // Restore original layout
	}

	selectButton := tview.NewButton("Select").SetSelectedFunc(func() {
		index := fileList.GetCurrentItem()
		selectedFileChan <- chatHistoryFiles[index] // Send selected file
		log.Printf("Selected chat history file: %s", chatHistoryFiles[index])
		closeModal()
		tv.pages = tv.pages.RemovePage(fileSelectionModalPageName) // Close the modal
		tv.app.SetRoot(tv.flex, true)
	})

	var filenameToBeDeleted string

	confirmationModal := tview.NewModal().
		SetText("Are you sure you want to delete this file?").
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Yes" {

				index := fileList.GetCurrentItem()
				if index >= 0 && index < len(chatHistoryFiles) {
					err := os.Remove(filepath.Join(getChatHistoryFolder(), chatHistoryFiles[index]))
					if err != nil {
						log.Printf("Error deleting chat history file: %v", err)
						// Consider showing an error message to the user
					}
					log.Printf("Deleted chat history file: %s", chatHistoryFiles[index])
				}
			}

			// Refresh the fileList
			if _, err := tv.refreshFileList(fileList); err != nil {
				log.Printf("Error refreshing file list: %v", err)
				// Consider showing an error message to the user
			}
			tv.pages = tv.pages.SwitchToPage(fileSelectionModalPageName)
		})
	confirmationModal.Box.SetBorder(true).
		SetRect(10, 10, 30, 5)

	deleteButton := tview.NewButton("Delete").SetSelectedFunc(func() {
		index := fileList.GetCurrentItem()
		if index >= 0 && index < len(chatHistoryFiles) {
			filenameToBeDeleted = chatHistoryFiles[index] // Store the filename
			confirmationText := fmt.Sprintf("Are you sure you want to delete:\n '%s'?", filenameToBeDeleted)
			confirmationModal.SetText(confirmationText) // Update the modal text
		} else {
			filenameToBeDeleted = "" //Clear previous text
			confirmationModal.SetText("Invalid file selected. Cannot proceed with deletion.")
			log.Println("Invalid file index selected")
		}
		tv.pages = tv.pages.SwitchToPage("confirmation")
		tv.app.SetRoot(tv.pages, true)
	})
	// Create buttons for selecting or deleting a file
	buttons := tview.NewFlex().
		AddItem(selectButton, 0, 1, true).
		AddItem(deleteButton, 0, 1, true)

		// Create a flex layout for the modal
	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(fileList, 0, 1, true).
		AddItem(buttons, 1, 1, false)

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
				tv.app.SetFocus(selectButton)
			} else if tv.app.GetFocus() == selectButton {
				tv.app.SetFocus(deleteButton)
			} else if tv.app.GetFocus() == deleteButton {
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

	// Add the modal to the pages
	tv.pages = tv.pages.AddPage(fileSelectionModalPageName, modal, true, true)
	// Add the confirmation page to the pages
	tv.pages = tv.pages.AddPage(confirmationPageName, confirmationModal, false, false)

	// Set the modal as the root of the application
	tv.pages.SwitchToPage(fileSelectionModalPageName)
	tv.app.SetRoot(tv.pages, true)
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
