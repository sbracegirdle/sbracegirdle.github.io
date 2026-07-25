package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/adrg/frontmatter"
)

// TestSlugifyTag covers the folding rules that make a tag safe for a filename
// and a URL.
func TestSlugifyTag(t *testing.T) {
	tests := []struct{ in, want string }{
		{"devops", "devops"},
		{"DevOps", "devops"},
		{"code_review", "code-review"},
		{"infrastructure-as-code", "infrastructure-as-code"},
		{"  aws  ", "aws"},
		{"C#", "c"},
		{"web 2.0", "web-2-0"},
		{"--messy--", "messy"},
		{"", ""},
		{"!!!", ""},
	}

	for _, tt := range tests {
		if got := slugifyTag(tt.in); got != tt.want {
			t.Errorf("slugifyTag(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestNormaliseTags checks the whole cleanup: splitting, folding, dropping
// duplicates and sorting.
func TestNormaliseTags(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "space separated scalar arrives as one field",
			in:   []string{"devops ci cd aws"},
			want: []string{"aws", "cd", "ci", "devops"},
		},
		{
			name: "carriage return does not create a phantom tag",
			in:   []string{"devops cdk\r"},
			want: []string{"cdk", "devops"},
		},
		{
			name: "duplicates collapse after folding",
			in:   []string{"AWS", "aws", "code_review", "code-review"},
			want: []string{"aws", "code-review"},
		},
		{
			name: "empty input stays empty",
			in:   nil,
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normaliseTags(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("normaliseTags(%v) = %v, want %v", tt.in, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("normaliseTags(%v) = %v, want %v", tt.in, got, tt.want)
					break
				}
			}
		})
	}
}

// TestTagListUnmarshal confirms both frontmatter spellings parse: the
// space-separated scalar every post in content/ uses, and a YAML sequence.
func TestTagListUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		want []string
	}{
		{
			name: "scalar",
			doc:  "---\ntitle: T\ntags: devops ci aws\n---\nbody",
			want: []string{"aws", "ci", "devops"},
		},
		{
			name: "sequence",
			doc:  "---\ntitle: T\ntags:\n  - devops\n  - ci\n  - aws\n---\nbody",
			want: []string{"aws", "ci", "devops"},
		},
		{
			name: "absent",
			doc:  "---\ntitle: T\n---\nbody",
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var meta FrontMatter
			if _, err := frontmatter.Parse(strings.NewReader(tt.doc), &meta); err != nil {
				t.Fatalf("parsing frontmatter: %v", err)
			}
			got := normaliseTags(meta.Tags)
			if len(got) != len(tt.want) {
				t.Fatalf("tags = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("tags = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// TestRenderTagChips checks the markup, the link target, and that an untagged
// post contributes nothing at all.
func TestRenderTagChips(t *testing.T) {
	if got := renderTagChips(nil); got != "" {
		t.Errorf("expected no markup for an untagged post, got %q", got)
	}

	got := renderTagChips([]string{"devops", "aws"})
	for _, want := range []string{
		`class="tag-list"`,
		`href="/tags/devops.html"`,
		`href="/tags/aws.html"`,
		`>devops<`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("tag chips missing %q, got %q", want, got)
		}
	}
}

// TestGroupByTag checks bucket ordering (count descending, then alphabetical)
// and that undated posts stay out of the listings.
func TestGroupByTag(t *testing.T) {
	d := func(s string) time.Time {
		parsed, _ := time.Parse("2006-01-02", s)
		return parsed
	}

	posts := []*BlogPost{
		{Title: "A", Date: d("2023-01-01"), OutputFile: "a.html", Tags: []string{"aws", "devops"}},
		{Title: "B", Date: d("2023-02-01"), OutputFile: "b.html", Tags: []string{"devops"}},
		{Title: "C", Date: d("2023-03-01"), OutputFile: "c.html", Tags: []string{"zzz", "devops"}},
		{Title: "About", OutputFile: "about.html", Tags: []string{"aws"}},
	}

	groups := groupByTag(posts)
	if len(groups) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(groups))
	}

	if groups[0].Tag != "devops" || len(groups[0].Posts) != 3 {
		t.Errorf("expected devops with 3 posts first, got %s with %d", groups[0].Tag, len(groups[0].Posts))
	}
	// aws and zzz both have one dated post, so they sort alphabetically.
	if groups[1].Tag != "aws" || groups[2].Tag != "zzz" {
		t.Errorf("expected aws then zzz, got %s then %s", groups[1].Tag, groups[2].Tag)
	}
	// The undated About page carries the aws tag but must not be listed.
	if len(groups[1].Posts) != 1 || groups[1].Posts[0].Title != "A" {
		t.Errorf("undated post should be excluded from tag listings, got %v", groups[1].Posts)
	}
	// Posts within a bucket run newest first.
	if groups[0].Posts[0].Title != "C" {
		t.Errorf("expected newest post first in bucket, got %s", groups[0].Posts[0].Title)
	}
}

// TestGenerateTagPages checks the files that land on disk and the paths
// reported back for the sitemap.
func TestGenerateTagPages(t *testing.T) {
	buildDir := t.TempDir()

	date, _ := time.Parse("2006-01-02", "2023-01-15")
	posts := []*BlogPost{
		{Title: "First Post", Date: date, OutputFile: "first.html", Tags: []string{"devops", "aws"}},
	}

	written, err := generateTagPages(posts, testTemplate, buildDir)
	if err != nil {
		t.Fatalf("generateTagPages: %v", err)
	}

	wantPaths := map[string]bool{"tags/devops.html": true, "tags/aws.html": true, "tags.html": true}
	if len(written) != len(wantPaths) {
		t.Fatalf("expected %d written paths, got %v", len(wantPaths), written)
	}
	for _, path := range written {
		if !wantPaths[path] {
			t.Errorf("unexpected written path %q", path)
		}
		if _, err := os.Stat(filepath.Join(buildDir, filepath.FromSlash(path))); err != nil {
			t.Errorf("reported path %q was not written: %v", path, err)
		}
	}

	tagPage, err := os.ReadFile(filepath.Join(buildDir, "tags", "devops.html"))
	if err != nil {
		t.Fatalf("reading tag page: %v", err)
	}
	// Links out of a tag page are root-absolute, so they resolve from /tags/.
	if !strings.Contains(string(tagPage), `href="/first.html"`) {
		t.Errorf("tag page should link posts root-absolutely, got %s", tagPage)
	}

	index, err := os.ReadFile(filepath.Join(buildDir, "tags.html"))
	if err != nil {
		t.Fatalf("reading tag index: %v", err)
	}
	if !strings.Contains(string(index), `class="tag-count"`) {
		t.Error("tag index should show post counts")
	}
}

// TestGenerateTagPagesUntagged confirms an untagged site writes no tag pages
// at all, rather than an empty index nothing links to.
func TestGenerateTagPagesUntagged(t *testing.T) {
	buildDir := t.TempDir()

	date, _ := time.Parse("2006-01-02", "2023-01-15")
	posts := []*BlogPost{{Title: "First Post", Date: date, OutputFile: "first.html"}}

	written, err := generateTagPages(posts, testTemplate, buildDir)
	if err != nil {
		t.Fatalf("generateTagPages: %v", err)
	}
	if len(written) != 0 {
		t.Errorf("expected no tag pages, got %v", written)
	}
	if _, err := os.Stat(filepath.Join(buildDir, "tags.html")); !os.IsNotExist(err) {
		t.Error("tags.html should not exist when nothing is tagged")
	}
}

// TestSlugifyTagCannotEscapeBuildDir asserts the property that matters rather
// than the folding rules that deliver it: a tag comes from post frontmatter and
// its slug is joined onto the build directory to make a filename, so no tag may
// produce a path that climbs out of build/tags/.
func TestSlugifyTagCannotEscapeBuildDir(t *testing.T) {
	hostile := []string{
		"../../etc/passwd",
		"/abs/path",
		"a/../b",
		"..",
		"....//....//",
		`..\..\windows`,
		"tags/../../secret",
		"日本語",
	}

	for _, in := range hostile {
		slug := slugifyTag(in)
		if strings.Contains(slug, "/") || strings.Contains(slug, `\`) || strings.Contains(slug, "..") {
			t.Errorf("slugifyTag(%q) = %q, which carries a path separator or a parent reference", in, slug)
		}
		if slug == "" {
			continue // an empty slug never reaches a filename
		}

		path := tagPagePath(slug)
		if filepath.Dir(path) != "tags" {
			t.Errorf("tagPagePath(slugifyTag(%q)) = %q, which lands outside tags/", in, path)
		}
		if cleaned := filepath.Clean(path); cleaned != path {
			t.Errorf("tagPagePath(slugifyTag(%q)) = %q, which is not already a clean path", in, path)
		}
	}
}

// TestPluralPosts covers the count suffix on generated tag-page descriptions.
// Every other tag test uses one post per tag, so the plural branch never ran.
func TestPluralPosts(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0 posts"},
		{1, "1 post"},
		{2, "2 posts"},
		{11, "11 posts"},
	}
	for _, tt := range tests {
		if got := pluralPosts(tt.n); got != tt.want {
			t.Errorf("pluralPosts(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
