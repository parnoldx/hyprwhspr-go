#!/bin/bash
# Example command script for taking quick notes
# Usage: Speak "note this is my quick note" to append to notes file

NOTE_TEXT="$1"
NOTES_FILE="$HOME/.local/share/hyprwhspr/quick-notes.txt"

# Create directory if it doesn't exist
mkdir -p "$(dirname "$NOTES_FILE")"

# Append note with timestamp
if [ -n "$NOTE_TEXT" ]; then
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $NOTE_TEXT" >> "$NOTES_FILE"
    echo "Note saved: $NOTE_TEXT"

    # Optional: Show notification
    if command -v notify-send &> /dev/null; then
        notify-send "Note Saved" "$NOTE_TEXT"
    fi
else
    echo "No note text provided"
fi
