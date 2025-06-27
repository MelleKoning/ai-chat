package fileio

import (
	"os"
	"path/filepath"
)

func StoreChatHistory(filename string, jsonData []byte) error {
	historyDir, err := historyDirectory()
	if err != nil {
		return err
	}

	// Construct the full file path
	filePath := filepath.Join(historyDir, filename)

	err = os.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return err
	}

	return nil
}

func LoadChatHistory(filename string) ([]byte, error) {
	historyDir, err := historyDirectory()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(historyDir, filename)

	// 3. Read the JSON data from the file
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func historyDirectory() (string, error) {
	// Get the user's configuration directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	// Create the chat history directory if it doesn't exist
	historyDir := filepath.Join(configDir, "ai-chat", "history")
	if _, err := os.Stat(historyDir); os.IsNotExist(err) {
		err := os.MkdirAll(historyDir, 0755)
		if err != nil {
			return "", err
		}
	}

	return historyDir, nil
}
