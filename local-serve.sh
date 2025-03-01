#!/bin/bash

# Exit on error
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Default port
PORT=8080
WATCH=false

# Parse command line arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        -p|--port) PORT="$2"; shift ;;
        -w|--watch) WATCH=true ;;
        *) echo "Unknown parameter: $1"; exit 1 ;;
    esac
    shift
done

# Function to build and generate
build_and_generate() {
    echo -e "${YELLOW}Building the static site generator...${NC}"
    go build -o ssg ./main.go
    
    echo -e "${YELLOW}Generating site...${NC}"
    mkdir -p build
    ./ssg
    
    echo -e "${GREEN}Build complete!${NC}"
}

# Function to serve the site
serve_site() {
    echo -e "${GREEN}Starting local server at http://localhost:${PORT}${NC}"
    echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
    
    # Try using Python's http.server first (Python 3)
    if command -v python3 &>/dev/null; then
        python3 -m http.server ${PORT} --directory build
    # Fall back to Python 2's SimpleHTTPServer
    elif command -v python &>/dev/null; then
        (cd build && python -m SimpleHTTPServer ${PORT})
    # If no Python, try using Go's built-in http.FileServer
    else
        echo -e "${RED}Python not found. Using Go to serve files.${NC}"
        go run -mod=mod $(go env GOROOT)/src/net/http/httpd.go -addr :${PORT} -directory ./build
    fi
}

# Build and generate initially
build_and_generate

# If watch mode is enabled and fswatch is available
if [ "$WATCH" = true ]; then
    if command -v fswatch &>/dev/null; then
        echo -e "${GREEN}Watch mode enabled. Will rebuild on file changes.${NC}"
        
        # Start server in background
        serve_site &
        SERVER_PID=$!
        
        # Trap Ctrl+C to kill the server process
        trap "kill $SERVER_PID; exit 0" INT
        
        # Watch for file changes and rebuild
        fswatch -o ./content ./main.go ./template.html | while read; do
            echo -e "${YELLOW}Changes detected, rebuilding...${NC}"
            build_and_generate
        done
    else
        echo -e "${RED}Watch mode requires fswatch but it's not installed.${NC}"
        echo "Install it with: brew install fswatch (macOS) or apt-get install fswatch (Linux)"
        # Fall back to serving without watch
        serve_site
    fi
else
    # Just serve without watching
    serve_site
fi
