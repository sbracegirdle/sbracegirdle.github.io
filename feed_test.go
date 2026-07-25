package main

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testDate(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("bad test date %q: %v", s, err)
	}
	return parsed
}

func feedTestPosts(t *testing.T) []*BlogPost {
	t.Helper()
	return []*BlogPost{
		{
			Title:       "First Post",
			Date:        testDate(t, "2023-01-15"),
			OutputFile:  "2023-01-15-first-post.html",
			Description: "The first one.",
			Tags:        []string{"aws", "devops"},
		},
		{
			Title:       "Second Post",
			Date:        testDate(t, "2023-03-20"),
			OutputFile:  "2023-03-20-second-post.html",
			Description: "The second one.",
		},
		// Undated pages are not articles and have no place in a feed.
		{Title: "About me", OutputFile: "about.html", Description: "A mini CV."},
	}
}

// TestBuildFeed checks ordering, canonical links, and the channel metadata.
func TestBuildFeed(t *testing.T) {
	feed := buildFeed(feedTestPosts(t))

	if len(feed.Channel.Items) != 2 {
		t.Fatalf("expected 2 items (undated pages excluded), got %d", len(feed.Channel.Items))
	}

	// Newest first.
	if feed.Channel.Items[0].Title != "Second Post" {
		t.Errorf("expected newest post first, got %q", feed.Channel.Items[0].Title)
	}

	item := feed.Channel.Items[1]
	wantLink := "https://letsbuild.cloud/2023-01-15-first-post.html"
	if item.Link != wantLink {
		t.Errorf("item link = %q, want %q", item.Link, wantLink)
	}
	if item.GUID.Value != wantLink || !item.GUID.IsPermaLink {
		t.Errorf("guid = %+v, want permalink %q", item.GUID, wantLink)
	}
	if _, err := time.Parse(time.RFC1123Z, item.PubDate); err != nil {
		t.Errorf("pubDate %q is not RFC1123Z: %v", item.PubDate, err)
	}
	if len(item.Categories) != 2 || item.Categories[0] != "aws" {
		t.Errorf("categories = %v, want the post's tags", item.Categories)
	}

	// lastBuildDate tracks the newest post, so an unchanged site rebuilds to an
	// identical feed.
	if feed.Channel.LastBuildDate != feed.Channel.Items[0].PubDate {
		t.Errorf("lastBuildDate = %q, want newest pubDate %q",
			feed.Channel.LastBuildDate, feed.Channel.Items[0].PubDate)
	}
	if feed.Channel.AtomLink.Href != "https://letsbuild.cloud/feed.xml" {
		t.Errorf("atom self link = %q", feed.Channel.AtomLink.Href)
	}
}

// TestGenerateFeed writes the file and reads it back through a parser, which is
// the check that matters: titles carrying markup-significant characters have to
// survive the round trip intact.
func TestGenerateFeed(t *testing.T) {
	buildDir := t.TempDir()

	posts := []*BlogPost{{
		Title:       `Ampersands & "quotes" <tags>`,
		Date:        testDate(t, "2023-01-15"),
		OutputFile:  "post.html",
		Description: `He said "LGTM" & meant it`,
	}}

	if err := generateFeed(posts, buildDir); err != nil {
		t.Fatalf("generateFeed: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(buildDir, "feed.xml"))
	if err != nil {
		t.Fatalf("reading feed: %v", err)
	}
	if !strings.HasPrefix(string(raw), xml.Header) {
		t.Error("feed should start with an XML declaration")
	}

	var parsed struct {
		Channel struct {
			Title string `xml:"title"`
			Items []struct {
				Title       string `xml:"title"`
				Description string `xml:"description"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("generated feed does not parse: %v", err)
	}
	if len(parsed.Channel.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(parsed.Channel.Items))
	}
	if parsed.Channel.Items[0].Title != posts[0].Title {
		t.Errorf("title round-tripped as %q, want %q", parsed.Channel.Items[0].Title, posts[0].Title)
	}
	if parsed.Channel.Items[0].Description != posts[0].Description {
		t.Errorf("description round-tripped as %q, want %q",
			parsed.Channel.Items[0].Description, posts[0].Description)
	}
}

// TestGenerateFeedNoPosts confirms an empty site writes no feed rather than an
// empty one.
func TestGenerateFeedNoPosts(t *testing.T) {
	buildDir := t.TempDir()

	if err := generateFeed([]*BlogPost{{Title: "About", OutputFile: "about.html"}}, buildDir); err != nil {
		t.Fatalf("generateFeed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(buildDir, "feed.xml")); !os.IsNotExist(err) {
		t.Error("feed.xml should not be written when there is nothing to syndicate")
	}
}

// TestBuildSitemap checks that pages and posts are both listed, and that only
// dated posts claim a lastmod.
func TestBuildSitemap(t *testing.T) {
	set := buildSitemap(feedTestPosts(t), []string{"index.html", "posts.html", "tags/aws.html"})

	if set.Xmlns != "http://www.sitemaps.org/schemas/sitemap/0.9" {
		t.Errorf("unexpected namespace %q", set.Xmlns)
	}
	if len(set.URLs) != 6 {
		t.Fatalf("expected 3 pages + 3 posts, got %d", len(set.URLs))
	}

	byLoc := make(map[string]sitemapURL, len(set.URLs))
	for _, u := range set.URLs {
		byLoc[u.Loc] = u
	}

	// index.html canonicalises to the bare root.
	if _, ok := byLoc["https://letsbuild.cloud/"]; !ok {
		t.Error("sitemap should list the site root")
	}
	if u, ok := byLoc["https://letsbuild.cloud/tags/aws.html"]; !ok {
		t.Error("sitemap should list tag pages")
	} else if u.LastMod != "" {
		t.Errorf("listing pages should not claim a lastmod, got %q", u.LastMod)
	}
	if u := byLoc["https://letsbuild.cloud/2023-01-15-first-post.html"]; u.LastMod != "2023-01-15" {
		t.Errorf("post lastmod = %q, want 2023-01-15", u.LastMod)
	}
	if u := byLoc["https://letsbuild.cloud/about.html"]; u.LastMod != "" {
		t.Errorf("undated page should have no lastmod, got %q", u.LastMod)
	}
}

// TestGenerateSitemap confirms the written document parses.
func TestGenerateSitemap(t *testing.T) {
	buildDir := t.TempDir()

	if err := generateSitemap(feedTestPosts(t), []string{"index.html"}, buildDir); err != nil {
		t.Fatalf("generateSitemap: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(buildDir, "sitemap.xml"))
	if err != nil {
		t.Fatalf("reading sitemap: %v", err)
	}

	var parsed sitemapURLSet
	if err := xml.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("generated sitemap does not parse: %v", err)
	}
	if len(parsed.URLs) != 4 {
		t.Errorf("expected 4 URLs, got %d", len(parsed.URLs))
	}
}

// TestGenerateRobots checks the crawler directives and the sitemap pointer.
func TestGenerateRobots(t *testing.T) {
	buildDir := t.TempDir()

	if err := generateRobots(buildDir); err != nil {
		t.Fatalf("generateRobots: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(buildDir, "robots.txt"))
	if err != nil {
		t.Fatalf("reading robots.txt: %v", err)
	}
	body := string(raw)
	for _, want := range []string{"User-agent: *", "Disallow:", "Sitemap: https://letsbuild.cloud/sitemap.xml"} {
		if !strings.Contains(body, want) {
			t.Errorf("robots.txt missing %q, got:\n%s", want, body)
		}
	}
}
