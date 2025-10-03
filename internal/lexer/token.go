package lexer

import "fmt"

// TokenType represents the type of a token
type TokenType int

const (
	// Special tokens
	ILLEGAL TokenType = iota
	EOF

	// Literals
	STRING   // "hello world"
	NUMBER   // 2.0, 42
	BOOLEAN  // true, false
	VARIABLE // $variable

	// Keywords
	VERSION  // version
	TASK     // task
	MEANS    // means
	PROJECT  // project
	SET      // set
	INCLUDE  // include
	BEFORE   // before
	AFTER    // after
	ANY      // any
	DRUN     // drun
	SETUP    // setup
	TEARDOWN // teardown
	DEPENDS  // depends
	ON       // on
	THEN     // then
	AND      // and
	OR       // or
	CALL     // call

	// Code reuse keywords
	PARAMETER // parameter
	SNIPPET   // snippet
	TEMPLATE  // template
	MIXIN     // mixin
	USES      // uses
	INCLUDES  // includes
	USE       // use

	// Import selectors (plural forms)
	SNIPPETS  // snippets
	TEMPLATES // templates
	TASKS     // tasks

	// Docker keywords
	DOCKER    // docker
	IMAGE     // image
	CONTAINER // container
	COMPOSE   // compose
	BUILD     // build
	PUSH      // push
	PULL      // pull
	TAG       // tag
	REMOVE    // remove
	START     // start
	STOP      // stop
	UP        // up
	DOWN      // down
	SCALE     // scale
	PORT      // port
	REGISTRY  // registry

	// Git keywords
	GIT        // git
	CLONE      // clone
	INIT       // init
	BRANCH     // branch
	SWITCH     // switch
	MERGE      // merge
	ADD        // add
	COMMIT     // commit
	FETCH      // fetch
	STATUS     // status
	LOG        // log
	SHOW       // show
	REPOSITORY // repository
	REMOTE     // remote
	CHANGES    // changes
	MESSAGE    // message
	FILES      // files
	CURRENT    // current
	ALL        // all
	WITH       // with
	INTO       // into
	CHECKOUT   // checkout

	// HTTP keywords
	HTTP      // http
	HTTPS     // https
	GET       // get
	POST      // post
	PUT       // put
	PATCH     // patch
	HEAD      // head
	OPTIONS   // options
	REQUEST   // request
	RESPONSE  // response
	BODY      // body
	HEADERS   // headers
	HEADER    // header
	URL       // url
	ENDPOINT  // endpoint
	API       // api
	JSON      // json
	XML       // xml
	FORM      // form
	DATA      // data
	TIMEOUT   // timeout
	RETRY     // retry
	FOLLOW    // follow
	REDIRECTS // redirects
	VERIFY    // verify
	SSL       // ssl
	AUTH      // auth
	BEARER    // bearer
	BASIC     // basic
	TOKEN     // token
	USER      // user
	PASSWORD  // password
	CONTENT   // content
	TYPE      // type
	ACCEPT    // accept
	SEND      // send
	RECEIVE   // receive
	DOWNLOAD  // download
	UPLOAD    // upload

	// Network keywords
	HEALTH     // health
	SERVICE    // service
	WAIT       // wait
	READY      // ready
	OPEN       // open
	PING       // ping
	HOST       // host
	CONNECTION // connection
	TEST       // test
	AT         // at
	BE         // be
	EXPECT     // expect

	// Smart Detection keywords
	DETECT      // detect
	AVAILABLE   // available
	NOT         // not
	INSTALLED   // installed
	TOOL        // tool
	FRAMEWORK   // framework
	ENVIRONMENT // environment
	NODE        // node
	NPM         // npm
	YARN        // yarn
	PNPM        // pnpm
	BUN         // bun
	PYTHON      // python
	PIP         // pip
	GO          // go
	GOLANG      // golang
	CARGO       // cargo
	JAVA        // java
	MAVEN       // maven
	GRADLE      // gradle
	RUBY        // ruby
	GEM         // gem
	PHP         // php
	COMPOSER    // composer
	RUST        // rust
	MAKE        // make
	KUBECTL     // kubectl
	HELM        // helm
	TERRAFORM   // terraform
	AWS         // aws
	GCP         // gcp
	AZURE       // azure
	CI          // ci
	LOCAL       // local
	PRODUCTION  // production
	STAGING     // staging
	DEVELOPMENT // development
	REACT       // react
	VUE         // vue
	ANGULAR     // angular
	DJANGO      // django
	RAILS       // rails
	EXPRESS     // express
	SPRING      // spring
	LARAVEL     // laravel

	// Advanced Control Flow keywords
	RANGE    // range
	WHERE    // where
	BREAK    // break
	CONTINUE // continue
	CONTAINS // contains
	STARTS   // starts
	ENDS     // ends
	MATCHES  // matches
	MATCHING // matching
	LINE     // line
	MATCH    // match
	PATTERN  // pattern
	BETWEEN  // between
	EMAIL    // email
	FORMAT   // format

	// Variable Operations keywords
	LET       // let
	CONCAT    // concat
	SPLIT     // split
	REPLACE   // replace
	TRIM      // trim
	UPPERCASE // uppercase
	LOWERCASE // lowercase
	PREPEND   // prepend
	JOIN      // join
	SLICE     // slice
	LENGTH    // length
	KEYS      // keys
	VALUES    // values
	TRANSFORM // transform
	SUBTRACT  // subtract
	MULTIPLY  // multiply
	DIVIDE    // divide
	MODULO    // modulo
	PROPERTY  // property

	// Advanced Variable Operations
	WITHOUT   // without
	FILTERED  // filtered
	SORTED    // sorted
	REVERSED  // reversed
	UNIQUE    // unique
	FIRST     // first
	LAST      // last
	BASENAME  // basename
	DIRNAME   // dirname
	EXTENSION // extension
	PREFIX    // prefix
	SUFFIX    // suffix

	// Comparison operators
	GTE // >=
	GT  // >
	LTE // <=
	LT  // <
	EQ  // ==
	NE  // !=

	// Action keywords (built-in actions)
	INFO    // info
	STEP    // step
	WARN    // warn
	ERROR   // error
	SUCCESS // success
	FAIL    // fail
	ECHO    // echo

	// Parameter keywords
	REQUIRES // requires
	GIVEN    // given
	ACCEPTS  // accepts
	DEFAULTS // defaults
	TO       // to
	FROM     // from
	AS       // as
	LIST     // list
	OF       // of

	// Control flow keywords
	WHEN      // when
	IF        // if
	ELSE      // else
	OTHERWISE // otherwise
	FOR       // for
	EACH      // each
	IN        // in
	PARALLEL  // parallel
	IS        // is
	ARE       // are

	// Built-in functions/conditions
	EXISTS // exists
	EMPTY  // empty

	// Shell operations
	RUN     // run
	EXEC    // exec
	SHELL   // shell
	CAPTURE // capture
	OUTPUT  // output
	CONFIG  // config

	// Type keywords
	STRING_TYPE  // string
	NUMBER_TYPE  // number
	BOOLEAN_TYPE // boolean
	LIST_TYPE    // list

	// File operations
	CREATE      // create
	COPY        // copy
	MOVE        // move
	DELETE      // delete
	READ        // read
	WRITE       // write
	APPEND      // append
	DIR         // dir
	BACKUP      // backup
	CHECK       // check
	SIZE        // size
	DIRECTORY   // directory
	ALLOW       // allow
	PERMISSIONS // permissions
	EXTRACT     // extract
	ARCHIVE     // archive

	// Error handling
	TRY     // try
	CATCH   // catch
	FINALLY // finally
	THROW   // throw
	RETHROW // rethrow
	IGNORE  // ignore

	// Identifiers and operators
	IDENT  // user-defined identifiers
	ASSIGN // :
	PLUS   // +
	MINUS  // -
	STAR   // *
	SLASH  // /
	EQUALS // =

	// Punctuation
	COLON    // :
	COMMA    // ,
	LPAREN   // (
	RPAREN   // )
	LBRACE   // {
	RBRACE   // }
	LBRACKET // [
	RBRACKET // ]

	// Indentation (Python-style)
	INDENT
	DEDENT
	NEWLINE

	// Comments
	COMMENT           // # comment
	MULTILINE_COMMENT // /* comment */
)

