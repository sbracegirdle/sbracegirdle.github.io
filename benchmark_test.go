package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkProcessMarkdownFile benchmarks the markdown processing function
func BenchmarkProcessMarkdownFile(b *testing.B) {
	// Setup
	tempDir, err := os.MkdirTemp("", "ssg-benchmark-*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file
	filePath := filepath.Join(tempDir, "benchmark.md")
	err = os.WriteFile(filePath, []byte(testMarkdownWithFrontmatter), 0644)
	if err != nil {
		b.Fatalf("Failed to write test file: %v", err)
	}

	// Run benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, err := processMarkdownFile(filePath, testTemplate)
		if err != nil {
			b.Fatalf("Error processing markdown: %v", err)
		}
	}
}

// BenchmarkGenerateSite benchmarks the entire site generation process
func BenchmarkGenerateSite(b *testing.B) {
	// Setup
	tempDir, err := os.MkdirTemp("", "ssg-benchmark-*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create content directory
	contentDir := filepath.Join(tempDir, "content")
	if err := os.Mkdir(contentDir, 0755); err != nil {
		b.Fatalf("Failed to create content directory: %v", err)
	}

	// Create build directory
	buildDir := filepath.Join(tempDir, "build")
	if err := os.Mkdir(buildDir, 0755); err != nil {
		b.Fatalf("Failed to create build directory: %v", err)
	}

	// Create template file
	templatePath := filepath.Join(tempDir, "template.html")
	if err := os.WriteFile(templatePath, []byte(testTemplate), 0644); err != nil {
		b.Fatalf("Failed to write template file: %v", err)
	}

	// Create test files - multiple to simulate a real site
	for i := 1; i <= 10; i++ {
		filename := filepath.Join(contentDir, fmt.Sprintf("test-%d.md", i))
		content := testMarkdownWithFrontmatter
		if i%2 == 0 {
			content = testMarkdownWithoutFrontmatter
		}
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			b.Fatalf("Failed to write markdown file: %v", err)
		}
	}

	// Run benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := generateSite(contentDir, buildDir, templatePath)
		if err != nil {
			b.Fatalf("Error generating site: %v", err)
		}
		// Clean build dir between runs
		os.RemoveAll(buildDir)
		os.Mkdir(buildDir, 0755)
	}
}
