#!/bin/bash

set -e

SCRIPT_DIR="$(dirname "$0")"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

SKILLS_SOURCE="$PROJECT_ROOT/.agent/skills"
TRAE_DIR="$PROJECT_ROOT/.trae"
KIRO_DIR="$PROJECT_ROOT/.kiro"
CURSOR_DIR="$PROJECT_ROOT/.cursor"
CLAUDE_DIR="$PROJECT_ROOT/.claude"

init_trae() {
    echo "Initializing Trae..."
    if [ ! -d "$TRAE_DIR/skills" ]; then
        mkdir -p "$TRAE_DIR/skills"
    fi
    if [ -d "$SKILLS_SOURCE" ]; then
        cp -r "$SKILLS_SOURCE/"* "$TRAE_DIR/skills/"
        echo "✓ Trae skills initialized"
    else
        echo "✗ No skills found at $SKILLS_SOURCE"
    fi
}

init_kiro() {
    echo "Initializing Kiro..."
    if [ ! -d "$KIRO_DIR/skills" ]; then
        mkdir -p "$KIRO_DIR/skills"
    fi
    if [ -d "$SKILLS_SOURCE" ]; then
        cp -r "$SKILLS_SOURCE/"* "$KIRO_DIR/skills/"
        echo "✓ Kiro skills initialized"
    else
        echo "✗ No skills found at $SKILLS_SOURCE"
    fi
}

init_cursor() {
    echo "Initializing Cursor..."
    if [ ! -d "$CURSOR_DIR/skills" ]; then
        mkdir -p "$CURSOR_DIR/skills"
    fi
    if [ -d "$SKILLS_SOURCE" ]; then
        cp -r "$SKILLS_SOURCE/"* "$CURSOR_DIR/skills/"
        echo "✓ Cursor skills initialized"
    else
        echo "✗ No skills found at $SKILLS_SOURCE"
    fi
}

init_claude() {
    echo "Initializing Claude..."
    if [ ! -d "$CLAUDE_DIR/skills" ]; then
        mkdir -p "$CLAUDE_DIR/skills"
    fi
    if [ -d "$SKILLS_SOURCE" ]; then
        cp -r "$SKILLS_SOURCE/"* "$CLAUDE_DIR/skills/"
        echo "✓ Claude skills initialized"
    else
        echo "✗ No skills found at $SKILLS_SOURCE"
    fi
}

show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Initialize AI tools with skills from .agent/skills"
    echo ""
    echo "Options:"
    echo "  --all         Initialize all AI tools (default)"
    echo "  --trae        Initialize Trae only"
    echo "  --kiro        Initialize Kiro only"
    echo "  --cursor      Initialize Cursor only"
    echo "  --claude      Initialize Claude only"
    echo "  --help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0              # Initialize all tools"
    echo "  $0 --all        # Initialize all tools"
    echo "  $0 --trae       # Initialize Trae only"
}

initialize_all() {
    init_trae
    init_kiro
    init_cursor
    init_claude
    echo ""
    echo "All AI tools initialized successfully!"
}

if [ $# -eq 0 ]; then
    initialize_all
else
    while [ $# -gt 0 ]; do
        case "$1" in
            --all)
                initialize_all
                shift
                ;;
            --trae)
                init_trae
                shift
                ;;
            --kiro)
                init_kiro
                shift
                ;;
            --cursor)
                init_cursor
                shift
                ;;
            --claude)
                init_claude
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
fi
