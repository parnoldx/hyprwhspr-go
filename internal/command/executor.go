package command

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Executor handles command mode execution
type Executor struct {
	enabled  bool
	commands map[string]string
}

// NewExecutor creates a new command executor
func NewExecutor(enabled bool, commands map[string]string) *Executor {
	return &Executor{
		enabled:  enabled,
		commands: commands,
	}
}

// Execute processes the transcribed text and either executes a command or returns false
// Returns (wasCommand, error)
func (e *Executor) Execute(text string) (bool, error) {
	if !e.enabled || text == "" {
		return false, nil
	}

	// Split text into words
	words := strings.Fields(text)
	if len(words) == 0 {
		return false, nil
	}

	// Check if first word is a command
	// Strip trailing punctuation from the first word to handle cases like "Note," or "Note."
	firstWord := strings.ToLower(strings.TrimRight(words[0], ".,!?;:"))
	scriptPath, exists := e.commands[firstWord]
	if !exists {
		return false, nil
	}

	// It's a command! Extract remaining text
	remainingText := ""
	if len(words) > 1 {
		remainingText = strings.Join(words[1:], " ")
	}

	fmt.Printf("🎯 Command mode: '%s' -> %s\n", firstWord, scriptPath)
	fmt.Printf("   Arguments: '%s'\n", remainingText)

	// Execute the script
	return true, e.executeScript(scriptPath, remainingText)
}

// executeScript runs the script with the provided text as arguments
func (e *Executor) executeScript(scriptPath, text string) error {
	// Expand home directory if needed
	if strings.HasPrefix(scriptPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			scriptPath = strings.Replace(scriptPath, "~", homeDir, 1)
		}
	}

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("script not found: %s", scriptPath)
	}

	// Check if script is executable
	info, err := os.Stat(scriptPath)
	if err != nil {
		return fmt.Errorf("cannot stat script: %w", err)
	}

	if info.Mode()&0111 == 0 {
		return fmt.Errorf("script is not executable: %s", scriptPath)
	}

	// Execute the script with text as argument
	cmd := exec.Command(scriptPath, text)
	cmd.Env = os.Environ()

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("script execution failed: %w\nOutput: %s", err, string(output))
	}

	if len(output) > 0 {
		fmt.Printf("📋 Script output: %s\n", string(output))
	}

	return nil
}

// IsEnabled returns whether command mode is enabled
func (e *Executor) IsEnabled() bool {
	return e.enabled
}

// GetCommands returns the command map
func (e *Executor) GetCommands() map[string]string {
	return e.commands
}

// GetStatus returns a status string for debugging
func (e *Executor) GetStatus() string {
	if !e.enabled {
		return "Command mode: disabled"
	}

	if len(e.commands) == 0 {
		return "Command mode: enabled (no commands configured)"
	}

	status := fmt.Sprintf("Command mode: enabled (%d commands)\n", len(e.commands))
	for cmd, script := range e.commands {
		status += fmt.Sprintf("  '%s' -> %s\n", cmd, script)
	}

	return status
}
