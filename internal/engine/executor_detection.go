package engine

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/detection"
)

// Domain: Detection Operations Execution
// This file contains executors for:
// - Tool and command detection
// - Environment and system detection

// executeDetection executes smart detection operations
func (e *Engine) executeDetection(detectionStmt *ast.DetectionStatement, ctx *ExecutionContext) error {
	detector := detection.NewDetector()

	switch detectionStmt.Type {
	case "detect":
		return e.executeDetectOperation(detector, detectionStmt, ctx)
	case "detect_available":
		return e.executeDetectAvailable(detector, detectionStmt, ctx)
	case "if_available":
		return e.executeIfAvailable(detector, detectionStmt, ctx)
	case "if_version":
		return e.executeIfVersion(detector, detectionStmt, ctx)
	case "when_environment":
		return e.executeWhenEnvironment(detector, detectionStmt, ctx)
	default:
		return fmt.Errorf("unknown detection type: %s", detectionStmt.Type)
	}
}
