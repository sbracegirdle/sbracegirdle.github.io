package main

import (
	"encoding/xml"
	"fmt"
	"html"
	htmltemplate "html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/adrg/frontmatter"
)

// Site identity. These feed canonical URLs, social cards, the RSS feed and the
// sitemap, so they are the one place the public address of the site is written
// down. siteURL carries no trailing slash.
const (
	siteURL         = "https://letsbuild.cloud"
	siteName        = "LetsBuild.cloud"
	siteAuthor      = "Simon Bracegirdle"
	siteTitle       = "Let's Build"
	siteDescription = "Notes on building software, shipping it, and the engineering practices in between — by Simon Bracegirdle, a software engineer in Perth, Western Australia."
	siteHomeHeading = "Let's build something."
	siteHomeIntro   = siteAuthor + ". Software engineer in Perth."
	siteHomeFocus   = "AI, web applications and cloud infrastructure."
)

// FrontMatter represents the metadata at the top of markdown files
type FrontMatter struct {
	Title       string  `yaml:"title"`
	Description string  `yaml:"description"`
	Tags        tagList `yaml:"tags"`
}

// BlogPost represents metadata about a blog post
type BlogPost struct {
	Title       string
	Date        time.Time
	Filename    string
	OutputFile  string
	Description string
	Tags        []string
}

// canonicalURL turns a build-relative output path into the absolute URL the
// page is served from. index.html maps to the bare site root so the home page
// has a single canonical form.
func canonicalURL(outputFile string) string {
	if outputFile == "index.html" {
		return siteURL + "/"
	}
	return siteURL + "/" + strings.TrimPrefix(outputFile, "/")
}

type pageMeta struct {
	Title       string // <title> and og:title
	Heading     string // visible <h1>; defaults to Title
	File        string
	Description string // meta description and og:description
	Canonical   string // absolute URL of this page
	OGType      string // og:type; defaults to "website"
	Content     string // rendered page body, inserted raw
	Kind        string
	Intro       string
	NoIndex     bool
	HideIntro   bool
	Date        time.Time
}

func renderPage(source string, m pageMeta) (string, error) {
	if m.Heading == "" {
		m.Heading = m.Title
	}
	if m.OGType == "" {
		m.OGType = "website"
	}
	if m.Kind == "" {
		m.Kind = "page"
	}
	if m.Intro == "" {
		m.Intro = m.Description
	}
	headingClass := ""
	if utf8.RuneCountInString(m.Heading) > 45 {
		headingClass = "long-title"
	}
	page, err := htmltemplate.New("page").Option("missingkey=error").Parse(source)
	if err != nil {
		return "", fmt.Errorf("parsing page template: %w", err)
	}
	data := struct {
		pageMeta
		Content      htmltemplate.HTML
		Author       string
		HomeFocus    string
		HeadingClass string
	}{
		pageMeta:     m,
		Content:      htmltemplate.HTML(m.Content),
		Author:       siteAuthor,
		HomeFocus:    siteHomeFocus,
		HeadingClass: headingClass,
	}
	var output strings.Builder
	if err := page.Execute(&output, data); err != nil {
		return "", fmt.Errorf("rendering %q: %w", m.Title, err)
	}
	return output.String(), nil
}

// defaultGoodreadsUserID is Simon's public Goodreads user ID. The "What I'm
// reading" section is built from public RSS, so there are no credentials to
// manage — only this ID, which already appears in the public profile URL.
// Override with the GOODREADS_USER_ID environment variable.
const defaultGoodreadsUserID = "28429269"

// maxBooksPerShelf caps how many books are shown under each shelf heading.
const maxBooksPerShelf = 3

type readingShelf struct {
	shelf string // Goodreads shelf slug
	label string // heading shown on the page
	sort  string // Goodreads RSS sort key ("" = feed default)
}

var featuredShelves = []readingShelf{
	{shelf: "currently-reading", label: "Currently reading", sort: ""},
	{shelf: "to-read", label: "Want to read", sort: "date_added"},
	{shelf: "read", label: "Recently finished", sort: "date_read"},
}

type Book struct {
	Title  string
	Author string
	Link   string
}

type ShelfBooks struct {
	Label string
	Books []Book
}

// goodreadsFeed mirrors the parts of the Goodreads RSS feed we care about.
type goodreadsFeed struct {
	Items []goodreadsItem `xml:"channel>item"`
}

