package parser

import (
	"context"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

type Language string

const (
	LangTypeScript Language = "typescript"
	LangGo         Language = "go"
	LangPython     Language = "python"
	LangRust       Language = "rust"
	LangPHP        Language = "php"
	LangUnknown    Language = "unknown"
)

type StepDefFound struct {
	Pattern      string
	FunctionName string
	Line         int
}

func DetectLanguage(filename string) Language {
	ext := filepath.Ext(filename)
	switch ext {
	case ".ts", ".tsx":
		return LangTypeScript
	case ".go":
		return LangGo
	case ".py":
		return LangPython
	case ".rs":
		return LangRust
	case ".php":
		return LangPHP
	}
	return LangUnknown
}

func getLanguage(lang Language) *sitter.Language {
	switch lang {
	case LangTypeScript:
		return typescript.GetLanguage()
	case LangGo:
		return golang.GetLanguage()
	case LangPython:
		return python.GetLanguage()
	case LangRust:
		return rust.GetLanguage()
	case LangPHP:
		return php.GetLanguage()
	default:
		return nil
	}
}

func ParseImports(content []byte, lang Language) ([]string, error) {
	sl := getLanguage(lang)
	if sl == nil {
		return nil, nil
	}

	parser := sitter.NewParser()
	parser.SetLanguage(sl)

	tree, _ := parser.ParseCtx(context.Background(), nil, content)
	root := tree.RootNode()

	var queryStr string
	switch lang {
	case LangTypeScript:
		queryStr = `
		(import_statement source: (string (string_fragment) @path))
		(export_statement source: (string (string_fragment) @path))
		`
	case LangGo:
		queryStr = `
		(import_spec path: (string_literal) @path)
		`
	case LangPython:
		queryStr = `
		(import_from_statement module_name: (dotted_name) @path)
		(import_statement name: (dotted_name) @path)
		`
	case LangRust:
		queryStr = `
		(use_declaration argument: (scoped_identifier) @path)
		`
	case LangPHP:
		queryStr = `
		(namespace_use_clause (qualified_name) @path)
		`
	}

	if queryStr == "" {
		return nil, nil
	}

	q, err := sitter.NewQuery([]byte(queryStr), sl)
	if err != nil {
		return nil, err
	}
	qc := sitter.NewQueryCursor()
	qc.Exec(q, root)

	var imports []string
	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		for _, c := range m.Captures {
			if c.Node != nil {
				text := string(content[c.Node.StartByte():c.Node.EndByte()])
				// Clean quotes for some languages
				text = strings.Trim(text, "\"'`")
				imports = append(imports, text)
			}
		}
	}

	return imports, nil
}

func ParseStepDefinitions(content []byte, lang Language) ([]StepDefFound, error) {
	sl := getLanguage(lang)
	if sl == nil {
		return nil, nil
	}

	parser := sitter.NewParser()
	parser.SetLanguage(sl)
	tree, _ := parser.ParseCtx(context.Background(), nil, content)
	root := tree.RootNode()

	// Queries for step definitions
	// TODO: Add Go (Godog), Python (Behave), Rust (Cucumber), PHP (Behat)
	var queryStr string
	switch lang {
	case LangTypeScript:
		queryStr = `
		(call_expression
			function: (identifier) @keyword
			arguments: (arguments
				(string (string_fragment) @pattern)
			)
		)
		`
	case LangGo:
		// Godog: ctx.Step(`^regex$`, handler)
		// Or: suite.Step(`^regex$`, handler)
		queryStr = `
		(call_expression
			function: (selector_expression field: (field_identifier) @method)
			arguments: (argument_list
				(raw_string_literal) @pattern
			)
			(#match? @method "^(Step|Given|When|Then)$")
		)
		`
	case LangPython:
		// Behave: @given("pattern")
		queryStr = `
		(decorated_definition
			decorator: (decorator
				call: (call
					function: (identifier) @keyword
					arguments: (argument_list (string) @pattern)
				)
			)
			definition: (function_definition name: (identifier) @method)
		)
		`
	// Rust and PHP would need specific framework queries. Leaving as TODO/Partial for now.
	}

	if queryStr == "" {
		return nil, nil
	}

	q, err := sitter.NewQuery([]byte(queryStr), sl)
	if err != nil {
		return nil, err
	}
	qc := sitter.NewQueryCursor()
	qc.Exec(q, root)

	var results []StepDefFound

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		var pattern, method string
		var line int

		for _, c := range m.Captures {
			name := q.CaptureNameForId(c.Index)
			if name == "pattern" {
				pattern = string(content[c.Node.StartByte():c.Node.EndByte()])
				pattern = strings.Trim(pattern, "\"`'")
				line = int(c.Node.StartPoint().Row) + 1
			} else if name == "method" {
				method = string(content[c.Node.StartByte():c.Node.EndByte()])
			}
		}

		if pattern != "" {
			results = append(results, StepDefFound{
				Pattern:      pattern,
				FunctionName: method,
				Line:         line,
			})
		}
	}

	return results, nil
}