// Token represents a single token
type Token struct {
	Type     TokenType
	Literal  string
	Line     int
	Column   int
	Position int
}

// String returns a string representation of the token type
func (t TokenType) String() string {
	switch t {
	case ILLEGAL:
		return "ILLEGAL"
	case EOF:
		return "EOF"
	case STRING:
		return "STRING"
	case NUMBER:
		return "NUMBER"
	case BOOLEAN:
		return "BOOLEAN"
	case VARIABLE:
		return "VARIABLE"
	case VERSION:
		return "VERSION"
	case TASK:
		return "TASK"
	case MEANS:
		return "MEANS"
	case PROJECT:
		return "PROJECT"
	case SET:
		return "SET"
	case INCLUDE:
		return "INCLUDE"
	case BEFORE:
		return "BEFORE"
	case AFTER:
		return "AFTER"
	case ANY:
		return "ANY"
	case DRUN:
		return "DRUN"
	case SETUP:
		return "SETUP"
	case TEARDOWN:
		return "TEARDOWN"
	case DEPENDS:
		return "DEPENDS"
	case ON:
		return "ON"
	case THEN:
		return "THEN"
	case AND:
		return "AND"
	case OR:
		return "OR"
	case CALL:
		return "CALL"
	case PARAMETER:
		return "PARAMETER"
	case SNIPPET:
		return "SNIPPET"
	case TEMPLATE:
		return "TEMPLATE"
	case MIXIN:
		return "MIXIN"
	case USES:
		return "USES"
	case INCLUDES:
		return "INCLUDES"
	case USE:
		return "USE"
	case SNIPPETS:
		return "SNIPPETS"
	case TEMPLATES:
		return "TEMPLATES"
	case TASKS:
		return "TASKS"
	case DOCKER:
		return "DOCKER"
	case IMAGE:
		return "IMAGE"
	case CONTAINER:
		return "CONTAINER"
	case COMPOSE:
		return "COMPOSE"
	case BUILD:
		return "BUILD"
	case PUSH:
		return "PUSH"
	case PULL:
		return "PULL"
	case TAG:
		return "TAG"
	case REMOVE:
		return "REMOVE"
	case START:
		return "START"
	case STOP:
		return "STOP"
	case UP:
		return "UP"
	case DOWN:
		return "DOWN"
	case SCALE:
		return "SCALE"
	case PORT:
		return "PORT"
	case REGISTRY:
		return "REGISTRY"
	case GIT:
		return "GIT"
	case CLONE:
		return "CLONE"
	case INIT:
		return "INIT"
	case BRANCH:
		return "BRANCH"
	case SWITCH:
		return "SWITCH"
	case MERGE:
		return "MERGE"
	case ADD:
		return "ADD"
	case COMMIT:
		return "COMMIT"
	case FETCH:
		return "FETCH"
	case STATUS:
		return "STATUS"
	case LOG:
		return "LOG"
	case SHOW:
		return "SHOW"
	case REPOSITORY:
		return "REPOSITORY"
	case REMOTE:
		return "REMOTE"
	case CHANGES:
		return "CHANGES"
	case MESSAGE:
		return "MESSAGE"
	case FILES:
		return "FILES"
	case CURRENT:
		return "CURRENT"
	case ALL:
		return "ALL"
	case WITH:
		return "WITH"
	case INTO:
		return "INTO"
	case CHECKOUT:
		return "CHECKOUT"
	case HTTP:
		return "HTTP"
	case HTTPS:
		return "HTTPS"
	case GET:
		return "GET"
	case POST:
		return "POST"
	case PUT:
		return "PUT"
	case PATCH:
		return "PATCH"
	case HEAD:
		return "HEAD"
	case OPTIONS:
		return "OPTIONS"
	case REQUEST:
		return "REQUEST"
	case RESPONSE:
		return "RESPONSE"
	case BODY:
		return "BODY"
	case HEADERS:
		return "HEADERS"
	case HEADER:
		return "HEADER"
	case URL:
		return "URL"
	case ENDPOINT:
		return "ENDPOINT"
	case API:
		return "API"
	case JSON:
		return "JSON"
	case XML:
		return "XML"
	case FORM:
		return "FORM"
	case DATA:
		return "DATA"
	case TIMEOUT:
		return "TIMEOUT"
	case RETRY:
		return "RETRY"
	case FOLLOW:
		return "FOLLOW"
	case REDIRECTS:
		return "REDIRECTS"
	case VERIFY:
		return "VERIFY"
	case SSL:
		return "SSL"
	case AUTH:
		return "AUTH"
	case BEARER:
		return "BEARER"
	case BASIC:
		return "BASIC"
	case TOKEN:
		return "TOKEN"
	case USER:
		return "USER"
	case PASSWORD:
		return "PASSWORD"
	case CONTENT:
		return "CONTENT"
	case TYPE:
		return "TYPE"
	case ACCEPT:
		return "ACCEPT"
	case SEND:
		return "SEND"
	case RECEIVE:
		return "RECEIVE"
	case DOWNLOAD:
		return "DOWNLOAD"
	case UPLOAD:
		return "UPLOAD"
	case HEALTH:
		return "HEALTH"
	case SERVICE:
		return "SERVICE"
	case WAIT:
		return "WAIT"
	case READY:
		return "READY"
	case OPEN:
		return "OPEN"
	case PING:
		return "PING"
	case HOST:
		return "HOST"
	case CONNECTION:
		return "CONNECTION"
	case TEST:
		return "TEST"
	case AT:
		return "AT"
	case BE:
		return "BE"
	case EXPECT:
		return "EXPECT"
	case DETECT:
		return "DETECT"
	case AVAILABLE:
		return "AVAILABLE"
	case NOT:
		return "NOT"
	case INSTALLED:
		return "INSTALLED"
	case TOOL:
		return "TOOL"
	case FRAMEWORK:
		return "FRAMEWORK"
	case ENVIRONMENT:
		return "ENVIRONMENT"
	case NODE:
		return "NODE"
	case NPM:
		return "NPM"
	case YARN:
		return "YARN"
	case PNPM:
		return "PNPM"
	case BUN:
		return "BUN"
	case PYTHON:
		return "PYTHON"
	case PIP:
		return "PIP"
	case GO:
		return "GO"
	case GOLANG:
		return "GOLANG"
	case CARGO:
		return "CARGO"
	case JAVA:
		return "JAVA"
	case MAVEN:
		return "MAVEN"
	case GRADLE:
		return "GRADLE"
	case RUBY:
		return "RUBY"
	case GEM:
		return "GEM"
	case PHP:
		return "PHP"
	case COMPOSER:
		return "COMPOSER"
	case RUST:
		return "RUST"
	case MAKE:
		return "MAKE"
	case KUBECTL:
		return "KUBECTL"
	case HELM:
		return "HELM"
	case TERRAFORM:
		return "TERRAFORM"
	case AWS:
		return "AWS"
	case GCP:
		return "GCP"
	case AZURE:
		return "AZURE"
	case CI:
		return "CI"
	case LOCAL:
		return "LOCAL"
	case PRODUCTION:
		return "PRODUCTION"
	case STAGING:
		return "STAGING"
	case DEVELOPMENT:
		return "DEVELOPMENT"
	case REACT:
		return "REACT"
	case VUE:
		return "VUE"
	case ANGULAR:
		return "ANGULAR"
	case DJANGO:
		return "DJANGO"
	case RAILS:
		return "RAILS"
	case EXPRESS:
		return "EXPRESS"
	case SPRING:
		return "SPRING"
	case LARAVEL:
		return "LARAVEL"
	case RANGE:
		return "RANGE"
	case WHERE:
		return "WHERE"
	case BREAK:
		return "BREAK"
	case CONTINUE:
		return "CONTINUE"
	case CONTAINS:
		return "CONTAINS"
	case STARTS:
		return "STARTS"
	case ENDS:
		return "ENDS"
	case MATCHES:
		return "MATCHES"
	case MATCHING:
		return "MATCHING"
	case LINE:
		return "LINE"
	case MATCH:
		return "MATCH"
	case PATTERN:
		return "PATTERN"
	case BETWEEN:
		return "BETWEEN"
	case EMAIL:
		return "EMAIL"
	case FORMAT:
		return "FORMAT"
	case LET:
		return "LET"
	case CONCAT:
		return "CONCAT"
	case SPLIT:
		return "SPLIT"
	case REPLACE:
		return "REPLACE"
	case TRIM:
		return "TRIM"
	case UPPERCASE:
		return "UPPERCASE"
	case LOWERCASE:
		return "LOWERCASE"
	case PREPEND:
		return "PREPEND"
	case JOIN:
		return "JOIN"
	case SLICE:
		return "SLICE"
	case LENGTH:
		return "LENGTH"
	case KEYS:
		return "KEYS"
	case VALUES:
		return "VALUES"
	case TRANSFORM:
		return "TRANSFORM"
	case SUBTRACT:
		return "SUBTRACT"
	case MULTIPLY:
		return "MULTIPLY"
	case DIVIDE:
		return "DIVIDE"
	case MODULO:
		return "MODULO"
	case PROPERTY:
		return "PROPERTY"
	case WITHOUT:
		return "WITHOUT"
	case FILTERED:
		return "FILTERED"
	case SORTED:
		return "SORTED"
	case REVERSED:
		return "REVERSED"
	case UNIQUE:
		return "UNIQUE"
	case FIRST:
		return "FIRST"
	case LAST:
		return "LAST"
	case BASENAME:
		return "BASENAME"
	case DIRNAME:
		return "DIRNAME"
	case EXTENSION:
		return "EXTENSION"
	case PREFIX:
		return "PREFIX"
	case SUFFIX:
		return "SUFFIX"
	case GTE:
		return "GTE"
	case GT:
		return "GT"
	case LTE:
		return "LTE"
	case LT:
		return "LT"
	case EQ:
		return "EQ"
	case NE:
		return "NE"
	case INFO:
		return "INFO"
	case STEP:
		return "STEP"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case SUCCESS:
		return "SUCCESS"
	case FAIL:
		return "FAIL"
	case ECHO:
		return "ECHO"
	case REQUIRES:
		return "REQUIRES"
	case GIVEN:
		return "GIVEN"
	case ACCEPTS:
		return "ACCEPTS"
	case DEFAULTS:
		return "DEFAULTS"
	case TO:
		return "TO"
	case FROM:
		return "FROM"
	case AS:
		return "AS"
	case LIST:
		return "LIST"
	case OF:
		return "OF"
	case WHEN:
		return "WHEN"
	case IF:
		return "IF"
	case ELSE:
		return "ELSE"
	case OTHERWISE:
		return "OTHERWISE"
	case FOR:
		return "FOR"
	case EACH:
		return "EACH"
	case IN:
		return "IN"
	case PARALLEL:
		return "PARALLEL"
	case IS:
		return "IS"
	case ARE:
		return "ARE"
	case EXISTS:
		return "EXISTS"
	case RUN:
		return "RUN"
	case EXEC:
		return "EXEC"
	case SHELL:
		return "SHELL"
	case CAPTURE:
		return "CAPTURE"
	case OUTPUT:
		return "OUTPUT"
	case CONFIG:
		return "CONFIG"
	case STRING_TYPE:
		return "STRING_TYPE"
	case NUMBER_TYPE:
		return "NUMBER_TYPE"
	case BOOLEAN_TYPE:
		return "BOOLEAN_TYPE"
	case LIST_TYPE:
		return "LIST_TYPE"
	case CREATE:
		return "CREATE"
	case COPY:
		return "COPY"
	case MOVE:
		return "MOVE"
	case DELETE:
		return "DELETE"
	case READ:
		return "READ"
	case WRITE:
		return "WRITE"
	case APPEND:
		return "APPEND"
	case DIR:
		return "DIR"
	case BACKUP:
		return "BACKUP"
	case CHECK:
		return "CHECK"
	case SIZE:
		return "SIZE"
	case DIRECTORY:
		return "DIRECTORY"
	case ALLOW:
		return "ALLOW"
	case PERMISSIONS:
		return "PERMISSIONS"
	case EXTRACT:
		return "EXTRACT"
	case ARCHIVE:
		return "ARCHIVE"
	case TRY:
		return "TRY"
	case CATCH:
		return "CATCH"
	case FINALLY:
		return "FINALLY"
	case THROW:
		return "THROW"
	case RETHROW:
		return "RETHROW"
	case IGNORE:
		return "IGNORE"
	case IDENT:
		return "IDENT"
	case ASSIGN:
		return "ASSIGN"
	case PLUS:
		return "PLUS"
	case MINUS:
		return "MINUS"
	case STAR:
		return "STAR"
	case SLASH:
		return "SLASH"
	case EQUALS:
		return "EQUALS"
	case COLON:
		return "COLON"
	case COMMA:
		return "COMMA"
	case LPAREN:
		return "LPAREN"
	case RPAREN:
		return "RPAREN"
	case LBRACE:
		return "LBRACE"
	case RBRACE:
		return "RBRACE"
	case LBRACKET:
		return "LBRACKET"
	case RBRACKET:
		return "RBRACKET"
	case INDENT:
		return "INDENT"
	case DEDENT:
		return "DEDENT"
	case NEWLINE:
		return "NEWLINE"
	case COMMENT:
		return "COMMENT"
	case MULTILINE_COMMENT:
		return "MULTILINE_COMMENT"
	default:
		return "UNKNOWN"
	}
}

