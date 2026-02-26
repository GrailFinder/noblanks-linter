package noblanks

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "noblanks",
	Doc:  "reports blank lines inside function bodies",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	if len(pass.Files) == 0 {
		return nil, nil
	}

	baseDir := getBaseDir(pass)

	for _, f := range pass.Files {
		checkFile(pass, f, baseDir)
	}
	return nil, nil
}

func getBaseDir(pass *analysis.Pass) string {
	cwd, err := os.Getwd()
	if err != nil {
		if len(pass.Files) == 0 {
			return ""
		}
		pos := pass.Fset.Position(pass.Files[0].Name.Pos())
		return filepath.Dir(pos.Filename)
	}
	return cwd
}

func makeRel(filename, baseDir string) string {
	rel, err := filepath.Rel(baseDir, filename)
	if err != nil {
		return filename
	}
	return rel
}

func checkFile(pass *analysis.Pass, f *ast.File, baseDir string) {
	fset := pass.Fset
	filename := fset.Position(f.Name.Pos()).Filename

	if !isInBaseDir(filename, baseDir) {
		return
	}

	ast.Inspect(f, func(n ast.Node) bool {
		var body *ast.BlockStmt
		var funcName string
		switch node := n.(type) {
		case *ast.FuncDecl:
			body = node.Body
			if node.Name != nil {
				funcName = node.Name.Name
			}
		case *ast.FuncLit:
			body = node.Body
			funcName = "anonymous"
		default:
			return true
		}
		if body == nil || len(body.List) == 0 {
			return true
		}
		checkBody(pass, fset, body, funcName, baseDir)
		return true
	})
}

func isInBaseDir(filename, baseDir string) bool {
	absFilename, err := filepath.Abs(filename)
	if err != nil {
		return false
	}
	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return false
	}
	return strings.HasPrefix(absFilename, absBaseDir)
}

func checkBody(pass *analysis.Pass, fset *token.FileSet, body *ast.BlockStmt, funcName, baseDir string) {
	stmts := body.List
	for i := 0; i < len(stmts)-1; i++ {
		pos1 := stmts[i].End()
		pos2 := stmts[i+1].Pos()
		posInfo1 := fset.Position(pos1)
		posInfo2 := fset.Position(pos2)
		if posInfo1.Filename != posInfo2.Filename {
			continue
		}
		if !isInBaseDir(posInfo1.Filename, baseDir) {
			continue
		}
		if posInfo2.Line-posInfo1.Line <= 1 {
			continue
		}
		file := fset.File(pos1)
		if file == nil {
			continue
		}
		if hasBlankLine(file, posInfo1.Line+1, posInfo2.Line-1) {
			relPath := makeRel(posInfo2.Filename, baseDir)
			strings.TrimPrefix(relPath, "-: ")
			pass.Reportf(token.NoPos, "%s:%d:%d: blank lines inside function body (%s)", relPath, posInfo2.Line, posInfo2.Column, funcName)
		}
	}
}

func hasBlankLine(file *token.File, startLine, endLine int) bool {
	for line := startLine; line <= endLine; line++ {
		start := file.LineStart(line)
		if start == token.NoPos {
			continue
		}
		offset := file.Offset(start)
		content := readFileContent(file.Name())
		if content == nil || offset >= len(content) {
			continue
		}
		end := offset
		for end < len(content) && content[end] != '\n' && content[end] != '\r' {
			end++
		}
		lineContent := content[offset:end]
		if !isBlank(lineContent) {
			return false
		}
	}
	return true
}

func readFileContent(filename string) []byte {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}
	return data
}

func isBlank(line []byte) bool {
	for _, b := range line {
		if b != ' ' && b != '\t' {
			return false
		}
	}
	return true
}
