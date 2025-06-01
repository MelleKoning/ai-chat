package terminal

import (
	"fmt"

	glamour "github.com/charmbracelet/glamour"
)

// Define ANSI escape codes for the desired color (e.g., green)
const colorGreen = "\033[32m"
const colorReset = "\033[0m"

// const colorYellow = "\033[33m"
const backGroundBlack = "\033[40m"

// const AttrReversed = "\033[7m"
const colorCyan = "\033[36m"

type GlamourRenderer interface {
	GetRendered(string) (string, error)
	FormatUserText(string, int) (string, error)
}

type glamourRenderer struct {
	gr *glamour.TermRenderer
}

func New() (GlamourRenderer, error) {
	selectedStyle := glamour.WithStandardStyle("dark")

	r, err := glamour.NewTermRenderer(selectedStyle,
		glamour.WithWordWrap(120))
	if err != nil {
		return nil, err
	}
	return &glamourRenderer{
		gr: r,
	}, nil
}

// GetRendered executs a Glamour action on a markdown string
// to colorize it with ANSI colour codes and returns the result
func (gr *glamourRenderer) GetRendered(str string) (string, error) {
	return gr.gr.Render(str)
}

func (gr *glamourRenderer) FormatUserText(str string, historyLength int) (string, error) {
	s := fmt.Sprintf(colorCyan+"History items: %d\n", historyLength)
	s = s + str
	return s, nil
}
func PrintGlamourString(theString string) {
	termRenderer, err := glamour.NewTermRenderer(glamour.WithWordWrap(120), glamour.WithStandardStyle("dracula"))
	if err != nil {
		fmt.Println("can not initialize termRenderer")
	}
	result, err := termRenderer.Render(theString)
	if err != nil {
		panic(err)
	}

	markdown := string(result)
	fmt.Print(markdown)
}

func PrintPrompt(historyLength int) {
	fmt.Printf("History items: %d\n", historyLength)
	fmt.Print(colorGreen + "('exit' to quit, `file` to upload, `prompt` to update systeminstruction) ")
	fmt.Println(colorCyan + backGroundBlack) // will be the typing colour
}

func PrintColourReset() {
	fmt.Print(colorReset)
}
