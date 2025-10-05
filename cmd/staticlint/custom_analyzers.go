// Пакет Main
package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// OsExitAnalyzer - кастомный анализатор, запрещает прямой вызов os.Exit из main.go
var OsExitAnalyzer = &analysis.Analyzer{
	Name: "osExitAnalyzer",
	Doc:  "prevents direct calling of os.Exit from main.go",
	Run:  runOsExitAnalysis,
}

// runOsExitAnalysis - функция анализатора OsExitAnalyzer, который запрещает прямой вызов os.Exit из main.go
func runOsExitAnalysis(pass *analysis.Pass) (interface{}, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			// Проверяем только вызовы функций
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}

			// Проверяем, что это селектор (вызов метода/функции из пакета)
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			// Проверяем, что это os.Exit
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "os" && sel.Sel.Name == "Exit" {
				pass.Reportf(call.Pos(), "os.Exit call detected in main package")
			}

			return true
		})
	}
	return nil, nil
}
