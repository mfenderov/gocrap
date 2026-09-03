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
	for _, fn := range funcDecls(file) {
		functions = append(functions, functionRange{
			Name:      funcName(fn),
			File:      filePath,
			StartLine: fset.Position(fn.Pos()).Line,
			EndLine:   fset.Position(fn.End()).Line,
		})
	}
	return functions, nil
}

func funcDecls(file *ast.File) []*ast.FuncDecl {
	var decls []*ast.FuncDecl
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Body != nil {
			decls = append(decls, fn)
		}
	}
	return decls
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
		*v.files = append(*v.files, path)
	}
	return nil
}

var skipDirs = map[string]bool{
	"testdata": true,
	"vendor":   true,
}

// isSkipDir mirrors gocyclo's walker: skip testdata, vendor, dot-dirs and
// "_"-prefixed dirs, so both passes see the same source tree. Without this,
// stale copies under e.g. .worktrees/ pollute the coverage join and produce
// non-deterministic 0% coverage attribution.
func isSkipDir(name string) bool {
	if skipDirs[name] {
		return true
	}
	if name == "." || name == ".." {
		return false
	}
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

func isGoSource(entry os.DirEntry) bool {
	return strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go")
}
