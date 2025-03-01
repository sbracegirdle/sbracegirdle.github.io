---
title: The world's simplest Static Site Generator in Go
description: A look at how this blog is generated using a custom-built static site generator written in Go.
---

I was bored on a Saturday afternoon, and thought that my personal site could do with some simplification. In the past I have used static site generators such as Astro and Jekyl. But these were surplus to needs, and instead I thought it's time to do the Software Engineer thing and re-invent the wheel.

All I needed was a script that traversed a directory of markdown files, converted them to html, and then inserted them into a simple HTML template. It turns out that this is very simple code to write, and can be done in a few hundred lines of Golang, or JS, or `<pick your poison>`.

In my case, the `processMarkdownFile` is the core of this, converting a file of markdown into html and injecting it into a template html file:

```go
// processMarkdownFile processes a single markdown file and returns the generated HTML
func processMarkdownFile(filePath, template string) (string, string, *BlogPost, error) {
    // Parse frontmatter
    var meta FrontMatter
    content, err := frontmatter.Parse(strings.NewReader(string(fileContent)), &meta)
    
    // Parse markdown to HTML
    htmlContent := markdown.ToHTML(content, nil, nil)

    // Replace template placeholders
    output := strings.Replace(template, "{{title}}", title, -1)
    output = strings.Replace(output, "{{content}}", string(htmlContent), -1)
    
    // ...rest of the function...
}
```

The other need was for an index page to list all my posts. This can be done by sorting and slicing them in date descending order, which I extracted from the standardised file name format (`[yyyy-mm-dd]-[title-bits].md`):

```go
// Sort posts by date in descending order (newest first)
sort.Slice(posts, func(i, j int) bool {
    return posts[i].Date.After(posts[j].Date)
})
```

Generating the post list:

```go
for _, post := range posts {
    if post.Date.IsZero() {
        continue // Skip posts without dates
    }

    formattedDate := post.Date.Format("January 2, 2006")
    contentBuilder.WriteString(fmt.Sprintf("<li><strong>%s</strong> - <a href=\"%s\">%s</a><p>%s</p></li>\n",
        formattedDate, post.OutputFile, post.Title, post.Description))
}
```

Supporting frontmatter is handy for metadata, but not absolutely essential:

```markdown
---
title: My Custom Page Title
description: A brief description of the page content
---
```

Parsed with:

```go
content, err := frontmatter.Parse(strings.NewReader(string(fileContent)), &meta)
```

[For more details, checkout the README and source code in the repo.](https://github.com/sbracegirdle/sbracegirdle.github.io/)
