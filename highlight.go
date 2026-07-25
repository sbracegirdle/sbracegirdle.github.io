package main

// A tiny dependency-free syntax highlighter. A single left-to-right scanner is
// driven by a per-language spec (comment markers, quote characters, keyword
// sets); anything it doesn't recognise stays unstyled, so unknown languages
// and odd constructs degrade to plain escaped text. Two renderers share the
// tokenizer: highlightCode emits flat spans for blog <pre><code> blocks, and
// highlightLines emits the line-numbered .cl/.ln/.cc rows used by the Rust
// quick reference (which previously built the same markup client-side in JS).

import (
	"fmt"
	"html"
	"io"
	"regexp"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	mdhtml "github.com/gomarkdown/markdown/html"
)

type tokenType string

const (
	tokPlain    tokenType = ""
	tokKeyword  tokenType = "kw"
	tokType     tokenType = "type"
	tokStr      tokenType = "str"
	tokNum      tokenType = "num"
	tokComment  tokenType = "comment"
	tokMacro    tokenType = "macro"
	tokLifetime tokenType = "lifetime"
	tokFn       tokenType = "fn"
	tokPrompt   tokenType = "prompt"
	tokFlag     tokenType = "flag"
)

type token struct {
	typ tokenType
	val string
}

// langSpec configures the scanner for one language. Zero-value fields switch
// the corresponding feature off, so most specs only set a handful.
type langSpec struct {
	lineComments []string  // markers running to end of line; "#" requires a word boundary
	blockComment [2]string // open/close pair; empty = none
	quotes       string    // string quote characters; ` may span lines, " and ' stop at EOL
	tripleQuotes bool      // python """…""" / '''…''' (multi-line)
	rawStrings   bool      // rust r"…" / r#"…"#
	byteStrings  bool      // rust b"…"
	lifetimes    bool      // rust 'a lifetimes and 'x' char literals
	macroBang    bool      // rust ident! → macro
	fnCalls      bool      // ident immediately before ( → fn
	capitalTypes bool      // Capitalised ident → type
	shellPrompt  bool      // leading "$ " → prompt
	shellFlags   bool      // -f / --flag after whitespace → flag
	yamlKeys     bool      // ident at line start followed by : → kw
	identExtra   string    // extra ident characters (yaml keys use "-.")
	keywords     map[string]bool
	types        map[string]bool
}

func wordSet(words string) map[string]bool {
	m := make(map[string]bool)
	for _, w := range strings.Fields(words) {
		m[w] = true
	}
	return m
}

var rustSpec = &langSpec{
	lineComments: []string{"//"},
	blockComment: [2]string{"/*", "*/"},
	quotes:       `"`,
	rawStrings:   true,
	byteStrings:  true,
	lifetimes:    true,
	macroBang:    true,
	fnCalls:      true,
	capitalTypes: true,
	keywords: wordSet(`as async await break const continue crate dyn else enum
		extern false fn for if impl in let loop match mod move mut pub ref
		return self Self static struct super trait true type unsafe use where
		while union box macro_rules yield try`),
	types: wordSet(`i8 i16 i32 i64 i128 isize u8 u16 u32 u64 u128 usize f32 f64
		bool char str`),
}

var goSpec = &langSpec{
	lineComments: []string{"//"},
	blockComment: [2]string{"/*", "*/"},
	quotes:       "\"'`",
	fnCalls:      true,
	capitalTypes: true,
	keywords: wordSet(`break case chan const continue default defer else
		fallthrough for func go goto if import interface map package range
		return select struct switch type var nil true false iota`),
	types: wordSet(`any bool byte comparable complex64 complex128 error float32
		float64 int int8 int16 int32 int64 rune string uint uint8 uint16
		uint32 uint64 uintptr`),
}

var pythonSpec = &langSpec{
	lineComments: []string{"#"},
	quotes:       `"'`,
	tripleQuotes: true,
	fnCalls:      true,
	capitalTypes: true,
	keywords: wordSet(`False None True and as assert async await break class
		continue def del elif else except finally for from global if import in
		is lambda match case nonlocal not or pass raise return try while with
		yield`),
}

var shellSpec = &langSpec{
	lineComments: []string{"#"},
	quotes:       `"'`,
	shellPrompt:  true,
	shellFlags:   true,
	keywords: wordSet(`if then else elif fi for while until do done case esac
		function in local export return exit set shift trap`),
}

