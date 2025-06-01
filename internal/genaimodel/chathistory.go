package genaimodel

import (
	"encoding/json"
	"os"
	"path/filepath"

	"google.golang.org/genai"
)

func (m *theModel) StoreChatHistory(filename string) error {
	// 1. Get the user's configuration directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	// 2. Create the chat history directory if it doesn't exist
	historyDir := filepath.Join(configDir, "ai-chat", "history")
	if _, err := os.Stat(historyDir); os.IsNotExist(err) {
		err := os.MkdirAll(historyDir, 0755)
		if err != nil {
			return err
		}
	}

	// 3. Construct the full file path
	filePath := filepath.Join(historyDir, filename)

	// 4. Serialize the chatHistory to JSON
	jsonData, err := json.MarshalIndent(m.chatHistory, "", "  ")
	if err != nil {
		return err
	}

	// 5. Write the JSON data to the file
	err = os.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (m *theModel) LoadChatHistory(filename string) ([]*genai.Content, error) {
	// 1. Get the user's configuration directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	// 2. Construct the full file path
	historyDir := filepath.Join(configDir, "ai-chat", "history")
	filePath := filepath.Join(historyDir, filename)

	// 3. Read the JSON data from the file
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// 4. Deserialize the JSON data into chatHistory
	err = json.Unmarshal(jsonData, &m.chatHistory)
	if err != nil {
		return nil, err
	}

	return m.chatHistory, nil
}
