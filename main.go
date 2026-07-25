package main

import (
	"encoding/xml"
	"flag"
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
	"sync"
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

// pageMeta carries everything template.html needs to render one page. The
// scalar fields are escaped by renderPage; Content and HeadExtra are inserted
// verbatim and so must already be valid HTML.
type pageMeta struct {
	Title       string // <title> and og:title
	Heading     string // visible <h1>; defaults to Title
	File        string // filename shown in the statusline
	Description string // meta description and og:description
	Canonical   string // absolute URL of this page
	OGType      string // og:type; defaults to "website"
	HeadExtra   string // extra <head> markup, inserted raw
	Content     string // rendered page body, inserted raw
}

// renderPage fills the template placeholders for a single page. Titles and
// descriptions now land in attribute values as well as element text, so every
// scalar is escaped on the way in — an unescaped quote in a description would
// otherwise close the meta content attribute early and mangle the head. Content
// and HeadExtra are substituted last, so a placeholder that happens to appear
// inside a post (in a code block, say) is never expanded.
func renderPage(template string, m pageMeta) string {
	if m.Heading == "" {
		m.Heading = m.Title
	}
	if m.OGType == "" {
		m.OGType = "website"
	}

	scalars := []struct{ placeholder, value string }{
		{"{{title}}", m.Title},
		{"{{heading}}", m.Heading},
		{"{{file}}", m.File},
		{"{{description}}", m.Description},
		{"{{canonical}}", m.Canonical},
		{"{{ogtype}}", m.OGType},
	}

	out := template
	for _, s := range scalars {
		out = strings.ReplaceAll(out, s.placeholder, html.EscapeString(s.value))
	}
	out = strings.ReplaceAll(out, "{{head_extra}}", m.HeadExtra)
	out = strings.ReplaceAll(out, "{{content}}", m.Content)
	return out
}

// defaultGoodreadsUserID is Simon's public Goodreads user ID. The "What I'm
// reading" section is built from public RSS, so there are no credentials to
// manage — only this ID, which already appears in the public profile URL.
// Override with the GOODREADS_USER_ID environment variable.
const defaultGoodreadsUserID = "28429269"

// maxBooksPerShelf caps how many books are shown under each shelf heading.
const maxBooksPerShelf = 3

// readingShelf describes one Goodreads shelf to feature, with the label shown
// on the page, the RSS sort key used to pick the most relevant books, and the
// theme hue that tints its card.
type readingShelf struct {
	shelf string // Goodreads shelf slug
	label string // heading shown on the page
	sort  string // Goodreads RSS sort key ("" = feed default)
	hue   string // theme.css hue modifier ("gold", "foam", "iris")
}

// featuredShelves are the shelves rendered in the "What I'm reading" block, in
// display order. "to-read" is sorted by when it was added and "read" by when it
// was finished, so each group shows the most recent few. Each shelf carries a
// distinct hue so the three cards read as three groups; the hue only repeats
// the label, which is always on the card, so nothing is carried by colour alone.
var featuredShelves = []readingShelf{
	{shelf: "currently-reading", label: "Currently reading", sort: "", hue: "gold"},
	{shelf: "to-read", label: "Want to read", sort: "date_added", hue: "foam"},
	{shelf: "read", label: "Recently finished", sort: "date_read", hue: "iris"},
}

// Book is a single book pulled from a public Goodreads shelf. Covers are
// deliberately not carried: the page draws a CSS book glyph instead of
// hot-linking Goodreads artwork, which ran to hundreds of kilobytes a cover for
// a 26px slot.
type Book struct {
	Title  string
	Author string
	Link   string
}

// ShelfBooks is a labelled group of books from one shelf, with the hue that
// tints its card.
type ShelfBooks struct {
	Label string
	Hue   string
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
		groups = append(groups, ShelfBooks{Label: s.label, Hue: s.hue, Books: books})
	}
	return groups
}

// shelfTTL is how long a cached Goodreads fetch is reused during watch mode,
// so repeated regenerations (one per file touch) don't hammer the public RSS
// feed.
const shelfTTL = 10 * time.Minute

// shelfCache memoises the Goodreads shelves for shelfTTL. In watch mode the
// same process regenerates the site many times; without this every file touch
// would re-fetch every shelf.
type shelfCache struct {
	mu        sync.Mutex
	shelves   []ShelfBooks
	fetchedAt time.Time
}

