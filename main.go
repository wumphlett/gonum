package main

import (
	_ "embed"
	"errors"
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
)

//go:embed enum.go.template
var t string

func main() {
	var (
		typeName    string
		fileName    = os.Getenv("GOFILE")
		lineNum     = os.Getenv("GOLINE")
		packageName = os.Getenv("GOPACKAGE")
	)
	flag.StringVar(&typeName, "type", "", "type to be generated for")
	flag.Parse()

	if err := process(typeName, fileName, lineNum, packageName); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}

func process(typeName, fileName, lineNum, packageName string) error {
	if typeName == "" || fileName == "" || lineNum == "" || packageName == "" {
		return errors.New("missing parameters")
	}

	inputCode, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fileName, inputCode, parser.ParseComments)
	if err != nil {
		return err
	}

	expectedLine, _ := strconv.Atoi(lineNum)
	expectedLine += 1

	specs := map[string]string{}

	ast.Inspect(f, func(astNode ast.Node) bool {
		node, ok := astNode.(*ast.GenDecl)
		if !ok || (node.Tok != token.CONST && node.Tok != token.VAR) {
			return true
		}

		position := fset.Position(node.Pos())
		if position.Line != expectedLine {
			return false
		}

		for _, astSpec := range node.Specs {
			spec, ok := astSpec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			if len(spec.Names) != 1 {
				break
			}

			specs[spec.Names[0].Name] = strcase.ToSnake(strings.TrimPrefix(spec.Names[0].Name, typeName))
		}

		return false
	})

	if len(specs) == 0 {
		return errors.New(fileName + ": unable to find values for enum type: " + typeName)
	}

	tmpl, err := template.New(typeName).Parse(t)
	if err != nil {
		return err
	}

	file, err := os.Create(filepath.Join(filepath.Dir(fileName), strcase.ToSnake(strings.ToLower(typeName))) + "_enum.go")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	tmpl.Execute(file, struct {
		Package string
		Type    string
		Values  map[string]string
	}{packageName, typeName, specs})

	return nil
}
