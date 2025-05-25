package tviewview

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/MelleKoning/ai-chat/internal/genaimodel"
	"github.com/rivo/tview"
)

// ProgressData encapsulates the data used to update the progress view.
type ProgressData struct {
	chunkCount      int
	length          int
	elapsedTime     time.Duration
	startTime       time.Time
	chunksPerSecond float64
}

func (pd *ProgressData) Update(chunk string) {
	pd.chunkCount++
	pd.length += len(chunk)
	pd.elapsedTime = time.Since(pd.startTime)
	if pd.elapsedTime > 0 {
		pd.chunksPerSecond = float64(pd.chunkCount) / pd.elapsedTime.Seconds()
	} else {
		pd.chunksPerSecond = 0
	}
}

func (pd *ProgressData) SetFinalResult(chunkCount int, length int) {
	pd.chunkCount = chunkCount
	pd.length = length

	pd.elapsedTime = time.Since(pd.startTime)
	if pd.elapsedTime > 0 {
		pd.chunksPerSecond = float64(pd.chunkCount) / pd.elapsedTime.Seconds()
	} else {
		pd.chunksPerSecond = 0
	}
}

// String returns a formatted string representation of the ProgressData.
func (pd *ProgressData) String() string {
	elapsedStr := fmt.Sprintf("%d.%03d s", int64(pd.elapsedTime.Seconds()),
		int64(pd.elapsedTime%time.Second/time.Millisecond))
	return fmt.Sprintf("Chunks: %d / Length: %d / Time: %s / Chunks/s: %.1f",
		pd.chunkCount, pd.length, elapsedStr, pd.chunksPerSecond)
}

type ModelResponseProgress struct {
	progressData               ProgressData
	originalOutputViewContents string
	// added to for each chunk
	chunksReceived strings.Builder
	tv             *tviewApp
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
	var sb strings.Builder
	sb.WriteString(p.tv.outputView.GetText(false))
	sb.WriteString(txtRendered)
	p.tv.outputView.SetText(sb.String())
}

func (p *ModelResponseProgress) runModelCommand(command string) {
	p.appendUserCommandToOutput(command)
	go func() {
		p.startProgress()
		// the callback -can- update the outputview for intermediate results
		result, chatErr := p.tv.aimodel.ChatMessage(command, p.onChunkReceived)
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
func (p *ModelResponseProgress) handleFinalModelResult(result genaimodel.ChatResult, chatErr error) {
	if chatErr != nil {
		p.tv.outputView.SetText(p.tv.outputView.GetText(false) + result.Response + "\n" + chatErr.Error())
	} else {
		renderedResult, _ := p.tv.mdRenderer.GetRendered(result.Response)
		txtRendered := tview.TranslateANSI(renderedResult)
		p.tv.outputView.SetText(p.tv.outputView.GetText(false) + txtRendered)
		// set last progress to progressView
		p.progressData.SetFinalResult(result.ChunkCount, len(result.Response))
		p.tv.progressView.SetText(p.progressData.String())
	}
	p.tv.app.SetFocus(p.tv.outputView)
}
func (p *ModelResponseProgress) startProgress() {
	p.originalOutputViewContents = p.tv.outputView.GetText(false)
	p.progressData = ProgressData{
		startTime: time.Now(),
	}
	p.chunksReceived = strings.Builder{}
}

func (p *ModelResponseProgress) onChunkReceived(chunk string) {
	p.progressData.Update(chunk)
	p.chunksReceived.WriteString(chunk)
	renderedProgress, _ := p.tv.mdRenderer.GetRendered(p.chunksReceived.String())
	tviewProgressRendered := tview.TranslateANSI(renderedProgress)
	p.updateUI(tviewProgressRendered)
}

func (p *ModelResponseProgress) updateUI(txtRendered string) {
	p.progressData.elapsedTime = time.Since(p.progressData.startTime)
	p.tv.app.QueueUpdateDraw(func() {
		p.tv.progressView.SetText(p.progressData.String())
		//p.tv.outputView.SetText(p.originalOutputViewContents + txtRendered)
		p.tv.outputView.SetText(txtRendered)
	})
}