// get returns the cached shelves if they are younger than ttl, otherwise it
// re-fetches and caches them. A ttl <= 0 always fetches (used for one-shot
// builds, preserving the original behaviour).
func (c *shelfCache) get(userID string, ttl time.Duration) []ShelfBooks {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ttl > 0 && time.Since(c.fetchedAt) < ttl {
		return c.shelves
	}
	c.shelves = fetchFeaturedShelves(userID)
	c.fetchedAt = time.Now()
	return c.shelves
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

	// No heading: the shelf cards carry their own titles, so a section heading
	// would just repeat them. The aria-label keeps the landmark named.
	var b strings.Builder
	b.WriteString("<section class=\"reading\" aria-label=\"What I'm reading\">")
	b.WriteString("<div class=\"card-grid\">")
	for _, group := range groups {
		if len(group.Books) == 0 {
			continue
		}
		cardClass := "card"
		if group.Hue != "" {
			cardClass += " shelf-" + html.EscapeString(group.Hue)
		}
		b.WriteString(fmt.Sprintf("<div class=\"%s\">", cardClass))
		b.WriteString(fmt.Sprintf("<span class=\"card-title\">%s</span>", html.EscapeString(group.Label)))
		b.WriteString("<ul class=\"book-list\">")
		for _, book := range group.Books {
			b.WriteString("<li class=\"book\">")

			openLink := book.Link != ""
			if openLink {
				b.WriteString(fmt.Sprintf("<a href=\"%s\" class=\"book-link\">", html.EscapeString(book.Link)))
			}
			// A CSS-drawn book, not an image: no request, no third party, and
			// nothing to announce, so it stays out of the accessibility tree.
			b.WriteString("<span class=\"book-glyph\" aria-hidden=\"true\"></span>")

			b.WriteString("<div class=\"book-meta\">")
			b.WriteString(fmt.Sprintf("<span class=\"book-title\">%s</span>", html.EscapeString(book.Title)))
			if book.Author != "" {
				b.WriteString(fmt.Sprintf("<span class=\"book-author\">%s</span>", html.EscapeString(book.Author)))
			}
			b.WriteString("</div>")
			if openLink {
				b.WriteString("</a>")
			}

			b.WriteString("</li>")
		}
		b.WriteString("</ul>")
		b.WriteString("</div>")
	}
	b.WriteString("</div>")
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
	headExtra := ""
	if !postDate.IsZero() {
		ogType = "article"
		headExtra = fmt.Sprintf("<meta property=\"article:published_time\" content=\"%s\" />",
			html.EscapeString(postDate.Format(time.RFC3339)))
	}

	output := renderPage(template, pageMeta{
		Title:       title,
		File:        outputFilename,
		Description: description,
		Canonical:   canonicalURL(outputFilename),
		OGType:      ogType,
		HeadExtra:   headExtra,
		// Tag chips sit at the top of the body, above the prose, the way a
		// file header states what a document is about.
		Content: renderTagChips(blogPost.Tags) + string(htmlContent),
	})

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

// renderPostList renders posts as ul.post-list, newest first: gold ISO date,
// linked title, one-line description. Callers pass an already-filtered,
// already-sorted slice. Links are root-absolute so the same markup works from
// the site root and from pages nested under /tags/.
func renderPostList(posts []*BlogPost) string {
	var b strings.Builder
	b.WriteString("<ul class=\"post-list\">")
	for _, post := range posts {
		formattedDate := post.Date.Format("2006-01-02")
		b.WriteString(fmt.Sprintf("<li><span class=\"date\">%s</span><a href=\"/%s\">%s</a><p>%s</p></li>\n",
			formattedDate, post.OutputFile, html.EscapeString(post.Title), html.EscapeString(post.Description)))
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
	contentBuilder.WriteString("<p>What I bring:</p><ul>")
	contentBuilder.WriteString("<li><em>AI features</em> — a research assistant that orchestrates agents and tools, plus overnight briefings and the embeddings pipeline behind semantic search.</li>")
	contentBuilder.WriteString("<li><em>Web applications</em> — React and TypeScript, Node APIs, Elasticsearch, serverless AWS.</li>")
	contentBuilder.WriteString("<li><em>Migrations</em> — a production webapp from JavaScript to TypeScript, about 30,000 lines, without pausing feature work.</li>")
	contentBuilder.WriteString("<li><em>Team tooling</em> — CI/CD, load testing, code review, and agent-assisted workflows.</li>")
	contentBuilder.WriteString("</ul>")
	contentBuilder.WriteString("<p><a href=\"/about.html\">More about me &rarr;</a></p>")
	contentBuilder.WriteString(renderReadingSection(shelves))
	contentBuilder.WriteString("<h2>Reference pages</h2>")
	contentBuilder.WriteString("<ul class=\"post-list\">")
	contentBuilder.WriteString("<li><a href=\"/rust-quick-reference.html\">Rust quick reference</a><p>A progressive tour of Rust — ownership, borrowing, traits, and cargo — styled like a TUI.</p></li>")
	contentBuilder.WriteString("<li><a href=\"/sports.html\">Sports 2026</a><p>What's left of the season in the six sports I follow, from the Vuelta to the Bledisloe to the Boxing Day Test.</p></li>")
	contentBuilder.WriteString("</ul>")
	contentBuilder.WriteString("<h2>Latest posts</h2>")

	latest := dated
	if len(latest) > latestPostCount {
		latest = latest[:latestPostCount]
	}
	contentBuilder.WriteString(renderPostList(latest))

	if len(dated) > len(latest) {
		contentBuilder.WriteString("<p><a href=\"/posts.html\">All posts &rarr;</a> &middot; <a href=\"/tags.html\">browse by tag &rarr;</a></p>")
	}

	output := renderPage(template, pageMeta{
		Title:       siteTitle,
		File:        "index.html",
		Description: siteDescription,
		Canonical:   canonicalURL("index.html"),
		Content:     contentBuilder.String(),
	})

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
	contentBuilder.WriteString(renderPostList(dated))
	contentBuilder.WriteString("<p><a href=\"/\">&larr; Home</a> &middot; <a href=\"/tags.html\">browse by tag &rarr;</a></p>")

	output := renderPage(template, pageMeta{
		Title:       "All posts",
		File:        "posts.html",
		Description: fmt.Sprintf("Every post on %s — %d of them, newest first.", siteName, len(dated)),
		Canonical:   canonicalURL("posts.html"),
		Content:     contentBuilder.String(),
	})

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

	output := renderPage(template, pageMeta{
		Title:       "404 — page not found",
		Heading:     "404",
		File:        "404.html",
		Description: "That page isn't here.",
		Canonical:   canonicalURL("404.html"),
		HeadExtra:   "<meta name=\"robots\" content=\"noindex\" />",
		Content:     contentBuilder.String(),
	})

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

// livereloadScript is injected into served HTML in watch mode so the browser
// refreshes automatically after each regeneration. It connects to the
// /__livereload SSE endpoint and reloads on any message.
const livereloadScript = `<script>(function(){var e=new EventSource("/__livereload");e.onmessage=function(){location.reload();};})();</script>`

// sseHub fans out a reload signal to every connected browser. Clients connect
// to /__livereload; when the watcher regenerates, notify() pushes a message
// that triggers each browser to reload.
type sseHub struct {
	mu      sync.Mutex
	clients map[chan struct{}]struct{}
}

func newSSEHub() *sseHub {
	return &sseHub{clients: make(map[chan struct{}]struct{})}
}

func (h *sseHub) subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *sseHub) unsubscribe(ch chan struct{}) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
}

// notify pushes a non-blocking reload signal to every connected client. A slow
// client that can't keep up is skipped rather than blocking the regenerator.
func (h *sseHub) notify() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// sseHandler keeps the connection open and writes a "reload" data event
// whenever the hub is notified. The connection is torn down when the client
// disconnects (request context cancelled).
func sseHandler(hub *sseHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		ch := hub.subscribe()
		defer hub.unsubscribe(ch)
		// Send a comment to flush headers so the browser sees a 200 right away.
		fmt.Fprintf(w, ": hello\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		for {
			select {
			case <-ch:
				fmt.Fprintf(w, "data: reload\n\n")
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			case <-r.Context().Done():
				return
			}
		}
	}
}

