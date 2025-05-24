package tviewview

import (
	"fmt"
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
