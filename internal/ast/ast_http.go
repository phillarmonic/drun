package ast

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/lexer"
)

// HTTPStatement represents HTTP operations
type HTTPStatement struct {
	Token   lexer.Token
	Method  string
	URL     string
	Body    string
	Headers map[string]string
	Auth    map[string]string
	Options map[string]string
}

func (hs *HTTPStatement) statementNode() {}
func (hs *HTTPStatement) String() string {
	out := strings.ToLower(hs.Method) + " request"

	if hs.URL != "" {
		out += fmt.Sprintf(" to \"%s\"", hs.URL)
	}

	for key, value := range hs.Headers {
		out += fmt.Sprintf(" with header \"%s: %s\"", key, value)
	}

	if hs.Body != "" {
		out += fmt.Sprintf(" with body \"%s\"", hs.Body)
	}

	for key, value := range hs.Auth {
		out += fmt.Sprintf(" with %s \"%s\"", key, value)
	}

	for key, value := range hs.Options {
		out += fmt.Sprintf(" %s \"%s\"", key, value)
	}

	return out
}

// DownloadStatement represents file download operations (like curl/wget)
type DownloadStatement struct {
	Token            lexer.Token
	URL              string
	Path             string
	AllowOverwrite   bool
	AllowPermissions []PermissionSpec
	ExtractTo        string
	RemoveArchive    bool
	Headers          map[string]string
	Auth             map[string]string
	Options          map[string]string
}

func (ds *DownloadStatement) statementNode() {}
func (ds *DownloadStatement) String() string {
	out := "download \"" + ds.URL + "\""

	if ds.ExtractTo != "" {
		out += " extract to \"" + ds.ExtractTo + "\""
		if ds.RemoveArchive {
			out += " remove archive"
		}
	} else {
		out += " to \"" + ds.Path + "\""
	}

	if ds.AllowOverwrite {
		out += " allow overwrite"
	}

	for _, perm := range ds.AllowPermissions {
		out += " allow permissions ["
		for i, p := range perm.Permissions {
			if i > 0 {
				out += ","
			}
			out += "\"" + p + "\""
		}
		out += "] to ["
		for i, t := range perm.Targets {
			if i > 0 {
				out += ","
			}
			out += "\"" + t + "\""
		}
		out += "]"
	}

	for key, value := range ds.Headers {
		out += fmt.Sprintf(" with header \"%s: %s\"", key, value)
	}

	for key, value := range ds.Auth {
		out += fmt.Sprintf(" with %s \"%s\"", key, value)
	}

	for key, value := range ds.Options {
		out += fmt.Sprintf(" %s \"%s\"", key, value)
	}

	return out
}

// PermissionSpec represents a permission specification for downloaded files
type PermissionSpec struct {
	Permissions []string
	Targets     []string
}
