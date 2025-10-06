package app

import (
	"fmt"
	"os"

	"github.com/phillarmonic/drun/internal/debug"
	"github.com/phillarmonic/drun/internal/lexer"
	"github.com/phillarmonic/drun/internal/parser"
)

// Domain: Debug Mode
// This file contains logic for debugging drun files (tokens, AST, errors)

// HandleDebugMode handles debug mode execution
func HandleDebugMode(
	configFile string,
	debugInput string,
	debugFull bool,
	debugTokens bool,
	debugAST bool,
	debugJSON bool,
	debugErrors bool,
) error {
	var content string

	// Get content from input string or file
	if debugInput != "" {
		content = debugInput
	} else {
		// Determine the config file to use
		actualConfigFile, err := FindConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("no drun task file found for debugging: %w\n\nTo get started:\n  drun --init          # Create .drun/spec.drun", err)
		}

		// Read the drun file
		data, err := os.ReadFile(actualConfigFile)
		if err != nil {
			return fmt.Errorf("failed to read drun file '%s': %w", actualConfigFile, err)
		}
		content = string(data)
	}

	// Handle specific debug flags
	if debugFull {
		debug.DebugFull(content)
		return nil
	}

	// Handle individual debug flags
	hasSpecificFlag := debugTokens || debugAST || debugJSON || debugErrors

	if debugTokens {
		debug.DebugTokens(content)
	}

	if debugAST || debugJSON || debugErrors {
		// Parse without full debug output
		l := lexer.NewLexer(content)
		p := parser.NewParser(l)
		program := p.ParseProgram()
		parseErrors := p.Errors()

		if debugErrors {
			debug.DebugParseErrors(parseErrors)
		}

		if debugAST {
			debug.DebugAST(program)
		}

		if debugJSON {
			debug.DebugJSON(program)
		}
	}

	// If no specific debug flags were set, show full debug by default
	if !hasSpecificFlag {
		debug.DebugFull(content)
	}

	return nil
}
