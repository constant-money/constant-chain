package common

// import (
// 	"bytes"
// 	"fmt"
// 	"io"
// 	"io/ioutil"
// 	"log"
// 	"os"
// 	"runtime"
// 	"strings"
// 	"sync"
// 	"sync/atomic"
// 	"time"

// 	"github.com/0xsirrush/color"
// )

// Logger is an interface which describes a level-based logger.  A default
// implementation of Logger is implemented by this package and can be created
// by calling (*Backend).Logger.
type Logger interface {
	// Tracef formats message according to format specifier and writes to
	// to log with LevelTrace.
	Tracef(format string, params ...interface{})

	// Debugf formats message according to format specifier and writes to
	// log with LevelDebug.
	Debugf(format string, params ...interface{})

	// Infof formats message according to format specifier and writes to
	// log with LevelInfo.
	Infof(format string, params ...interface{})

	// Warnf formats message according to format specifier and writes to
	// to log with LevelWarn.
	Warnf(format string, params ...interface{})

	// Errorf formats message according to format specifier and writes to
	// to log with LevelError.
	Errorf(format string, params ...interface{})

	// Criticalf formats message according to format specifier and writes to
	// log with LevelCritical.
	Criticalf(format string, params ...interface{})

	// Trace formats message using the default formats for its operands
	// and writes to log with LevelTrace.
	Trace(v ...interface{})

	// Debug formats message using the default formats for its operands
	// and writes to log with LevelDebug.
	Debug(v ...interface{})

	// Info formats message using the default formats for its operands
	// and writes to log with LevelInfo.
	Info(v ...interface{})

	// Warn formats message using the default formats for its operands
	// and writes to log with LevelWarn.
	Warn(v ...interface{})

	// Error formats message using the default formats for its operands
	// and writes to log with LevelError.
	Error(v ...interface{})

	// Critical formats message using the default formats for its operands
	// and writes to log with LevelCritical.
	Critical(v ...interface{})

	// Level returns the current logging level.
	Level() Level

	// SetLevel changes the logging level to the passed level.
	SetLevel(level Level)
}

// defaultFlags specifies changes to the default logger behavior.  It is set
// during package init and configured using the LOGFLAGS environment variable.
// Zero logger backends can override these default flags using WithFlags.
// var defaultFlags uint32 = 2

// // Flags to modify Backend's behavior.
// const (
// 	// Llongfile modifies the logger output to include full path and line number
// 	// of the logging callsite, e.g. /a/b/c/main.go:123.
// 	Llongfile uint32 = 1 << iota

// 	// Lshortfile modifies the logger output to include filename and line number
// 	// of the logging callsite, e.g. main.go:123.  Overrides Llongfile.
// 	Lshortfile
// )

// var (
// 	FgGreen   = color.New(color.FgGreen).FprintfFunc()
// 	FgRed     = color.New(color.FgRed).FprintfFunc()
// 	FgYellow  = color.New(color.FgYellow).FprintfFunc()
// 	FgBlue    = color.New(color.FgBlue).FprintfFunc()
// 	FgCyan    = color.New(color.FgCyan).FprintfFunc()
// 	FgMagenta = color.New(color.FgMagenta).FprintfFunc()
// 	FgWhite   = color.New(color.FgWhite).FprintfFunc()
// )

// // Read logger flags from the LOGFLAGS environment variable.  Multiple flags can
// // be set at once, separated by commas.
// func init() {
// 	for _, f := range strings.Split(os.Getenv("LOGFLAGS"), ",") {
// 		switch f {
// 		case "longfile":
// 			defaultFlags |= Llongfile
// 		case "shortfile":
// 			defaultFlags |= Lshortfile
// 		}
// 	}
// }

// Level is the level at which a logger is configured.  All messages sent
// to a level which is below the current level are filtered.
type Level uint32

// Level constants.
const (
	LevelTrace Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelCritical
	LevelOff
)

// levelStrs defines the human-readable names for each logging level.
// var levelStrs = [...]string{"TRC", "DBG", "INF", "WRN", "ERR", "CRT", "OFF"}