// String returns a string representation of the token
func (t Token) String() string {
	return fmt.Sprintf("Token{Type: %s, Literal: %q, Line: %d, Column: %d}",
		t.Type, t.Literal, t.Line, t.Column)
}

// Keywords maps string literals to their token types
var keywords = map[string]TokenType{
	"version":     VERSION,
	"task":        TASK,
	"means":       MEANS,
	"project":     PROJECT,
	"set":         SET,
	"include":     INCLUDE,
	"before":      BEFORE,
	"after":       AFTER,
	"any":         ANY,
	"drun":        DRUN,
	"setup":       SETUP,
	"teardown":    TEARDOWN,
	"depends":     DEPENDS,
	"on":          ON,
	"then":        THEN,
	"and":         AND,
	"or":          OR,
	"call":        CALL,
	"parameter":   PARAMETER,
	"snippet":     SNIPPET,
	"template":    TEMPLATE,
	"mixin":       MIXIN,
	"uses":        USES,
	"includes":    INCLUDES,
	"use":         USE,
	"snippets":    SNIPPETS,
	"templates":   TEMPLATES,
	"tasks":       TASKS,
	"docker":      DOCKER,
	"image":       IMAGE,
	"container":   CONTAINER,
	"compose":     COMPOSE,
	"build":       BUILD,
	"push":        PUSH,
	"pull":        PULL,
	"tag":         TAG,
	"remove":      REMOVE,
	"start":       START,
	"stop":        STOP,
	"up":          UP,
	"down":        DOWN,
	"scale":       SCALE,
	"port":        PORT,
	"registry":    REGISTRY,
	"git":         GIT,
	"clone":       CLONE,
	"init":        INIT,
	"branch":      BRANCH,
	"switch":      SWITCH,
	"merge":       MERGE,
	"add":         ADD,
	"commit":      COMMIT,
	"fetch":       FETCH,
	"status":      STATUS,
	"log":         LOG,
	"show":        SHOW,
	"repository":  REPOSITORY,
	"remote":      REMOTE,
	"changes":     CHANGES,
	"message":     MESSAGE,
	"files":       FILES,
	"current":     CURRENT,
	"all":         ALL,
	"with":        WITH,
	"into":        INTO,
	"checkout":    CHECKOUT,
	"http":        HTTP,
	"https":       HTTPS,
	"get":         GET,
	"post":        POST,
	"put":         PUT,
	"patch":       PATCH,
	"head":        HEAD,
	"options":     OPTIONS,
	"request":     REQUEST,
	"response":    RESPONSE,
	"body":        BODY,
	"headers":     HEADERS,
	"header":      HEADER,
	"url":         URL,
	"endpoint":    ENDPOINT,
	"api":         API,
	"json":        JSON,
	"xml":         XML,
	"form":        FORM,
	"data":        DATA,
	"timeout":     TIMEOUT,
	"retry":       RETRY,
	"follow":      FOLLOW,
	"redirects":   REDIRECTS,
	"verify":      VERIFY,
	"ssl":         SSL,
	"auth":        AUTH,
	"bearer":      BEARER,
	"basic":       BASIC,
	"token":       TOKEN,
	"user":        USER,
	"password":    PASSWORD,
	"content":     CONTENT,
	"type":        TYPE,
	"accept":      ACCEPT,
	"send":        SEND,
	"receive":     RECEIVE,
	"download":    DOWNLOAD,
	"upload":      UPLOAD,
	"health":      HEALTH,
	"service":     SERVICE,
	"wait":        WAIT,
	"ready":       READY,
	"open":        OPEN,
	"ping":        PING,
	"host":        HOST,
	"connection":  CONNECTION,
	"test":        TEST,
	"at":          AT,
	"be":          BE,
	"expect":      EXPECT,
	"detect":      DETECT,
	"available":   AVAILABLE,
	"not":         NOT,
	"installed":   INSTALLED,
	"tool":        TOOL,
	"framework":   FRAMEWORK,
	"environment": ENVIRONMENT,
	"node":        NODE,
	"npm":         NPM,
	"yarn":        YARN,
	"pnpm":        PNPM,
	"bun":         BUN,
	"python":      PYTHON,
	"pip":         PIP,
	"go":          GO,
	"golang":      GOLANG,
	"cargo":       CARGO,
	"java":        JAVA,
	"maven":       MAVEN,
	"gradle":      GRADLE,
	"ruby":        RUBY,
	"gem":         GEM,
	"php":         PHP,
	"composer":    COMPOSER,
	"rust":        RUST,
	"make":        MAKE,
	"kubectl":     KUBECTL,
	"helm":        HELM,
	"terraform":   TERRAFORM,
	"aws":         AWS,
	"gcp":         GCP,
	"azure":       AZURE,
	"ci":          CI,
	"local":       LOCAL,
	"production":  PRODUCTION,
	"staging":     STAGING,
	"development": DEVELOPMENT,
	"react":       REACT,
	"vue":         VUE,
	"angular":     ANGULAR,
	"django":      DJANGO,
	"rails":       RAILS,
	"express":     EXPRESS,
	"spring":      SPRING,
	"laravel":     LARAVEL,
	"range":       RANGE,
	"where":       WHERE,
	"break":       BREAK,
	"continue":    CONTINUE,
	"contains":    CONTAINS,
	"starts":      STARTS,
	"ends":        ENDS,
	"matches":     MATCHES,
	"matching":    MATCHING,
	"line":        LINE,
	"match":       MATCH,
	"pattern":     PATTERN,
	"between":     BETWEEN,
	"email":       EMAIL,
	"format":      FORMAT,
	"let":         LET,
	"concat":      CONCAT,
	"split":       SPLIT,
	"replace":     REPLACE,
	"trim":        TRIM,
	"uppercase":   UPPERCASE,
	"lowercase":   LOWERCASE,
	"prepend":     PREPEND,
	"join":        JOIN,
	"slice":       SLICE,
	"length":      LENGTH,
	"keys":        KEYS,
	"values":      VALUES,
	"transform":   TRANSFORM,
	"subtract":    SUBTRACT,
	"multiply":    MULTIPLY,
	"divide":      DIVIDE,
	"modulo":      MODULO,
	"property":    PROPERTY,
	"without":     WITHOUT,
	"filtered":    FILTERED,
	"sorted":      SORTED,
	"reversed":    REVERSED,
	"unique":      UNIQUE,
	"first":       FIRST,
	"last":        LAST,
	"basename":    BASENAME,
	"dirname":     DIRNAME,
	"extension":   EXTENSION,
	"prefix":      PREFIX,
	"suffix":      SUFFIX,
	">=":          GTE,
	">":           GT,
	"<=":          LTE,
	"<":           LT,
	"==":          EQ,
	"!=":          NE,
	"info":        INFO,
	"step":        STEP,
	"warn":        WARN,
	"error":       ERROR,
	"success":     SUCCESS,
	"fail":        FAIL,
	"echo":        ECHO,
	"requires":    REQUIRES,
	"given":       GIVEN,
	"accepts":     ACCEPTS,
	"defaults":    DEFAULTS,
	"to":          TO,
	"from":        FROM,
	"as":          AS,
	"list":        LIST,
	"of":          OF,
	"when":        WHEN,
	"if":          IF,
	"else":        ELSE,
	"otherwise":   OTHERWISE,
	"for":         FOR,
	"each":        EACH,
	"in":          IN,
	"parallel":    PARALLEL,
	"is":          IS,
	"are":         ARE,
	"exists":      EXISTS,
	"empty":       EMPTY,
	"run":         RUN,
	"exec":        EXEC,
	"shell":       SHELL,
	"capture":     CAPTURE,
	"output":      OUTPUT,
	"config":      CONFIG,
	"string":      STRING_TYPE,
	"number":      NUMBER_TYPE,
	"boolean":     BOOLEAN_TYPE,
	"create":      CREATE,
	"copy":        COPY,
	"move":        MOVE,
	"delete":      DELETE,
	"read":        READ,
	"write":       WRITE,
	"append":      APPEND,
	"dir":         DIR,
	"backup":      BACKUP,
	"check":       CHECK,
	"size":        SIZE,
	"directory":   DIRECTORY,
	"allow":       ALLOW,
	"permissions": PERMISSIONS,
	"extract":     EXTRACT,
	"archive":     ARCHIVE,
	"try":         TRY,
	"catch":       CATCH,
	"finally":     FINALLY,
	"throw":       THROW,
	"rethrow":     RETHROW,
	"ignore":      IGNORE,
	"true":        BOOLEAN,
	"false":       BOOLEAN,
}

// LookupIdent checks if an identifier is a keyword
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
