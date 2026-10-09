package main

import (
	"errors"
	"fmt"
	"os"
	"ruli/checker"
	"ruli/evaluator"
	"ruli/lexer"
	"ruli/object"
	"ruli/parser"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: ruli <script>")
		os.Exit(1)
	}

	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	l := lexer.New(string(src))
	p := parser.New(l)
	program := p.ParseProgram()

	if errs := p.Errors(); len(errs) > 0 {
		parseErrs := make([]error, 0, len(errs))
		for _, e := range errs {
			parseErrs = append(parseErrs, &e)
		}
		return errors.Join(parseErrs...)
	}

	if errs := checker.Check(program); len(errs) > 0 {
		typeErrs := make([]error, 0, len(errs))
		for i := range errs {
			typeErrs = append(typeErrs, &errs[i])
		}
		return errors.Join(typeErrs...)
	}

	env := object.NewEnvironment()
	result := evaluator.Eval(program, env)
	if errObj, ok := result.(*object.Error); ok {
		return errors.New(errObj.Inspect())
	}

	return nil
}
