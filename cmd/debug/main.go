package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/phillarmonic/drun/internal/v2/debug"
)

func main() {
	var (
		file   = flag.String("f", "", "drun file to debug")
		tokens = flag.Bool("tokens", false, "show lexer tokens")
		ast    = flag.Bool("ast", false, "show AST structure")
		json   = flag.Bool("json", false, "show AST as JSON")
		errors = flag.Bool("errors", false, "show parse errors only")
		full   = flag.Bool("full", false, "show full debug output")
		input  = flag.String("input", "", "direct input string to debug")
	)
	flag.Parse()

	var content string

	if *input != "" {
		content = *input
	} else if *file != "" {
		data, err := ioutil.ReadFile(*file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
		content = string(data)
	} else {
		fmt.Fprintf(os.Stderr, "Usage: debug -f <file> or -input <string> [options]\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *full {
		debug.DebugFull(content)
		return
	}

	if *tokens {
		debug.DebugTokens(content)
	}

	if *ast || *json || *errors {
		program, parseErrors := debug.DebugFull(content)

		if *errors {
			debug.DebugParseErrors(parseErrors)
		}

		if *ast {
			debug.DebugAST(program)
		}

		if *json {
			debug.DebugJSON(program)
		}
	}

	if !*tokens && !*ast && !*json && !*errors {
		// Default: show full debug
		debug.DebugFull(content)
	}
}
