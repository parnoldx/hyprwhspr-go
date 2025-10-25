package inject

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"
)

// Injector handles text injection into focused applications
type Injector struct {
	wtypeAvailable   bool
	ydotoolAvailable bool
}

// New creates a new text injector
func New() *Injector {
	return &Injector{
		wtypeAvailable:   checkCommand("wtype"),
		ydotoolAvailable: checkCommand("ydotool"),
	}
}

// checkCommand checks if a command is available
func checkCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// Inject injects text into the focused application
func (inj *Injector) Inject(text string) error {
	// Priority 1: wtype (no clipboard pollution)
	if inj.wtypeAvailable {
		return inj.injectViaWtype(text)
	}

	// Priority 2: ydotool (with clipboard)
	if inj.ydotoolAvailable {
		if err := inj.injectViaYdotool(text); err != nil {
			return err
		}
		// Clear clipboard after 2 seconds
		go func() {
			time.Sleep(2 * time.Second)
			exec.Command("wl-copy", "").Run()
		}()
		return nil
	}

	// Priority 3: clipboard only (fallback)
	return inj.copyToClipboard(text)
}

// injectViaWtype injects text using wtype (Wayland-native, no clipboard)
func (inj *Injector) injectViaWtype(text string) error {
	fmt.Printf("💡 Injecting text with wtype (no clipboard): %d chars\n", len(text))

	// Small delay for window focus
	time.Sleep(50 * time.Millisecond)

	cmd := exec.Command("wtype", "-")
	cmd.Stdin = bytes.NewBufferString(text)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wtype failed: %w", err)
	}

	fmt.Println("✅ Text injected successfully (wtype)")
	return nil
}

// injectViaYdotool injects text using ydotool with clipboard
func (inj *Injector) injectViaYdotool(text string) error {
	fmt.Printf("📋 Injecting text via clipboard+ydotool: %d chars\n", len(text))

	// Copy to clipboard
	if err := inj.copyToClipboard(text); err != nil {
		return err
	}

	// Wait for clipboard to settle
	time.Sleep(120 * time.Millisecond)

	// Simulate Ctrl+Shift+V
	cmd := exec.Command("ydotool", "key", "29:1", "42:1", "47:1", "47:0", "42:0", "29:0")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ydotool failed: %w", err)
	}

	fmt.Println("✅ Text injected successfully (ydotool+clipboard)")
	return nil
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
	if inj.wtypeAvailable {
		return "✅ Text injection: wtype (Wayland-native, no clipboard)"
	} else if inj.ydotoolAvailable {
		return "⚠️  Text injection: ydotool (uses clipboard)"
	} else {
		return "⚠️  Text injection: clipboard only (manual paste needed)"
	}
}
