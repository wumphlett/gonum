package gonum

import (
	_ "embed"
	"errors"
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
)

//go:embed enum.go.template
var template string

func main() {
	var (
		typeName string
		// TODO case
		fileName    = os.Getenv("GOFILE")
		lineNum     = os.Getenv("GOLINE")
		packageName = os.Getenv("GOPACKAGE")
	)
	flag.StringVar(&typeName, "type", "", "type to be generated for")
	flag.Parse()

	if err := process(typeName, fileName, lineNum, packageName); err != nil {
		os.Stderr.WriteString(err.Error())
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

	var specs [][2]string

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

			tag, ok := "", false
			for _, field := range strings.Fields(spec.Comment.Text()) {
				if strings.HasPrefix(field, "json:") {
					tag, ok = field[len("json:\""):len(field)-1], true
					break
				}
			}
			if ok {
				specs = append(specs, [2]string{spec.Names[0].Name, tag})
			}
		}

		return false
	})

	if len(specs) == 0 {
		return errors.New(fileName + ": unable to find values for enum type: " + typeName)
	}

	return nil
}
