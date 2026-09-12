package ast

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// HTTPStatement represents HTTP operations
type HTTPStatement struct {
	Headers map[string]string
	Auth    map[string]string
	Options map[string]string
	Method  string
	URL     string
	Body    string
	Token   lexer.Token
}

func (hs *HTTPStatement) statementNode() {}
func (hs *HTTPStatement) String() string {
	out := strings.ToLower(hs.Method) + " request"

	if hs.URL != "" {
		out += fmt.Sprintf(" to \"%s\"", hs.URL)
	}

	var outSb29 strings.Builder
	for key, value := range hs.Headers {
		fmt.Fprintf(&outSb29, " with header \"%s: %s\"", key, value)
	}
	out += outSb29.String()

	if hs.Body != "" {
		out += fmt.Sprintf(" with body \"%s\"", hs.Body)
	}

	var outSb37 strings.Builder
	for key, value := range hs.Auth {
		fmt.Fprintf(&outSb37, " with %s \"%s\"", key, value)
	}
	out += outSb37.String()

	var outSb41 strings.Builder
	for key, value := range hs.Options {
		fmt.Fprintf(&outSb41, " %s \"%s\"", key, value)
	}
	out += outSb41.String()

	return out
}

// DownloadStatement represents file download operations (like curl/wget)
type DownloadStatement struct {
	Headers          map[string]string
	Auth             map[string]string
	Options          map[string]string
	URL              string
	Path             string
	ExtractTo        string
	AllowPermissions []PermissionSpec
	Token            lexer.Token
	AllowOverwrite   bool
	RemoveArchive    bool
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

	var outSb79 strings.Builder
	var outSb86 strings.Builder
	for _, perm := range ds.AllowPermissions {
		outSb79.WriteString(" allow permissions [")
		var outSb81 strings.Builder
		for i, p := range perm.Permissions {
			if i > 0 {
				outSb81.WriteString(",")
			}
			outSb81.WriteString("\"" + p + "\"")
		}
		outSb86.WriteString(outSb81.String())
		outSb79.WriteString("] to [")
		var outSb88 strings.Builder
		for i, t := range perm.Targets {
			if i > 0 {
				outSb88.WriteString(",")
			}
			outSb88.WriteString("\"" + t + "\"")
		}
		outSb86.WriteString(outSb88.String())
		outSb79.WriteString("]")
	}
	out += outSb86.String()
	out += outSb79.String()

	var outSb97 strings.Builder
	for key, value := range ds.Headers {
		fmt.Fprintf(&outSb97, " with header \"%s: %s\"", key, value)
	}
	out += outSb97.String()

	var outSb101 strings.Builder
	for key, value := range ds.Auth {
		fmt.Fprintf(&outSb101, " with %s \"%s\"", key, value)
	}
	out += outSb101.String()

	var outSb105 strings.Builder
	for key, value := range ds.Options {
		fmt.Fprintf(&outSb105, " %s \"%s\"", key, value)
	}
	out += outSb105.String()

	return out
}

// PermissionSpec represents a permission specification for downloaded files
type PermissionSpec struct {
	Permissions []string
	Targets     []string
}
