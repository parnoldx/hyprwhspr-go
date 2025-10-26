package inject

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"
)

// Injector handles text injection into focused applications
type Injector struct {
	wlClipboardAvailable bool // wl-copy/wl-paste availability
}

// New creates a new text injector
func New() *Injector {
	return &Injector{
		wlClipboardAvailable: checkCommand("wl-copy") && checkCommand("wl-paste") && checkCommand("wtype"),
	}
}

// checkCommand checks if a command is available
func checkCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// Inject injects text into the focused application
func (inj *Injector) Inject(text string) error {
	// Smart clipboard with wtype (reliable with all layouts, keeps clipboard clean)
	if inj.wlClipboardAvailable {
		return inj.injectViaSmartClipboardWtype(text)
	}

	// Fallback: clipboard only (manual paste needed)
	return inj.copyToClipboard(text)
}

// injectViaSmartClipboardWtype injects text using smart clipboard with wtype for paste
func (inj *Injector) injectViaSmartClipboardWtype(text string) error {
	fmt.Printf("📋 Injecting text via smart clipboard (wtype): %d chars\n", len(text))

	// Save current clipboard content
	oldClipboard, err := inj.getCurrentClipboard()
	if err != nil {
		fmt.Printf("[WARN] Failed to save current clipboard: %v\n", err)
		oldClipboard = ""
	}

	// Copy new text to clipboard
	if err := inj.copyToClipboard(text); err != nil {
		return fmt.Errorf("failed to copy text to clipboard: %w", err)
	}

	// Wait for clipboard to settle
	time.Sleep(120 * time.Millisecond)

	// Paste with wtype using Ctrl+Shift+V (safer, doesn't conflict with system bindings)
	pasteCmd := exec.Command("wtype", "-M", "ctrl", "-M", "shift", "v", "-m", "ctrl", "-m", "shift")
	if err := pasteCmd.Run(); err != nil {
		return fmt.Errorf("wtype paste failed: %w", err)
	}

	// Schedule clipboard restoration in background
	go func() {
		time.Sleep(500 * time.Millisecond) // Wait 0.5 seconds for paste to complete

		if oldClipboard != "" {
			if err := inj.copyToClipboard(oldClipboard); err != nil {
				fmt.Printf("[WARN] Failed to restore clipboard: %v\n", err)
			} else {
				fmt.Println("📋 Clipboard restored")
			}
		} else {
			// Clear clipboard if it was empty before
			clearCmd := exec.Command("wl-copy", "")
			if err := clearCmd.Run(); err != nil {
				fmt.Printf("[WARN] Failed to clear clipboard: %v\n", err)
			} else {
				fmt.Println("📋 Clipboard cleared")
			}
		}
	}()

	fmt.Println("✅ Text injected successfully (smart clipboard)")
	return nil
}

// getCurrentClipboard retrieves current clipboard content
func (inj *Injector) getCurrentClipboard() (string, error) {
	cmd := exec.Command("wl-paste")
	output, err := cmd.Output()
	if err != nil {
		// wl-paste returns exit status 1 when clipboard is empty, which is normal
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return "", nil
		}
		return "", err
	}
	return string(output), nil
}

// copyToClipboard copies text to clipboard
func (inj *Injector) copyToClipboard(text string) error {
	cmd := exec.Command("wl-copy")
	cmd.Stdin = bytes.NewBufferString(text)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}

	fmt.Println("📋 Text copied to clipboard")
	return nil
}

// GetStatus returns the current injection method
func (inj *Injector) GetStatus() string {
	if inj.wlClipboardAvailable {
		return "✅ Text injection: Smart clipboard (wl-copy/wl-paste + wtype, keeps clipboard clean)"
	} else {
		return "⚠️  Text injection: clipboard only (manual paste needed)"
	}
}
