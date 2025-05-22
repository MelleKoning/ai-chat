package tviewview

import (
	"fmt"
	"log"

	"github.com/MelleKoning/ai-chat/internal/genaimodel"
	"github.com/MelleKoning/ai-chat/internal/terminal"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ModelResponseProgress struct {
	progressCount  int
	length         int
	beforeContents string
	// added to for each chunk
	progressString string
}
type tviewApp struct {
	app          *tview.Application
	mdRenderer   terminal.GlamourRenderer // can render markdown colours
	flex         *tview.Flex
	textArea     *tview.TextArea
	dropDown     *tview.DropDown
	outputView   *tview.TextView
	submitButton *tview.Button
	progressView *tview.TextView
	promptView   *tview.TextArea
	progress     ModelResponseProgress
	aimodel      genaimodel.Action
}

type TviewApp interface {
	Run() error
	SetDefaultView()
}

// New will create a new VIEW on the terminal
// Always call a view, for example "SetDefaultView"
// to initialize the view container with a default view
// TODO Expose a good interface for this
func New(mdrenderer terminal.GlamourRenderer,
	aimodel genaimodel.Action) TviewApp {
	tv := &tviewApp{
		app:        tview.NewApplication(),
		mdRenderer: mdrenderer,
		aimodel:    aimodel,
		flex: tview.NewFlex().SetDirection(
			tview.FlexRow,
		),
	}
	tv.createOutputView()
	tv.createTextArea()
	tv.createSubmitButton()
	tv.createDropDown()
	tv.createProgressView()
	tv.SetDefaultView()
	tv.app.SetRoot(tv.flex, true)

	return tv
}

func (tv *tviewApp) createProgressView() {
	tv.progressView = tview.NewTextView().
		SetText("").SetDynamicColors(true)
	//.SetChangedFunc(func() {
	// redraw when text changes
	// see onChunkReceived
	//tv.app.Draw()
	//})
	tv.progressView.SetBorder(false)

}

func (tv *tviewApp) onChunkReceived(str string) {

	tv.progress.progressCount++
	tv.progress.length += len(str)

	// keep track of the progress,
	// progress is reset in handleModelResult
	tv.progress.progressString += str
	renderedResult, _ := tv.mdRenderer.GetRendered(tv.progress.progressString)
	txtRendered := tview.TranslateANSI(renderedResult)

	tv.app.QueueUpdateDraw(func() {
		tv.progressView.SetText(fmt.Sprintf("Progress: %d/%d", tv.progress.progressCount, tv.progress.length))
		tv.outputView.SetText(tv.progress.beforeContents + txtRendered)
	})

}

func (tv *tviewApp) Run() error {
	err := tv.app.Run()
	if err != nil {
		return err
	}

	return nil
}
func (tv *tviewApp) createOutputView() {
	// Create a text view for displaying output
	// contains the logic for rendering
	tv.outputView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetSize(0, 0).
		SetChangedFunc(func() {
			tv.outputView.ScrollToEnd()
		}).SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyTAB {
			tv.app.SetFocus(tv.dropDown)
		}
	})

	tv.outputView.SetBorder(true).SetBackgroundColor(tcell.ColorBlack)
}

func (tv *tviewApp) createTextArea() {
	// Create an input field for user input
	tv.textArea = tview.NewTextArea().
		SetLabel("Enter command: ")
	tv.textArea.SetBorder(true)
	// Capture key events for the text area
	tv.textArea.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTAB {
			tv.app.SetFocus(tv.submitButton) // Move focus to the submit button
			return nil                       // Consume the event
		}
		if event.Key() == tcell.KeyEnter {
			return event // nil // Consume other events - do nothing!
		}
		return event
	})

	// we have to enable pasting for the user
	// for the whole app so that user can
	// paste into the Text Area
	tv.app.SetFocus(tv.textArea).EnablePaste(true)
}

func (tv *tviewApp) appendUserCommandToOutput(command string) {
	tv.app.SetFocus(tv.progressView) // remove highlight from button
	txtRendered, err := tv.mdRenderer.FormatUserText(command,
		tv.aimodel.GetHistoryLength())
	if err != nil {
		log.Print(err)
	}
	txtRendered = tview.TranslateANSI(txtRendered)
	tv.outputView.SetText(tv.outputView.GetText(false) + txtRendered)
}