// devHandler serves files from buildDir using the same URL rules as GitHub
// Pages, so a link that works locally works in production and vice versa:
// an extensionless path resolves to `.html`, a directory to its `index.html`,
// and anything unresolvable to `404.html` with a 404 status. HTML responses get
// the livereload script appended so the browser auto-refreshes after a rebuild;
// everything else falls through to http.ServeFile. Path traversal is guarded so
// a crafted URL can't escape buildDir.
func devHandler(buildDir string) http.HandlerFunc {
	cleanBuildDir := filepath.Clean(buildDir)

	isFile := func(p string) bool {
		info, err := os.Stat(p)
		return err == nil && !info.IsDir()
	}

	// resolve maps a cleaned request path to a file on disk, or "" if nothing
	// matches. Mirrors tests/browser/serve.mjs, which mimics GitHub Pages.
	resolve := func(full string) string {
		if info, err := os.Stat(full); err == nil && info.IsDir() {
			if index := filepath.Join(full, "index.html"); isFile(index) {
				return index
			}
			return ""
		}
		if isFile(full) {
			return full
		}
		if filepath.Ext(full) == "" && isFile(full+".html") {
			return full + ".html"
		}
		return ""
	}

	// serveHTML writes an HTML file with the livereload script appended.
	serveHTML := func(w http.ResponseWriter, path string, status int) bool {
		data, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		w.Write(data)
		w.Write([]byte(livereloadScript))
		return true
	}

	notFound := func(w http.ResponseWriter, r *http.Request) {
		if serveHTML(w, filepath.Join(cleanBuildDir, "404.html"), http.StatusNotFound) {
			return
		}
		http.NotFound(w, r)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// r.URL.Path is already percent-decoded by net/http.
		rel := filepath.Clean(strings.TrimLeft(r.URL.Path, "/"))
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			notFound(w, r)
			return
		}
		full := resolve(filepath.Join(cleanBuildDir, rel))
		if full == "" {
			notFound(w, r)
			return
		}
		if strings.HasSuffix(full, ".html") {
			serveHTML(w, full, http.StatusOK)
			return
		}
		http.ServeFile(w, r, full)
	}
}