type goodreadsItem struct {
	Title      string `xml:"title"`
	BookID     string `xml:"book_id"`
	AuthorName string `xml:"author_name"`
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

// goodreadsBaseURL is the origin the shelf feeds are fetched from. It is a var
// rather than a const solely so tests can point it at an httptest server —
// without that seam every test that builds the whole site makes three live
// requests to goodreads.com.
var goodreadsBaseURL = "https://www.goodreads.com"

// fetchShelf retrieves a public Goodreads shelf as RSS at build time, sorted by
// the given key (empty for the feed default) and truncated to limit books. No
// authentication is involved; the feed is public. Failures are returned to the
// caller so the build can carry on without the section rather than aborting.
func fetchShelf(userID, shelf, sort string, limit int) ([]Book, error) {
	url := fmt.Sprintf("%s/review/list_rss/%s?shelf=%s", goodreadsBaseURL, userID, shelf)
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
		// Build a clean canonical book link from the ID, dropping the
		// tracking parameters Goodreads attaches to the RSS <link>.
		link := ""
		if id := strings.TrimSpace(item.BookID); id != "" {
			link = "https://www.goodreads.com/book/show/" + id
		}

		books = append(books, Book{
			Title:  strings.TrimSpace(item.Title),
			Author: strings.TrimSpace(item.AuthorName),
			Link:   link,
		})
	}
	return books, nil
}

func renderReadingSection(groups []ShelfBooks) string {
	var b strings.Builder
	for _, group := range groups {
		if len(group.Books) == 0 {
			continue
		}
		b.WriteString("<div class=\"reading-shelf\">")
		b.WriteString(fmt.Sprintf("<h3>%s</h3><ul class=\"book-list\">", html.EscapeString(group.Label)))
		for _, book := range group.Books {
			b.WriteString("<li>")
			title := html.EscapeString(book.Title)
			if book.Link != "" {
				b.WriteString(fmt.Sprintf("<a href=\"%s\" class=\"book-title\">%s</a>", html.EscapeString(book.Link), title))
			} else {
				b.WriteString(fmt.Sprintf("<span class=\"book-title\">%s</span>", title))
			}
			if book.Author != "" {
				b.WriteString(fmt.Sprintf("<span class=\"book-author\">%s</span>", html.EscapeString(book.Author)))
			}
			b.WriteString("</li>")
		}
		b.WriteString("</ul></div>")
	}
	return b.String()
}

func renderAboutReading(shelves []ShelfBooks) string {
	userID := os.Getenv("GOODREADS_USER_ID")
	if userID == "" {
		userID = defaultGoodreadsUserID
	}
	return "<section class=\"reading\" id=\"reading\" aria-labelledby=\"reading-title\"><h2 id=\"reading-title\">What I'm reading</h2>" +
		renderReadingSection(shelves) +
		fmt.Sprintf("<p class=\"reading-profile\"><a href=\"https://www.goodreads.com/review/list/%s\">My books on Goodreads &rarr;</a></p></section>", html.EscapeString(userID))
}

// processMarkdownFile processes a single markdown file and returns the generated HTML
func processMarkdownFile(filePath, template string) (string, string, *BlogPost, error) {
	return processMarkdownFileWithAppendix(filePath, template, "")
}

func processMarkdownFileWithAppendix(filePath, template, appendix string) (string, string, *BlogPost, error) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", nil, fmt.Errorf("error reading file %s: %v", filePath, err)
	}

	// Parse frontmatter
	var meta FrontMatter
	content, err := frontmatter.Parse(strings.NewReader(string(fileContent)), &meta)
	if err != nil {
		// A file with no frontmatter at all parses cleanly, so an error here
		// means the YAML between the --- delimiters is genuinely broken — an
		// unquoted colon in a title being the classic. The old behaviour was to
		// fall back to the whole file as the body, which published the
		// delimiters and the frontmatter keys as prose and made the first of
		// them the meta description. Refusing the file is louder and cheaper to
		// notice: the caller logs it and skips the post.
		return "", "", nil, fmt.Errorf("error parsing frontmatter in %s: %w", filePath, err)
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

	// Parse markdown to HTML (code blocks are syntax-highlighted at build time)
	htmlContent := renderMarkdown(content)

	outputFilename := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".html"

	// Create blog post metadata
	blogPost := &BlogPost{
		Title:       title,
		Date:        postDate,
		Filename:    filename,
		OutputFile:  outputFilename,
		Description: description,
		Tags:        normaliseTags(meta.Tags),
	}

	// Dated posts are articles; undated pages (about, and anything else) are
	// ordinary pages. Only articles carry a published time.
	ogType := "website"
	kind := "page"
	if !postDate.IsZero() {
		ogType = "article"
		kind = "article"
	}

	output, err := renderPage(template, pageMeta{
		Title:       title,
		File:        outputFilename,
		Description: description,
		Canonical:   canonicalURL(outputFilename),
		OGType:      ogType,
		Kind:        kind,
		Date:        postDate,
		// Tag chips sit at the top of the body, above the prose, the way a
		// file header states what a document is about.
		Content: renderTagChips(blogPost.Tags) + string(htmlContent) + appendix,
	})

	if err != nil {
		return "", "", nil, err
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
			return truncateRunes(p, 150)
		}
	}

	// Fallback to first 150 chars if no paragraph found
	return truncateRunes(strings.TrimSpace(text), 150)
}

