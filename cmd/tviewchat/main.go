package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/MelleKoning/ai-chat/internal/genaimodel"
	"github.com/MelleKoning/ai-chat/internal/terminal"
	"github.com/MelleKoning/ai-chat/internal/tviewview"
)

// Config holds the configuration for the application.
type Config struct {
	DisplayStyle terminal.GlamourStyle `json:"DisplayStyle"`
}

func (c Config) loadConfig() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		c.DisplayStyle = terminal.GlamourStyleDracula // Default style
		return
	}

	configFilePath := filepath.Join(configDir, "ai-chat", "config.json")
	fileOpen, err := os.Open(configFilePath)
	if err != nil {
		log.Println("Error opening config file:", err)
		c.DisplayStyle = terminal.GlamourStyleDracula // Default style
		return
	}

	defer func() {
		err = fileOpen.Close()
		if err != nil {
			log.Println("Error closing config file:", err)
		}
	}()

	decoder := json.NewDecoder(fileOpen)
	if err := decoder.Decode(&c); err != nil {
		log.Println("Error decoding config file:", err)
		c.DisplayStyle = terminal.GlamourStyleDracula // Default style
		return
	}
}

func (c Config) saveConfig() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(configDir, "ai-chat")
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return err
	}
	configFilePath := filepath.Join(configPath, "config.json")
	file, err := os.Create(configFilePath)
	if err != nil {
		return err
	}
	defer func() {
		err = file.Close()
		if err != nil {
			log.Println("Error closing config file:", err)
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(c)
}

func main() {
	var config Config
	config.loadConfig()
	mdRenderer, err := terminal.New(config.DisplayStyle)
	if err != nil {
		fmt.Println(err)
		return
	}

	systemPrompt := `Be a supportive technical assistant.`

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("Error: GEMINI_API_KEY environment variable not set. Please set it before running.")
	}

	ctx := context.Background()
	genaiClient, err := genaimodel.NewGeminiClient(ctx, apiKey)
	if err != nil {
		log.Fatal("Error creating Gemini client: ", err)
	}
	modelAction, err := genaimodel.NewModel(ctx, genaiClient, systemPrompt)
	if err != nil {
		log.Fatal("Error creating AI model: ", err)
	}
	// Create the console view
	tviewApp := tviewview.New(mdRenderer, modelAction)

	// We want to have a default log
	closeFile := SetupLogging()
	defer closeFile()
	// Run the application
	if err := tviewApp.Run(); err != nil {
		log.Fatal(err)
	}

	SaveAppConfig(tviewApp.CurrentGlamourStyle(), &config)
	fmt.Println(tviewApp.Output())

}

func SaveAppConfig(style terminal.GlamourStyle, config *Config) {
	config.DisplayStyle = style

	err := config.saveConfig()
	if err != nil {
		log.Println("Error saving config:", err)
	} else {
		log.Println("Configuration saved successfully.")
	}

}
func SetupLogging() func() {
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
		log.Println("Application exiting")
		err := logFile.Close()
		if err != nil {
			log.Println("Error closing log file:", err)
		} else {
			log.Println("Log file closed")
		}
	}
}
