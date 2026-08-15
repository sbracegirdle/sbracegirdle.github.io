package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

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
	err := generateIndex(blogPosts, testTemplate, buildDir, nil)
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

	// Check title. Titles are HTML-escaped on the way into the template — the
	// apostrophe becomes an entity, which browsers render back as "Let's Build".
	if !strings.Contains(string(content), "<title>Let&#39;s Build</title>") {
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

	// Check formatted dates appear in the content (ISO, per the post tree)
	if !strings.Contains(string(content), "2023-01-15") {
		t.Error("First post date not found in index")
	}
	if !strings.Contains(string(content), "2023-03-20") {
		t.Error("Second post date not found in index")
	}
	if !strings.Contains(string(content), "2023-02-10") {
		t.Error("Third post date not found in index")
	}
}

// sampleGoodreadsRSS mirrors the structure of a real Goodreads shelf feed,
// trimmed to the fields we parse. Kept inline so the parser test never touches
// the network.
const sampleGoodreadsRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Simon's bookshelf: currently-reading</title>
    <item>
      <title><![CDATA[Oathbringer (The Stormlight Archive, #3)]]></title>
      <book_id>34002132</book_id>
      <book_image_url><![CDATA[https://example.com/small.jpg]]></book_image_url>
      <book_large_image_url><![CDATA[https://example.com/large.jpg]]></book_large_image_url>
      <author_name>Brandon Sanderson</author_name>
    </item>
    <item>
      <title><![CDATA[A Book Without A Link]]></title>
      <author_name>Some Author</author_name>
    </item>
  </channel>
</rss>`

// TestParseShelf verifies the Goodreads RSS parsing without any network access.
func TestParseShelf(t *testing.T) {
	books, err := parseShelf([]byte(sampleGoodreadsRSS))
	if err != nil {
		t.Fatalf("parseShelf returned error: %v", err)
	}

	if len(books) != 2 {
		t.Fatalf("expected 2 books, got %d", len(books))
	}

	first := books[0]
	if first.Title != "Oathbringer (The Stormlight Archive, #3)" {
		t.Errorf("unexpected title: %q", first.Title)
	}
	if first.Author != "Brandon Sanderson" {
		t.Errorf("unexpected author: %q", first.Author)
	}
	if first.Link != "https://www.goodreads.com/book/show/34002132" {
		t.Errorf("expected canonical book link, got %q", first.Link)
	}

	// Second item carries no book_id, so it should parse with an empty link
	// rather than a broken one.
	if books[1].Link != "" {
		t.Errorf("expected empty link without a book_id, got %q", books[1].Link)
	}
}

// TestParseShelfIgnoresCovers pins the decision to stop hot-linking Goodreads
// artwork: the feed still carries cover URLs and the parser must drop them, so
// no page can pick one up again by accident.
func TestParseShelfIgnoresCovers(t *testing.T) {
	books, err := parseShelf([]byte(sampleGoodreadsRSS))
	if err != nil {
		t.Fatalf("parseShelf returned error: %v", err)
	}
	section := renderReadingSection([]ShelfBooks{{Label: "Currently reading", Books: books}})
	for _, unwanted := range []string{"example.com", "<img", "book-cover"} {
		if strings.Contains(section, unwanted) {
			t.Errorf("reading section should carry no cover images, found %q\ngot: %s", unwanted, section)
		}
	}
}

// TestParseShelfEmpty ensures an empty (but valid) feed yields no books and no error.
func TestParseShelfEmpty(t *testing.T) {
	books, err := parseShelf([]byte(`<rss><channel></channel></rss>`))
	if err != nil {
		t.Fatalf("parseShelf returned error: %v", err)
	}
	if len(books) != 0 {
		t.Errorf("expected no books, got %d", len(books))
	}
}

// TestRenderReadingSection checks that labelled shelves render into the index
// and that no books at all produces no section.
func TestRenderReadingSection(t *testing.T) {
	if got := renderReadingSection(nil); got != "" {
		t.Errorf("expected empty string for no shelves, got %q", got)
	}
	if got := renderReadingSection([]ShelfBooks{{Label: "Want to read"}}); got != "" {
		t.Errorf("expected empty string when every shelf is empty, got %q", got)
	}

	shelves := []ShelfBooks{
		{
			Label: "Currently reading",
			Hue:   "gold",
			Books: []Book{{
				Title:  "Oathbringer",
				Author: "Brandon Sanderson",
				Link:   "https://www.goodreads.com/book/show/34002132",
			}},
		},
		{
			Label: "Want to read",
			Hue:   "foam",
			Books: []Book{{Title: "Wind and Truth", Author: "Brandon Sanderson"}},
		},
	}
	html := renderReadingSection(shelves)

	for _, want := range []string{
		"What I'm reading",
		"Currently reading",
		"Want to read",
		"Oathbringer",
		"Wind and Truth",
		"https://www.goodreads.com/book/show/34002132",
		// The CSS book glyph replaces the cover, and is decoration only.
		`<span class="book-glyph" aria-hidden="true"></span>`,
		// Each shelf card carries its hue modifier.
		`class="card shelf-gold"`,
		`class="card shelf-foam"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered section missing %q\ngot: %s", want, html)
		}
	}

	// Group order should be preserved: "Currently reading" before "Want to read".
	if strings.Index(html, "Currently reading") > strings.Index(html, "Want to read") {
		t.Error("shelf groups rendered out of order")
	}
}

// TestGenerateIndexWithBooks confirms the reading section lands in the index
// when books are supplied, ahead of the "Latest posts" heading.
func TestGenerateIndexWithBooks(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()
	buildDir := filepath.Join(testDir, "build")

	date, _ := time.Parse("2006-01-02", "2023-01-15")
	posts := []*BlogPost{{
		Title:      "First Post",
		Date:       date,
		OutputFile: "2023-01-15-first-post.html",
	}}
	shelves := []ShelfBooks{{
		Label: "Currently reading",
		Books: []Book{{Title: "Oathbringer", Author: "Brandon Sanderson"}},
	}}

	if err := generateIndex(posts, testTemplate, buildDir, shelves); err != nil {
		t.Fatalf("Error generating index: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(buildDir, "index.html"))
	if err != nil {
		t.Fatalf("Error reading index file: %v", err)
	}
	body := string(content)

	readingIdx := strings.Index(body, "What I'm reading")
	postsIdx := strings.Index(body, "Latest posts")
	if readingIdx == -1 {
		t.Fatal("reading section not found in index")
	}
	if postsIdx == -1 || readingIdx > postsIdx {
		t.Error("reading section should appear before the Latest posts heading")
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
		nil,
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
		"posts.html",
		"feed.xml",
		"sitemap.xml",
		"robots.txt",
		"404.html",
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

// testMetaTemplate mirrors the head of the real template closely enough to
// exercise every placeholder renderPage fills.
const testMetaTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<title>{{title}}</title>
<meta name="description" content="{{description}}" />
<link rel="canonical" href="{{canonical}}" />
<meta property="og:type" content="{{ogtype}}" />
<meta property="og:title" content="{{title}}" />
<meta property="og:description" content="{{description}}" />
<meta property="og:url" content="{{canonical}}" />
{{head_extra}}
</head>
<body>
<span class="seg seg-c">{{file}}</span>
<h1>{{heading}}</h1>
<main>{{content}}</main>
</body>
</html>`

// TestCanonicalURL checks that the home page collapses to the bare root while
// every other page keeps its filename.
func TestCanonicalURL(t *testing.T) {
	tests := []struct{ in, want string }{
		{"index.html", "https://letsbuild.cloud/"},
		{"posts.html", "https://letsbuild.cloud/posts.html"},
		{"tags/aws.html", "https://letsbuild.cloud/tags/aws.html"},
		{"/already-rooted.html", "https://letsbuild.cloud/already-rooted.html"},
	}
	for _, tt := range tests {
		if got := canonicalURL(tt.in); got != tt.want {
			t.Errorf("canonicalURL(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestRenderPageEscaping is the regression guard for metadata: a description
// containing a quote used to be impossible, but now that descriptions land in
// attribute values an unescaped one would close the attribute early and break
// the whole head.
func TestRenderPageEscaping(t *testing.T) {
	out := renderPage(testMetaTemplate, pageMeta{
		Title:       `Ampersands & "quotes"`,
		File:        "post.html",
		Description: `Do you slap the trusty "LGTM!" on pull requests? Tom & Jerry <script>`,
		Canonical:   "https://letsbuild.cloud/post.html",
		OGType:      "article",
		Content:     "<p>Body &amp; content</p>",
	})

	if strings.Contains(out, `content="Do you slap the trusty "LGTM!"`) {
		t.Error("description was inserted unescaped and broke out of the attribute")
	}
	if !strings.Contains(out, `Do you slap the trusty &#34;LGTM!&#34; on pull requests?`) {
		t.Errorf("expected an escaped description, got:\n%s", out)
	}
	if strings.Contains(out, "<script>") {
		t.Error("markup from a description reached the page unescaped")
	}
	// Content is trusted, pre-rendered HTML and must survive verbatim.
	if !strings.Contains(out, "<p>Body &amp; content</p>") {
		t.Error("page content should be inserted raw")
	}
	// Heading defaults to the title.
	if !strings.Contains(out, `<h1>Ampersands &amp; &#34;quotes&#34;</h1>`) {
		t.Errorf("heading should default to the title, got:\n%s", out)
	}
	if !strings.Contains(out, `content="article"`) {
		t.Error("og:type was not filled")
	}
}

// TestRenderPageDefaults checks the two fields that fall back.
func TestRenderPageDefaults(t *testing.T) {
	out := renderPage(testMetaTemplate, pageMeta{Title: "Plain", Content: "x"})

	if !strings.Contains(out, "<h1>Plain</h1>") {
		t.Error("heading should default to the title")
	}
	if !strings.Contains(out, `content="website"`) {
		t.Error("og:type should default to website")
	}
	if strings.Contains(out, "{{") {
		t.Errorf("template placeholders were left unfilled:\n%s", out)
	}
}

// TestPostMetadata checks the head a real post ends up with: its own
// description, canonical URL, article type and published time.
func TestPostMetadata(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	post := "---\ntitle: Tagged Post\ndescription: A short summary.\ntags: devops AWS code_review\n---\n\nBody text."
	postPath := filepath.Join(testDir, "content", "2024-05-01-tagged.md")
	if err := os.WriteFile(postPath, []byte(post), 0644); err != nil {
		t.Fatalf("writing post: %v", err)
	}

	_, output, meta, err := processMarkdownFile(postPath, testMetaTemplate)
	if err != nil {
		t.Fatalf("processMarkdownFile: %v", err)
	}

	for _, want := range []string{
		`content="A short summary."`,
		`href="https://letsbuild.cloud/2024-05-01-tagged.html"`,
		`content="article"`,
		`content="2024-05-01T00:00:00Z"`,
		`href="/tags/devops.html"`,
		`href="/tags/code-review.html"`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("post head missing %q", want)
		}
	}

	wantTags := []string{"aws", "code-review", "devops"}
	if len(meta.Tags) != len(wantTags) {
		t.Fatalf("tags = %v, want %v", meta.Tags, wantTags)
	}
	for i, tag := range wantTags {
		if meta.Tags[i] != tag {
			t.Errorf("tags = %v, want %v", meta.Tags, wantTags)
			break
		}
	}
}

// TestUndatedPageMetadata checks that a page without a date is not advertised
// as an article with a publication time.
func TestUndatedPageMetadata(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	page := "---\ntitle: About me\ndescription: A mini CV.\n---\n\nBody text."
	pagePath := filepath.Join(testDir, "content", "about.md")
	if err := os.WriteFile(pagePath, []byte(page), 0644); err != nil {
		t.Fatalf("writing page: %v", err)
	}

	_, output, _, err := processMarkdownFile(pagePath, testMetaTemplate)
	if err != nil {
		t.Fatalf("processMarkdownFile: %v", err)
	}

	if !strings.Contains(output, `content="website"`) {
		t.Error("an undated page should be og:type website")
	}
	if strings.Contains(output, "article:published_time") {
		t.Error("an undated page should not claim a publication time")
	}
}

// TestGenerateNotFound checks the 404 page is written and kept out of the index.
func TestGenerateNotFound(t *testing.T) {
	buildDir := t.TempDir()

	if err := generateNotFound(testMetaTemplate, buildDir); err != nil {
		t.Fatalf("generateNotFound: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(buildDir, "404.html"))
	if err != nil {
		t.Fatalf("reading 404 page: %v", err)
	}
	body := string(raw)
	if !strings.Contains(body, `name="robots" content="noindex"`) {
		t.Error("404 page should be noindex")
	}
	if !strings.Contains(body, "<h1>404</h1>") {
		t.Error("404 page should head with 404")
	}
	if !strings.Contains(body, `href="/posts.html"`) {
		t.Error("404 page should offer a way back")
	}
}

// TestCopyStaticDirReportsPages checks the HTML pages reported for the sitemap.
func TestCopyStaticDirReportsPages(t *testing.T) {
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")
	buildDir := filepath.Join(tempDir, "build")
	if err := os.MkdirAll(filepath.Join(staticDir, "nested"), 0755); err != nil {
		t.Fatalf("creating static dir: %v", err)
	}
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatalf("creating build dir: %v", err)
	}

	files := map[string]string{
		"guide.html":        "<p>guide</p>",
		"theme.css":         "body{}",
		"nested/inner.html": "<p>inner</p>",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(staticDir, filepath.FromSlash(name)), []byte(body), 0644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	pages, err := copyStaticDir(staticDir, buildDir)
	if err != nil {
		t.Fatalf("copyStaticDir: %v", err)
	}

	want := []string{"guide.html", "nested/inner.html"}
	if len(pages) != len(want) {
		t.Fatalf("pages = %v, want %v", pages, want)
	}
	for i := range want {
		if pages[i] != want[i] {
			t.Fatalf("pages = %v, want %v (HTML only, forward slashes, sorted)", pages, want)
		}
	}

	// Non-HTML files still copy, they just aren't reported as pages.
	if _, err := os.Stat(filepath.Join(buildDir, "theme.css")); err != nil {
		t.Errorf("static asset was not copied: %v", err)
	}
}

// TestExtractDescriptionTruncatesOnRuneBoundary guards the meta description
// against invalid UTF-8. The description reaches <meta name="description"> and
// og:description as an attribute value; truncating it mid-rune emits a partial
// byte sequence that html.EscapeString has no way to repair, and the site's
// prose is full of em dashes and curly quotes.
func TestExtractDescriptionTruncatesOnRuneBoundary(t *testing.T) {
	// Position an em dash (3 bytes) so it straddles the old 147-byte cut.
	para := strings.Repeat("a", 146) + "—" + strings.Repeat("b", 60)

	got := extractDescription([]byte(para))

	if !utf8.ValidString(got) {
		t.Errorf("description is not valid UTF-8: %q", got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("expected a truncated description ending in an ellipsis, got %q", got)
	}
	if n := utf8.RuneCountInString(got); n > 150 {
		t.Errorf("description is %d runes, want at most 150", n)
	}
}

// TestExtractDescriptionFallbackIsRuneSafe covers the no-paragraph branch,
// which does its own truncation.
func TestExtractDescriptionFallbackIsRuneSafe(t *testing.T) {
	// One long line with no blank line, so the paragraph split yields a single
	// oversized chunk of multi-byte runes.
	got := extractDescription([]byte(strings.Repeat("é", 400)))

	if !utf8.ValidString(got) {
		t.Errorf("description is not valid UTF-8: %q", got)
	}
	if n := utf8.RuneCountInString(got); n > 150 {
		t.Errorf("description is %d runes, want at most 150", n)
	}
}

// TestDatedPostsSameDateOrderIsDeterministic pins the ordering of posts that
// share a date. content/ already has a pair on 2021-12-20, and sort.Slice is
// explicitly not stable, so without a total order the index, the archive and
// the feed could reshuffle on a Go upgrade with no source change.
func TestDatedPostsSameDateOrderIsDeterministic(t *testing.T) {
	day := time.Date(2021, 12, 20, 0, 0, 0, 0, time.UTC)
	older := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	mk := func(file string, d time.Time) *BlogPost {
		return &BlogPost{Title: file, OutputFile: file, Date: d}
	}

	forward := []*BlogPost{
		mk("2021-12-20-cdk-cr.html", day),
		mk("2021-12-20-robo-devops.html", day),
		mk("2020-01-01-old.html", older),
	}
	reversed := []*BlogPost{
		mk("2020-01-01-old.html", older),
		mk("2021-12-20-robo-devops.html", day),
		mk("2021-12-20-cdk-cr.html", day),
	}

	order := func(posts []*BlogPost) []string {
		got := datedPostsNewestFirst(posts)
		names := make([]string, len(got))
		for i, p := range got {
			names[i] = p.OutputFile
		}
		return names
	}

	want := []string{
		"2021-12-20-cdk-cr.html",
		"2021-12-20-robo-devops.html",
		"2020-01-01-old.html",
	}

	for _, tc := range []struct {
		name  string
		posts []*BlogPost
	}{
		{"forward", forward},
		{"reversed", reversed},
	} {
		got := order(tc.posts)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s input: got %v, want %v", tc.name, got, want)
		}
	}
}

// TestGenerateIndexLimitsToLatest covers the homepage's "latest 5, then link to
// the archive" rule. Every other index test uses one to three posts, so the
// truncation and the archive link — the production path, with 15 posts live —
// never ran.
func TestGenerateIndexLimitsToLatest(t *testing.T) {
	mk := func(n int, day int) *BlogPost {
		return &BlogPost{
			Title:       fmt.Sprintf("Post %d", n),
			OutputFile:  fmt.Sprintf("2023-01-%02d-post-%d.html", day, n),
			Date:        time.Date(2023, 1, day, 0, 0, 0, 0, time.UTC),
			Description: "A description.",
		}
	}

	t.Run("more posts than the limit", func(t *testing.T) {
		buildDir := t.TempDir()
		// Seven posts, newest is Post 7.
		posts := make([]*BlogPost, 0, 7)
		for i := 1; i <= 7; i++ {
			posts = append(posts, mk(i, i))
		}

		if err := generateIndex(posts, testMetaTemplate, buildDir, nil); err != nil {
			t.Fatalf("generateIndex: %v", err)
		}
		body := readFile(t, filepath.Join(buildDir, "index.html"))

		for _, want := range []string{"Post 7", "Post 6", "Post 5", "Post 4", "Post 3"} {
			if !strings.Contains(body, want) {
				t.Errorf("index is missing %q", want)
			}
		}
		for _, notWant := range []string{"Post 2", "Post 1"} {
			if strings.Contains(body, notWant) {
				t.Errorf("index lists %q, but should stop at the latest %d", notWant, latestPostCount)
			}
		}
		if !strings.Contains(body, `href="/posts.html"`) {
			t.Error("index does not link to the archive at /posts.html")
		}
		// Newest first.
		if strings.Index(body, "Post 7") > strings.Index(body, "Post 3") {
			t.Error("index is not ordered newest first")
		}
	})

	t.Run("fewer posts than the limit", func(t *testing.T) {
		buildDir := t.TempDir()
		posts := []*BlogPost{mk(1, 1), mk(2, 2)}

		if err := generateIndex(posts, testMetaTemplate, buildDir, nil); err != nil {
			t.Fatalf("generateIndex: %v", err)
		}
		body := readFile(t, filepath.Join(buildDir, "index.html"))

		if strings.Contains(body, `href="/posts.html"`) {
			t.Error("index links to the archive even though it lists every post")
		}
	})
}

// readFile is a small helper for the assertions below.
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(b)
}

// shelfRSS builds a minimal Goodreads-shaped feed for the fake server below.
func shelfRSS(titles ...string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><rss><channel>`)
	for i, title := range titles {
		fmt.Fprintf(&b,
			`<item><title>%s</title><book_id>%d</book_id><author_name>An Author</author_name></item>`,
			title, i+1)
	}
	b.WriteString(`</channel></rss>`)
	return b.String()
}

// TestFetchShelfHandlesFailure covers the error paths fetchShelf promises the
// caller it will return rather than abort on. None of them were reachable
// before goodreadsBaseURL became a seam, because the URL was built inline.
func TestFetchShelfHandlesFailure(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{
			name: "non-200 response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
		},
		{
			name: "malformed XML",
			handler: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "<rss><channel><item></rss")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			defer withGoodreadsBase(t, srv.URL)()

			if _, err := fetchShelf("123", "read", "", 5); err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}

// TestFetchShelfLimits checks the truncation to maxBooksPerShelf.
func TestFetchShelfLimits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, shelfRSS("One", "Two", "Three", "Four"))
	}))
	defer srv.Close()
	defer withGoodreadsBase(t, srv.URL)()

	books, err := fetchShelf("123", "read", "date_read", 2)
	if err != nil {
		t.Fatalf("fetchShelf: %v", err)
	}
	if len(books) != 2 {
		t.Fatalf("got %d books, want 2", len(books))
	}
	if books[0].Title != "One" {
		t.Errorf("got first book %q, want %q", books[0].Title, "One")
	}
}

// TestFetchFeaturedShelvesSkipsFailingShelf pins the behaviour the comment on
// fetchFeaturedShelves promises: one bad shelf is logged and skipped, and the
// rest of the section still renders.
func TestFetchFeaturedShelvesSkipsFailingShelf(t *testing.T) {
	// Serve the first shelf and fail the rest, whichever order they arrive in.
	var served int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served++
		if served == 1 {
			fmt.Fprint(w, shelfRSS("A Good Book"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	defer withGoodreadsBase(t, srv.URL)()

	groups := fetchFeaturedShelves("123")

	if len(groups) != 1 {
		t.Fatalf("got %d shelf groups, want 1 (the others failed and should be skipped)", len(groups))
	}
	if len(groups[0].Books) != 1 || groups[0].Books[0].Title != "A Good Book" {
		t.Errorf("unexpected books in the surviving shelf: %+v", groups[0].Books)
	}
}

// withGoodreadsBase points the shelf fetcher at a test server and returns a
// func that restores the real origin.
func withGoodreadsBase(t *testing.T, base string) func() {
	t.Helper()
	original := goodreadsBaseURL
	goodreadsBaseURL = base
	return func() { goodreadsBaseURL = original }
}

// TestPostMetadataEscapedEndToEnd checks escaping at the seam, not just in the
// unit. TestRenderPageEscaping proves renderPage escapes; it does not prove
// processMarkdownFile still goes through renderPage. Reintroducing a raw
// strings.Replace there — the exact regression AGENTS.md warns about — would
// break the <head> of every post with every existing test green.
func TestPostMetadataEscapedEndToEnd(t *testing.T) {
	nasty := `Do you slap a "LGTM!" on it? Tom & Jerry <script>alert(1)</script>`

	tests := []struct {
		name string
		body string
	}{
		{
			name: "description from frontmatter",
			body: "---\ntitle: Escaping\ndescription: " + nasty + "\n---\n\nBody text.\n",
		},
		{
			name: "description extracted from the first paragraph",
			body: "---\ntitle: Escaping\n---\n\n" + nasty + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "2024-01-01-escaping.md")
			if err := os.WriteFile(path, []byte(tt.body), 0644); err != nil {
				t.Fatalf("writing fixture: %v", err)
			}

			_, output, post, err := processMarkdownFile(path, testMetaTemplate)
			if err != nil {
				t.Fatalf("processMarkdownFile: %v", err)
			}

			head := output[:strings.Index(output, "</head>")]

			// The quote must be escaped, or it closes the content attribute
			// early and takes the rest of the <head> with it.
			if strings.Contains(head, `"LGTM!"`) {
				t.Error("an unescaped double quote reached a head attribute value")
			}
			if !strings.Contains(head, "&#34;LGTM!&#34;") {
				t.Errorf("expected the quote to be escaped in the head:\n%s", head)
			}
			if strings.Contains(head, "<script>") {
				t.Error("an unescaped <script> tag reached the head")
			}
			if !strings.Contains(head, "&amp;") {
				t.Error("the ampersand was not escaped")
			}
			// The head must still be intact: one description meta, one og:description.
			if n := strings.Count(head, `name="description"`); n != 1 {
				t.Errorf("found %d description meta tags, want 1 — the head is malformed", n)
			}
			if !strings.Contains(head, `property="og:description"`) {
				t.Error("og:description was lost, so the head broke early")
			}
			// The raw text should survive intact on the post itself.
			if !strings.Contains(post.Description, `"LGTM!"`) {
				t.Errorf("post description lost its content: %q", post.Description)
			}
		})
	}
}

// TestMalformedFrontmatterDoesNotLeakIntoPage pins what happens when the YAML
// fails to parse — the classic being an unquoted colon in a title. The parser
// error is swallowed and the whole file becomes the body, so the delimiters and
// the frontmatter keys render as prose and become the meta description.
func TestMalformedFrontmatterDoesNotLeakIntoPage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "2024-01-01-broken.md")
	body := "---\ntitle: Foo: bar\n---\n\nReal body text.\n"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	_, output, _, err := processMarkdownFile(path, testMetaTemplate)

	if err == nil {
		t.Fatalf("expected an error for malformed frontmatter, got none.\noutput:\n%s", output)
	}
	if !strings.Contains(err.Error(), "frontmatter") {
		t.Errorf("error should name the frontmatter as the cause, got: %v", err)
	}
	// The file must not be published with its YAML as the body.
	if strings.Contains(output, "title: Foo") {
		t.Error("the raw frontmatter line rendered as page content")
	}
}

// TestNoFrontmatterIsNotAnError keeps the check above from over-reaching: a
// post with no --- block at all is legitimate, and the title falls back to the
// filename.
func TestNoFrontmatterIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "2024-01-01-plain-post.md")
	if err := os.WriteFile(path, []byte("# Plain\n\nBody text.\n"), 0644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	_, output, post, err := processMarkdownFile(path, testMetaTemplate)
	if err != nil {
		t.Fatalf("a post with no frontmatter should build, got: %v", err)
	}
	if !strings.Contains(output, "Body text.") {
		t.Error("the post body was lost")
	}
	if post.Title != "plain post" {
		t.Errorf("got title %q, want the filename-derived %q", post.Title, "plain post")
	}
}

// TestInvalidFilenameDateDropsPostFromIndex documents what an impossible date
// costs: the page is still written, but a zero date drops it from the index,
// the archive, the tag pages and the feed, reachable only by direct URL.
func TestInvalidFilenameDateDropsPostFromIndex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "2023-02-30-impossible.md")
	if err := os.WriteFile(path, []byte("---\ntitle: Impossible\n---\n\nBody.\n"), 0644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	outName, _, post, err := processMarkdownFile(path, testMetaTemplate)
	if err != nil {
		t.Fatalf("processMarkdownFile: %v", err)
	}

	if outName != "2023-02-30-impossible.html" {
		t.Errorf("got output file %q, want the page to still be written", outName)
	}
	if !post.Date.IsZero() {
		t.Errorf("expected a zero date for an impossible calendar date, got %v", post.Date)
	}
	if got := datedPostsNewestFirst([]*BlogPost{post}); len(got) != 0 {
		t.Error("a post with an unparseable date should be dropped from the dated listing")
	}
}

// TestGenerateSiteStaticPagesAndCollisions covers the wiring between
// copyStaticDir and the rest of the build. copyStaticDir is tested in
// isolation, but every other end-to-end test runs in a temp dir with no
// static/, so staticPages is always nil: drop the append that feeds them to the
// sitemap and /style-guide.html, /sports.html and the quick reference vanish
// from it with a green suite. The collision rule — generated pages win — is
// enforced only by statement order, and nothing asserted it.
func TestGenerateSiteStaticPagesAndCollisions(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	staticDir := filepath.Join(testDir, "static")
	if err := os.Mkdir(staticDir, 0755); err != nil {
		t.Fatalf("creating static dir: %v", err)
	}
	files := map[string]string{
		"guide.html": "<html><body>the guide</body></html>",
		"theme.css":  "body { color: red }",
		// A decoy the generated homepage must win against.
		"index.html": "<html><body>DECOY HOMEPAGE</body></html>",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(staticDir, name), []byte(body), 0644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	originalWd, _ := os.Getwd()
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Could not change to test directory: %v", err)
	}
	defer os.Chdir(originalWd)

	buildDir := filepath.Join(testDir, "build")
	err := generateSite(
		filepath.Join(testDir, "content"),
		buildDir,
		filepath.Join(testDir, "template.html"),
		nil,
	)
	if err != nil {
		t.Fatalf("generateSite: %v", err)
	}

	// Static assets are copied verbatim.
	if got := readFile(t, filepath.Join(buildDir, "theme.css")); got != files["theme.css"] {
		t.Errorf("theme.css was not copied verbatim, got %q", got)
	}
	if got := readFile(t, filepath.Join(buildDir, "guide.html")); got != files["guide.html"] {
		t.Errorf("guide.html was not copied verbatim, got %q", got)
	}

	// Static HTML pages reach the sitemap; a stylesheet does not.
	sitemap := readFile(t, filepath.Join(buildDir, "sitemap.xml"))
	if !strings.Contains(sitemap, "guide.html") {
		t.Errorf("sitemap.xml omits the static page guide.html:\n%s", sitemap)
	}
	if strings.Contains(sitemap, "theme.css") {
		t.Error("sitemap.xml lists a stylesheet, which is not a page")
	}

	// The generated homepage wins the collision.
	index := readFile(t, filepath.Join(buildDir, "index.html"))
	if strings.Contains(index, "DECOY HOMEPAGE") {
		t.Error("a static index.html overwrote the generated homepage")
	}
	if !strings.Contains(index, "First Post") {
		t.Error("the generated homepage is missing its post list")
	}
}

// TestGenerateSiteWithNoPosts covers a fresh clone: an empty content directory
// should still produce the machine-readable files and the 404 page, and should
// not leave an empty index or archive behind.
func TestGenerateSiteWithNoPosts(t *testing.T) {
	testDir := t.TempDir()
	for _, dir := range []string{"content", "build"} {
		if err := os.Mkdir(filepath.Join(testDir, dir), 0755); err != nil {
			t.Fatalf("creating %s: %v", dir, err)
		}
	}
	templatePath := filepath.Join(testDir, "template.html")
	if err := os.WriteFile(templatePath, []byte(testMetaTemplate), 0644); err != nil {
		t.Fatalf("writing template: %v", err)
	}

	originalWd, _ := os.Getwd()
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Could not change to test directory: %v", err)
	}
	defer os.Chdir(originalWd)

	buildDir := filepath.Join(testDir, "build")
	if err := generateSite(filepath.Join(testDir, "content"), buildDir, templatePath, nil); err != nil {
		t.Fatalf("generateSite on an empty site: %v", err)
	}

	for _, name := range []string{"404.html", "robots.txt"} {
		if _, err := os.Stat(filepath.Join(buildDir, name)); err != nil {
			t.Errorf("%s was not written: %v", name, err)
		}
	}
	for _, name := range []string{"index.html", "posts.html"} {
		if _, err := os.Stat(filepath.Join(buildDir, name)); !os.IsNotExist(err) {
			t.Errorf("%s was written even though there are no posts", name)
		}
	}
}

// TestMain points the shelf fetcher at a local server for the whole package, so
// no test reaches goodreads.com. TestFileGeneration calls main(), which fetches
// every featured shelf; against the real origin that is three live HTTPS
// requests per run, each with a 15-second timeout — slow, flaky, and dependent
// on a third party being up for the suite to pass. Individual tests still
// override the base URL for their own fixtures.
func TestMain(m *testing.M) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, shelfRSS("A Cached Book"))
	}))
	goodreadsBaseURL = srv.URL

	code := m.Run()

	srv.Close()
	os.Exit(code)
}
