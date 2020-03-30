package atreugo

import (
	"net"
	"os"
	"time"

	fastrouter "github.com/fasthttp/router"
	logger "github.com/savsgio/go-logger"
	"github.com/savsgio/gotils/nocopy"
	"github.com/valyala/fasthttp"
)

// Atreugo implements high performance HTTP server
//
// It is prohibited copying Atreugo values. Create new values instead.
type Atreugo struct {
	noCopy nocopy.NoCopy // nolint:structcheck,unused

	server *fasthttp.Server
	log    *logger.Logger
	cfg    Config

	*Router
}

// Router dispatchs requests to different
// views via configurable routes (paths)
//
// It is prohibited copying Router values. Create new values instead.
type Router struct {
	noCopy nocopy.NoCopy // nolint:structcheck,unused

	router    *fastrouter.Router
	parent    *Router
	beginPath string

	errorView ErrorView

	paths             []*Path
	customOptionsURLS []string
	middlewares       Middlewares

	log *logger.Logger
}

// Path configuration of the registered view
//
// It is prohibited copying Path values.
type Path struct {
	noCopy nocopy.NoCopy // nolint:structcheck,unused

	handlerBuilder func(View, Middlewares) fasthttp.RequestHandler

	method      string
	url         string
	view        View
	middlewares Middlewares

	withTimeout bool
	timeout     time.Duration
	timeoutMsg  string
	timeoutCode int
}