// // LevelFromString returns a level based on the input string s.  If the input
// // can't be interpreted as a valid log level, the info level and false is
// // returned.
// func LevelFromString(s string) (l Level, ok bool) {
// 	switch strings.ToLower(s) {
// 	case "trace", "trc":
// 		return LevelTrace, true
// 	case "debug", "dbg":
// 		return LevelDebug, true
// 	case "info", "inf":
// 		return LevelInfo, true
// 	case "warn", "wrn":
// 		return LevelWarn, true
// 	case "error", "err":
// 		return LevelError, true
// 	case "critical", "crt":
// 		return LevelCritical, true
// 	case "off":
// 		return LevelOff, true
// 	default:
// 		return LevelInfo, false
// 	}
// }

// // String returns the tag of the logger used in log messages, or "OFF" if
// // the level will not produce any log output.
// func (l Level) String() string {
// 	if l >= LevelOff {
// 		return "OFF"
// 	}
// 	return levelStrs[l]
// }

// // NewBackend creates a logger backend from a Writer.
// func NewBackend(w io.Writer, opts ...BackendOption) *Backend {
// 	b := &Backend{w: w, flag: defaultFlags}
// 	for _, o := range opts {
// 		o(b)
// 	}
// 	return b
// }

// // Backend is a logging backend.  Subsystems created from the backend write to
// // the backend's Writer.  Backend provides atomic writes to the Writer from all
// // subsystems.
// type Backend struct {
// 	w    io.Writer
// 	mu   sync.Mutex // ensures atomic writes
// 	flag uint32
// }

// // BackendOption is a function used to modify the behavior of a Backend.
// type BackendOption func(b *Backend)

// // WithFlags configures a Backend to use the specified flags rather than using
// // the package's defaults as determined through the LOGFLAGS environment
// // variable.
// func WithFlags(flags uint32) BackendOption {
// 	return func(b *Backend) {
// 		b.flag = flags
// 	}
// }

// // bufferPool defines a concurrent safe free list of byte slices used to provide
// // temporary buffers for formatting log messages prior to outputting them.
// var bufferPool = sync.Pool{
// 	New: func() interface{} {
// 		b := make([]byte, 0, 120)
// 		return &b // pointer to slice to avoid boxing alloc
// 	},
// }

// // buffer returns a byte slice from the free list.  A new buffer is allocated if
// // there are not any available on the free list.  The returned byte slice should
// // be returned to the fee list by using the recycleBuffer function when the
// // caller is done with it.
// func buffer() *[]byte {
// 	return bufferPool.Get().(*[]byte)
// }

// // recycleBuffer puts the provided byte slice, which should have been obtain via
// // the buffer function, back on the free list.
// func recycleBuffer(b *[]byte) {
// 	*b = (*b)[:0]
// 	bufferPool.Put(b)
// }

// // From stdlib log package.
// // Cheap integer to fixed-width decimal ASCII.  Give a negative width to avoid
// // zero-padding.
// func itoa(buf *[]byte, i int, wid int) {
// 	// Assemble decimal in reverse order.
// 	var b [20]byte
// 	bp := len(b) - 1
// 	for i >= 10 || wid > 1 {
// 		wid--
// 		q := i / 10
// 		b[bp] = byte('0' + i - q*10)
// 		bp--
// 		i = q
// 	}
// 	// i < 10
// 	b[bp] = byte('0' + i)
// 	*buf = append(*buf, b[bp:]...)
// }

// // Appends a header in the default format 'YYYY-MM-DD hh:mm:ss.sss [LVL] TAG: '.
// // If either of the Lshortfile or Llongfile flags are specified, the file named
// // and line number are included after the tag and before the final colon.
// func formatHeader(buf *[]byte, t time.Time, lvl, tag string, file string, line int) {
// 	year, month, day := t.Date()
// 	hour, min, sec := t.Clock()
// 	ms := t.Nanosecond() / 1e6

