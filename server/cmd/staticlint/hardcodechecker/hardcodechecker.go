package hardcodechecker

import (
	"go/ast"
	"go/token"
	"golang.org/x/tools/go/analysis"
	"strings"
)

var Analyzer = &analysis.Analyzer{
	Name: "hardcodechecker",
	Doc:  "Запрещает хардкод JWT secret, client_secret, password и других.",
	Run:  run,
}

var forbidden = []string{
	"jwt_secret", "secret", "password", "client_secret", "api_key",
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.BasicLit:
				if x.Kind != token.STRING {
					s := strings.Trim(x.Value, `"`)
					for _, f := range forbidden {
						if strings.Contains(strings.ToLower(s), f) {
							pass.Reportf(x.Pos(), "Хардкод ключа '%s' запрещен, внесите в .env", f)
						}
					}
				}
			}
			return true
		})

	}
	return nil, nil
}