// Config configuration to run server
//
// Default settings should satisfy the majority of Server users.
// Adjust Server settings only if you really understand the consequences.
type Config struct { // nolint:maligned
	Addr string

	// TLS/SSL options
	TLSEnable bool
	CertKey   string
	CertFile  string

	// Server name for sending in response headers. (default: Atreugo)
	Name string

	// Default: atreugo
	LogName string

	// See levels in https://github.com/savsgio/go-logger#levels
	LogLevel string

	// Kind of network listener (default: tcp4)
	// The network must be "tcp", "tcp4", "tcp6" or "unix".
	Network string

	// Compress transparently the response body generated by handler if the request contains 'gzip' or 'deflate'
	// in 'Accept-Encoding' header.
	Compress bool

	// Run server with a TCP listener with SO_REUSEPORT option set.
	// Just supported tcp4 and tcp6, and not with Windows OS.
	//
	// It is recommended to scale linearly, executing several instances (as many as usable logical CPUs)
	// of the same server in different processes, significantly increasing performance.
	//
	//It is IMPORTANT that each of these processes be executed with GOMAXPROCS=1.
	Reuseport bool

	// Shutdown gracefully shuts down the server without interrupting any active connections.
	// Shutdown works by first closing all open listeners and then waiting indefinitely for all connections
	// to return to idle and then shut down.
	GracefulShutdown bool

	// Configurable view which is called when no matching route is
	// found. If it is not set, http.NotFound is used.
	NotFoundView View

	// Configurable view which is called when a request
	// cannot be routed.
	// If it is not set, http.Error with http.StatusMethodNotAllowed is used.
	// The "Allow" header with allowed request methods is set before the handler
	// is called.
	MethodNotAllowedView View

	// Function to handle error returned by view.
	// It should be used to generate a error page and return the http error code
	// 500 (Internal Server Error).
	ErrorView ErrorView

	// Function to handle panics recovered from views.
	// It should be used to generate a error page and return the http error code
	// 500 (Internal Server Error).
	// The handler can be used to keep your server from crashing because of
	// unrecovered panics.
	PanicView PanicView

	//
	// --- fasthttp server configuration ---
	//

	// HeaderReceived is called after receiving the header
	//
	// non zero RequestConfig field values will overwrite the default configs
	HeaderReceived func(header *fasthttp.RequestHeader) fasthttp.RequestConfig

	// The maximum number of concurrent connections the server may serve.
	//
	// DefaultConcurrency is used if not set.
	Concurrency int

	// Whether to disable keep-alive connections.
	//
	// The server will close all the incoming connections after sending
	// the first response to client if this option is set to true.
	//
	// By default keep-alive connections are enabled.
	DisableKeepalive bool

	// Per-connection buffer size for requests' reading.
	// This also limits the maximum header size.
	//
	// Increase this buffer if your clients send multi-KB RequestURIs
	// and/or multi-KB headers (for example, BIG cookies).
	//
	// Default buffer size is used if not set.
	ReadBufferSize int

	// Per-connection buffer size for responses' writing.
	//
	// Default buffer size is used if not set.
	WriteBufferSize int

	// ReadTimeout is the amount of time allowed to read
	// the full request including body. The connection's read
	// deadline is reset when the connection opens, or for
	// keep-alive connections after the first byte has been read.
	//
	// By default request read timeout is unlimited.
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out
	// writes of the response. It is reset after the request handler
	// has returned.
	//
	// By default response write timeout is unlimited.
	WriteTimeout time.Duration

	// IdleTimeout is the maximum amount of time to wait for the
	// next request when keep-alive is enabled. If IdleTimeout
	// is zero, the value of ReadTimeout is used.
	IdleTimeout time.Duration

	// Maximum number of concurrent client connections allowed per IP.
	//
	// By default unlimited number of concurrent connections
	// may be established to the server from a single IP address.
	MaxConnsPerIP int

	// Maximum number of requests served per connection.
	//
	// The server closes connection after the last request.
	// 'Connection: close' header is added to the last response.
	//
	// By default unlimited number of requests may be served per connection.
	MaxRequestsPerConn int

	// Maximum request body size.
	//
	// The server rejects requests with bodies exceeding this limit.
	//
	// Request body size is limited by DefaultMaxRequestBodySize by default.
	MaxRequestBodySize int

	// Aggressively reduces memory usage at the cost of higher CPU usage
	// if set to true.
	//
	// Try enabling this option only if the server consumes too much memory
	// serving mostly idle keep-alive connections. This may reduce memory
	// usage by more than 50%.
	//
	// Aggressive memory usage reduction is disabled by default.
	ReduceMemoryUsage bool

	// Rejects all non-GET requests if set to true.
	//
	// This option is useful as anti-DoS protection for servers
	// accepting only GET requests. The request size is limited
	// by ReadBufferSize if GetOnly is set.
	//
	// Server accepts all the requests by default.
	GetOnly bool

	// Logs all errors, including the most frequent
	// 'connection reset by peer', 'broken pipe' and 'connection timeout'
	// errors. Such errors are common in production serving real-world
	// clients.
	//
	// By default the most frequent errors such as
	// 'connection reset by peer', 'broken pipe' and 'connection timeout'
	// are suppressed in order to limit output log traffic.
	LogAllErrors bool

	// Header names are passed as-is without normalization
	// if this option is set.
	//
	// Disabled header names' normalization may be useful only for proxying
	// incoming requests to other servers expecting case-sensitive
	// header names. See https://github.com/valyala/fasthttp/issues/57
	// for details.
	//
	// By default request and response header names are normalized, i.e.
	// The first letter and the first letters following dashes
	// are uppercased, while all the other letters are lowercased.
	// Examples:
	//
	//     * HOST -> Host
	//     * content-type -> Content-Type
	//     * cONTENT-lenGTH -> Content-Length
	DisableHeaderNamesNormalizing bool

	// SleepWhenConcurrencyLimitsExceeded is a duration to be slept of if
	// the concurrency limit in exceeded (default [when is 0]: don't sleep
	// and accept new connections immidiatelly).
	SleepWhenConcurrencyLimitsExceeded time.Duration

	// NoDefaultServerHeader, when set to true, causes the default Server header
	// to be excluded from the Response.
	//
	// The default Server header value is the value of the Name field or an
	// internal default value in its absence. With this option set to true,
	// the only time a Server header will be sent is if a non-zero length
	// value is explicitly provided during a request.
	NoDefaultServerHeader bool

	// NoDefaultContentType, when set to true, causes the default Content-Type
	// header to be excluded from the Response.
	//
	// The default Content-Type header value is the internal default value. When
	// set to true, the Content-Type will not be present.
	NoDefaultContentType bool

	// ConnState specifies an optional callback function that is
	// called when a client connection changes state. See the
	// ConnState type and associated constants for details.
	ConnState func(net.Conn, fasthttp.ConnState)

	// KeepHijackedConns is an opt-in disable of connection
	// close by fasthttp after connections' HijackHandler returns.
	// This allows to save goroutines, e.g. when fasthttp used to upgrade
	// http connections to WS and connection goes to another handler,
	// which will close it when needed.
	KeepHijackedConns bool

	socketFileMode os.FileMode
}