// 	itoa(buf, year, 4)
// 	*buf = append(*buf, '-')
// 	itoa(buf, int(month), 2)
// 	*buf = append(*buf, '-')
// 	itoa(buf, day, 2)
// 	*buf = append(*buf, ' ')
// 	itoa(buf, hour, 2)
// 	*buf = append(*buf, ':')
// 	itoa(buf, min, 2)
// 	*buf = append(*buf, ':')
// 	itoa(buf, sec, 2)
// 	*buf = append(*buf, '.')
// 	itoa(buf, ms, 3)

// 	if file != "" {
// 		*buf = append(*buf, ' ')
// 		*buf = append(*buf, file...)
// 		*buf = append(*buf, ':')
// 		itoa(buf, line, -1)
// 	}

// 	*buf = append(*buf, " ["...)
// 	*buf = append(*buf, lvl...)
// 	*buf = append(*buf, "] "...)
// 	*buf = append(*buf, tag...)

// 	*buf = append(*buf, ": "...)
// }

// // calldepth is the call depth of the callsite function relative to the
// // caller of the subsystem logger.  It is used to recover the filename and line
// // number of the logging call if either the short or long file flags are
// // specified.
// const calldepth = 3

// // callsite returns the file name and line number of the callsite to the
// // subsystem logger.
// func callsite(flag uint32) (string, int) {
// 	_, file, line, ok := runtime.Caller(calldepth)
// 	if !ok {
// 		return "???", 0
// 	}
// 	if flag&Lshortfile != 0 {
// 		short := file
// 		for i := len(file) - 1; i > 0; i-- {
// 			if os.IsPathSeparator(file[i]) {
// 				short = file[i+1:]
// 				break
// 			}
// 		}
// 		file = short
// 	}
// 	return file, line
// }

// // print outputs a log message to the writer associated with the backend after
// // creating a prefix for the given level and tag according to the formatHeader
// // function and formatting the provided arguments using the default formatting
// // rules.
// func (b *Backend) print(lvl, tag string, args ...interface{}) {
// 	t := time.Now() // get as early as possible

// 	bytebuf := buffer()

// 	var file string
// 	var line int
// 	if b.flag&(Lshortfile|Llongfile) != 0 {
// 		file, line = callsite(b.flag)
// 	}

// 	formatHeader(bytebuf, t, lvl, tag, file, line)
// 	buf := bytes.NewBuffer(*bytebuf)
// 	fmt.Fprintln(buf, args...)
// 	*bytebuf = buf.Bytes()

// 	b.colorPrint(lvl, *bytebuf)
// 	// @hunghd SEND LOG TO AGGREGATION LOG SERVER
// 	if isAggregationLogMode() {
// 		loggerSeparator := "[" + lvl + "]"
// 		mes := bytes.Split(*bytebuf, []byte(loggerSeparator))
// 		if len(mes) >= 2 {
// 			HandleCaptureMessage(string(mes[1]), lvl)
// 		}
// 	}

// 	recycleBuffer(bytebuf)
// }

// // printf outputs a log message to the writer associated with the backend after
// // creating a prefix for the given level and tag according to the formatHeader
// // function and formatting the provided arguments according to the given format
// // specifier.
// func (b *Backend) printf(lvl, tag string, format string, args ...interface{}) {
// 	t := time.Now() // get as early as possible

// 	bytebuf := buffer()

// 	var file string
// 	var line int
// 	if b.flag&(Lshortfile|Llongfile) != 0 {
// 		file, line = callsite(b.flag)
// 	}
// 	formatHeader(bytebuf, t, lvl, tag, file, line)
// 	buf := bytes.NewBuffer(*bytebuf)
// 	fmt.Fprintf(buf, format, args...)
// 	*bytebuf = append(buf.Bytes(), '\n')

// 	b.colorPrint(lvl, *bytebuf)
// 	// @hunghd SEND LOG TO AGGREGATION LOG SERVER
// 	if isAggregationLogMode() {
// 		loggerSeparator := "[" + lvl + "]"
// 		mes := bytes.Split(*bytebuf, []byte(loggerSeparator))
// 		if len(mes) >= 2 {
// 			HandleCaptureMessage(string(mes[1]), lvl)
// 		}
// 	}