// snapshotFiles records the modtime of every file under the given roots (files
// or directory trees). The poller compares snapshots to detect changes.
func snapshotFiles(roots ...string) map[string]time.Time {
	m := make(map[string]time.Time)
	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			m[root] = info.ModTime()
			continue
		}
		_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			m[p] = info.ModTime()
			return nil
		})
	}
	return m
}

func modtimesEqual(a, b map[string]time.Time) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || !v.Equal(w) {
			return false
		}
	}
	return true
}

// runWatch generates the site once, then serves buildDir on port and
// regenerates whenever a file under contentDir or templatePath changes. The
// Goodreads shelves are fetched through cache with shelfTTL so repeated
// regenerations don't re-hit the feed. Connected browsers auto-refresh via SSE.
func runWatch(contentDir, buildDir, templatePath, userID, port string, cache *shelfCache) {
	if err := generateSite(contentDir, buildDir, templatePath, cache.get(userID, shelfTTL)); err != nil {
		log.Printf("initial generation error: %v", err)
	}

	hub := newSSEHub()
	mux := http.NewServeMux()
	mux.HandleFunc("/__livereload", sseHandler(hub))
	mux.HandleFunc("/", devHandler(buildDir))

	srv := &http.Server{Addr: ":" + port, Handler: mux}
	go func() {
		log.Printf("serving http://localhost:%s (watch mode — live reload on)", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	staticDir := filepath.Join(".", "static")
	watchTargets := []string{contentDir, templatePath, staticDir}
	prev := snapshotFiles(watchTargets...)
	ticker := time.NewTicker(800 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		cur := snapshotFiles(watchTargets...)
		if modtimesEqual(prev, cur) {
			continue
		}
		prev = cur
		log.Println("change detected, regenerating…")
		if err := generateSite(contentDir, buildDir, templatePath, cache.get(userID, shelfTTL)); err != nil {
			log.Printf("regeneration error: %v", err)
			continue
		}
		hub.notify()
		log.Println("reloaded")
	}
}

func main() {
	watch := flag.Bool("watch", false, "watch for file changes, serve, and live-reload")
	port := flag.String("port", "8080", "port for the dev server (watch mode)")
	flag.Parse()

	contentDir := filepath.Join(".", "content")
	buildDir := filepath.Join(".", "build")
	templatePath := filepath.Join(".", "template.html")

	// Pull the "What I'm reading" shelves from public Goodreads RSS at build
	// time. This is best-effort: any shelf that can't be fetched is logged and
	// skipped rather than failing the build. In watch mode the result is
	// cached for shelfTTL so repeated regenerations don't hammer the feed.
	userID := os.Getenv("GOODREADS_USER_ID")
	if userID == "" {
		userID = defaultGoodreadsUserID
	}
	cache := &shelfCache{}

	if *watch {
		runWatch(contentDir, buildDir, templatePath, userID, *port, cache)
		return
	}

	// ttl <= 0 always fetches, preserving the original one-shot behaviour.
	shelves := cache.get(userID, 0)
	if err := generateSite(contentDir, buildDir, templatePath, shelves); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Site generation complete!")
}