var yamlSpec = &langSpec{
	lineComments: []string{"#"},
	quotes:       `"'`,
	yamlKeys:     true,
	identExtra:   "-._",
	keywords:     wordSet(`true false null yes no on off`),
}

var jsSpec = &langSpec{
	lineComments: []string{"//"},
	blockComment: [2]string{"/*", "*/"},
	quotes:       "\"'`",
	fnCalls:      true,
	capitalTypes: true,
	keywords: wordSet(`async await break case catch class const continue
		debugger default delete do else export extends finally for from
		function if import in instanceof let new of return static super switch
		this throw try typeof var void while with yield true false null
		undefined`),
}

// langSpecs maps fence labels (and common aliases) to specs. Unlisted
// languages fall back to plain escaping.
var langSpecs = map[string]*langSpec{
	"rust": rustSpec, "rs": rustSpec,
	"go": goSpec, "golang": goSpec,
	"python": pythonSpec, "py": pythonSpec,
	"sh": shellSpec, "bash": shellSpec, "shell": shellSpec, "zsh": shellSpec, "console": shellSpec,
	"yaml": yamlSpec, "yml": yamlSpec,
	"js": jsSpec, "jsx": jsSpec, "javascript": jsSpec, "ts": jsSpec, "tsx": jsSpec, "typescript": jsSpec,
}

