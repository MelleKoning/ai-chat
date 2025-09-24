package tviewview

import (
	"log"

	"github.com/MelleKoning/ai-chat/internal/terminal"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (tv *tviewApp) showStyleModal() {
	log.Println("showStyleModal: Starting")
	styles := tv.mdRenderer.AvailableGlamourStyles()
	styleTexts := make([]string, len(styles))
	for i, style := range styles {
		styleTexts[i] = style.String()
	}

	log.Printf("showStyleModal: Available styles: %v\n", styles)
	modal := tview.NewModal().
		SetText("Select a style:").
		AddButtons(styleTexts).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			log.Printf("showStyleModal: Button %d (%s) clicked\n", buttonIndex, buttonLabel)
			if buttonIndex >= 0 && buttonIndex < len(styles) {
				tv.reRenderOutputView(styles[buttonIndex])
			}

			log.Println("showStyleModal: Closing modal and returning to main view")
			tv.app.SetRoot(tv.flex, true)

			/*tv.app.QueueUpdateDraw(func() {
				log.Println("showStyleModal: Closing modal")
				tv.app.SetRoot(tv.flex, true) // Set the main view back to the flex layout
			})*/
		})

	log.Println("showStyleModal: Showing modal")
	tv.app.SetRoot(modal, true)
}

// reRenderOutputView function
func (tv *tviewApp) reRenderOutputView(selectedStyle terminal.GlamourStyle) {
	log.Println("reRenderOutputView: Starting to re-render output view")

	log.Println("showStyleModal: Creating new renderer")
	var err error
	tv.mdRenderer, err = terminal.New(selectedStyle)
	if err != nil {
		log.Printf("Error creating new renderer: %v", err)
		return
	}

	tv.progressView.SetText("Re-rendering output view with style: " + selectedStyle.String())

	// Render the markdown content with the new renderer
	// Using the handleFinalModelresult that knows how to
	// handle the re-rendering of the output view
	log.Println("reRenderOutputView: Rendering markdown content")

	// Translate current text to new Style
	markdownContent := tv.outputView.GetText(true)
	renderedResult, err := tv.mdRenderer.GetRendered(markdownContent)
	if err != nil {
		log.Printf("Error rendering markdown content: %v", err)
		tv.outputView.SetText("Error rendering markdown content: " + err.Error())
		return
	}
	txtRendered := tview.TranslateANSI(renderedResult)
	tv.outputView.SetText(txtRendered)

	// Set background color based on the current style
	tv.outputView.SetBackgroundColor(tcell.ColorBlack)
	if tv.mdRenderer.CurrentStyle() == terminal.GlamourStyleLight {
		tv.outputView.SetBackgroundColor(tcell.ColorWhite)
	}

	log.Println("reRenderOutputView: Finished re-rendering output view")

}
