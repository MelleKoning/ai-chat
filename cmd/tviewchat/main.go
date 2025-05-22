package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/MelleKoning/ai-chat/internal/genaimodel"
	"github.com/MelleKoning/ai-chat/internal/terminal"
	"github.com/MelleKoning/ai-chat/internal/tviewview"
)

func main() {
	mdRenderer, err := terminal.New()
	if err != nil {
		fmt.Println(err)
		return
	}

	systemPrompt := `Be a supportive technical assistant.`

	ctx := context.Background()
	modelAction, _ := genaimodel.NewModel(ctx, systemPrompt)

	// Create the console view
	tviewApp := tviewview.New(mdRenderer, modelAction)

	// We want to have a default log
	closeFile := OpenTheLog()
	defer closeFile()
	// Run the application
	if err := tviewApp.Run(); err != nil {
		log.Fatal(err)
	}
}

func OpenTheLog() func() {
	// --- Logging Setup ---
	logFile, err := os.OpenFile("tviewapp.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err) // Print to console if logfile fails
		os.Exit(1)                                  // Exit if we can't log
	}
	//defer logFile.Close() // Moved defer closer to end of main()

	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile) // Include timestamp andline number
	log.Println("Application started")
	//... Rest of your code

	return func() {
		err := logFile.Close()
		if err != nil {
			log.Println("Error closing log file:", err)
		} else {
			log.Println("Log file closed")
		}
	}
}
