package main

import (
	noblanks "github.com/GrailFinder/noblanks-linter"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(noblanks.Analyzer)
}
