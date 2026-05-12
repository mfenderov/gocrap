package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type functionRange struct {
	Name      string
	File      string
	StartLine int
	EndLine   int
}

func extractFunctions(filePath string) ([]functionRange, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil, err
	}

	var functions []functionRange
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !isFuncDecl(ok, fn) {
			continue
		}
		name := funcName(fn)
		functions = append(functions, functionRange{
			Name:      name,
			File:      filePath,
			StartLine: fset.Position(fn.Pos()).Line,
			EndLine:   fset.Position(fn.End()).Line,
		})
	}
	return functions, nil
}

func isFuncDecl(ok bool, fn *ast.FuncDecl) bool {
	return ok && fn.Body != nil
}

func funcName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	return receiverName(fn.Recv.List[0].Type) + "." + fn.Name.Name
}

func receiverName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return receiverName(t.X)
	default:
		return ""
	}
}

func extractAllFunctions(paths []string) ([]functionRange, error) {
	sourceFiles, err := findSourceFiles(paths)
	if err != nil {
		return nil, err
	}
	var functions []functionRange
	for _, f := range sourceFiles {
		fns, err := extractFunctions(f)
		if err != nil {
			return nil, err
		}
		functions = append(functions, fns...)
	}
	return functions, nil
}

func findSourceFiles(paths []string) ([]string, error) {
	var files []string
	seen := map[string]bool{}
	for _, root := range paths {
		if err := walkSourceDir(root, &files, seen); err != nil {
			return nil, err
		}
	}
	return files, nil
}

func walkSourceDir(root string, files *[]string, seen map[string]bool) error {
	v := &walkVisitor{root: root, files: files, seen: seen}
	return filepath.WalkDir(root, v.visit)
}

type walkVisitor struct {
	root  string
	files *[]string
	seen  map[string]bool
}

func (v *walkVisitor) visit(path string, entry os.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if entry.IsDir() {
		return v.handleDir(entry)
	}
	if !isGoSource(entry) {
		return nil
	}
	return v.addFile(path)
}

func (v *walkVisitor) handleDir(entry os.DirEntry) error {
	if isSkipDir(entry.Name()) {
		return filepath.SkipDir
	}
	return nil
}

func (v *walkVisitor) addFile(path string) error {
	rel, _ := filepath.Rel(v.root, path)
	if rel == "" {
		rel = path
	}
	rel = filepath.ToSlash(rel)
	if !v.seen[rel] {
		v.seen[rel] = true
		*v.files = append(*v.files, rel)
	}
	return nil
}

func isSkipDir(name string) bool {
	switch name {
	case ".git", "vendor", "testdata":
		return true
	}
	return false
}

func isGoSource(entry os.DirEntry) bool {
	return strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go")
}
