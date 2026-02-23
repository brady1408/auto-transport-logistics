package handler

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"math"
	"strings"
)

//go:embed all:templates
var templateFS embed.FS

// TemplateMap holds a separate parsed template for each page.
// Each page template is cloned from the base layout, so each page's
// "title"/"content" block definitions don't collide.
type TemplateMap struct {
	pages    map[string]*template.Template // page name -> cloned base with page blocks
	partials *template.Template            // base + all partials (for standalone rendering)
}

// RenderPage executes a page template by rendering the "base" layout
// with the page's overridden blocks (title, content).
func (m *TemplateMap) RenderPage(w io.Writer, name string, data any) error {
	t, ok := m.pages[name]
	if !ok {
		return fmt.Errorf("page template %q not found", name)
	}
	return t.ExecuteTemplate(w, "base", data)
}

// RenderTemplate executes a named template directly (for self-contained
// templates like login.html, or partials like customer_table).
func (m *TemplateMap) RenderTemplate(w io.Writer, name string, data any) error {
	return m.partials.ExecuteTemplate(w, name, data)
}

var funcMap = template.FuncMap{
	"add": func(a, b int) int { return a + b },
	"sub": func(a, b int) int { return a - b },
	"mul": func(a, b int) int { return a * b },
	"div": func(a, b int) int {
		if b == 0 {
			return 0
		}
		return a / b
	},
	"mod": func(a, b int) int {
		if b == 0 {
			return 0
		}
		return a % b
	},
	"seq": func(start, end int) []int {
		var s []int
		for i := start; i <= end; i++ {
			s = append(s, i)
		}
		return s
	},
	"totalPages": func(total, pageSize int) int {
		if pageSize == 0 {
			return 0
		}
		return int(math.Ceil(float64(total) / float64(pageSize)))
	},
	"selected": func(a, b string) template.HTMLAttr {
		if a == b {
			return template.HTMLAttr("selected")
		}
		return ""
	},
	"checked": func(v bool) template.HTMLAttr {
		if v {
			return template.HTMLAttr("checked")
		}
		return ""
	},
	"deref": func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	},
	"derefInt": func(i *int) int {
		if i == nil {
			return 0
		}
		return *i
	},
	"dict": func(values ...any) map[string]any {
		d := make(map[string]any, len(values)/2)
		for i := 0; i < len(values)-1; i += 2 {
			key, ok := values[i].(string)
			if ok {
				d[key] = values[i+1]
			}
		}
		return d
	},
}

func ParseTemplates() (*TemplateMap, error) {
	m := &TemplateMap{pages: make(map[string]*template.Template)}

	// Parse layout templates as the base
	base, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/layout/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse layout: %w", err)
	}

	// Walk all page template files
	var allFiles []string
	err = fs.WalkDir(templateFS, "templates/pages", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".html") {
			allFiles = append(allFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk pages: %w", err)
	}

	// Classify each file: page template (has nested title/content defines)
	// or standalone (partial/self-contained like login.html).
	type fileEntry struct {
		path      string
		content   string
		isPage    bool
		pageName  string
		pageInner string // page-specific defines (title, content)
		extras    string // standalone defines after the outer wrapper (partials)
	}

	var files []fileEntry
	for _, f := range allFiles {
		raw, err := fs.ReadFile(templateFS, f)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f, err)
		}
		s := string(raw)

		fe := fileEntry{path: f, content: s}

		// A page template contains nested {{define "title"}} and {{define "content"}}
		// inside an outer {{define "page_name"}} wrapper.
		if strings.Contains(s, `define "title"`) && strings.Contains(s, `define "content"`) {
			name, pageInner, extras, ok := unwrapPageTemplate(s)
			if ok {
				fe.isPage = true
				fe.pageName = name
				fe.pageInner = pageInner
				fe.extras = extras
			}
		}

		files = append(files, fe)
	}

	// First pass: parse all non-page files (pure partials + self-contained templates)
	// AND the "extras" from page files (partial defines that live in the same file
	// as a page but outside the outer wrapper) into base.
	for _, fe := range files {
		if fe.isPage {
			// Parse standalone partials from this page file into base
			if strings.TrimSpace(fe.extras) != "" {
				if _, err := base.Parse(fe.extras); err != nil {
					return nil, fmt.Errorf("parse extras from %s: %w", fe.path, err)
				}
			}
			continue
		}
		if _, err := base.Parse(fe.content); err != nil {
			return nil, fmt.Errorf("parse partial %s: %w", fe.path, err)
		}
	}

	// Store enriched base for partial/standalone rendering
	m.partials = base

	// Second pass: for each page template, clone the enriched base
	// (which has all partials) and parse the page-specific blocks.
	for _, fe := range files {
		if !fe.isPage {
			continue
		}

		clone, err := base.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone base for %s: %w", fe.path, err)
		}

		if _, err := clone.Parse(fe.pageInner); err != nil {
			return nil, fmt.Errorf("parse page %s (%s): %w", fe.pageName, fe.path, err)
		}

		m.pages[fe.pageName] = clone
	}

	return m, nil
}

