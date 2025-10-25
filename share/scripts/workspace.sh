#!/bin/bash
# Example command script for switching Hyprland workspaces
# Usage: Speak "workspace 3" to switch to workspace 3

WORKSPACE_NUM="$1"

# Check if workspace number is provided
if [ -z "$WORKSPACE_NUM" ]; then
    echo "No workspace number provided"
    exit 1
fi

# Use hyprctl to switch workspace
if command -v hyprctl &> /dev/null; then
    hyprctl dispatch workspace "$WORKSPACE_NUM"
    echo "Switched to workspace $WORKSPACE_NUM"
else
    echo "hyprctl not found - are you running Hyprland?"
    exit 1
fi
