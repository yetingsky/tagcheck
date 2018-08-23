package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

func main() {
	var (
		flagPath = flag.String("path", ".", "directory name")
	)
	flag.Parse()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, *flagPath, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	fileErrs := make(map[string]FieldErrors)
	for pkgName, pkg := range pkgs {
		for fileName, file := range pkg.Files {
			_, _ = pkgName, fileName
			errs := check(fset, file)
			if errs != nil {
				fileErrs[fileName] = errs
			}
		}
	}

	if len(fileErrs) > 0 {
		for _, errs := range fileErrs {
			fmt.Fprintln(os.Stderr, errs)
		}
		os.Exit(1)
	}
}

type FieldError struct {
	FileName  string
	FieldName string
	Line      int
	Column    int
}

func (f FieldError) Error() string {
	return fmt.Sprintf("%s:%s:%d:%d", f.FileName, f.FieldName, f.Line, f.Column)
}

type FieldErrors []FieldError

func (f FieldErrors) String() string {
	var b strings.Builder
	for i, e := range f {
		fmt.Fprintf(&b, "%s", e)
		if i < len(f)-1 {
			fmt.Fprint(&b, "\n")
		}
	}
	return b.String()
}

func check(fset *token.FileSet, file *ast.File) FieldErrors {
	var errs FieldErrors
	skipStructs := collectSkipStructs(file)

	check := func(n ast.Node) bool {
		if isSkipStruct(n, skipStructs) {
			// not traverses ast
			return false
		}

		x, ok := n.(*ast.StructType)
		if !ok {
			return true
		}

		for _, f := range x.Fields.List {
			if f.Tag != nil {
				continue
			}

			// anonymous field
			if f.Names == nil {
				continue
			}

			fieldName := ""
			if len(f.Names) != 0 {
				fieldName = f.Names[0].Name
			}
			errs = append(errs, FieldError{
				fset.Position(f.Pos()).Filename,
				fieldName,
				fset.Position(f.Pos()).Line,
				fset.Position(f.Pos()).Column,
			})
		}
		return true
	}
	ast.Inspect(file, check)
	return errs
}

func isSkipStruct(node ast.Node, skipStructs map[string]bool) bool {
	switch x := node.(type) {
	case *ast.TypeSpec:
		if _, ok := x.Type.(*ast.StructType); ok {
			structName := x.Name.Name
			return skipStructs[structName]
		}
	}
	return false
}

func collectSkipStructs(file *ast.File) map[string]bool {
	skipStructs := make(map[string]bool)
	var sts []string

	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if strings.Contains(c.Text, "notagcheck") {
				i := strings.Index(c.Text, ":")
				sts = append(sts, strings.Split(c.Text[i+1:], ",")...)
			}
		}
	}

	for _, st := range sts {
		if st != "" {
			skipStructs[st] = true
		}
	}
	return skipStructs
}
