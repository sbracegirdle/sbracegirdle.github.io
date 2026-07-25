package main

import (
	"strings"
	"testing"
)

// contains asserts every want fragment appears in got.
func assertContains(t *testing.T, got string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("output missing %q\nfull output:\n%s", w, got)
		}
	}
}

func assertNotContains(t *testing.T, got string, avoids ...string) {
	t.Helper()
	for _, a := range avoids {
		if strings.Contains(got, a) {
			t.Errorf("output should not contain %q\nfull output:\n%s", a, got)
		}
	}
}

func TestHighlightUnknownLangEscapesOnly(t *testing.T) {
	got := highlightCode("<b>&x</b>", "cmake")
	if got != "&lt;b&gt;&amp;x&lt;/b&gt;" {
		t.Errorf("unknown lang should be plain-escaped, got %q", got)
	}
	if strings.Contains(got, "<span") {
		t.Errorf("unknown lang should have no spans, got %q", got)
	}
}

func TestHighlightEmptyLangEscapesOnly(t *testing.T) {
	got := highlightCode("a < b && c > d", "")
	if got != "a &lt; b &amp;&amp; c &gt; d" {
		t.Errorf("plain block mis-escaped: %q", got)
	}
}

func TestHighlightEscapesInsideTokens(t *testing.T) {
	got := highlightCode(`s = "<b> & </b>"`, "python")
	assertContains(t, got, `<span class="t-str">&#34;&lt;b&gt; &amp; &lt;/b&gt;&#34;</span>`)
	assertNotContains(t, got, "<b>")
}

func TestHighlightGo(t *testing.T) {
	code := "// greet\nfunc main() {\n\tmsg := \"hi\"\n\tfmt.Println(msg, 42)\n}"
	got := highlightCode(code, "go")
	assertContains(t, got,
		`<span class="t-comment">// greet</span>`,
		`<span class="t-kw">func</span>`,
		`<span class="t-fn">main</span>`,
		`<span class="t-str">&#34;hi&#34;</span>`,
		`<span class="t-type">Println</span>`,
		`<span class="t-num">42</span>`,
	)
}

func TestHighlightGoBuiltinTypesAndRawString(t *testing.T) {
	got := highlightCode("var s string = `multi\nline`", "go")
	assertContains(t, got,
		`<span class="t-kw">var</span>`,
		`<span class="t-type">string</span>`,
		"<span class=\"t-str\">`multi\nline`</span>",
	)
}

func TestHighlightPython(t *testing.T) {
	code := "def load(path):\n    # read it\n    return open(path).read()  # trailing"
	got := highlightCode(code, "py")
	assertContains(t, got,
		`<span class="t-kw">def</span>`,
		`<span class="t-fn">load</span>`,
		`<span class="t-comment"># read it</span>`,
		`<span class="t-comment"># trailing</span>`,
		`<span class="t-kw">return</span>`,
		`<span class="t-fn">open</span>`,
	)
}

func TestHighlightPythonHashInsideStringIsNotComment(t *testing.T) {
	got := highlightCode(`tag = "issue #42"`, "python")
	assertContains(t, got, `<span class="t-str">&#34;issue #42&#34;</span>`)
	assertNotContains(t, got, "t-comment")
}

func TestHighlightPythonTripleQuotedString(t *testing.T) {
	code := "doc = \"\"\"first\nsecond # not a comment\n\"\"\"\nx = 1"
	got := highlightCode(code, "python")
	assertContains(t, got, "<span class=\"t-str\">&#34;&#34;&#34;first\nsecond # not a comment\n&#34;&#34;&#34;</span>")
	assertNotContains(t, got, "t-comment")
}

func TestHighlightKeywordNeedsWordBoundary(t *testing.T) {
	// "format" contains "for", "classic" contains "class": neither is a keyword.
	got := highlightCode("format = classic", "python")
	assertNotContains(t, got, "t-kw")
}

func TestHighlightShell(t *testing.T) {
	code := "$ cargo build --release -v  # compile\nif true; then echo hi; fi"
	got := highlightCode(code, "sh")
	assertContains(t, got,
		`<span class="t-prompt">$</span>`,
		`<span class="t-flag">--release</span>`,
		`<span class="t-flag">-v</span>`,
		`<span class="t-comment"># compile</span>`,
		`<span class="t-kw">if</span>`,
		`<span class="t-kw">then</span>`,
	)
}

