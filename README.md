# Simple Static Site Generator

A Go-based static site generator that converts Markdown files to HTML.

## Features

- Processes markdown files to HTML
- Uses a customizable HTML template
- Supports frontmatter for metadata
- Generates HTML files in a build directory

## Setup

1. Install dependencies:
   ```
   go mod download
   ```

2. Build the project:
   ```
   go build -o ssg
   ```

3. Run the generator:
   ```
   ./ssg
   ```

## Local Development

To test the site locally before deploying, use the included scripts:

### On macOS/Linux:

```bash
# Make the script executable
chmod +x ./local-serve.sh

# Basic usage (serves on port 8080)
./local-serve.sh

# Custom port
./local-serve.sh --port 3000

# Watch mode (requires fswatch)
./local-serve.sh --watch
```

## Project Structure

- `content/` - Markdown files for your site
- `build/` - Generated HTML output
- `template.html` - HTML template for the site

## Template Syntax

The template uses simple placeholders:
- `{{title}}`: Will be replaced with the title from frontmatter (or filename if not specified)
- `{{content}}`: Will be replaced with the HTML converted from markdown

## Markdown Frontmatter

You can add metadata to your markdown files using YAML frontmatter:

```markdown
---
title: My Page Title
---

Content goes here...
```

## Deployment

This site is automatically deployed to GitHub Pages when changes are pushed to the main branch.