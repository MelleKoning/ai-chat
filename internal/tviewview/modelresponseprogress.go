package tviewview

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/rivo/tview"
)

type ModelResponseProgress struct {
	progressCount              int
	length                     int
	originalOutputViewContents string
	// added to for each chunk
	progressString   strings.Builder
	progressRendered strings.Builder
	startTime        time.Time
	tv               *tviewApp
}

func (p *ModelResponseProgress) StartCommand() {
	command := p.tv.commandArea.GetText()
	// validation
	if command == "" {
		// TODO: create modal dialog
		// to inform the userthat command is empty.
		// postpone this when we support tview.pages

		return
	}
	p.appendUserCommandToOutput(command)
	// Execute model
	p.runModelCommand(command)

}

func (p *ModelResponseProgress) appendUserCommandToOutput(command string) {
	p.tv.app.SetFocus(p.tv.progressView) // remove highlight from button
	txtRendered, err := p.tv.mdRenderer.FormatUserText(command,
		p.tv.aimodel.GetHistoryLength())
	if err != nil {
		log.Print(err)
	}
	txtRendered = tview.TranslateANSI(txtRendered)
	p.tv.outputView.SetText(p.tv.outputView.GetText(false) + txtRendered)
}

func (p *ModelResponseProgress) runModelCommand(command string) {
	go func() {
		p.startProgress()
		// the callback -can- update the outputview for intermediate results

		result, chatErr := p.tv.aimodel.ChatMessage(command, p.tv.onChunkReceived)
		// as we run in an async routine we have
		// to use the QueueUpdateDraw for all following
		// UI updates
		p.tv.app.QueueUpdateDraw(func() {
			p.tv.outputView.SetText(p.tv.progress.originalOutputViewContents) // reset back
			p.handleFinalModelResult(result, chatErr)
			// replace the command box
			p.tv.commandArea.Replace(0, len(command), "")
		})
	}()
}

// handleFinalModelResult is called async from the main thread
// therefore the app.QueueUpdateDraw is used to update the UI
// we can safely write to all the UI elements because
// this func is already called from QueueUpdateDraw
func (p *ModelResponseProgress) handleFinalModelResult(result string, chatErr error) {
	if chatErr != nil {
		p.tv.outputView.SetText(p.tv.outputView.GetText(false) + result + "\n" + chatErr.Error())
	} else {
		renderedResult, _ := p.tv.mdRenderer.GetRendered(result)
		txtRendered := tview.TranslateANSI(renderedResult)
		p.tv.outputView.SetText(p.tv.outputView.GetText(false) + txtRendered)
		// reset the progressview
		p.tv.progress.resetProgressString()
	}
	p.tv.app.SetFocus(p.tv.outputView)
}
func (p *ModelResponseProgress) startProgress() {
	p.originalOutputViewContents = p.tv.outputView.GetText(false)
	p.startTime = time.Now()
}
func (p *ModelResponseProgress) updateProgressPerChunk(chunk string) {
	p.progressCount++
	p.length += len(chunk)
	elapsedStr := p.formatElapsedTime()
	p.progressString.WriteString(chunk)
	renderedChunk, _ := p.tv.mdRenderer.GetRendered(chunk)
	chunkRendered := tview.TranslateANSI(renderedChunk)
	p.progressRendered.WriteString(chunkRendered)
	p.updateUI(elapsedStr, p.progressString)
}

func (p *ModelResponseProgress) formatElapsedTime() string {
	elapsed := time.Since(p.startTime)
	seconds := int64(elapsed.Seconds())
	milliseconds := int64(elapsed % time.Second / time.Millisecond)
	return fmt.Sprintf("%d.%03d s", seconds, milliseconds)
}

func (p *ModelResponseProgress) updateUI(elapsedStr string, txtRendered strings.Builder) {
	p.tv.app.QueueUpdateDraw(func() {
		p.tv.progressView.SetText(fmt.Sprintf("Chunks: %d / Length: %d / Time: %s", p.tv.progress.progressCount,
			p.tv.progress.length,
			elapsedStr))
		p.tv.outputView.SetText(p.originalOutputViewContents + txtRendered.String())
	})
}
func (p *ModelResponseProgress) resetProgressString() {
	p.progressString = strings.Builder{}
	p.progressRendered = strings.Builder{}
	p.length = 0
	p.progressCount = 0
}
