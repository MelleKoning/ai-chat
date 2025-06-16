package tviewview

import (
	"fmt"
	"log"
	"strings"
	"sync"
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

// startSpinner is a helper function to show a spinner in the terminal
func (p *ModelResponseProgress) startSpinnerGoroutine(stopChan chan struct{}) {
	chars := `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`
	spinnerRunes := []rune(chars)
	i := 0

	// Create a ticker that fires every X milliseconds
	// Adjust the duration for desired spinner speed. 75ms might be more visible than 49ms.
	ticker := time.NewTicker(75 * time.Millisecond)
	defer ticker.Stop() // Ensure the ticker is stopped when the goroutine exits

	// --- IMPORTANT ADDITION ---
	// Queue the very first spinner frame immediately, before entering the loop.
	// This ensures that even if the first chunk comes back extremely fast (e.g., within 1ms),
	// there's a chance for at least one spinner character to be queued and displayed.
	p.tv.app.QueueUpdateDraw(func() {
		p.tv.outputView.SetText(p.userCommandRendered)
		thinkingString := fmt.Sprintf("Thinking... %c (%d)", spinnerRunes[0], 0)
		p.tv.progressView.SetText(thinkingString)
	})
	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			elapsedDuration := time.Since(p.progressData.startTime)

			p.tv.app.QueueUpdateDraw(func() {
				thinkingString := fmt.Sprintf("Thinking... %c (%s)", spinnerRunes[i], elapsedDuration.String())
				p.tv.progressView.SetText(thinkingString)
			})
			i = (i + 1) % len(spinnerRunes)
		}
	}
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
	userCommandRendered        string
	chunksReceived             strings.Builder
	tv                         *tviewApp
	StopSpinner                chan struct{}
	closeSpinnerOnce           sync.Once
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

// appendUserCommandToOutput appends the user command to the output view
// this is important so that the originalOutput also contains the
// user command
func (p *ModelResponseProgress) appendUserCommandToOutput(command string) {
	p.tv.app.SetFocus(p.tv.progressView) // remove highlight from button
	txtRendered, err := p.tv.mdRenderer.FormatUserText(command,
		p.tv.aimodel.GetHistoryLength())
	if err != nil {
		log.Print(err)
	}
	txtRendered = tview.TranslateANSI(txtRendered)
	p.userCommandRendered = txtRendered
	var sb strings.Builder
	sb.WriteString(p.tv.outputView.GetText(false))
	sb.WriteString(txtRendered)
	p.tv.outputView.SetText(sb.String())
}

// runModelCommand does a few things in a specific order:
// 1. Perform an immediate UI update ( appendUserCommandToOutput ) on the main thread.
// 2. Then, in a background goroutine, asynchronously get a snapshot of that UI state
// for the originalOutputView, which requires  QueueUpdateDraw
// to go back to the main thread).
// 3. Manage ongoing streaming updates via p.onChunkReceived (each requiring  QueueUpdateDraw ).
// 4. Finally, perform a concluding update ( QueueUpdateDraw ).
func (p *ModelResponseProgress) runModelCommand(command string) {
	p.appendUserCommandToOutput(command)
	// Start async operationas for model call, spinner, final result handling
	go func() {
		p.tv.app.QueueUpdateDraw(func() {
			// CRITICAL: Capture the outputView's content *after* the user command
			// has been appended and processed by the main UI goroutine.
			// This `GetText(false)` will now correctly contain the history + the user's command.
			p.originalOutputViewContents = p.tv.outputView.GetText(false)

			// Set focus to the progress view as part of the preparation for showing progress.
			p.tv.app.SetFocus(p.tv.progressView)
		})

		p.startProgress()
		// the callback -can- update the outputview for intermediate results
		result, chatErr := p.tv.aimodel.ChatMessage(command, p.onChunkReceived)
		// Final UI update after the model returns the result
		p.tv.app.QueueUpdateDraw(func() {
			p.tv.outputView.SetText(p.tv.progress.originalOutputViewContents) // reset back
			p.handleFinalModelResult(result, chatErr)
			// clear the command area
			p.tv.commandArea.Replace(0, len(command), "")
		})
	}()
}

// handleFinalModelResult is called async from the main thread
// therefore the app.QueueUpdateDraw is used to update the UI
// we can safely write to all the UI elements because
// this func is already called from QueueUpdateDraw
func (p *ModelResponseProgress) handleFinalModelResult(result genaimodel.ChatResult, chatErr error) {
	// Ensure the spinner is stopped, even if no chunks were received (onChunkReceived was never called).
	// This is safe due to sync.Once; it will only execute if it hasn't already been executed by onChunkReceived.
	p.closeSpinnerOnce.Do(func() {
		close(p.StopSpinner)
	})

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
	//p.originalOutputViewContents = p.tv.outputView.GetText(false)
	p.progressData = ProgressData{
		startTime: time.Now(),
	}
	p.chunksReceived = strings.Builder{}
	p.StopSpinner = make(chan struct{})       // Create a NEW channel for each command
	p.closeSpinnerOnce = sync.Once{}          // IMPORTANT: Reinitialize sync.Once for each command
	go p.startSpinnerGoroutine(p.StopSpinner) // Start the spinner goroutine with the new channel
}

func (p *ModelResponseProgress) onChunkReceived(chunk string) {
	p.closeSpinnerOnce.Do(func() {
		close(p.StopSpinner)
	})
	p.progressData.Update(chunk)
	p.chunksReceived.WriteString(chunk)
	renderedProgress, _ := p.tv.mdRenderer.GetRendered(p.chunksReceived.String())
	tviewProgressRendered := tview.TranslateANSI(renderedProgress)
	p.updateUI(p.userCommandRendered + "\n" + tviewProgressRendered)
}

func (p *ModelResponseProgress) updateUI(txtRendered string) {
	p.progressData.elapsedTime = time.Since(p.progressData.startTime)
	p.tv.app.QueueUpdateDraw(func() {
		p.tv.progressView.SetText(p.progressData.String())
		p.tv.outputView.SetText(txtRendered)
	})
}
