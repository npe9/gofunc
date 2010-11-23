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
	goprint "go/printer"
	"go/token"
	"log"
	"reflect"
	"io"
)


/*
	make install && make test
*/

var (
	ErrPlace    = os.NewError("Im a placeholder")
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


// A FieldFilter may be provided to Fprint to control the output.
type FieldFilter func(name string, value reflect.Value) bool


// NotNilFilter returns true for field values that are not nil;
// it returns false otherwise.
func NotNilFilter(_ string, value reflect.Value) bool {
	v, ok := value.(interface {
		IsNil() bool
	})
	return !ok || !v.IsNil() || value.Type() == reflect.Typeof(token.Position{})
}


// Fprint prints the (sub-)tree starting at AST node x to w.
//
// A non-nil FieldFilter f may be provided to control the output:
// struct fields for which f(fieldname, fieldvalue) is true are
// are printed; all others are filtered from the output.
//
func Fprint(w io.Writer, x interface{}, f FieldFilter) (n int, err os.Error) {
	// setup printer
	p := printer{
		output: w,
		filter: f,
		ptrmap: make(map[interface{}]int),
		last:   '\n', // force printing of line number on first line
	}

	// install error handler
	defer func() {
		n = p.written
		if e := recover(); e != nil {
			err = e.(localError).err // re-panics if it's not a localError
		}
	}()

	// print x
	if x == nil {
		p.printf("nil\n")
		return
	}
	p.print(reflect.NewValue(x))
	p.printf("\n")

	return
}


// Print prints x to standard output, skipping nil fields.
// Print(x) is the same as Fprint(os.Stdout, x, NotNilFilter).
func Print(x interface{}) (int, os.Error) {
	return Fprint(os.Stdout, x, NotNilFilter)
}


type printer struct {
	output  io.Writer
	filter  FieldFilter
	ptrmap  map[interface{}]int // *reflect.PtrValue -> line number
	written int                 // number of bytes written to output
	indent  int                 // current indentation level
	last    byte                // the last byte processed by Write
	line    int                 // current line number
}


var indent = []byte("  ")

func (p *printer) Write(data []byte) (n int, err os.Error) {
	var m int
	for i, b := range data {
		// invariant: data[0:n] has been written
		if b == '\n' {
			m, err = p.output.Write(data[n : i+1])
			n += m
			if err != nil {
				return
			}
			p.line++
		} else if p.last == '\n' {
			for j := p.indent; j > 0; j-- {
				_, err = p.output.Write(indent)
				if err != nil {
					return
				}
			}
		}
		p.last = b
	}
	m, err = p.output.Write(data[n:])
	n += m
	return
}


// localError wraps locally caught os.Errors so we can distinguish
// them from genuine panics which we don't want to return as errors.
type localError struct {
	err os.Error
}


// printf is a convenience wrapper that takes care of print errors.
func (p *printer) printf(format string, args ...interface{}) {
	n, err := fmt.Fprintf(p, format, args...)
	p.written += n
	if err != nil {
		panic(localError{err})
	}
}


// Implementation note: Print is written for AST nodes but could be
// used to print arbitrary data structures; such a version should
// probably be in a different package.

func (p *printer) print(x reflect.Value) {
	// Note: This test is only needed because AST nodes
	//       embed a token.Position, and thus all of them
	//       understand the String() method (but it only
	//       applies to the Position field).
	// TODO: Should reconsider this AST design decision.


	if !NotNilFilter("", x) {
		p.printf("nil")
		return
	}

	switch v := x.(type) {
	case *reflect.InterfaceValue:
		p.print(v.Elem())

	case *reflect.MapValue:
		p.printf("%v {\n", x.Type())
		p.indent++
		for _, key := range v.Keys() {
			p.print(key)
			p.printf(": ")
			p.print(v.Elem(key))
		}
		p.indent--
		p.printf("}")

	case *reflect.PtrValue:
		p.printf("&")
		// type-checked ASTs may contain cycles - use ptrmap
		// to keep track of objects that have been printed
		// already and print the respective line number instead
		ptr := v.Interface()
		if line, exists := p.ptrmap[ptr]; exists {
			p.printf("(obj @ %d)", line)
		} else {
			p.ptrmap[ptr] = p.line
			p.print(v.Elem())
		}

	case *reflect.SliceValue:
		if s, ok := v.Interface().([]byte); ok {
			p.printf("%#v", s)
			return
		}
		p.indent++
		p.printf("%s{\n", v.Type())
		for i, n := 0, v.Len(); i < n; i++ {
			p.print(v.Elem(i))
			p.printf(",\n")
		}
		p.indent--
		p.printf("}")

	case *reflect.StructValue:
		if x.Type() == reflect.Typeof(token.Position{}) {
			p.printf("token.Position{}")
			return
		}

		p.printf("%v {\n", x.Type())
		p.indent++
		t := v.Type().(*reflect.StructType)
		for i, n := 0, t.NumField(); i < n; i++ {
			name := t.Field(i).Name
			value := v.Field(i)
			if p.filter == nil || p.filter(name, value) {
				p.printf("%s: ", name)
				p.print(value)
				p.printf(",\n")
			}
		}
		p.indent--
		p.printf("}")

	default:
		p.printf("%#v", x.Interface())
	}
}


/*
parse the file
get the functions
convert the function definitions to function calls. 

if the function call wants a return add it
if it's a no return just reformat the function call as a snippet.

We want to be able to go from an interface to a structure



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

type funcFinder struct {
	funcs map[string]int
}

func (f *funcFinder) Visit(node interface{}) ast.Visitor {
	switch node.(type) {
	case *ast.FuncDecl:
		// when you get the function you need to know the package.
		// if node.(*ast.FuncDecl).Name.Name != "Fprintf" {
		// 	return nil
		// }
		_, err := (&goprint.Config{0, 4, nil}).Fprint(os.Stdout, node)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			return nil
		}
		fmt.Println()
		Print(node)

		//Print(node)
	case *ast.ExprStmt:
		_, err := (&goprint.Config{0, 4, nil}).Fprint(os.Stdout, node)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			return nil
		}
		fmt.Println()
		n := node.(*ast.ExprStmt).X
		switch n.(type) {
		case *ast.CallExpr:
			switch n.(*ast.CallExpr).Fun.(type) {
			case *ast.SelectorExpr:
				Print(n)
			case *ast.Ident:
				Print(n)
			default:
				Print(n)
			}
		default:
			Print(n)
		}

		//	case *ast.File:
		//		ast.FileExports(node.(*ast.File))
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
		basedir = os.Getenv("GOROOT") + "/src/cmd/expfunc"
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

	/* parse yourself for function definitions */
	funcsInPkg(basedir)
	os.Exit(0)
	fn := &ast.FuncDecl {
	  Name: &ast.Ident {
	    NamePos: token.Position{},
	    Name: "fn",
	  },
	  Type: &ast.FuncType {
	    Func: token.Position{},
	    Params: &ast.FieldList {
	      Opening: token.Position{},
	      List: []*ast.Field{
	        &ast.Field {
	          Names: []*ast.Ident{
	            &ast.Ident {
	              NamePos: token.Position{},
	              Name: "i",
	            },
	          },
	          Type: &ast.Ident {
	            NamePos: token.Position{},
	            Name: "int",
	          },
	        },
	      },
	      Closing: token.Position{},
	    },
	    Results: &ast.FieldList {
	      Opening: token.Position{},
	      List: []*ast.Field{
	        &ast.Field {
	          Type: &ast.SelectorExpr {
	            X: &ast.Ident {
	              NamePos: token.Position{},
	              Name: "os",
	            },
	            Sel: &ast.Ident {
	              NamePos: token.Position{},
	              Name: "Error",
	            },
	          },
	        },
	      },
	      Closing: token.Position{},
	    },
	  },
	}



/*
	how to handle structures here? 
	we still have to traverse these guys. 
	
*/
	for _, value := range fn.Type.Params.List {
		// so you are stuck here. you can get it to be able to make its own thing. 
		// but now you are doing it so that it just works. well now I have the names. 
		// the real test will be to test and see if I can generate a valid snippet.
		// now how?
		fmt.Println(value.Names[0].Name)
	}
	for _, value := range fn.Type.Results.List {
//		Print(value)
		fmt.Println(value.Type.(*ast.SelectorExpr).Sel)
	}
	Print(fn)
	fmt.Println(fn.Name.Name)
	call :=   &ast.CallExpr {
        Fun: &ast.Ident {
          NamePos: token.Position{},
          Name: fn.Name.Name,
        },
        Lparen: token.Position{},
        Args: []ast.Expr{
        },
        Ellipsis: token.Position{},
        Rparen: token.Position{},
      }
	_, err := (&goprint.Config{0, 4, nil}).Fprint(os.Stdout, call)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		return
	}
	fmt.Println()
	Print(call)
	// _, err := (&goprint.Config{0, 4, nil}).Fprint(os.Stdout, fn)
	// if err != nil {
	// 	fmt.Fprint(os.Stderr, err)
	// 	return
	// }
	// fmt.Println()
	// Print(fn)
	os.Exit(0)
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
