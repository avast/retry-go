// Command genreadme generates the project README from the package
// documentation, using only the standard library (go/doc renders doc
// comments as Markdown since Go 1.19). The layout lives in README.tmpl.
package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

//go:embed README.tmpl
var readmeTemplate string

const importPath = "github.com/avast/retry-go/v5"

type documentedFunc struct {
	Receiver  string
	Name      string
	Signature string
	Doc       string
}

type documentedType struct {
	Name      string
	Signature string
	Doc       string
	Funcs     []documentedFunc
	Methods   []documentedFunc
}

type documentedPackage struct {
	Name     string
	Synopsis string
	Doc      string
	Funcs    []documentedFunc
	Types    []documentedType
}

func main() {
	out, err := generate(".")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(out)
}

// generate renders the README for the package in pkgDir.
func generate(pkgDir string) (string, error) {
	fset := token.NewFileSet()
	files, err := packageSourceFiles(fset, pkgDir)
	if err != nil {
		return "", err
	}

	docPkg, err := doc.NewFromFiles(fset, files, importPath)
	if err != nil {
		return "", err
	}

	pkg := documentedPackage{
		Name:     docPkg.Name,
		Synopsis: docPkg.Synopsis(docPkg.Doc),
		Doc:      markdownFor(docPkg, docPkg.Doc),
	}
	for _, fn := range docPkg.Funcs {
		pkg.Funcs = append(pkg.Funcs, documentedFunc{
			Name:      fn.Name,
			Signature: declaration(fset, fn.Decl),
			Doc:       markdownFor(docPkg, fn.Doc),
		})
	}
	for _, typ := range docPkg.Types {
		entry := documentedType{
			Name:      typ.Name,
			Signature: declaration(fset, typ.Decl),
			Doc:       markdownFor(docPkg, typ.Doc),
		}
		for _, fn := range typ.Funcs {
			entry.Funcs = append(entry.Funcs, documentedFunc{
				Name:      fn.Name,
				Signature: declaration(fset, fn.Decl),
				Doc:       markdownFor(docPkg, fn.Doc),
			})
		}
		for _, method := range typ.Methods {
			entry.Methods = append(entry.Methods, documentedFunc{
				Receiver:  method.Recv,
				Name:      method.Name,
				Signature: declaration(fset, method.Decl),
				Doc:       markdownFor(docPkg, method.Doc),
			})
		}
		pkg.Types = append(pkg.Types, entry)
	}

	tmpl := template.Must(template.New("readme").Parse(readmeTemplate))
	var out bytes.Buffer
	if err := tmpl.Execute(&out, pkg); err != nil {
		return "", err
	}
	return blankLines.ReplaceAllString(out.String(), "\n\n"), nil
}

// packageSourceFiles parses all non-test Go files of the package in dir,
// in a deterministic order.
func packageSourceFiles(fset *token.FileSet, dir string) ([]*ast.File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.Type().IsRegular() && strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	files := make([]*ast.File, 0, len(names))
	for _, name := range names {
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

// declaration renders a declaration as Go source, without function bodies.
func declaration(fset *token.FileSet, decl ast.Decl) string {
	if fn, ok := decl.(*ast.FuncDecl); ok {
		fn.Body = nil
	}

	var buf bytes.Buffer
	_ = printer.Fprint(&buf, fset, decl)
	return strings.ReplaceAll(buf.String(), "\t", "    ")
}

// markdownFor renders a doc comment as Markdown for GitHub: pkg.go.dev-style
// heading anchors are stripped (GitHub would show them as literal text),
// tab-indented code blocks become fenced Go code, and escaped backticks are
// restored so that the doc comment backquote convention renders as code.
func markdownFor(pkg *doc.Package, text string) string {
	out := string(pkg.Markdown(text))
	out = headingAnchor.ReplaceAllString(out, "## $1")
	out = fenceIndentedCodeBlocks(out)
	out = strings.ReplaceAll(out, "\\`", "`")
	return blankLines.ReplaceAllString(out, "\n\n")
}

var (
	headingAnchor = regexp.MustCompile(`(?m)^### (.*) \{#hdr-[^}]+\}$`)
	blankLines    = regexp.MustCompile(`\n{3,}`)
)

func fenceIndentedCodeBlocks(markdown string) string {
	lines := strings.Split(markdown, "\n")
	out := make([]string, 0, len(lines))
	inCode := false
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "\t"):
			if !inCode {
				out = append(out, "```go")
				inCode = true
			}
			out = append(out, line[1:])
		case line == "" && inCode:
			out = append(out, line)
		default:
			if inCode {
				closeCodeFence(&out)
				inCode = false
			}
			out = append(out, line)
		}
	}
	if inCode {
		closeCodeFence(&out)
	}
	return strings.Join(out, "\n")
}

func closeCodeFence(out *[]string) {
	for len(*out) > 0 && (*out)[len(*out)-1] == "" {
		*out = (*out)[:len(*out)-1]
	}
	*out = append(*out, "```")
}