// unwrapPageTemplate extracts the page name and inner content from a
// template file that uses the pattern:
//
//	{{define "page_name"}}
//	{{template "base" .}}
//	{{define "title"}}...{{end}}
//	{{define "content"}}...{{end}}
//	{{end}}
//
// It removes the outer {{define}} wrapper and the {{template "base" .}} line,
// leaving the inner block definitions (title, content) as top-level defines.
// Any standalone defines after the outer wrapper (like partial templates
// defined in the same file) are preserved.
func unwrapPageTemplate(content string) (pageName string, pageInner string, extras string, ok bool) {
	lines := strings.Split(content, "\n")

	// Find the outer {{define "name"}} and its matching {{end}}
	outerStart := -1
	outerEnd := -1
	depth := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Count ALL opening actions that require {{end}} to close,
		// not just define/block — otherwise if/range/with ends
		// throw off the depth count.
		opens := 0
		for _, kw := range []string{"{{define ", "{{block ", "{{if ", "{{range ", "{{with "} {
			opens += strings.Count(trimmed, kw)
		}
		closes := strings.Count(trimmed, "{{end}}")

		if outerStart == -1 {
			if opens > 0 {
				outerStart = i
				// Extract the define name
				pageName = extractDefineName(trimmed)
				depth = opens - closes
				if depth == 0 {
					// Single-line define (unlikely for a page)
					outerEnd = i
				}
			}
			continue
		}

		if outerEnd == -1 {
			depth += opens - closes
			if depth == 0 {
				outerEnd = i
				break
			}
		}
	}

	if outerStart == -1 || outerEnd == -1 || pageName == "" {
		return "", "", "", false
	}

	// Extract page-specific inner content (between outer define and its end),
	// minus {{template "base" .}}
	var innerLines []string
	for _, line := range lines[outerStart+1 : outerEnd] {
		trimmed := strings.TrimSpace(line)
		if trimmed == `{{template "base" .}}` {
			continue
		}
		innerLines = append(innerLines, line)
	}

	// Extract extras: any content after the outer {{end}} (standalone partial defines)
	var extraLines []string
	if outerEnd+1 < len(lines) {
		extraLines = append(extraLines, lines[outerEnd+1:]...)
	}

	return pageName, strings.Join(innerLines, "\n"), strings.Join(extraLines, "\n"), true
}

// extractDefineName extracts the template name from a line like:
//
//	{{define "my_page.html"}}
func extractDefineName(line string) string {
	const prefix = `{{define "`
	idx := strings.Index(line, prefix)
	if idx == -1 {
		return ""
	}
	rest := line[idx+len(prefix):]
	end := strings.Index(rest, `"`)
	if end == -1 {
		return ""
	}
	return rest[:end]
}