func TestHighlightShellHashNeedsBoundary(t *testing.T) {
	got := highlightCode(`echo ${#arr}`, "bash")
	assertNotContains(t, got, "t-comment")
}

func TestHighlightShellDoubleDashAloneIsNotFlag(t *testing.T) {
	got := highlightCode("cargo test -- --nocapture", "sh")
	assertContains(t, got, `<span class="t-flag">--nocapture</span>`)
	// the bare separator "--" stays plain
	assertNotContains(t, got, `<span class="t-flag">--</span>`)
}

func TestHighlightYaml(t *testing.T) {
	code := "name: deploy\non:\n  push:\n    branches: [main]\njobs:\n  build:\n    runs-on: ubuntu-latest\n    enabled: true\n    retries: 3  # attempts"
	got := highlightCode(code, "yml")
	assertContains(t, got,
		`<span class="t-kw">name</span>`,
		`<span class="t-kw">push</span>`,
		`<span class="t-kw">runs-on</span>`,
		`<span class="t-kw">true</span>`,
		`<span class="t-num">3</span>`,
		`<span class="t-comment"># attempts</span>`,
	)
}

func TestHighlightYamlListItemKey(t *testing.T) {
	got := highlightCode("steps:\n  - uses: actions/checkout@v4\n  - run: make", "yaml")
	assertContains(t, got,
		`<span class="t-kw">uses</span>`,
		`<span class="t-kw">run</span>`,
	)
}

func TestHighlightYamlValueIsNotKey(t *testing.T) {
	// "deploy" is a value, not followed by ':', and not at line start
	got := highlightCode("name: deploy", "yaml")
	assertNotContains(t, got, `<span class="t-kw">deploy</span>`)
}

func TestHighlightRust(t *testing.T) {
	code := "fn main() {\n    let mut n: u32 = 0xFF;\n    println!(\"n = {}\", n); // show\n}"
	got := highlightCode(code, "rust")
	assertContains(t, got,
		`<span class="t-kw">fn</span>`,
		`<span class="t-fn">main</span>`,
		`<span class="t-kw">let</span>`,
		`<span class="t-kw">mut</span>`,
		`<span class="t-type">u32</span>`,
		`<span class="t-num">0xFF</span>`,
		`<span class="t-macro">println</span>`,
		`<span class="t-str">&#34;n = {}&#34;</span>`,
		`<span class="t-comment">// show</span>`,
	)
}

func TestHighlightRustLifetimesAndCharLiterals(t *testing.T) {
	got := highlightCode(`fn first<'a>(s: &'a str) -> char { 'x' }`, "rust")
	assertContains(t, got,
		`<span class="t-lifetime">&#39;a</span>`,
		`<span class="t-type">str</span>`,
		`<span class="t-num">&#39;x&#39;</span>`,
	)
}

func TestHighlightRustNotEqualIsNotMacro(t *testing.T) {
	got := highlightCode("if x != y { }", "rust")
	assertNotContains(t, got, "t-macro")
}

func TestHighlightRustRawAndByteStrings(t *testing.T) {
	got := highlightCode(`let re = r#"a "quoted" str"#; let b = b"bytes";`, "rust")
	assertContains(t, got,
		`<span class="t-str">r#&#34;a &#34;quoted&#34; str&#34;#</span>`,
		`<span class="t-str">b&#34;bytes&#34;</span>`,
	)
}

func TestHighlightRustBlockComment(t *testing.T) {
	got := highlightCode("/* multi\nline */ fn f() {}", "rs")
	assertContains(t, got, "<span class=\"t-comment\">/* multi\nline */</span>")
}

func TestHighlightJsx(t *testing.T) {
	code := "const App = () => {\n  // render\n  return <div>{`hi ${name}`}</div>;\n};"
	got := highlightCode(code, "jsx")
	assertContains(t, got,
		`<span class="t-kw">const</span>`,
		`<span class="t-type">App</span>`,
		`<span class="t-comment">// render</span>`,
		`<span class="t-kw">return</span>`,
	)
	// JSX tags degrade to plain escaped text
	assertContains(t, got, "&lt;")
}