// 	recycleBuffer(bytebuf)
// }

// func (b *Backend) colorPrint(lvl string, bytebuf []byte) {
// 	b.mu.Lock()
// 	switch lvl {
// 	case "INF":
// 		{
// 			FgGreen(b.w, string(bytebuf[:]))
// 		}
// 	case "ERR":
// 		{
// 			FgRed(b.w, string(bytebuf[:]))
// 		}
// 	case "WRN":
// 		{
// 			FgYellow(b.w, string(bytebuf[:]))
// 		}
// 	case "CRT":
// 		{
// 			FgCyan(b.w, string(bytebuf[:]))
// 		}
// 	case "DBG":
// 		{
// 			FgMagenta(b.w, string(bytebuf[:]))
// 		}
// 	default:
// 		FgWhite(b.w, string(bytebuf[:]))
// 	}
// 	b.mu.Unlock()
// }

// // Logger returns a new logger for a particular subsystem that writes to the
// // Backend b.  A tag describes the subsystem and is included in all log
// // messages.  The logger uses the info verbosity level by default.
// func (b *Backend) Logger(subsystemTag string, disable bool) Logger {
// 	return &slog{LevelInfo, subsystemTag, b, disable}
// }

// // slog is a subsystem logger for a Backend.  Implements the Logger interface.
// type slog struct {
// 	lvl     Level // atomic
// 	tag     string
// 	b       *Backend
// 	disable bool
// }

// // Trace formats message using the default formats for its operands, prepends
// // the prefix as necessary, and writes to log with LevelTrace.
// //
// // This is part of the Logger interface implementation.
// func (l *slog) Trace(args ...interface{}) {
// 	if l == nil {
// 		return
// 	}
// 	lvl := l.Level()
// 	if lvl <= LevelTrace {
// 		if !l.disable {
// 			l.b.print("TRC", l.tag, args...)
// 		}
// 	}
// }

// // Tracef formats message according to format specifier, prepends the prefix as
// // necessary, and writes to log with LevelTrace.
// //
// // This is part of the Logger interface implementation.
// func (l *slog) Tracef(format string, args ...interface{}) {
// 	if l == nil {
// 		return
// 	}
// 	lvl := l.Level()
// 	if lvl <= LevelTrace {
// 		if !l.disable {
// 			l.b.printf("TRC", l.tag, format, args...)
// 		}
// 	}
// }

// // Debug formats message using the default formats for its operands, prepends
// // the prefix as necessary, and writes to log with LevelDebug.
// //
// // This is part of the Logger interface implementation.
// func (l *slog) Debug(args ...interface{}) {
// 	if l == nil {
// 		return
// 	}
// 	lvl := l.Level()
// 	if lvl <= LevelDebug {
// 		if !l.disable {
// 			l.b.print("DBG", l.tag, args...)
// 		}
// 	}
// }

// // Debugf formats message according to format specifier, prepends the prefix as
// // necessary, and writes to log with LevelDebug.
// //
// // This is part of the Logger interface implementation.
// func (l *slog) Debugf(format string, args ...interface{}) {
// 	if l == nil {
// 		return
// 	}
// 	lvl := l.Level()
// 	if lvl <= LevelDebug {
// 		if !l.disable {
// 			l.b.printf("DBG", l.tag, format, args...)
// 		}
// 	}
// }

// // Info formats message using the default formats for its operands, prepends
// // the prefix as necessary, and writes to log with LevelInfo.
// //
// // This is part of the Logger interface implementation.
// func (l *slog) Info(args ...interface{}) {
// 	if l == nil {
// 		return
// 	}
// 	lvl := l.Level()
// 	if lvl <= LevelInfo {
// 		if !l.disable {
// 			l.b.print("INF", l.tag, args...)
// 		}
// 	}
// }

// // Infof formats message according to format specifier, prepends the prefix as
// // necessary, and writes to log with LevelInfo.
// //
// // This is part of the Logger interface implementation.
// func (l *slog) Infof(format string, args ...interface{}) {
// 	if l == nil {
// 		return
// 	}
// 	lvl := l.Level()
// 	if lvl <= LevelInfo {
// 		if !l.disable {
// 			l.b.printf("INF", l.tag, format, args...)
// 		}
// 	}
// }

