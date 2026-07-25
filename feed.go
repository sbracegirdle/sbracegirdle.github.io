package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ── RSS ───────────────────────────────────────────────────────────────────
//
// A plain RSS 2.0 feed at /feed.xml, marshalled through encoding/xml so escaping
// is the standard library's problem rather than ours. Items carry the post
// summary rather than the full article: the rendered HTML contains root-relative
// links that would resolve against the reader's own host, and rewriting them all
// is more machinery than a summary feed is worth.

type rssDocument struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	AtomNS  string     `xml:"xmlns:atom,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	LastBuildDate string    `xml:"lastBuildDate,omitempty"`
	AtomLink      atomLink  `xml:"atom:link"`
	Items         []rssItem `xml:"item"`
}

// atomLink is the rel="self" pointer feed validators ask for.
type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type rssItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	GUID        rssGUID  `xml:"guid"`
	PubDate     string   `xml:"pubDate"`
	Description string   `xml:"description"`
	Categories  []string `xml:"category,omitempty"`
}

type rssGUID struct {
	IsPermaLink bool   `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

// buildFeed assembles the feed document from the dated posts, newest first.
// lastBuildDate tracks the newest post rather than the wall clock, so rebuilding
// an unchanged site produces an unchanged feed.
func buildFeed(posts []*BlogPost) rssDocument {
	dated := datedPostsNewestFirst(posts)

	items := make([]rssItem, 0, len(dated))
	for _, post := range dated {
		link := canonicalURL(post.OutputFile)
		items = append(items, rssItem{
			Title:       post.Title,
			Link:        link,
			GUID:        rssGUID{IsPermaLink: true, Value: link},
			PubDate:     post.Date.Format(time.RFC1123Z),
			Description: post.Description,
			Categories:  post.Tags,
		})
	}

	lastBuild := ""
	if len(dated) > 0 {
		lastBuild = dated[0].Date.Format(time.RFC1123Z)
	}

	return rssDocument{
		Version: "2.0",
		AtomNS:  "http://www.w3.org/2005/Atom",
		Channel: rssChannel{
			Title:         siteName,
			Link:          siteURL + "/",
			Description:   siteDescription,
			Language:      "en-au",
			LastBuildDate: lastBuild,
			AtomLink: atomLink{
				Href: canonicalURL("feed.xml"),
				Rel:  "self",
				Type: "application/rss+xml",
			},
			Items: items,
		},
	}
}

// generateFeed writes build/feed.xml. A site with no dated posts has nothing to
// syndicate, so no file is written rather than an empty feed.
func generateFeed(posts []*BlogPost, buildDir string) error {
	feed := buildFeed(posts)
	if len(feed.Channel.Items) == 0 {
		return nil
	}

	body, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling feed: %w", err)
	}

	outputPath := filepath.Join(buildDir, "feed.xml")
	if err := os.WriteFile(outputPath, append([]byte(xml.Header), body...), 0644); err != nil {
		return fmt.Errorf("writing feed: %w", err)
	}

	fmt.Printf("Generated feed: %s (%d items)\n", outputPath, len(feed.Channel.Items))
	return nil
}

// ── Sitemap ───────────────────────────────────────────────────────────────

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

// buildSitemap lists the site root, every generated page passed in, and every
// post. Posts carry a lastmod taken from their date; listing and standalone
// pages have no meaningful modification date to report, so they carry none
// rather than a date invented at build time.
func buildSitemap(posts []*BlogPost, pages []string) sitemapURLSet {
	set := sitemapURLSet{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9"}

	for _, page := range pages {
		// index.html canonicalises to the bare root, which is already the
		// first entry for every site with content.
		set.URLs = append(set.URLs, sitemapURL{Loc: canonicalURL(page)})
	}

	for _, post := range posts {
		entry := sitemapURL{Loc: canonicalURL(post.OutputFile)}
		if !post.Date.IsZero() {
			entry.LastMod = post.Date.Format("2006-01-02")
		}
		set.URLs = append(set.URLs, entry)
	}

	return set
}

// generateSitemap writes build/sitemap.xml.
func generateSitemap(posts []*BlogPost, pages []string, buildDir string) error {
	set := buildSitemap(posts, pages)
	if len(set.URLs) == 0 {
		return nil
	}

	body, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling sitemap: %w", err)
	}

	outputPath := filepath.Join(buildDir, "sitemap.xml")
	if err := os.WriteFile(outputPath, append([]byte(xml.Header), body...), 0644); err != nil {
		return fmt.Errorf("writing sitemap: %w", err)
	}

	fmt.Printf("Generated sitemap: %s (%d URLs)\n", outputPath, len(set.URLs))
	return nil
}

// ── robots.txt ────────────────────────────────────────────────────────────

// robotsBody allows everything and points crawlers at the sitemap. An empty
// Disallow is the canonical "nothing is off limits".
const robotsBody = `User-agent: *
Disallow:

Sitemap: ` + siteURL + `/sitemap.xml
`

// generateRobots writes build/robots.txt.
func generateRobots(buildDir string) error {
	outputPath := filepath.Join(buildDir, "robots.txt")
	if err := os.WriteFile(outputPath, []byte(robotsBody), 0644); err != nil {
		return fmt.Errorf("writing robots.txt: %w", err)
	}
	fmt.Printf("Generated: %s\n", outputPath)
	return nil
}