func specFor(lang string) *langSpec {
	return langSpecs[strings.ToLower(strings.TrimSpace(lang))]
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentStart(c byte) bool { return isLetter(c) || c == '_' }

func isIdentChar(c byte) bool { return isIdentStart(c) || (c >= '0' && c <= '9') }

// isNumChar is deliberately loose: it accepts hex digits, separators, and the
// suffix/exponent letters that appear in numeric literals (0xFF, 1_000, 2.5e3,
// 5u32). Over-matching inside a number is harmless for display purposes.
func isNumChar(c byte) bool {
	switch {
	case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		return true
	}
	return strings.IndexByte("_.xXoui", c) >= 0
}

func isSpaceByte(c byte) bool { return c == ' ' || c == '\t' || c == '\n' }

// tokenize scans src into a flat token stream. Strings and comments are
// consumed before anything else, so a '#' inside a Python string never becomes
// a comment and keywords inside strings stay strings.
func tokenize(src string, spec *langSpec) []token {
	var out []token
	push := func(t tokenType, v string) {
		if v == "" {
			return
		}
		if t == tokPlain && len(out) > 0 && out[len(out)-1].typ == tokPlain {
			out[len(out)-1].val += v
			return
		}
		out = append(out, token{t, v})
	}

	i, n := 0, len(src)
	lineStart := true // nothing but whitespace (or yaml list dashes) so far on this line
	for i < n {
		c := src[i]

		if c == '\n' {
			push(tokPlain, "\n")
			i++
			lineStart = true
			continue
		}
		if c == ' ' || c == '\t' {
			j := i
			for j < n && (src[j] == ' ' || src[j] == '\t') {
				j++
			}
			push(tokPlain, src[i:j])
			i = j
			continue
		}

		if spec.blockComment[0] != "" && strings.HasPrefix(src[i:], spec.blockComment[0]) {
			j := n
			if end := strings.Index(src[i+len(spec.blockComment[0]):], spec.blockComment[1]); end >= 0 {
				j = i + len(spec.blockComment[0]) + end + len(spec.blockComment[1])
			}
			push(tokComment, src[i:j])
			i = j
			lineStart = false
			continue
		}

		lineComment := ""
		for _, m := range spec.lineComments {
			if !strings.HasPrefix(src[i:], m) {
				continue
			}
			// '#' only opens a comment at a word boundary, so ${#var} and
			// URL fragments stay plain.
			if m == "#" && i > 0 && !isSpaceByte(src[i-1]) {
				continue
			}
			lineComment = m
			break
		}
		if lineComment != "" {
			j := i
			for j < n && src[j] != '\n' {
				j++
			}
			push(tokComment, src[i:j])
			i = j
			lineStart = false
			continue
		}

		if spec.rawStrings && c == 'r' && i+1 < n && (src[i+1] == '"' || src[i+1] == '#') {
			j, hashes := i+1, 0
			for j < n && src[j] == '#' {
				hashes++
				j++
			}
			if j < n && src[j] == '"' {
				closer := `"` + strings.Repeat("#", hashes)
				k := n
				if end := strings.Index(src[j+1:], closer); end >= 0 {
					k = j + 1 + end + len(closer)
				}
				push(tokStr, src[i:k])
				i = k
				lineStart = false
				continue
			}
		}

		if spec.byteStrings && c == 'b' && i+1 < n && src[i+1] == '"' {
			j := i + 2
			for j < n && src[j] != '"' {
				if src[j] == '\\' && j+1 < n {
					j++
				}
				j++
			}
			if j < n {
				j++
			}
			push(tokStr, src[i:j])
			i = j
			lineStart = false
			continue
		}

		if spec.tripleQuotes && (strings.HasPrefix(src[i:], `"""`) || strings.HasPrefix(src[i:], "'''")) {
			q := src[i : i+3]
			j := n
			if end := strings.Index(src[i+3:], q); end >= 0 {
				j = i + 3 + end + 3
			}
			push(tokStr, src[i:j])
			i = j
			lineStart = false
			continue
		}

		if strings.IndexByte(spec.quotes, c) >= 0 {
			j := i + 1
			for j < n {
				if src[j] == c {
					j++
					break
				}
				// " and ' don't span lines; an unterminated string ends at EOL.
				if src[j] == '\n' && c != '`' {
					break
				}
				if src[j] == '\\' && c != '`' && j+1 < n {
					j++
				}
				j++
			}
			push(tokStr, src[i:j])
			i = j
			lineStart = false
			continue
		}

		if spec.lifetimes && c == '\'' {
			j := i + 1
			if j < n && src[j] == '\\' { // escaped char literal: '\n'
				j++
				for j < n && src[j] != '\'' {
					j++
				}
				if j < n {
					j++
				}
				push(tokNum, src[i:j])
				i = j
				lineStart = false
				continue
			}
			if j+1 < n && src[j+1] == '\'' { // char literal: 'x'
				push(tokNum, src[i:j+2])
				i = j + 2
				lineStart = false
				continue
			}
			k := j
			for k < n && isIdentChar(src[k]) {
				k++
			}
			if k > j { // lifetime or loop label: 'a
				push(tokLifetime, src[i:k])
				i = k
				lineStart = false
				continue
			}
			push(tokPlain, "'")
			i++
			lineStart = false
			continue
		}

		if spec.shellPrompt && lineStart && c == '$' && i+1 < n && src[i+1] == ' ' {
			push(tokPrompt, "$")
			i++
			lineStart = false
			continue
		}

		if spec.shellFlags && c == '-' && (i == 0 || isSpaceByte(src[i-1])) {
			j := i + 1
			if j < n && src[j] == '-' {
				j++
			}
			if j < n && isLetter(src[j]) {
				for j < n && (isIdentChar(src[j]) || src[j] == '-') {
					j++
				}
				push(tokFlag, src[i:j])
				i = j
				lineStart = false
				continue
			}
		}

		// A leading "- " keeps yaml list items in key position: `- name: x`.
		if spec.yamlKeys && lineStart && c == '-' && (i+1 >= n || src[i+1] == ' ') {
			push(tokPlain, "-")
			i++
			continue
		}

		if c >= '0' && c <= '9' {
			j := i
			for j < n && isNumChar(src[j]) {
				j++
			}
			push(tokNum, src[i:j])
			i = j
			lineStart = false
			continue
		}

		if isIdentStart(c) {
			atLineStart := lineStart
			j := i
			for j < n && (isIdentChar(src[j]) || strings.IndexByte(spec.identExtra, src[j]) >= 0) {
				j++
			}
			word := src[i:j]
			i = j
			lineStart = false
			switch {
			case spec.yamlKeys && atLineStart && j < n && src[j] == ':':
				push(tokKeyword, word)
			case spec.macroBang && j < n && src[j] == '!' && (j+1 >= n || src[j+1] != '='):
				push(tokMacro, word)
			case spec.keywords[word]:
				push(tokKeyword, word)
			case spec.types[word], spec.capitalTypes && word[0] >= 'A' && word[0] <= 'Z':
				push(tokType, word)
			case spec.fnCalls && j < n && src[j] == '(':
				push(tokFn, word)
			default:
				push(tokPlain, word)
			}
			continue
		}

		push(tokPlain, string(c))
		i++
		lineStart = false
	}
	return out
}

func renderTokens(b *strings.Builder, toks []token) {
	for _, tk := range toks {
		if tk.typ == tokPlain {
			b.WriteString(html.EscapeString(tk.val))
			continue
		}
		b.WriteString(`<span class="t-`)
		b.WriteString(string(tk.typ))
		b.WriteString(`">`)
		b.WriteString(html.EscapeString(tk.val))
		b.WriteString(`</span>`)
	}
}

// highlightCode returns code as escaped HTML with token spans, ready to sit
// inside <pre><code>. Unknown languages are escaped with no spans.
func highlightCode(code, lang string) string {
	spec := specFor(lang)
	if spec == nil {
		return html.EscapeString(code)
	}
	var b strings.Builder
	renderTokens(&b, tokenize(code, spec))
	return b.String()
}

// highlightLines renders code as the site's line-numbered rows: one .cl per
// line holding a .ln gutter number and the highlighted .cc content. The
// gutter width fits the largest line number (minimum 2).
func highlightLines(code, lang string) string {
	code = strings.Trim(code, "\n")
	spec := specFor(lang)
	toks := []token{{tokPlain, code}}
	if spec != nil {
		toks = tokenize(code, spec)
	}

	lines := [][]token{{}}
	for _, tk := range toks {
		for k, part := range strings.Split(tk.val, "\n") {
			if k > 0 {
				lines = append(lines, nil)
			}
			if part != "" {
				lines[len(lines)-1] = append(lines[len(lines)-1], token{tk.typ, part})
			}
		}
	}

	width := len(fmt.Sprint(len(lines)))
	if width < 2 {
		width = 2
	}
	var b strings.Builder
	for idx, ln := range lines {
		var body strings.Builder
		renderTokens(&body, ln)
		content := body.String()
		if content == "" {
			content = " "
		}
		fmt.Fprintf(&b, `<div class="cl"><span class="ln">%*d</span><span class="cc">%s</span></div>`, width, idx+1, content)
	}
	return b.String()
}

// renderCodeBlock renders any code block — fenced in a post or sourced from a
// static page — as the site's single code component: a line-numbered
// pre.code, with a language-<lang> class when the language is known.
func renderCodeBlock(code, lang string) string {
	class := "code"
	if lang != "" {
		class += " language-" + html.EscapeString(lang)
	}
	return `<pre class="` + class + `">` + highlightLines(code, lang) + "</pre>\n"
}

// codeBlockHook intercepts fenced/indented code blocks during markdown
// rendering and substitutes highlighted output for the default escaping.
func codeBlockHook(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	cb, ok := node.(*ast.CodeBlock)
	if !ok {
		return ast.GoToNext, false
	}
	lang := ""
	if fields := strings.Fields(string(cb.Info)); len(fields) > 0 {
		lang = fields[0]
	}
	io.WriteString(w, renderCodeBlock(string(cb.Literal), lang))
	return ast.GoToNext, true
}

// renderMarkdown converts post markdown to HTML with highlighted code blocks.
// Parser and flags match gomarkdown's defaults; only code rendering differs.
func renderMarkdown(content []byte) []byte {
	renderer := mdhtml.NewRenderer(mdhtml.RendererOptions{
		Flags:          mdhtml.CommonFlags,
		RenderNodeHook: codeBlockHook,
	})
	return markdown.ToHTML(content, nil, renderer)
}

var codeScriptRe = regexp.MustCompile(`(?s)<script type="text/(rust|shell)">(.*?)</script>`)

// renderStaticCodeScripts pre-renders <script type="text/rust|shell"> source
// blocks in static HTML into the same pre.code component posts get, at build
// time. Pages without such blocks pass through untouched.
func renderStaticCodeScripts(page string) string {
	return codeScriptRe.ReplaceAllStringFunc(page, func(m string) string {
		sub := codeScriptRe.FindStringSubmatch(m)
		lang := sub[1]
		if lang == "shell" {
			lang = "sh"
		}
		return renderCodeBlock(sub[2], lang)
	})
}
