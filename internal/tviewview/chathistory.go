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

	"github.com/MelleKoning/ai-chat/internal/fileio"
	"github.com/MelleKoning/ai-chat/internal/genaimodel"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var ErrNoChatHistoryFiles = errors.New("no chat history files found")

const (
	fileSelectionModalPageName = "fileSelectionModal"
	confirmationPageName       = "confirmation"
)

// ChatHistoryHandler encapsulates chat history logic and UI interactions.
type ChatHistoryHandler struct {
	tv *tviewApp
}

func NewChatHistoryHandler(tv *tviewApp) *ChatHistoryHandler { return &ChatHistoryHandler{tv: tv} }

// OnStatus reports status/progress messages via tview's event loop.
func (h *ChatHistoryHandler) OnStatus(msg string) {
	if h.tv != nil && h.tv.app != nil && h.tv.progressView != nil {
		h.tv.app.QueueUpdate(func() { h.tv.progressView.SetText(msg) })
	}
}

func (h *ChatHistoryHandler) storeChatHistory() {
	filename, _ := h.GenerateChatHistoryFilename()
	jsonData, err := h.tv.aimodel.GetChatHistory()
	if err != nil {
		log.Printf("Error storing chat history: %v", err)
		h.OnStatus(fmt.Sprintf("Error storing chat history: %v", err))
	}
	err = fileio.StoreChatHistory(filename, jsonData)
	if err != nil {
		log.Printf("Error storing chat history: %v", err)
		h.OnStatus(fmt.Sprintf("Error storing chat history: %v", err))
	} else {
		log.Printf("Chat history stored to: %s", filename)
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Chat history stored to:\n%s", filename)).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				h.tv.app.SetRoot(h.tv.flex, true) // Set focus back to the main layout
			})
		h.OnStatus(filename)
		h.tv.app.SetRoot(modal, false)
	}
}