// truncateRunes shortens s to at most limit characters, appending an ellipsis
// when it cuts. It counts runes rather than bytes: this text lands in
// <meta description> and og:description, and slicing a multi-byte rune down the
// middle — an em dash or a curly quote straddling the cut — emits invalid UTF-8
// that escaping cannot repair.
func truncateRunes(s string, limit int) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	return string([]rune(s)[:limit-3]) + "..."
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
	// Two posts can share a date — content/ has a pair on 2021-12-20 — so the
	// comparator falls back to the output filename. Without a total order the
	// result would rest on sort.Slice's unspecified behaviour over directory
	// order, and a Go upgrade could silently reshuffle the index, the archive
	// and the feed.
	sort.SliceStable(dated, func(i, j int) bool {
		if !dated[i].Date.Equal(dated[j].Date) {
			return dated[i].Date.After(dated[j].Date)
		}
		return dated[i].OutputFile < dated[j].OutputFile
	})
	return dated
}

func renderPostList(posts []*BlogPost) string {
	var b strings.Builder
	b.WriteString("<ul class=\"post-list\">")
	for _, post := range posts {
		b.WriteString(fmt.Sprintf("<li><time class=\"date\" datetime=\"%s\">%s</time><a href=\"/%s\">%s</a><p>%s</p></li>\n",
			post.Date.Format("2006-01-02"), post.Date.Format("02 Jan 2006"), html.EscapeString(post.OutputFile), html.EscapeString(post.Title), html.EscapeString(post.Description)))
	}
	b.WriteString("</ul>")
	return b.String()
}

func generateIndex(posts []*BlogPost, template string, buildDir string, _ []ShelfBooks) error {
	dated := datedPostsNewestFirst(posts)

	var contentBuilder strings.Builder

	contentBuilder.WriteString("<div class=\"list-heading\"><h2>Latest posts</h2>")

	latest := dated
	if len(latest) > latestPostCount {
		latest = latest[:latestPostCount]
	}
	if len(dated) > len(latest) {
		contentBuilder.WriteString("<a href=\"/posts.html\">All posts &rarr;</a>")
	}
	contentBuilder.WriteString("</div>")
	contentBuilder.WriteString(renderPostList(latest))

	output, err := renderPage(template, pageMeta{
		Title:       siteTitle,
		Heading:     siteHomeHeading,
		Kind:        "home",
		Intro:       siteHomeIntro,
		File:        "index.html",
		Description: siteDescription,
		Canonical:   canonicalURL("index.html"),
		Content:     contentBuilder.String(),
	})
	if err != nil {
		return err
	}

	// Write the index file
	outputPath := filepath.Join(buildDir, "index.html")
	err = os.WriteFile(outputPath, []byte(output), 0644)
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
	contentBuilder.WriteString(renderPostList(dated))
	contentBuilder.WriteString("<p><a href=\"/\">&larr; Home</a> &middot; <a href=\"/tags.html\">browse by tag &rarr;</a></p>")

	output, err := renderPage(template, pageMeta{
		Title:       "All posts",
		File:        "posts.html",
		Description: "Posts by " + siteAuthor + ".",
		HideIntro:   true,
		Canonical:   canonicalURL("posts.html"),
		Content:     contentBuilder.String(),
	})
	if err != nil {
		return err
	}

	outputPath := filepath.Join(buildDir, "posts.html")
	if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("error writing archive file: %v", err)
	}

	fmt.Printf("Generated archive: %s\n", outputPath)
	return nil
}

