package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/gomarkdown/markdown"
)

// FrontMatter represents the metadata at the top of markdown files
type FrontMatter struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

// BlogPost represents metadata about a blog post
type BlogPost struct {
	Title       string
	Date        time.Time
	Filename    string
	OutputFile  string
	Description string
}

// processMarkdownFile processes a single markdown file and returns the generated HTML
func processMarkdownFile(filePath, template string) (string, string, *BlogPost, error) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", nil, fmt.Errorf("error reading file %s: %v", filePath, err)
	}

	// Parse frontmatter
	var meta FrontMatter
	content, err := frontmatter.Parse(strings.NewReader(string(fileContent)), &meta)
	if err != nil {
		// If frontmatter parsing fails, use the whole content
		content = fileContent
	}

	// Get filename and extract date
	filename := filepath.Base(filePath)
	title := meta.Title

	// Extract date from filename (yyyy-mm-dd-title.md)
	dateRegex := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-(.+)$`)
	matches := dateRegex.FindStringSubmatch(strings.TrimSuffix(filename, filepath.Ext(filename)))

	var postDate time.Time
	var filenameTitle string

	if len(matches) == 3 {
		// Parse the date from the filename
		postDate, err = time.Parse("2006-01-02", matches[1])
		if err != nil {
			postDate = time.Time{} // Zero value if date parsing fails
		}

		// Convert hyphens to spaces in filename for title
		filenameTitle = strings.ReplaceAll(matches[2], "-", " ")
	} else {
		// No date in filename, just convert hyphens to spaces for the whole filename
		filenameTitle = strings.ReplaceAll(
			strings.TrimSuffix(filename, filepath.Ext(filename)),
			"-",
			" ",
		)
	}

	// Use frontmatter title if available, otherwise use filename-based title
	if title == "" {
		title = filenameTitle
	}

	// Get description from frontmatter or extract from content
	description := meta.Description
	if description == "" {
		description = extractDescription(content)
	}

	// Parse markdown to HTML
	htmlContent := markdown.ToHTML(content, nil, nil)

	// Replace template placeholders
	output := strings.Replace(template, "{{title}}", title, -1)
	output = strings.Replace(output, "{{content}}", string(htmlContent), -1)

	outputFilename := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".html"

	// Create blog post metadata
	blogPost := &BlogPost{
		Title:       title,
		Date:        postDate,
		Filename:    filename,
		OutputFile:  outputFilename,
		Description: description,
	}

	return outputFilename, output, blogPost, nil
}

// extractDescription extracts a brief description from the content
func extractDescription(content []byte) string {
	// Simple approach: get first paragraph or first 150 chars
	text := string(content)
	// Remove any markdown formatting
	text = strings.ReplaceAll(text, "#", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "_", "")

	// Find first paragraph
	paragraphs := strings.Split(text, "\n\n")
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p != "" {
			if len(p) > 150 {
				return p[:147] + "..."
			}
			return p
		}
	}

	// Fallback to first 150 chars if no paragraph found
	if len(text) > 150 {
		return strings.TrimSpace(text[:147]) + "..."
	}
	return strings.TrimSpace(text)
}

// generateIndex generates an index page with links to all blog posts
func generateIndex(posts []*BlogPost, template string, buildDir string) error {
	// Sort posts by date in descending order (newest first)
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})

	// Generate HTML for the list of posts
	var contentBuilder strings.Builder
	contentBuilder.WriteString("<p>Hi! I'm <a href=\"https://github.com/sbracegirdle\" rel=\"author\"><em>Simon</em></a>, a Consultant and Software Engineer from Perth, Western Australia. I work for <a href=\"https://govconnex.com/\">GovConnex</a>, and in the past worked for <a href=\"https://mechanicalrock.io\">Mechanical Rock</a> and <a href=\"https://seqta.com.au\">SEQTA Software</a>.</p>")
	contentBuilder.WriteString("<p>I've been developing software and helping teams with their practices and products for over 15 years. During that time I've built an interest in writing, learning, and solving challenging problems. This blog is a place for me to share my thoughts, experiences, and learnings.</p>")
	contentBuilder.WriteString("<h2>Latest posts</h2>")

	for _, post := range posts {
		if post.Date.IsZero() {
			continue // Skip posts without dates
		}

		formattedDate := post.Date.Format("January 2, 2006")
		contentBuilder.WriteString(fmt.Sprintf("<li><strong>%s</strong> - <a href=\"%s\">%s</a><p>%s</p></li>\n",
			formattedDate, post.OutputFile, post.Title, post.Description))
	}

	contentBuilder.WriteString("</ul>")

	// Replace template placeholders
	output := strings.Replace(template, "{{title}}", "Let's Build", -1)
	output = strings.Replace(output, "{{content}}", contentBuilder.String(), -1)

	// Write the index file
	outputPath := filepath.Join(buildDir, "index.html")
	err := os.WriteFile(outputPath, []byte(output), 0644)
	if err != nil {
		return fmt.Errorf("error writing index file: %v", err)
	}

	fmt.Printf("Generated index: %s\n", outputPath)
	return nil
}

// generateSite processes all markdown files in the content directory
func generateSite(contentDir, buildDir, templatePath string) error {
	// Check if build directory exists, create if not
	if _, err := os.Stat(buildDir); os.IsNotExist(err) {
		err = os.MkdirAll(buildDir, 0755)
		if err != nil {
			return fmt.Errorf("error creating build directory: %v", err)
		}
	}

	// Get template
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return fmt.Errorf("template file not found at %s", templatePath)
	}

	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("error reading template: %v", err)
	}
	template := string(templateBytes)

	// Check content directory
	if _, err := os.Stat(contentDir); os.IsNotExist(err) {
		return fmt.Errorf("content directory not found at %s", contentDir)
	}

	// Get markdown files
	files, err := os.ReadDir(contentDir)
	if err != nil {
		return fmt.Errorf("error reading content directory: %v", err)
	}

	// Collection of blog posts for the index
	var blogPosts []*BlogPost

	// Process each markdown file
	for _, file := range files {
		// Skip directories and non-markdown files
		if file.IsDir() ||
			(!strings.HasSuffix(file.Name(), ".md") && !strings.HasSuffix(file.Name(), ".markdown")) {
			continue
		}

		filePath := filepath.Join(contentDir, file.Name())
		outputFilename, outputContent, blogPost, err := processMarkdownFile(filePath, template)
		if err != nil {
			log.Printf("%v", err)
			continue
		}

		// Add to collection of blog posts
		if blogPost != nil {
			blogPosts = append(blogPosts, blogPost)
		}

		// Write output file
		outputPath := filepath.Join(buildDir, outputFilename)
		err = os.WriteFile(outputPath, []byte(outputContent), 0644)
		if err != nil {
			log.Printf("Error writing output file %s: %v", outputPath, err)
			continue
		}

		fmt.Printf("Generated: %s\n", outputPath)
	}

	// Generate index page
	if len(blogPosts) > 0 {
		err = generateIndex(blogPosts, template, buildDir)
		if err != nil {
			log.Printf("Error generating index: %v", err)
		}
	}

	return nil
}

func main() {
	contentDir := filepath.Join(".", "content")
	buildDir := filepath.Join(".", "build")
	templatePath := filepath.Join(".", "template.html")

	err := generateSite(contentDir, buildDir, templatePath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Site generation complete!")
}
