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

The site uses Go's standard `html/template` package. Page structure, conditional introductions and article bylines live in `template.html`; Go supplies the data.

- `{{.Title}}` and `{{.Heading}}`: document title and visible page heading
- `{{.Kind}}`: page layout (`home`, `article` or `page`)
- `{{.Intro}}` and `{{.HideIntro}}`: masthead description and its visibility
- `{{.Author}}` and `{{.Date}}`: article author and publication date
- `{{.Content}}`: rendered Markdown and generated page content

Scalar values are escaped automatically. Only the rendered body is passed as trusted HTML. Template parsing and rendering errors are returned to the generator.

The homepage lists the five newest posts. Reading is a compact section at `/about.html#reading`. Writing, About, Sports and Reading are in the header; reference pages and tags are in the footer. Shared styles are in `static/theme.css`, with the design documented in `static/style-guide.html`.

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
