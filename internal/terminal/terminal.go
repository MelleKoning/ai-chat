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

const CyanBackGroundWhiteForeground = "\033[44;37m"

type GlamourStyle int

const (
	GlamourStyleDark GlamourStyle = iota
	GlamourStyleLight
	GlamourStyleNotty
	GlamourStyleDracula
	GlamourStyleTokyoNight
)

func (s GlamourStyle) String() string {
	switch s {
	case GlamourStyleDark:
		return "Dark"
	case GlamourStyleLight:
		return "Light"
	case GlamourStyleNotty:
		return "Notty"
	case GlamourStyleDracula:
		return "Dracula"
	case GlamourStyleTokyoNight:
		return "Tokyo Night"
	default:
		return "Dracula"
	}
}

type GlamourRenderer interface {
	GetRendered(string) (string, error)
	FormatUserText(string, int) (string, error)
	CurrentStyle() GlamourStyle
	AvailableGlamourStyles() []GlamourStyle
}

type glamourRenderer struct {
	gr        *glamour.TermRenderer
	styleUsed GlamourStyle
}

func getStyle(style GlamourStyle) glamour.TermRendererOption {
	switch style {
	case GlamourStyleDark:
		return glamour.WithStandardStyle("dark")
	case GlamourStyleLight:
		return glamour.WithStandardStyle("light")
	case GlamourStyleNotty:
		return glamour.WithStandardStyle("notty")
	case GlamourStyleDracula:
		return glamour.WithStandardStyle("dracula")
	case GlamourStyleTokyoNight:
		return glamour.WithStandardStyle("tokyo-night")
	default:
		return glamour.WithStandardStyle("dark")
	}
}
func New(style GlamourStyle) (GlamourRenderer, error) {
	selectedStyle := getStyle(style)

	r, err := glamour.NewTermRenderer(selectedStyle,
		glamour.WithWordWrap(120))
	if err != nil {
		return nil, err
	}
	return &glamourRenderer{
		gr:        r,
		styleUsed: style,
	}, nil
}

// GetRendered executs a Glamour action on a markdown string
// to colorize it with ANSI colour codes and returns the result
func (gr *glamourRenderer) GetRendered(str string) (string, error) {
	return gr.gr.Render(str)
}

func (gr *glamourRenderer) FormatUserText(str string, historyLength int) (string, error) {
	s := fmt.Sprintf(colorCyan+"History items: %d\n", historyLength)
	s = s + fmt.Sprintf(CyanBackGroundWhiteForeground+"%s\n"+colorReset, str)
	return s, nil
}

func (g *glamourRenderer) CurrentStyle() GlamourStyle {
	return g.styleUsed
}

// ...existing code...

func (g *glamourRenderer) AvailableGlamourStyles() []GlamourStyle {
	var allStyles []GlamourStyle
	for i := 0; i <= int(GlamourStyleTokyoNight); i++ {
		allStyles = append(allStyles, GlamourStyle(i))
	}
	return allStyles
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
