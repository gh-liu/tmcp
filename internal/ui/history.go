package ui

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

const historyLimit = 500

func LoadHistory() ([]string, error) {
	path, err := historyPath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var history []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		history = append(history, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return history, nil
}

func AppendHistory(history []string, line string) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return history
	}

	if n := len(history); n > 0 && history[n-1] == line {
		return history
	}

	history = append(history, line)
	if len(history) > historyLimit {
		history = history[len(history)-historyLimit:]
	}

	return history
}

func SaveHistory(history []string) error {
	path, err := historyPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range history {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return err
		}
	}

	return writer.Flush()
}

func historyPath() (string, error) {
	if stateHome := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); stateHome != "" {
		return filepath.Join(stateHome, "tmcp", "history"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".local", "state", "tmcp", "history"), nil
}
