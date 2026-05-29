package main

import (
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
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

// defaultGoodreadsUserID is Simon's public Goodreads user ID. The "What I'm
// reading" section is built from public RSS, so there are no credentials to
// manage — only this ID, which already appears in the public profile URL.
// Override with the GOODREADS_USER_ID environment variable.
const defaultGoodreadsUserID = "28429269"

// maxBooksPerShelf caps how many books are shown under each shelf heading.
const maxBooksPerShelf = 3

// readingShelf describes one Goodreads shelf to feature, with the label shown
// on the page and the RSS sort key used to pick the most relevant books.
type readingShelf struct {
	shelf string // Goodreads shelf slug
	label string // heading shown on the page
	sort  string // Goodreads RSS sort key ("" = feed default)
}

// featuredShelves are the shelves rendered in the "What I'm reading" block, in
// display order. "to-read" is sorted by when it was added and "read" by when it
// was finished, so each group shows the most recent few.
var featuredShelves = []readingShelf{
	{shelf: "currently-reading", label: "Currently reading", sort: ""},
	{shelf: "to-read", label: "Want to read", sort: "date_added"},
	{shelf: "read", label: "Recently finished", sort: "date_read"},
}

// Book is a single book pulled from a public Goodreads shelf.
type Book struct {
	Title    string
	Author   string
	Link     string
	ImageURL string
}

// ShelfBooks is a labelled group of books from one shelf.
type ShelfBooks struct {
	Label string
	Books []Book
}

// goodreadsFeed mirrors the parts of the Goodreads RSS feed we care about.
type goodreadsFeed struct {
	Items []goodreadsItem `xml:"channel>item"`
}

type goodreadsItem struct {
	Title          string `xml:"title"`
	BookID         string `xml:"book_id"`
	AuthorName     string `xml:"author_name"`
	BookImageURL   string `xml:"book_image_url"`
	BookLargeImage string `xml:"book_large_image_url"`
}

// fetchFeaturedShelves pulls every shelf in featuredShelves, capped at
// maxBooksPerShelf each, and returns the non-empty groups in display order. A
// single failing shelf is logged and skipped so the rest still render; the
// caller decides what to do when nothing comes back at all.
func fetchFeaturedShelves(userID string) []ShelfBooks {
	groups := make([]ShelfBooks, 0, len(featuredShelves))
	for _, s := range featuredShelves {
		books, err := fetchShelf(userID, s.shelf, s.sort, maxBooksPerShelf)
		if err != nil {
			log.Printf("warning: could not fetch Goodreads shelf %q: %v", s.shelf, err)
			continue
		}
		if len(books) == 0 {
			continue
		}
		groups = append(groups, ShelfBooks{Label: s.label, Books: books})
	}
	return groups
}

// fetchShelf retrieves a public Goodreads shelf as RSS at build time, sorted by
// the given key (empty for the feed default) and truncated to limit books. No
// authentication is involved; the feed is public. Failures are returned to the
// caller so the build can carry on without the section rather than aborting.
func fetchShelf(userID, shelf, sort string, limit int) ([]Book, error) {
	url := fmt.Sprintf("https://www.goodreads.com/review/list_rss/%s?shelf=%s", userID, shelf)
	if sort != "" {
		// order=d gives newest-first for date-based sorts.
		url += fmt.Sprintf("&sort=%s&order=d", sort)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching shelf: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("goodreads returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	books, err := parseShelf(body)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(books) > limit {
		books = books[:limit]
	}
	return books, nil
}

// parseShelf turns a Goodreads RSS document into a slice of books. It is split
// out from fetchShelf so it can be tested without touching the network.
func parseShelf(data []byte) ([]Book, error) {
	var feed goodreadsFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("parsing feed: %w", err)
	}

	books := make([]Book, 0, len(feed.Items))
	for _, item := range feed.Items {
		// Prefer the larger cover; fall back to the thumbnail.
		image := strings.TrimSpace(item.BookLargeImage)
		if image == "" {
			image = strings.TrimSpace(item.BookImageURL)
		}

		// Build a clean canonical book link from the ID, dropping the
		// tracking parameters Goodreads attaches to the RSS <link>.
		link := ""
		if id := strings.TrimSpace(item.BookID); id != "" {
			link = "https://www.goodreads.com/book/show/" + id
		}

		books = append(books, Book{
			Title:    strings.TrimSpace(item.Title),
			Author:   strings.TrimSpace(item.AuthorName),
			Link:     link,
			ImageURL: image,
		})
	}
	return books, nil
}

// renderReadingSection builds the "What I'm reading" HTML block from one or more
// labelled shelves. It returns an empty string when there are no books at all,
// so an unavailable feed simply omits the section instead of leaving an empty
// heading behind.
func renderReadingSection(groups []ShelfBooks) string {
	hasBooks := false
	for _, g := range groups {
		if len(g.Books) > 0 {
			hasBooks = true
			break
		}
	}
	if !hasBooks {
		return ""
	}

	var b strings.Builder
	b.WriteString("<section class=\"reading\">")
	b.WriteString("<h2>What I'm reading</h2>")
	for _, group := range groups {
		if len(group.Books) == 0 {
			continue
		}
		b.WriteString("<div class=\"shelf\">")
		b.WriteString(fmt.Sprintf("<h3 class=\"shelf-label\">%s</h3>", html.EscapeString(group.Label)))
		b.WriteString("<ul class=\"book-list\">")
		for _, book := range group.Books {
			b.WriteString("<li class=\"book\">")

			openLink := book.Link != ""
			if openLink {
				b.WriteString(fmt.Sprintf("<a href=\"%s\">", html.EscapeString(book.Link)))
			}
			if book.ImageURL != "" {
				b.WriteString(fmt.Sprintf(
					"<img class=\"book-cover\" src=\"%s\" alt=\"Cover of %s\" loading=\"lazy\" />",
					html.EscapeString(book.ImageURL), html.EscapeString(book.Title)))
			}
			if openLink {
				b.WriteString("</a>")
			}

			b.WriteString("<div class=\"book-meta\">")
			b.WriteString(fmt.Sprintf("<span class=\"book-title\">%s</span>", html.EscapeString(book.Title)))
			if book.Author != "" {
				b.WriteString(fmt.Sprintf("<span class=\"book-author\">%s</span>", html.EscapeString(book.Author)))
			}
			b.WriteString("</div>")

			b.WriteString("</li>")
		}
		b.WriteString("</ul>")
		b.WriteString("</div>")
	}
	b.WriteString("</section>")
	return b.String()
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

// latestPostCount is how many posts the landing page lists before linking out
// to the full archive on posts.html.
const latestPostCount = 5

// datedPostsNewestFirst returns the posts that have a valid date, sorted newest
// first. Posts without a date are dropped, since the date drives ordering.
func datedPostsNewestFirst(posts []*BlogPost) []*BlogPost {
	dated := make([]*BlogPost, 0, len(posts))
	for _, post := range posts {
		if !post.Date.IsZero() {
			dated = append(dated, post)
		}
	}
	sort.Slice(dated, func(i, j int) bool {
		return dated[i].Date.After(dated[j].Date)
	})
	return dated
}

// renderPostList renders a <ul> linking to each post, newest first. Callers
// pass an already-filtered, already-sorted slice.
func renderPostList(posts []*BlogPost) string {
	var b strings.Builder
	b.WriteString("<ul class=\"post-list\">")
	for _, post := range posts {
		formattedDate := post.Date.Format("January 2, 2006")
		b.WriteString(fmt.Sprintf("<li><strong>%s</strong> - <a href=\"%s\">%s</a><p>%s</p></li>\n",
			formattedDate, post.OutputFile, post.Title, post.Description))
	}
	b.WriteString("</ul>")
	return b.String()
}

// generateIndex generates the landing page: a short intro and the latest posts,
// with a link to the full archive when there are more.
func generateIndex(posts []*BlogPost, template string, buildDir string, shelves []ShelfBooks) error {
	dated := datedPostsNewestFirst(posts)

	var contentBuilder strings.Builder
	contentBuilder.WriteString("<p>Notes on building software, shipping it, and the engineering practices in between.</p>")
	contentBuilder.WriteString("<p>Hi! I'm <a href=\"https://github.com/sbracegirdle\" rel=\"author\"><em>Simon</em></a>, a software engineer and consultant in Perth, Western Australia. I've spent 20+ years building products and helping teams improve how they work — now at <a href=\"https://govconnex.com/\">GovConnex</a>, after <a href=\"https://mechanicalrock.io\">Mechanical Rock</a> and <a href=\"https://seqta.com.au\">SEQTA Software</a>.</p>")
	contentBuilder.WriteString(renderReadingSection(shelves))
	contentBuilder.WriteString("<h2>Latest posts</h2>")

	latest := dated
	if len(latest) > latestPostCount {
		latest = latest[:latestPostCount]
	}
	contentBuilder.WriteString(renderPostList(latest))

	if len(dated) > len(latest) {
		contentBuilder.WriteString("<p><a href=\"posts.html\">All posts &rarr;</a></p>")
	}

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

// generateArchive generates posts.html: every dated post, newest first.
func generateArchive(posts []*BlogPost, template string, buildDir string) error {
	dated := datedPostsNewestFirst(posts)

	var contentBuilder strings.Builder
	contentBuilder.WriteString("<h1>All posts</h1>")
	contentBuilder.WriteString(renderPostList(dated))
	contentBuilder.WriteString("<p><a href=\"index.html\">&larr; Home</a></p>")

	output := strings.Replace(template, "{{title}}", "All posts", -1)
	output = strings.Replace(output, "{{content}}", contentBuilder.String(), -1)

	outputPath := filepath.Join(buildDir, "posts.html")
	if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("error writing archive file: %v", err)
	}

	fmt.Printf("Generated archive: %s\n", outputPath)
	return nil
}

// generateSite processes all markdown files in the content directory
func generateSite(contentDir, buildDir, templatePath string, shelves []ShelfBooks) error {
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

	// Generate index and archive pages
	if len(blogPosts) > 0 {
		if err = generateIndex(blogPosts, template, buildDir, shelves); err != nil {
			log.Printf("Error generating index: %v", err)
		}
		if err = generateArchive(blogPosts, template, buildDir); err != nil {
			log.Printf("Error generating archive: %v", err)
		}
	}

	return nil
}

func main() {
	contentDir := filepath.Join(".", "content")
	buildDir := filepath.Join(".", "build")
	templatePath := filepath.Join(".", "template.html")

	// Pull the "What I'm reading" shelves from public Goodreads RSS at build
	// time. This is best-effort: any shelf that can't be fetched is logged and
	// skipped rather than failing the build.
	userID := os.Getenv("GOODREADS_USER_ID")
	if userID == "" {
		userID = defaultGoodreadsUserID
	}
	shelves := fetchFeaturedShelves(userID)

	if err := generateSite(contentDir, buildDir, templatePath, shelves); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Site generation complete!")
}