// // Warn formats message using the default formats for its operands, prepends
// // the prefix as necessary, and writes to log with LevelWarn.
// //
// // This is part of the Logger interface implementation.
// func (l *slog) Warn(args ...interface{}) {
// 	if l == nil {
// 		return
// 	}
// 	lvl := l.Level()
// 	if lvl <= LevelWarn {
// 		if !l.disable {
// 			l.b.print("WRN", l.tag, args...)
// 		}
// 	}
// }

// // Warnf formats message according to format specifier, prepends the prefix as
// // necessary, and writes to log with LevelWarn.
// //
// // This is part of the Logger interface implementation.
// func (l *slog) Warnf(format string, args ...interface{}) {
// 	if l == nil {
// 		return
// 	}
// 	lvl := l.Level()
// 	if lvl <= LevelWarn {
// 		if !l.disable {
// 			l.b.printf("WRN", l.tag, format, args...)
// 		}
// 	}
// }

// // Error formats message using the default formats for its operands, prepends
// // the prefix as necessary, and writes to log with LevelError.
// //
// // This is part of the Logger interface implementation.
// func (l *slog) Error(args ...interface{}) {
// 	if l == nil {
// 		return
// 	}
// 	lvl := l.Level()
// 	if lvl <= LevelError {
// 		if !l.disable {
// 			l.b.print("ERR", l.tag, args...)
// 		}
// 	}
// }

// // Errorf formats message according to format specifier, prepends the prefix as
// // necessary, and writes to log with LevelError.
// //
// // This is part of the Logger interface implementation.
// func (l *slog) Errorf(format string, args ...interface{}) {
// 	if l == nil {
// 		return
// 	}
// 	lvl := l.Level()
// 	if lvl <= LevelError {
// 		if !l.disable {
// 			l.b.printf("ERR", l.tag, format, args...)
// 		}
// 	}
// }

// // Critical formats message using the default formats for its operands, prepends
// // the prefix as necessary, and writes to log with LevelCritical.
// //
// // This is part of the Logger interface implementation.
// func (l *slog) Critical(args ...interface{}) {
// 	if l == nil {
// 		return
// 	}
// 	lvl := l.Level()
// 	if lvl <= LevelCritical {
// 		if !l.disable {
// 			l.b.print("CRT", l.tag, args...)
// 		}
// 	}
// }

// // Criticalf formats message according to format specifier, prepends the prefix
// // as necessary, and writes to log with LevelCritical.
// //
// // This is part of the Logger interface implementation.
// func (l *slog) Criticalf(format string, args ...interface{}) {
// 	if l == nil {
// 		return
// 	}
// 	lvl := l.Level()
// 	if lvl <= LevelCritical {
// 		if !l.disable {
// 			l.b.printf("CRT", l.tag, format, args...)
// 		}
// 	}
// }

// // Level returns the current logging level
// //
// // This is part of the Logger interface implementation.
// func (l *slog) Level() Level {
// 	return Level(atomic.LoadUint32((*uint32)(&l.lvl)))
// }

// // SetLevel changes the logging level to the passed level.
// //
// // This is part of the Logger interface implementation.
// func (l *slog) SetLevel(level Level) {
// 	atomic.StoreUint32((*uint32)(&l.lvl), uint32(level))
// }

// // Disabled is a Logger that will never output anything.
// var Disabled Logger

// func init() {
// 	Disabled = &slog{lvl: LevelOff, b: NewBackend(ioutil.Discard)}

// 	// @hunghd SEND LOG TO AGGREGATION LOG SERVER
// 	if isAggregationLogMode() {
// 		log.Println("init aggreation log mode")
// 		AggregationLogInit()
// 	}
// }

// func isAggregationLogMode() bool {
// 	aggreLogMode := os.Getenv("AGGRE_LOG_MODE")
// 	return aggreLogMode == "true"
// }