// StaticFS represents settings for serving static files
// from the local filesystem.
//
// It is prohibited copying StaticFS values. Create new values instead.
type StaticFS struct {
	noCopy nocopy.NoCopy // nolint:structcheck,unused

	// Path to the root directory to serve files from.
	Root string

	// List of index file names to try opening during directory access.
	//
	// For example:
	//
	//     * index.html
	//     * index.htm
	//     * my-super-index.xml
	//
	// By default the list is empty.
	IndexNames []string

	// Index pages for directories without files matching IndexNames
	// are automatically generated if set.
	//
	// Directory index generation may be quite slow for directories
	// with many files (more than 1K), so it is discouraged enabling
	// index pages' generation for such directories.
	//
	// By default index pages aren't generated.
	GenerateIndexPages bool

	// Transparently compresses responses if set to true.
	//
	// The server tries minimizing CPU usage by caching compressed files.
	// It adds CompressedFileSuffix suffix to the original file name and
	// tries saving the resulting compressed file under the new file name.
	// So it is advisable to give the server write access to Root
	// and to all inner folders in order to minimize CPU usage when serving
	// compressed responses.
	//
	// Transparent compression is disabled by default.
	Compress bool

	// Enables byte range requests if set to true.
	//
	// Byte range requests are disabled by default.
	AcceptByteRange bool

	// Path rewriting function.
	//
	// By default request path is not modified.
	PathRewrite PathRewriteFunc

	// PathNotFound fires when file is not found in filesystem
	// this functions tries to replace "Cannot open requested path"
	// server response giving to the programmer the control of server flow.
	//
	// By default PathNotFound returns
	// "Cannot open requested path"
	PathNotFound View

	// Expiration duration for inactive file handlers.
	//
	// FSHandlerCacheDuration is used by default.
	CacheDuration time.Duration

	// Suffix to add to the name of cached compressed file.
	//
	// This value has sense only if Compress is set.
	//
	// FSCompressedFileSuffix is used by default.
	CompressedFileSuffix string
}

// RequestCtx context wrapper of fasthttp.RequestCtx to adds extra funtionality
//
// It is prohibited copying RequestCtx values. Create new values instead.
//
// View should avoid holding references to incoming RequestCtx and/or
// its' members after the return.
// If holding RequestCtx references after the return is unavoidable
// (for instance, ctx is passed to a separate goroutine and ctx lifetime cannot
// be controlled), then the View MUST call ctx.TimeoutError()
// before return.
//
// It is unsafe modifying/reading RequestCtx instance from concurrently
// running goroutines. The only exception is TimeoutError*, which may be called
// while other goroutines accessing RequestCtx.
type RequestCtx struct {
	noCopy nocopy.NoCopy // nolint:structcheck,unused

	next     bool
	skipView bool

	// Flag to avoid stack overflow when this context has been embedded in the attached context
	searchingOnAttachedCtx bool

	*fasthttp.RequestCtx
}

// View must process incoming requests.
type View func(*RequestCtx) error

// ErrorView must process error returned by view.
type ErrorView func(*RequestCtx, error, int)

// PanicView must process panics recovered from views, if it's defined in configuration.
type PanicView func(*RequestCtx, interface{})

// Middleware must process all incoming requests before/after defined views.
type Middleware View

// Middlewares is a collection of middlewares with the order of execution and which to skip
type Middlewares struct {
	Before []Middleware
	After  []Middleware
	Skip   []Middleware
}

// PathRewriteFunc must return new request path based on arbitrary ctx
// info such as ctx.Path().
//
// Path rewriter is used in StaticFS for translating the current request
// to the local filesystem path relative to StaticFS.Root.
//
// The returned path must not contain '/../' substrings due to security reasons,
// since such paths may refer files outside StaticFS.Root.
//
// The returned path may refer to ctx members. For example, ctx.Path().
type PathRewriteFunc func(ctx *RequestCtx) []byte

// JSON is a map whose key is a string and whose value an interface
type JSON map[string]interface{}