// generateNotFound writes 404.html, which GitHub Pages serves for any unknown
// path. It is marked noindex — a soft 404 in the search index helps nobody.
func generateNotFound(template, buildDir string) error {
	var contentBuilder strings.Builder
	contentBuilder.WriteString("<p>No such file or directory. The page you asked for isn't here — it may have moved, or the link may be wrong.</p>")
	contentBuilder.WriteString("<ul>")
	contentBuilder.WriteString("<li><a href=\"/\">Home</a></li>")
	contentBuilder.WriteString("<li><a href=\"/posts.html\">All posts</a></li>")
	contentBuilder.WriteString("<li><a href=\"/tags.html\">Browse by tag</a></li>")
	contentBuilder.WriteString("</ul>")

	output, err := renderPage(template, pageMeta{
		Title:       "404 — page not found",
		Heading:     "404",
		File:        "404.html",
		Description: "That page isn't here.",
		Canonical:   canonicalURL("404.html"),
		NoIndex:     true,
		Content:     contentBuilder.String(),
	})
	if err != nil {
		return err
	}

	outputPath := filepath.Join(buildDir, "404.html")
	if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("error writing 404 page: %v", err)
	}

	fmt.Printf("Generated: %s\n", outputPath)
	return nil
}

// copyStaticDir copies every file under staticDir into buildDir, preserving
// relative paths and subdirectories. Files in static/ bypass the markdown
// rendering pipeline entirely, so a standalone HTML resource (e.g. a
// self-contained quick-reference page) can be served and linked from the site
// without being wrapped in the post template. The one transformation applied:
// <script type="text/rust|shell"> source blocks in HTML files are pre-rendered
// into highlighted <pre class="code"> markup (see renderStaticCodeScripts).
// A missing staticDir is a no-op. Generated pages are written after this runs,
// so a generated file always wins on a name collision with a static one.
//
// It returns the site-relative URL path of every HTML page copied, so standalone
// pages can be listed in the sitemap without being enumerated by hand.
func copyStaticDir(staticDir, buildDir string) ([]string, error) {
	var pages []string
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		return nil, nil
	}
	err := filepath.Walk(staticDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(staticDir, p)
		if err != nil {
			return err
		}
		dst := filepath.Join(buildDir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.HasSuffix(p, ".html") {
			data = []byte(renderStaticCodeScripts(string(data)))
			pages = append(pages, filepath.ToSlash(rel))
		}
		return os.WriteFile(dst, data, 0644)
	})
	sort.Strings(pages)
	return pages, err
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

	// Copy standalone resources from static/ before generating posts, so any
	// generated page takes precedence on a name collision.
	staticDir := filepath.Join(".", "static")
	staticPages, err := copyStaticDir(staticDir, buildDir)
	if err != nil {
		log.Printf("warning: could not copy static directory: %v", err)
	}

	// Get template
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return fmt.Errorf("template file not found at %s", templatePath)
	}

	templateBytes, readErr := os.ReadFile(templatePath)
	if readErr != nil {
		return fmt.Errorf("error reading template: %v", readErr)
	}
	template := string(templateBytes)
	if _, err := renderPage(template, pageMeta{}); err != nil {
		return err
	}

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
		appendix := ""
		if strings.TrimSuffix(file.Name(), filepath.Ext(file.Name())) == "about" {
			appendix = renderAboutReading(shelves)
		}
		outputFilename, outputContent, blogPost, err := processMarkdownFileWithAppendix(filePath, template, appendix)
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

	// Generate index, archive and tag pages
	var tagPages []string
	if len(blogPosts) > 0 {
		if err := generateIndex(blogPosts, template, buildDir, shelves); err != nil {
			log.Printf("Error generating index: %v", err)
		}
		if err := generateArchive(blogPosts, template, buildDir); err != nil {
			log.Printf("Error generating archive: %v", err)
		}
		var err error
		if tagPages, err = generateTagPages(blogPosts, template, buildDir); err != nil {
			log.Printf("Error generating tag pages: %v", err)
		}
	}

	// Machine-readable outputs: feed for readers, sitemap and robots.txt for
	// crawlers. Each is best-effort — a failure here shouldn't lose the pages
	// that already generated.
	if err := generateFeed(blogPosts, buildDir); err != nil {
		log.Printf("Error generating feed: %v", err)
	}
	// Pages the sitemap lists beyond the posts themselves. The generated
	// listings only exist when there was at least one post to list, and the tag
	// pages only when at least one post carried a tag — generateTagPages
	// reports exactly what it wrote.
	var pages []string
	if len(blogPosts) > 0 {
		pages = append(pages, "index.html", "posts.html")
	}
	pages = append(pages, tagPages...)
	pages = append(pages, staticPages...)
	if err := generateSitemap(blogPosts, pages, buildDir); err != nil {
		log.Printf("Error generating sitemap: %v", err)
	}
	if err := generateRobots(buildDir); err != nil {
		log.Printf("Error generating robots.txt: %v", err)
	}
	if err := generateNotFound(template, buildDir); err != nil {
		log.Printf("Error generating 404 page: %v", err)
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

	if err := generateSite(contentDir, buildDir, templatePath, fetchFeaturedShelves(userID)); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Site generation complete!")
}
