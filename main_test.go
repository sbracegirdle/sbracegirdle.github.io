package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/gomarkdown/markdown"
)

// Test data constants
const (
	testTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{title}}</title>
</head>
<body>
    <h1>{{title}}</h1>
    <div>{{content}}</div>
    <footer>Created by: sbracegirdle on 2025-02-28 12:29:25</footer>
</body>
</html>`

	testMarkdownWithFrontmatter = `---
title: Test Title
---
# Heading
This is a test.`

	testMarkdownWithoutFrontmatter = `# No Frontmatter
This is a test without frontmatter.`

	testDateMarkdown1 = `---
title: First Post
---
# First Post
This is the first test post with a date.`

	testDateMarkdown2 = `---
title: Second Post
---
# Second Post
This is the second test post with a date.`

	testDateMarkdown3 = `# Third Post
This is the third test post with a date but no frontmatter.`

	testMarkdownWithFrontmatterAndDescription = `---
title: Test Title with Description
description: This is a custom description from frontmatter.
---
# Heading
This is a test with a custom description in frontmatter.`
)

// Setup function to create a test environment
func setupTestEnv(t *testing.T) (string, func()) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "ssg-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create content directory
	contentDir := filepath.Join(tempDir, "content")
	if err := os.Mkdir(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create content directory: %v", err)
	}

	// Create build directory
	buildDir := filepath.Join(tempDir, "build")
	if err := os.Mkdir(buildDir, 0755); err != nil {
		t.Fatalf("Failed to create build directory: %v", err)
	}

	// Create template file
	templatePath := filepath.Join(tempDir, "template.html")
	if err := os.WriteFile(templatePath, []byte(testTemplate), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	// Create test markdown files
	mdPath1 := filepath.Join(contentDir, "test-with-frontmatter.md")
	if err := os.WriteFile(mdPath1, []byte(testMarkdownWithFrontmatter), 0644); err != nil {
		t.Fatalf("Failed to write markdown file: %v", err)
	}

	mdPath2 := filepath.Join(contentDir, "test-without-frontmatter.md")
	if err := os.WriteFile(mdPath2, []byte(testMarkdownWithoutFrontmatter), 0644); err != nil {
		t.Fatalf("Failed to write markdown file: %v", err)
	}

	// Create dated markdown files for index testing
	mdPath3 := filepath.Join(contentDir, "2023-01-15-first-post.md")
	if err := os.WriteFile(mdPath3, []byte(testDateMarkdown1), 0644); err != nil {
		t.Fatalf("Failed to write dated markdown file: %v", err)
	}

	mdPath4 := filepath.Join(contentDir, "2023-03-20-second-post.md")
	if err := os.WriteFile(mdPath4, []byte(testDateMarkdown2), 0644); err != nil {
		t.Fatalf("Failed to write dated markdown file: %v", err)
	}

	mdPath5 := filepath.Join(contentDir, "2023-02-10-third-post.md")
	if err := os.WriteFile(mdPath5, []byte(testDateMarkdown3), 0644); err != nil {
		t.Fatalf("Failed to write dated markdown file: %v", err)
	}

	// Create test file with description in frontmatter
	mdPath6 := filepath.Join(contentDir, "test-with-description.md")
	if err := os.WriteFile(mdPath6, []byte(testMarkdownWithFrontmatterAndDescription), 0644); err != nil {
		t.Fatalf("Failed to write markdown file with description: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// Test helper to process a single file
func processFile(t *testing.T, filePath string, template string) (string, string) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Error reading file %s: %v", filePath, err)
	}

	var meta FrontMatter
	content, err := frontmatter.Parse(strings.NewReader(string(fileContent)), &meta)
	if err != nil {
		// If frontmatter parsing fails, use the whole content
		content = fileContent
	}

	title := meta.Title
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	}

	htmlContent := markdown.ToHTML(content, nil, nil)
	output := strings.Replace(template, "{{title}}", title, -1)
	output = strings.Replace(output, "{{content}}", string(htmlContent), -1)

	return title, output
}

// TestFrontmatterParsing tests the frontmatter parsing functionality
func TestFrontmatterParsing(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Test with frontmatter
	filePath := filepath.Join(testDir, "content", "test-with-frontmatter.md")
	title, _ := processFile(t, filePath, testTemplate)

	if title != "Test Title" {
		t.Errorf("Expected title 'Test Title', got '%s'", title)
	}

	// Test without frontmatter
	filePath = filepath.Join(testDir, "content", "test-without-frontmatter.md")
	title, _ = processFile(t, filePath, testTemplate)

	if title != "test-without-frontmatter" {
		t.Errorf("Expected title 'test-without-frontmatter', got '%s'", title)
	}
}

// TestHTMLGeneration tests the HTML generation process
func TestHTMLGeneration(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Test HTML generation for file with frontmatter
	filePath := filepath.Join(testDir, "content", "test-with-frontmatter.md")
	_, output := processFile(t, filePath, testTemplate)

	// Check if title is in the output
	if !strings.Contains(output, "<title>Test Title</title>") {
		t.Error("HTML output does not contain the expected title")
	}

	// Check if content is in the output
	if !strings.Contains(output, "<h1>Heading</h1>") {
		t.Error("HTML output does not contain the expected heading")
	}

	// Check if the footer with metadata is in the output
	if !strings.Contains(output, "Created by: sbracegirdle on 2025-02-28 12:29:25") {
		t.Error("HTML output does not contain the expected footer")
	}
}

// TestFileGeneration tests the file generation process
func TestFileGeneration(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Temporarily change working directory to test directory
	originalWd, _ := os.Getwd()
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Could not change to test directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Run the main function (which will use the test directory)
	main()

	// Check if the output files were created
	file1 := filepath.Join(testDir, "build", "test-with-frontmatter.html")
	if _, err := os.Stat(file1); os.IsNotExist(err) {
		t.Error("Expected output file was not created:", file1)
	}

	file2 := filepath.Join(testDir, "build", "test-without-frontmatter.html")
	if _, err := os.Stat(file2); os.IsNotExist(err) {
		t.Error("Expected output file was not created:", file2)
	}

	// Check content of generated files
	content1, err := os.ReadFile(file1)
	if err != nil {
		t.Fatalf("Could not read generated file: %v", err)
	}
	if !strings.Contains(string(content1), "<title>Test Title</title>") {
		t.Error("Generated HTML does not contain the expected title" + string(content1))
	}

	content2, err := os.ReadFile(file2)
	if err != nil {
		t.Fatalf("Could not read generated file: %v", err)
	}
	if !strings.Contains(string(content2), "<title>test without frontmatter</title>") {
		t.Error("Generated HTML does not contain the expected title" + string(content2))
	}
}

// TestExtractDescription tests the description extraction functionality
func TestExtractDescription(t *testing.T) {
	// Define the repeat string once to ensure consistency
	repeatStr := "Very long paragraph with repetitive content. "

	tests := []struct {
		name        string
		content     []byte
		expected    string
		description string
	}{
		{
			name:        "Short paragraph",
			content:     []byte("This is a short paragraph."),
			expected:    "This is a short paragraph.",
			description: "Should return the full paragraph when under 150 chars",
		},
		{
			name:    "Long paragraph",
			content: []byte(strings.Repeat(repeatStr, 10)),
			expected: func() string {
				// Calculate how many repetitions fit in 147 chars
				fullText := strings.Repeat(repeatStr, 10)
				if len(fullText) > 147 {
					return fullText[:147] + "..."
				}
				return fullText // In case the string is shorter than expected
			}(),
			description: "Should truncate long paragraphs to 150 chars",
		},
		{
			name:        "Multiple paragraphs",
			content:     []byte("# Heading\n\nFirst paragraph.\n\nSecond paragraph."),
			expected:    "Heading",
			description: "Should extract the first non-empty paragraph after headers",
		},
		{
			name:        "With markdown formatting",
			content:     []byte("# Heading\n\n**Bold text** and *italic* formatting."),
			expected:    "Heading",
			description: "Should remove markdown formatting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDescription(tt.content)
			if result != tt.expected {
				t.Errorf("%s: expected '%s', got '%s'", tt.description, tt.expected, result)
			}
		})
	}
}

// TestDateExtraction tests the date extraction from filenames
func TestDateExtraction(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	tests := []struct {
		filename      string
		expectedDate  string
		expectedTitle string
	}{
		{
			filename:      "2023-01-15-first-post.md",
			expectedDate:  "2023-01-15",
			expectedTitle: "First Post", // From frontmatter
		},
		{
			filename:      "2023-02-10-third-post.md",
			expectedDate:  "2023-02-10",
			expectedTitle: "third post", // From filename (no frontmatter)
		},
		{
			filename:      "regular-post-without-date.md",
			expectedDate:  "", // No date expected
			expectedTitle: "regular post without date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			filePath := filepath.Join(testDir, "content", tt.filename)

			// For the regular post that doesn't exist yet in our setup
			if tt.filename == "regular-post-without-date.md" {
				err := os.WriteFile(filePath, []byte("# Regular Post\nThis is a post without a date in filename."), 0644)
				if err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
			}

			_, _, post, err := processMarkdownFile(filePath, testTemplate)
			if err != nil {
				t.Fatalf("Error processing file: %v", err)
			}

			// Check date extraction
			var expectedDate time.Time
			if tt.expectedDate != "" {
				expectedDate, _ = time.Parse("2006-01-02", tt.expectedDate)
				if !post.Date.Equal(expectedDate) {
					t.Errorf("Expected date %v, got %v", expectedDate, post.Date)
				}
			} else {
				if !post.Date.IsZero() {
					t.Errorf("Expected zero date, got %v", post.Date)
				}
			}

			// Check title extraction
			if post.Title != tt.expectedTitle {
				t.Errorf("Expected title '%s', got '%s'", tt.expectedTitle, post.Title)
			}
		})
	}
}

// TestFrontmatterDescription tests the extraction of description from frontmatter
func TestFrontmatterDescription(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Test with description in frontmatter
	filePath := filepath.Join(testDir, "content", "test-with-description.md")
	_, _, post, err := processMarkdownFile(filePath, testTemplate)
	if err != nil {
		t.Fatalf("Error processing file: %v", err)
	}

	expectedDescription := "This is a custom description from frontmatter."
	if post.Description != expectedDescription {
		t.Errorf("Expected description '%s', got '%s'", expectedDescription, post.Description)
	}

	// Test with no description in frontmatter (should extract from content)
	filePath = filepath.Join(testDir, "content", "test-with-frontmatter.md")
	_, _, post, err = processMarkdownFile(filePath, testTemplate)
	if err != nil {
		t.Fatalf("Error processing file: %v", err)
	}

	// The extracted description should contain "This is a test"
	if !strings.Contains(post.Description, "This is a test") {
		t.Errorf("Expected description to contain 'This is a test', got '%s'", post.Description)
	}
}

// TestGenerateIndex tests the index generation functionality
func TestGenerateIndex(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()
	buildDir := filepath.Join(testDir, "build")

	// Create test blog posts with different dates
	date1, _ := time.Parse("2006-01-02", "2023-01-15")
	date2, _ := time.Parse("2006-01-02", "2023-03-20")
	date3, _ := time.Parse("2006-01-02", "2023-02-10")

	blogPosts := []*BlogPost{
		{
			Title:       "First Post",
			Date:        date1,
			Filename:    "2023-01-15-first-post.md",
			OutputFile:  "2023-01-15-first-post.html",
			Description: "This is the first test post with a date.",
		},
		{
			Title:       "Second Post",
			Date:        date2,
			Filename:    "2023-03-20-second-post.md",
			OutputFile:  "2023-03-20-second-post.html",
			Description: "This is the second test post with a date.",
		},
		{
			Title:       "Third Post",
			Date:        date3,
			Filename:    "2023-02-10-third-post.md",
			OutputFile:  "2023-02-10-third-post.html",
			Description: "This is the third test post with a date but no frontmatter.",
		},
	}

	// Generate index
	err := generateIndex(blogPosts, testTemplate, buildDir)
	if err != nil {
		t.Fatalf("Error generating index: %v", err)
	}

	// Verify index.html exists
	indexPath := filepath.Join(buildDir, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Fatal("Index file was not created")
	}

	// Verify index content
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Error reading index file: %v", err)
	}

	// Check title
	if !strings.Contains(string(content), "<title>Let's Build</title>") {
		t.Error("Index page does not have the correct title")
	}

	// Check posts are sorted by date (newest first)
	// The second post should appear before the third post, which should appear before the first post
	secondIndex := strings.Index(string(content), "Second Post")
	thirdIndex := strings.Index(string(content), "Third Post")
	firstIndex := strings.Index(string(content), "First Post")

	if secondIndex == -1 || thirdIndex == -1 || firstIndex == -1 {
		t.Error("One or more posts not found in index page")
	} else if !(secondIndex < thirdIndex && thirdIndex < firstIndex) {
		t.Error("Posts not sorted correctly by date in descending order")
	}

	// Check formatted dates appear in the content
	if !strings.Contains(string(content), "January 15, 2023") {
		t.Error("First post date not found in index")
	}
	if !strings.Contains(string(content), "March 20, 2023") {
		t.Error("Second post date not found in index")
	}
	if !strings.Contains(string(content), "February 10, 2023") {
		t.Error("Third post date not found in index")
	}
}

// TestFullSiteGeneration tests the complete site generation process including index
func TestFullSiteGeneration(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Temporarily change working directory to test directory
	originalWd, _ := os.Getwd()
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Could not change to test directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Run the main function (which will use the test directory)
	err := generateSite(
		filepath.Join(testDir, "content"),
		filepath.Join(testDir, "build"),
		filepath.Join(testDir, "template.html"),
	)
	if err != nil {
		t.Fatalf("Error generating site: %v", err)
	}

	// Check if all expected output files were created
	expectedFiles := []string{
		"test-with-frontmatter.html",
		"test-without-frontmatter.html",
		"2023-01-15-first-post.html",
		"2023-03-20-second-post.html",
		"2023-02-10-third-post.html",
		"index.html",
	}

	for _, filename := range expectedFiles {
		filePath := filepath.Join(testDir, "build", filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected output file was not created: %s", filename)
		}
	}

	// Verify index.html content
	indexPath := filepath.Join(testDir, "build", "index.html")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Error reading index file: %v", err)
	}

	// Check post order by date (newest first)
	secondIndex := strings.Index(string(content), "Second Post")
	thirdIndex := strings.Index(string(content), "Third Post")
	firstIndex := strings.Index(string(content), "First Post")

	if secondIndex == -1 || thirdIndex == -1 || firstIndex == -1 {
		t.Error("One or more posts not found in index page")
	} else if !(secondIndex < thirdIndex && thirdIndex < firstIndex) {
		t.Error("Posts not sorted correctly by date in descending order in the index")
	}
}