func TestHighlightKeywordInsideStringStaysString(t *testing.T) {
	got := highlightCode(`s = "return def class"`, "python")
	assertNotContains(t, got, `<span class="t-kw">return</span>`)
	assertContains(t, got, `<span class="t-str">&#34;return def class&#34;</span>`)
}

func TestHighlightLinesNumbersAndStructure(t *testing.T) {
	got := highlightLines("\nfn main() {\n\n}\n", "rust")
	assertContains(t, got,
		`<div class="cl"><span class="ln"> 1</span><span class="cc">`,
		`<div class="cl"><span class="ln"> 2</span><span class="cc"> </span></div>`,
		`<div class="cl"><span class="ln"> 3</span><span class="cc">`,
	)
	// leading/trailing blank lines are trimmed: exactly 3 lines
	if n := strings.Count(got, `<div class="cl">`); n != 3 {
		t.Errorf("want 3 lines, got %d:\n%s", n, got)
	}
}

func TestHighlightLinesUnknownLangEscapes(t *testing.T) {
	got := highlightLines("<tag>", "mystery")
	assertContains(t, got, "&lt;tag&gt;")
	assertNotContains(t, got, "<tag>")
}

func TestRenderCodeBlockKeepsLanguageClass(t *testing.T) {
	got := renderCodeBlock("x = 1\n", "py")
	assertContains(t, got,
		`<pre class="code language-py">`,
		`<span class="t-num">1</span>`,
		`<span class="ln"> 1</span>`,
		"</pre>",
	)
}

func TestRenderCodeBlockNoLang(t *testing.T) {
	got := renderCodeBlock("plain <text>\n", "")
	if !strings.HasPrefix(got, `<pre class="code">`) {
		t.Errorf("bare code block should have no language class: %q", got)
	}
	assertContains(t, got, "plain &lt;text&gt;")
	assertNotContains(t, got, `class="t-`)
}

func TestRenderCodeBlockGutterWidensForLongBlocks(t *testing.T) {
	got := renderCodeBlock(strings.Repeat("x\n", 100), "")
	assertContains(t, got,
		`<span class="ln">  1</span>`,
		`<span class="ln">100</span>`,
	)
}

func TestRenderMarkdownHighlightsFencedBlock(t *testing.T) {
	md := "Some text.\n\n```go\nfunc main() {}\n```\n"
	got := string(renderMarkdown([]byte(md)))
	assertContains(t, got,
		`<pre class="code language-go">`,
		`<span class="t-kw">func</span>`,
	)
}

func TestRenderMarkdownPlainFenceEscapedNoTokens(t *testing.T) {
	md := "```\na < b\n```\n"
	got := string(renderMarkdown([]byte(md)))
	assertContains(t, got, "a &lt; b")
	assertNotContains(t, got, `class="t-`)
}

func TestRenderMarkdownFenceInfoWithExtras(t *testing.T) {
	md := "```py title=x\nx = 1\n```\n"
	got := string(renderMarkdown([]byte(md)))
	assertContains(t, got, `class="code language-py"`, `<span class="t-num">1</span>`)
}

func TestRenderMarkdownNonCodeUnaffected(t *testing.T) {
	md := "# Title\n\nA [link](https://example.com) and *emphasis*.\n"
	got := string(renderMarkdown([]byte(md)))
	assertContains(t, got, "<h1", `<a href="https://example.com"`, "<em>emphasis</em>")
}

func TestRenderStaticCodeScripts(t *testing.T) {
	page := `<p>before</p>
<script type="text/rust">fn main() {
    println!("hi");
}</script>
<script type="text/shell">$ cargo run --release</script>
<script>keep_me();</script>`
	got := renderStaticCodeScripts(page)
	assertContains(t, got,
		`<pre class="code language-rust">`,
		`<pre class="code language-sh">`,
		`<span class="t-kw">fn</span>`,
		`<span class="t-macro">println</span>`,
		`<span class="t-prompt">$</span>`,
		`<span class="t-flag">--release</span>`,
		`<script>keep_me();</script>`, // regular scripts untouched
		"<p>before</p>",
	)
	assertNotContains(t, got, `type="text/rust"`, `type="text/shell"`)
}

func TestRenderStaticCodeScriptsNoBlocksPassThrough(t *testing.T) {
	page := "<html><body><p>nothing here</p></body></html>"
	if got := renderStaticCodeScripts(page); got != page {
		t.Errorf("page without code scripts should be unchanged, got %q", got)
	}
}

