package statements

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/lexer"
)

// FileStatement represents file system operations
type FileStatement struct {
	Token      lexer.Token
	Action     string
	Target     string
	Source     string
	Content    string
	IsDir      bool
	CaptureVar string
}

func (fs *FileStatement) statementNode() {}
func (fs *FileStatement) String() string {
	switch fs.Action {
	case "create":
		if fs.IsDir {
			return fmt.Sprintf("create dir \"%s\"", fs.Target)
		}
		return fmt.Sprintf("create file \"%s\"", fs.Target)
	case "copy":
		return fmt.Sprintf("copy \"%s\" to \"%s\"", fs.Source, fs.Target)
	case "move":
		return fmt.Sprintf("move \"%s\" to \"%s\"", fs.Source, fs.Target)
	case "delete":
		if fs.IsDir {
			return fmt.Sprintf("delete dir \"%s\"", fs.Target)
		}
		return fmt.Sprintf("delete file \"%s\"", fs.Target)
	case "read":
		if fs.CaptureVar != "" {
			return fmt.Sprintf("read file \"%s\" as %s", fs.Target, fs.CaptureVar)
		}
		return fmt.Sprintf("read file \"%s\"", fs.Target)
	case "write":
		return fmt.Sprintf("write \"%s\" to file \"%s\"", fs.Content, fs.Target)
	case "append":
		return fmt.Sprintf("append \"%s\" to file \"%s\"", fs.Content, fs.Target)
	default:
		return fmt.Sprintf("%s \"%s\"", fs.Action, fs.Target)
	}
}
