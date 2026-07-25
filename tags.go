package main

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// tagList holds the `tags` frontmatter value. Posts in this repo write tags as
// a single space-separated scalar ("devops ci cd aws"), but a YAML sequence is
// the other obvious spelling, so both are accepted.
type tagList []string

// UnmarshalYAML implements the yaml.v2 unmarshaler so a scalar and a sequence
// both land in the same slice. Anything else is ignored rather than failing the
// build — a malformed tag line shouldn't cost you the whole post.
func (t *tagList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var seq []string
	if err := unmarshal(&seq); err == nil {
		*t = seq
		return nil
	}

	var scalar string
	if err := unmarshal(&scalar); err == nil {
		*t = strings.Fields(scalar)
		return nil
	}

	*t = nil
	return nil
}

// normaliseTags cleans raw frontmatter tags into the canonical form used for
// both display and URLs: lowercase, punctuation folded to hyphens, deduplicated
// and sorted. Folding means "code_review" and "code review" arrive at the same
// tag, and stray whitespace (a CRLF line ending, say) can't split one tag into
// two.
func normaliseTags(raw []string) []string {
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		// A scalar that reached us unsplit (e.g. from a YAML sequence entry
		// holding several words) still splits on whitespace here.
		for _, field := range strings.Fields(r) {
			tag := slugifyTag(field)
			if tag == "" {
				continue
			}
			if _, dup := seen[tag]; dup {
				continue
			}
			seen[tag] = struct{}{}
			out = append(out, tag)
		}
	}
	sort.Strings(out)
	return out
}

// slugifyTag reduces one tag to lowercase alphanumerics separated by single
// hyphens, safe to use verbatim in a filename and a URL.
func slugifyTag(tag string) string {
	var b strings.Builder
	lastHyphen := true // leading hyphens are trimmed by starting "in" one
	for _, r := range strings.ToLower(strings.TrimSpace(tag)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastHyphen = false
		case !lastHyphen:
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

// tagPagePath is the build-relative path of the listing page for one tag.
func tagPagePath(tag string) string {
	return "tags/" + tag + ".html"
}

// renderTagChips renders a post's tags as linked chips. It returns an empty
// string for an untagged post so nothing but whitespace is added to the page.
func renderTagChips(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<nav class=\"tag-list\" aria-label=\"Tags\">")
	for _, tag := range tags {
		b.WriteString(fmt.Sprintf("<a class=\"tag\" href=\"/%s\">%s</a>",
			html.EscapeString(tagPagePath(tag)), html.EscapeString(tag)))
	}
	b.WriteString("</nav>")
	return b.String()
}

// tagCount pairs a tag with the posts carrying it, for the tag index.
type tagCount struct {
	Tag   string
	Posts []*BlogPost
}

// groupByTag buckets dated posts by tag, newest first within each bucket. The
// result is ordered by post count descending, then alphabetically, so the tags
// that carry the most writing lead the index.
func groupByTag(posts []*BlogPost) []tagCount {
	byTag := make(map[string][]*BlogPost)
	for _, post := range datedPostsNewestFirst(posts) {
		for _, tag := range post.Tags {
			byTag[tag] = append(byTag[tag], post)
		}
	}

	groups := make([]tagCount, 0, len(byTag))
	for tag, tagged := range byTag {
		groups = append(groups, tagCount{Tag: tag, Posts: tagged})
	}
	sort.Slice(groups, func(i, j int) bool {
		if len(groups[i].Posts) != len(groups[j].Posts) {
			return len(groups[i].Posts) > len(groups[j].Posts)
		}
		return groups[i].Tag < groups[j].Tag
	})
	return groups
}

// generateTagPages writes one listing page per tag under build/tags/, plus the
// tags.html index that links them all. It returns the build-relative paths of
// every page written, for the sitemap.
func generateTagPages(posts []*BlogPost, template, buildDir string) ([]string, error) {
	groups := groupByTag(posts)
	if len(groups) == 0 {
		return nil, nil
	}

	tagsDir := filepath.Join(buildDir, "tags")
	if err := os.MkdirAll(tagsDir, 0755); err != nil {
		return nil, fmt.Errorf("creating tags directory: %w", err)
	}

	written := make([]string, 0, len(groups)+1)
	for _, group := range groups {
		var body strings.Builder
		body.WriteString(renderPostList(group.Posts))
		body.WriteString("<p><a href=\"/tags.html\">&larr; All tags</a></p>")

		outputPath := tagPagePath(group.Tag)
		page := renderPage(template, pageMeta{
			Title:   "Tagged: " + group.Tag,
			File:    filepath.Base(outputPath),
			Content: body.String(),
			Description: fmt.Sprintf("%s tagged %q on %s.",
				pluralPosts(len(group.Posts)), group.Tag, siteName),
			Canonical: canonicalURL(outputPath),
		})

		if err := os.WriteFile(filepath.Join(buildDir, filepath.FromSlash(outputPath)), []byte(page), 0644); err != nil {
			return written, fmt.Errorf("writing tag page %s: %w", outputPath, err)
		}
		written = append(written, outputPath)
	}

	if err := generateTagIndex(groups, template, buildDir); err != nil {
		return written, err
	}
	written = append(written, "tags.html")

	fmt.Printf("Generated %d tag pages and %s\n", len(groups), filepath.Join(buildDir, "tags.html"))
	return written, nil
}

// generateTagIndex writes tags.html: every tag as a chip carrying its post
// count, most-used first.
func generateTagIndex(groups []tagCount, template, buildDir string) error {
	var body strings.Builder
	body.WriteString("<p>Every tag across the archive, most-used first.</p>")
	body.WriteString("<nav class=\"tag-cloud\" aria-label=\"All tags\">")
	for _, group := range groups {
		body.WriteString(fmt.Sprintf("<a class=\"tag\" href=\"/%s\">%s<span class=\"tag-count\">%d</span></a>",
			html.EscapeString(tagPagePath(group.Tag)), html.EscapeString(group.Tag), len(group.Posts)))
	}
	body.WriteString("</nav>")
	body.WriteString("<p><a href=\"/posts.html\">&larr; All posts</a></p>")

	page := renderPage(template, pageMeta{
		Title:       "Tags",
		File:        "tags.html",
		Description: fmt.Sprintf("Browse %s by topic — %d tags across the archive.", siteName, len(groups)),
		Canonical:   canonicalURL("tags.html"),
		Content:     body.String(),
	})

	outputPath := filepath.Join(buildDir, "tags.html")
	if err := os.WriteFile(outputPath, []byte(page), 0644); err != nil {
		return fmt.Errorf("writing tag index: %w", err)
	}
	return nil
}

// pluralPosts renders a post count with the right noun, for descriptions.
func pluralPosts(n int) string {
	if n == 1 {
		return "1 post"
	}
	return fmt.Sprintf("%d posts", n)
}