// loadChatHistory loads chat history from a file and updates UI via QueueUpdate/Draw.
func (h *ChatHistoryHandler) loadChatHistory(filename string) {
	// Start progress *before* loading from disk to measure total time.
	h.tv.app.QueueUpdate(func() {
		h.tv.progress.startProgress()
		h.tv.outputView.Clear() // Clear the output view *before* starting the load
	})
	jsonHistory, err := fileio.LoadChatHistory(filename)
	if err != nil {
		log.Printf("Error loading chat history: %v", err)
		h.OnStatus(fmt.Sprintf("Error loading chat history: %v", err))
		return
	}
	contentList, err := h.tv.aimodel.LoadChatHistory(jsonHistory)
	if err != nil {
		log.Printf("Error loading chat history: %v", err)
		h.OnStatus(fmt.Sprintf("Error loading chat history: %v", err))
		return
	}
	log.Printf("Chat history loaded from: %s", filename)

	// Update the outputView with the loaded chat history
	for _, content := range contentList {
		if content.Role == "user" {
			h.tv.app.QueueUpdate(func() {
				h.tv.progress.appendUserCommandToOutput(content.Parts[0].Text)
			})
		} else {
			h.tv.app.QueueUpdate(func() {
				h.tv.progress.handleFinalModelResult(genaimodel.ChatResult{
					Response:   content.Parts[0].Text,
					ChunkCount: 1,
				}, nil)
			})
		}
	}

	h.tv.app.QueueUpdateDraw(func() {
		h.tv.app.SetRoot(h.tv.flex, true)
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

func (h *ChatHistoryHandler) refreshFileList(fileList *tview.List) ([]string, error) {
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

func (h *ChatHistoryHandler) createChatHistorySelectionModal(onSelected func(string)) {
	fileList := tview.NewList()
	chatHistoryFiles, err := h.refreshFileList(fileList) // Capture the returned slice
	if err != nil {
		log.Printf("Error refreshing file list: %v", err)
		h.OnStatus(fmt.Sprintf("Error refreshing chat history files: %v", err))
		onSelected("")
		return
	}

	fileList.SetBorder(true).SetTitle("Select Chat History File (ESC to exit)")
	// Create a function to close the modal and reset the UI
	closeModal := func() {
		h.tv.app.SetInputCapture(nil)                     // undo the override of the TAB and ESC key
		h.tv.pages.RemovePage(fileSelectionModalPageName) // Remove the modal from pages
		h.tv.pages.RemovePage(confirmationPageName)       // Remove confirmation page if it exist
		h.tv.app.SetRoot(h.tv.flex, true)                 // Restore original layout
	}

	selectButton := tview.NewButton("Select").SetSelectedFunc(func() {
		index := fileList.GetCurrentItem()
		if index >= 0 && index < len(chatHistoryFiles) {
			filename := chatHistoryFiles[index] // Get selected file
			log.Printf("Selected chat history file: %s", filename)
			closeModal()
			h.tv.pages = h.tv.pages.RemovePage(fileSelectionModalPageName) // Close the modal
			h.tv.app.SetRoot(h.tv.flex, true)
			onSelected(filename) // callback with filename
		} else {
			onSelected("") // invalid selection
		}
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
			if _, err := h.refreshFileList(fileList); err != nil {
				log.Printf("Error refreshing file list: %v", err)
				// Consider showing an error message to the user
			}
			h.tv.pages = h.tv.pages.SwitchToPage(fileSelectionModalPageName)
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
		h.tv.pages = h.tv.pages.SwitchToPage("confirmation")
		h.tv.app.SetRoot(h.tv.pages, true)
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
		if index >= 0 && index < len(chatHistoryFiles) {
			filename := chatHistoryFiles[index] // selected file
			log.Printf("Selected chat history file: %s", filename)
			closeModal() // Signal modal is closed
			onSelected(filename)
		} else {
			onSelected("")
		}
	})

	// Set input capture to handle TAB and ESCAPE
	h.tv.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTAB:
			if h.tv.app.GetFocus() == fileList {
				h.tv.app.SetFocus(selectButton)
			} else if h.tv.app.GetFocus() == selectButton {
				h.tv.app.SetFocus(deleteButton)
			} else if h.tv.app.GetFocus() == deleteButton {
				h.tv.app.SetFocus(fileList)
			}
			return nil // Consume the event
		case tcell.KeyEscape:
			closeModal()           // Signal modal is closed
			onSelected("") // Signal cancellation
			return nil             // Consume the event
		}
		return event // Pass other events through
	})

	// Add the modal to the pages
	h.tv.pages = h.tv.pages.AddPage(fileSelectionModalPageName, modal, true, true)
	// Add the confirmation page to the pages
	h.tv.pages = h.tv.pages.AddPage(confirmationPageName, confirmationModal, false, false)

	// Set the modal as the root of the application
	h.tv.pages.SwitchToPage(fileSelectionModalPageName)
	h.tv.app.SetRoot(h.tv.pages, true)
}

func (h *ChatHistoryHandler) SelectChatHistoryFile(onSelected func(string)) {
	h.createChatHistorySelectionModal(func(filename string) {
		if filename == "" {
			log.Println("Chat history selection cancelled or failed.")
			return
		}
		// Invoke the callback asynchronously to avoid blocking tview's event loop.
		go onSelected(filename)
	})
}

// GenerateChatHistoryFilename returns a sanitized filename based on a model-generated summary.
func (h *ChatHistoryHandler) GenerateChatHistoryFilename() (string, error) {
	var summary string
	summary, err := h.tv.aimodel.GenerateChatSummary()
	if err != nil {
		log.Printf("Error generating chat summary: %v", err)
		// use default filename - it will be sanitized further
		summary = "chat_summary"
	}
	filename := fmt.Sprintf("chat_%s_%s.json", time.Now().Format("20060102150405"), summary)

	// Sanitize and format the summary for use as a filename
	sanitizedSummary := sanitizeFilename(filename)

	return sanitizedSummary, nil
}

func sanitizeFilename(filename string) string {
	// Remove spaces and convert to lowercase
	filename = strings.ToLower(strings.TrimSpace(filename))

	reg := regexp.MustCompile("[^a-zA-Z0-9._-]+") // Allow alphanumeric, dot, underscore, and dash
	filename = reg.ReplaceAllString(strings.ReplaceAll(filename, " ", "_"), "")
	return filename
}