// TestHighlightPreservesNonASCII guards against mojibake in code blocks. The
// tokenizer walks bytes, so emitting a byte >= 0x80 via string(c) would
// re-encode it and turn "café" into "cafÃ©".
func TestHighlightPreservesNonASCII(t *testing.T) {
	for _, lang := range []string{"python", "rust", "go", "js", "shell", "yaml"} {
		t.Run(lang, func(t *testing.T) {
			got := highlightCode("x = café ÷ naïve", lang)
			assertContains(t, got, "café", "÷", "naïve")
			assertNotContains(t, got, "Ã©", "Ã·", "Ã¯")
		})
	}
}

// TestHighlightStringEscapesDontTerminate covers the backslash-escape branch.
// Without it a `\"` ends the string early, the rest of the literal highlights
// as code, and the closing quote opens a new string that runs to end of line —
// a visibly broken code block.
func TestHighlightStringEscapesDontTerminate(t *testing.T) {
	for _, lang := range []string{"go", "python", "js", "rust"} {
		t.Run(lang, func(t *testing.T) {
			got := highlightCode(`s = "say \"hi\" now" if x`, lang)
			assertContains(t, got, `<span class="t-str">&#34;say \&#34;hi\&#34; now&#34;</span>`)
			// `if` sits outside the literal, so it must still be a keyword;
			// the point is that the string ended where it should.
			assertNotContains(t, got, `<span class="t-str">&#34; if x`)
		})
	}
}

// TestHighlightUnterminatedStringStopsAtEOL keeps one stray quote from
// swallowing every following line of a block.
func TestHighlightUnterminatedStringStopsAtEOL(t *testing.T) {
	got := highlightCode("let s = \"oops\nlet n = 1;", "rust")

	if n := strings.Count(got, `<span class="t-kw">let</span>`); n != 2 {
		t.Errorf("got %d `let` keywords, want 2 — line two stopped tokenizing", n)
	}
	assertContains(t, got, `<span class="t-num">1</span>`)
}

// TestHighlightUnterminatedTokensRunToEOF pins the "no closer found"
// fall-throughs for block comments, raw strings and triple-quoted strings.
// Each defaults to consuming the rest of the input; a regression that consumed
// nothing, or sliced past the end, would panic at build time.
func TestHighlightUnterminatedTokensRunToEOF(t *testing.T) {
	tests := []struct {
		name string
		src  string
		lang string
		want string
	}{
		{
			name: "block comment",
			src:  "/* forever\nand ever",
			lang: "rust",
			want: `<span class="t-comment">/* forever
and ever</span>`,
		},
		{
			name: "raw string",
			src:  `let r = r#"abc`,
			lang: "rust",
			want: `<span class="t-str">r#&#34;abc</span>`,
		},
		{
			name: "triple quote",
			src:  `doc = """abc`,
			lang: "python",
			want: `<span class="t-str">&#34;&#34;&#34;abc</span>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertContains(t, highlightCode(tt.src, tt.lang), tt.want)
		})
	}
}

// TestHighlightRustEscapedCharLiteral covers the branch that tells a char
// literal from a lifetime — the distinction the rust spec's lifetimes flag
// exists for, and one the quick reference exercises for real.
func TestHighlightRustEscapedCharLiteral(t *testing.T) {
	got := highlightCode(`let nl = '\n'; let t = '\t';`, "rust")

	assertContains(t,
		got,
		`<span class="t-num">&#39;\n&#39;</span>`,
		`<span class="t-num">&#39;\t&#39;</span>`,
	)
	assertNotContains(t, got, "t-lifetime")
}

// TestSpecForResolvesEveryAlias checks every language alias resolves. A dropped
// map entry degrades silently to plain text, which no other test would notice.
func TestSpecForResolvesEveryAlias(t *testing.T) {
	for alias, want := range langSpecs {
		if got := specFor(alias); got != want {
			t.Errorf("specFor(%q) resolved to the wrong spec", alias)
		}
	}
	// The info string comes off a markdown fence, so it is trimmed and folded.
	if specFor("  RUST  ") != rustSpec {
		t.Error("specFor does not trim and lowercase the fence info string")
	}
	if specFor("cmake") != nil {
		t.Error("an unknown language should resolve to no spec, so it stays plain")
	}
}
