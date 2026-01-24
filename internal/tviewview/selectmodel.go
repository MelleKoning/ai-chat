package tviewview

import (
	"log"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	modelSelectionModalPageName = "modelSelectionModal"
)

func (tv *tviewApp) SelectModel(onSelect func(modelName string)) {
	// Call ListModels to get the string of model names
	modelString, err := tv.aimodel.ListModels()
	if err != nil {
		log.Printf("Error listing models: %v", err)
		return
	}

	// Parse the string into individual model names
	// The current implementation returns models separated by ", \n"
	rawModels := strings.Split(modelString, ",")
	var modelNames []string
	for _, m := range rawModels {
		trimmed := strings.TrimSpace(m)
		if trimmed != "" && trimmed != "nil" {
			modelNames = append(modelNames, trimmed)
		}
	}

	if len(modelNames) == 0 {
		log.Println("No models found.")
		return
	}

	modelList := tview.NewList()
	for _, name := range modelNames {
		modelList.AddItem(name, "", 0, nil)
	}

	modelList.SetBorder(true).SetTitle("Select Gemini Model (ESC to cancel)")

	closeModal := func() {
		tv.app.SetInputCapture(nil)
		tv.pages.RemovePage(modelSelectionModalPageName)
		tv.app.SetRoot(tv.flex, true)
	}

	modelList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		selectedModel := mainText
		tv.aimodel.UpdateModelName(selectedModel)
		onSelect(selectedModel) // Call the callback with the selected model
		closeModal()
	})

	tv.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			closeModal()
			return nil
		}
		return event
	})

	// Ensure pages is set up correctly.
	// SetDefaultView initializes tv.pages and adds "main".
	tv.pages.AddPage(modelSelectionModalPageName, modelList, true, true)
	tv.pages.SwitchToPage(modelSelectionModalPageName)
	tv.app.SetRoot(tv.pages, true)
}