func (tv *tviewApp) runModelCommand(command string) {
	go func() {
		// remember the original contents of the output view
		tv.progress.beforeContents = tv.outputView.GetText(false)
		// the callback -can- update the outputview for intermediate results
		result, chatErr := tv.aimodel.ChatMessage(command, tv.onChunkReceived)
		// as we run in an async routine we have
		// to use the QueueUpdateDraw for all following
		// UI updates
		tv.app.QueueUpdateDraw(func() {
			tv.outputView.SetText(tv.progress.beforeContents) // reset back
			tv.handleModelResult(result, chatErr)
		})
	}()
}

// handleModelResult is called async from the main thread
// therefore the app.QueueUpdateDraw is used to update the UI
// we can safely write to all the UI elements because
// this func is already called from QueueUpdateDraw
func (tv *tviewApp) handleModelResult(result string, chatErr error) {
	if chatErr != nil {
		tv.outputView.SetText(tv.outputView.GetText(false) + chatErr.Error())
	} else {
		renderedResult, _ := tv.mdRenderer.GetRendered(result)
		txtRendered := tview.TranslateANSI(renderedResult)
		tv.outputView.SetText(tv.outputView.GetText(false) + txtRendered)
		// reset the progressview
		tv.progress = ModelResponseProgress{}
	}
	tv.app.SetFocus(tv.outputView)
}

func (tv *tviewApp) createSubmitButton() {
	tv.submitButton = tview.NewButton("Submit").SetSelectedFunc(
		func() {
			command := tv.textArea.GetText()
			tv.appendUserCommandToOutput(command)
			// Execute model
			tv.runModelCommand(command)

		}).
		SetExitFunc(func(key tcell.Key) {
			if key == tcell.KeyTAB {
				tv.app.SetFocus(tv.outputView)
			}
		},
		)
}

// we create a dropdown, but it should be fed with
// some model data instead of hardcoded static data
func (tv *tviewApp) createDropDown() {
	// Create a dropdown for selecting options
	tv.dropDown = tview.NewDropDown().
		SetLabel("Select option: ").
		SetOptions([]string{
			"Continue",
			"ReviewFile",
			"OutputView",
			"PromptView",
			"SystemPrompt",
			"Exit"}, func(option string, index int) {
			switch option {
			case "Exit":
				tv.app.Stop()
			case "OutputView":
				tv.SetDefaultView()
			case "PromptView":
				// TODO: move switching of view
				// to a function
				tv.flex.RemoveItem(tv.outputView)
				tv.flex.AddItem(tv.promptView, 0, 10, true)
			case "SystemPrompt":
				response := tv.aimodel.SendSystemPrompt()
				renderedResult, _ := tv.mdRenderer.GetRendered(response)
				txtRendered := tview.TranslateANSI(renderedResult)
				tv.outputView.SetText(tv.outputView.GetText(false) + txtRendered)
			case "ReviewFile":
				// Prompt user for file path (simple version: use textArea input)
				filePath := "gitdiff.txt"
				tv.appendUserCommandToOutput("[ReviewFile] " + filePath)
				go func() {
					result, err := tv.aimodel.ReviewFile(tv.onChunkReceived)
					tv.app.QueueUpdateDraw(func() {
						if err != nil {
							tv.outputView.SetText(tv.outputView.GetText(false) + "[ReviewFile Error] " + err.Error())
						} else {
							renderedResult, _ := tv.mdRenderer.GetRendered(result)
							txtRendered := tview.TranslateANSI(renderedResult)
							tv.outputView.SetText(tv.outputView.GetText(false) + txtRendered)
						}
						tv.app.SetFocus(tv.outputView)
					})
				}()
			}
			if option == "Exit" {
				tv.app.Stop()
			}
			tv.outputView.SetText(tv.outputView.GetText(false) + option)
		}).SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyTAB {
			tv.app.SetFocus(tv.textArea)
		}
	})
}

// SetDefaultView will set the default view
// of the tviewApp
func (tv *tviewApp) SetDefaultView() {
	tv.flex.Clear()
	buttonRow := tview.NewFlex().
		AddItem(tv.submitButton, 0, 1, false).
		AddItem(tv.progressView, 0, 1, false)
	tv.flex.
		SetDirection(tview.FlexRow).
		AddItem(tv.outputView, 0, 10, true).
		AddItem(tv.dropDown, 1, 1, true).
		AddItem(tv.textArea, 0, 3, true).
		AddItem(buttonRow, 1, 1, true)
}
