//
//  parser
//
//  Created by Noah Evans on 2010-11-13.
//  Copyright (c) 2010 __MyCompanyName__. All rights reserved.
//

package main

import (
	"flag"
	"os"
	"fmt"
	"strings"
	"go/ast"
	"go/parser"
	"log"
)

var (
	NotFoundErr = os.NewError("type not found")
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: textmate [flags] [path ...]")
	flag.PrintDefaults()
	os.Exit(2)
}

var (
	place       string
	placeHolder = flag.Bool(place, false, "dont print final newline")
)
/*
parse the file, 
	get the import statements
	parse the imports in their dirs
	create a list of symbols based on this. 

	have all sorts of refactoring tricks.
		rename
		wrap in a function
		essentially what I want to do is write a go command
			then refactor all of the stuff so I can move it to a package
		add/remove all the necessary packages.	

	need all the *global* func names and struct names

	so we can find all the global funcs. 
	I need to be able to do it that way. 
	when you write the initial program. 


	if there's a word
		get the package from the current word
	if there's nothing here,
		get all the files from the Makefile in this directory
		make a list of all of their symbols.

	Here's another trick. 
		the autocompleter should return a function with the proper return. 
		a function call you should be able to call a function with the proper values and returns automatically. 

		for function pointers and function values figure out all of the things that match those two. 

	Need it to create my gomakefile
		that is relatively simple.
	Need it to create my tests too
		test creation is just a matter of writing a bunch of test cases for my particular function. 
			reflect on my own source code. 
	Need to make a codereview bundle

	Well now we have the kind of auto completion i want. 
	but what do want 

	now I have function autocomplete. 
	NEXT: get lookup so I can find definitions of bits of code. 
	Let's test this out. 
	BUG: textmate needs you to save before it will autocomplete for you. 

	when you are doing function arguments, autocomplete based on the type of the function.
		how do you do that? call gofunc with the context you are in. 
		gofunc is like gofmt, except that the context of your go stuff in the parsed program will give you different candidates.


	take the usual patterns, a web server, a daemon, a 9p server, a shell command, a go parse tree traverser. 

*/
/*
type ${finderName:$0} struct {}
func (f *${finderName:$0}) Visit(node interface{}) ast.Visitor {
	fd, ok := s.(*ast.FuncDecl)
	if !ok {
		return nil
	}
	fd := w.(*ast.FuncDecl)
	fmt.Printf("%v\n", fd.Name.Name)
	return f
}
ast.Walk(&f, m)
*/
func isGoFile(f *os.FileInfo) bool {
	return strings.HasSuffix(f.Name, ".go") && !strings.HasSuffix(f.Name, "_test.go")
}

type importFinder struct {
	imports []string
}

func (f *importFinder) Visit(node interface{}) ast.Visitor {
	is, ok := node.(*ast.ImportSpec)
	if ok {
		f.imports = append(f.imports, strings.Trim(string(is.Path.Value), "\""))
	}
	return f
}

func importsInDir(path string) (imports []string, err os.Error) {
	m, err := parser.ParseDir(path, isGoFile, parser.ImportsOnly)
	if err != nil {
		return
	}
	var f importFinder
	for _, pkg := range m {
		ast.Walk(&f, pkg)
	}
	imports = f.imports
	return
}

type funcFinder struct{ funcs map[string]int }

func (f *funcFinder) Visit(node interface{}) ast.Visitor {
	switch node.(type) {
	case *ast.FuncDecl:
		f.funcs[node.(*ast.FuncDecl).Name.Name]=1
	case *ast.File:
		ast.FileExports(node.(*ast.File))
	}
	return f
}

func funcsInPkg(pkg string) os.Error {
	m, err := parser.ParseDir(pkg, isGoFile, 0)
	if err != nil {
		return err
	}
	f := &funcFinder{make(map[string]int)}
	for _, pkg := range m {
		ast.Walk(f, pkg)
	}
	for s := range f.funcs {
		fmt.Println(s)
	}
	return nil
}



func main() {
	flag.Usage = usage
	flag.Parse()

	basedir := ""
	pkg := ""
	switch flag.NArg() {
	case 0:
		basedir = os.Getenv("GOROOT") + "/src/cmd/gofunc"
	case 1:
		basedir = flag.Arg(0)
	case 2:
		basedir = flag.Arg(0)
		pkg = flag.Arg(1)
	default:
		flag.Usage()
	}
	if pkg != "" {
		err := funcsInPkg(os.Getenv("GOROOT") + "/src/pkg/" + pkg)
		if err != nil {
			log.Exit(err)
		}
		
		log.Exit(0)
	}
	imports, err := importsInDir(basedir)
	if err != nil {
		log.Exit(err)
	}

	for _, pkg := range imports {
		err = funcsInPkg(os.Getenv("GOROOT") + "/src/pkg/" + pkg)
		if err != nil {
			log.Exit(err)
		}
	}
	os.Exit(0)
}
