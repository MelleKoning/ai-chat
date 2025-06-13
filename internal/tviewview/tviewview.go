package tviewview

import (
	"github.com/MelleKoning/ai-chat/internal/genaimodel"
	"github.com/MelleKoning/ai-chat/internal/terminal"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type tviewApp struct {
	app            *tview.Application
	mdRenderer     terminal.GlamourRenderer // can render markdown colours
	flex           *tview.Flex              // the main screen set to root
	commandArea    *tview.TextArea
	dropDown       *tview.DropDown
	titleView      *tview.TextView
	outputView     *tview.TextView
	submitButton   *tview.Button
	progressView   *tview.TextView
	progress       ModelResponseProgress
	aimodel        genaimodel.Action
	selectedPrompt string
	pages          *tview.Pages // to support modal dialog
}

type TviewApp interface {
	Run() error
	SetDefaultView()
	Output() string
}

func (tv *tviewApp) Output() string {
	outputViewText := tv.outputView.GetText(true)

	renderedTxt, _ := tv.mdRenderer.GetRendered(outputViewText)

	return renderedTxt
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
	tv.progress = ModelResponseProgress{tv: tv}
	tv.createTitleView()
	tv.createOutputView()
	tv.createTextArea()
	tv.createSubmitButton()
	tv.createDropDown()
	tv.createProgressView()
	tv.SetDefaultView()
	tv.app.SetRoot(tv.flex, true)

	return tv
}

func (tv *tviewApp) createTitleView() {
	tv.titleView = tview.NewTextView().
		SetText("AI Chat").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	tv.titleView.SetBorder(false)

	tv.titleView.SetBackgroundColor(tcell.ColorDefault)
	tv.titleView.SetTextColor(tcell.ColorDefault)
}

func (tv *tviewApp) createProgressView() {
	tv.progressView = tview.NewTextView().
		SetText("").SetDynamicColors(true)
	tv.progressView.SetBorder(false)

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
	tv.outputView.SetBorder(false).
		SetFocusFunc(func() {
			tv.titleView.SetTextColor(tcell.ColorWhite)
			tv.titleView.SetBackgroundColor(tcell.ColorDarkBlue)
		}).SetBlurFunc(func() {
		tv.titleView.SetTextColor(tcell.ColorGray)
		tv.titleView.SetBackgroundColor(tcell.ColorDarkBlue)
	})
	//tv.outputView.SetTextStyle(tcell.StyleDefault)
}

func (tv *tviewApp) createTextArea() {
	// Create an input field for user input
	tv.commandArea = tview.NewTextArea()

	tv.commandArea.SetBorder(true)
	tv.commandArea.SetTitle("Enter command: ")
	// Capture key events for the text area
	tv.commandArea.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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
	tv.app.SetFocus(tv.commandArea).EnablePaste(true)
}

func (tv *tviewApp) createSubmitButton() {
	tv.submitButton = tview.NewButton("Submit").SetSelectedFunc(
		tv.progress.StartCommand,
	).
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
			"ReviewFile",
			"Select system prompt",
			"Store Chat History",
			"Load Chat History",
			"ListModels",
			"Exit"}, func(option string, index int) {
			switch option {
			case "Exit":
				tv.app.Stop()

			case "Select system prompt":
				tv.SelectSystemPrompt()
			case "ReviewFile":
				// Prompt user for file path (simple version: use textArea input)
				filePath := "gitdiff.txt"
				tv.progress.appendUserCommandToOutput("[ReviewFile] " + filePath)
				tv.progress.originalOutputViewContents = tv.outputView.GetText(false)
				go func() { // async for the chunk updates
					result, err := tv.aimodel.ReviewFile(tv.progress.onChunkReceived)
					tv.UpdateOutputView(result, err)
				}()
			case "Store Chat History":
				tv.storeChatHistory()

			case "Load Chat History":
				tv.SelectChatHistoryFile()

			case "ListModels":
				go func() {
					tv.UpdateOutputView(tv.aimodel.ListModels())
				}()
			}
		}).SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyTAB {
			tv.app.SetFocus(tv.commandArea)
		}
	})
}

func (tv *tviewApp) UpdateOutputView(result string, err error) {
	tv.app.QueueUpdateDraw(func() {
		if err != nil {
			tv.outputView.SetText(tv.outputView.GetText(false) + err.Error())
		} else {
			renderedResult, _ := tv.mdRenderer.GetRendered(result)
			txtRendered := tview.TranslateANSI(renderedResult)
			tv.outputView.SetText(tv.progress.originalOutputViewContents + txtRendered)
		}
		tv.app.SetFocus(tv.outputView)
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
		AddItem(tv.titleView, 1, 1, false).
		AddItem(tv.outputView, 0, 10, true).
		AddItem(tv.dropDown, 1, 1, true).
		AddItem(tv.commandArea, 0, 3, true).
		AddItem(buttonRow, 1, 1, true)

	tv.pages = tview.NewPages()

	tv.pages.AddPage("main", tv.flex, true, true)

}
