package tviewview

import (
	"log"

	"github.com/MelleKoning/ai-chat/internal/prompts"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// createPromptSelectionModal creates a modal to select a prompt
// the returned channel contains the selected prompt
func (tv *tviewApp) createPromptSelectionModal() chan prompts.Prompt {
	selectedPromptChan := make(chan prompts.Prompt)
	// Helper function to wrap text at a specified width
	truncateText := func(text string, width int) string {
		if len(text) <= width {
			return text
		}
		return text[:width-3] + "..."
	}

	// Create a list to display prompts
	promptList := tview.NewList()
	for _, prompt := range prompts.PromptList {
		promptList.AddItem(truncateText(prompt.Name, 25), "", 0, nil)
	}
	promptList.SetBorder(true)

	// Create a text view to display the selected prompt's content
	selectedPromptView := tview.NewTextView().
		SetDynamicColors(true).SetScrollable(true)
	selectedPromptView.SetBorder(true)

	// Update the selected prompt's content when the selection changes
	promptList.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		selectedPromptView.SetText(
			prompts.PromptList[index].Name + "\n\n" +
				prompts.PromptList[index].Prompt)
	})

	// Set input capture to switch focus between list and text view
	tv.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTAB {
			if tv.app.GetFocus() == promptList {
				tv.app.SetFocus(selectedPromptView)
			} else {
				tv.app.SetFocus(promptList)
			}
			return nil
		}
		return event
	})

	// Create a modal with the list on the left and the selected prompt view on the right
	modal := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(promptList, 0, 1, true).
		AddItem(selectedPromptView, 0, 3, false)

	// Handle prompt selection
	promptList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		// This sets the selected prompt for further use
		selectedPromptChan <- prompts.PromptList[index]
		log.Printf("selected prompt %s", tv.selectedPrompt)
		tv.app.SetInputCapture(nil)   // undo the override of the TAB key
		tv.app.SetRoot(tv.flex, true) // Close the modal
	})

	// Set the modal as the root of the application
	tv.app.SetRoot(modal, true)

	return selectedPromptChan
}

func (tv *tviewApp) SelectSystemPrompt() {
	tv.progress.originalOutputViewContents = tv.outputView.GetText(false)

	// Open the prompt selection modal
	selectedPromptChan := tv.createPromptSelectionModal()

	// to enable tview to draw on the main thread
	// we have to await the modal response in a goroutine
	go func() {
		prompt := <-selectedPromptChan
		tv.selectedPrompt = prompt.Prompt
		log.Println("Selected prompt:", prompt.Name)
		tv.aimodel.UpdateSystemInstruction(tv.selectedPrompt)
		// the callback -can- update the outputview for intermediate results
		tv.progress.startProgress()
		finalResult, chatErr := tv.aimodel.SendSystemPrompt(tv.progress.onChunkReceived)
		// as we run in an async routine we have
		// to use the QueueUpdateDraw for UI updates
		tv.app.QueueUpdateDraw(func() {
			tv.outputView.SetText(tv.progress.originalOutputViewContents) // reset back
			tv.progress.handleFinalModelResult(finalResult, chatErr)
		})
	}()

}
