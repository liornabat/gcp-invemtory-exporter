package logger

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode"
)

// A few constants, which are used more like flags
const (
	notATrace     = -1
	noTraceOutput = -1
)

// The known log levels
const (
	levelNone = iota
	levelFatal
	levelErr
	levelWarn
	levelInfo
	levelDebug
	levelTrace
)

// Translation map from level to string representation
var levelStrings = map[int]string{
	levelDebug: "debug",
	levelInfo:  "info",
	levelWarn:  "warn",
	levelErr:   "error",
	levelFatal: "fatal",
	levelNone:  "none",
}

// Translation from level string to number.
var levelNumbers = map[string]int{
	"debug": levelDebug,
	"info":  levelInfo,
	"warn":  levelWarn,
	"error": levelErr,
	"fatal": levelFatal,
	"none":  levelNone,
}

type Logger struct {
	initMutex             sync.RWMutex
	settingsName          string
	settingShowCallerInfo bool        // whether we log caller info
	settingDateTimeFormat string      // flags for date/time output
	logWriterStream       *log.Logger // the first writer to which output is sent
	logWriterFile         *log.Logger // the second writer to which output is sent
	logFilterSpec         *filterSpec // filters for log messages
	currentLogFile        *os.File    // the logfile currently in use
	currentLogFileName    string      // name of current log file
	bufferSink            *bufferSink // buffer for log messages
}

func NewLogger(name string, level string) *Logger {
	cfg := newLoggerConfig().
		SetName(name).
		SetLogLevel(level).
		SetShowCallerInfo(true)

	l := newLogger()
	err := l.init(cfg)
	if err != nil {
		panic(err)
	}
	return l
}

func NewLoggerWithConfig(cfg *Config) *Logger {
	l := newLogger()
	err := l.init(cfg)
	if err != nil {
		panic(err)
	}
	return l
}

func newLogger() *Logger {
	return &Logger{
		initMutex: sync.RWMutex{},
	}
}

func (l *Logger) init(cfg *Config) error {
	l.initMutex.Lock()
	defer l.initMutex.Unlock()
	l.settingsName = capitalize(cfg.Name)
	l.settingShowCallerInfo = cfg.ShowCallerInfo
	l.settingDateTimeFormat = l.getTimeFormat(cfg)
	newLogFilterSpec := new(filterSpec)
	if err := newLogFilterSpec.fromString(cfg.LogLevel, false, levelInfo); err != nil {
		return err
	}
	l.logFilterSpec = newLogFilterSpec

	if cfg.LogStream == "stdout" {
		l.logWriterStream = log.New(os.Stdout, "", 0)
	} else if cfg.LogStream == "none" {
		l.logWriterStream = nil
	} else {
		l.logWriterStream = log.New(os.Stderr, "", 0)
	}
	var newLogFile *os.File
	if l.currentLogFileName != cfg.LogFile { // something changed
		if cfg.LogFile == "" {
			// no more log output to a file
			l.logWriterFile = nil
		} else {
			// Check if the logfile was changed or was set for the first
			// time. Only then do we need to open/create a new file.
			// We also do this if for some reason we don't have a log writer
			// yet.
			if l.currentLogFileName != cfg.LogFile || l.logWriterFile == nil {
				var err error
				newLogFile, err = os.OpenFile(cfg.LogFile,
					os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
				if err == nil {
					l.logWriterFile = log.New(newLogFile, "", 0)
				} else {
					return fmt.Errorf("Unable to open log file: %s", err.Error())
				}
			}
		}
		// Close the old logfile, since we are now writing to a new file
		if l.currentLogFileName != "" {
			_ = l.currentLogFile.Close()
			l.currentLogFileName = cfg.LogFile
			l.currentLogFile = newLogFile
		}
	}
	l.bufferSink = newBufferSink().setEnabled(cfg.LogBufferSync)
	return nil
}

func (l *Logger) getTimeFormat(cfg *Config) string {
	settingDateTimeFormat := ""
	logNoTime := cfg.LogNoTime
	if !logNoTime {
		// Store the format string for date/time logging. Allowed values are
		// all the constants specified in
		// https://golang.org/src/time/format.go.
		var f string
		switch strings.ToUpper(cfg.LogTimeFormat) {
		case "ANSIC":
			f = time.ANSIC
		case "UNIXDATE":
			f = time.UnixDate
		case "RUBYDATE":
			f = time.RubyDate
		case "RFC822":
			f = time.RFC822
		case "RFC822Z":
			f = time.RFC822Z
		case "RFC1123":
			f = time.RFC1123
		case "RFC1123Z":
			f = time.RFC1123Z
		case "RFC3339":
			f = time.RFC3339
		case "RFC3339NANO":
			f = time.RFC3339Nano
		case "KITCHEN":
			f = time.Kitchen
		default:
			if cfg.LogTimeFormat != "" {
				f = cfg.LogTimeFormat
			} else {
				f = time.RFC3339
			}
		}
		settingDateTimeFormat = f + " "
	}
	return settingDateTimeFormat
}

func (l *Logger) GetLogs() []string {
	if !l.bufferSink.enabled {
		return nil
	}
	return l.bufferSink.getBufferLogs()
}

func (l *Logger) SetEnableBufferLogs(value bool) {
	l.bufferSink = newBufferSink().setEnabled(value)
}

func (l *Logger) basicLog(logLevel int, format string, prefixAddition string, a ...interface{}) {
	if !l.logFilterSpec.matchfilters("", logLevel) {
		return
	}
	now := time.Now()
	callerInfo := ""
	if l.settingShowCallerInfo {
		// Extract information about the caller of the log function, if requested.

		var moduleAndFileName string
		_, fullFilePath, line, ok := runtime.Caller(2)
		if ok {
			dirPath, fileName := path.Split(fullFilePath)
			var moduleName string
			if dirPath != "" {
				dirPath = dirPath[:len(dirPath)-1]
				_, moduleName = path.Split(dirPath)
			}
			moduleAndFileName = moduleName + "/" + fileName
		}
		callerInfo = fmt.Sprintf("[%s:%d] ", moduleAndFileName, line)

	}

	// Assemble the actual log line
	var msg string
	if format != "" {
		msg = fmt.Sprintf(format, a...)
	} else {
		msg = fmt.Sprintln(a...)
	}
	levelDecoration := levelStrings[logLevel] + prefixAddition
	logLine := fmt.Sprintf("%s%-4s %-6s: %s %s",
		now.Format(l.settingDateTimeFormat), capitalize(levelDecoration), l.settingsName, capitalize(msg), callerInfo)
	if l.logWriterStream != nil {
		l.logWriterStream.Print(logLine)
	}
	if l.logWriterFile != nil {
		l.logWriterFile.Print(logLine)
	}
}

func (l *Logger) Debug(a ...interface{}) {
	l.basicLog(levelDebug, "", "", a...)
}

func (l *Logger) Debugf(format string, a ...interface{}) {
	l.basicLog(levelDebug, format, "", a...)
}

func (l *Logger) Info(a ...interface{}) {
	l.basicLog(levelInfo, "", "", a...)
}

func (l *Logger) Infof(format string, a ...interface{}) {
	l.basicLog(levelInfo, format, "", a...)
}

func (l *Logger) Println(a ...interface{}) {
	l.basicLog(levelInfo, "", "", a...)
}

func (l *Logger) Printf(format string, a ...interface{}) {
	l.basicLog(levelInfo, format, "", a...)
}

func (l *Logger) Warn(a ...interface{}) {
	l.basicLog(levelWarn, "", "", a...)
}

func (l *Logger) Warnf(format string, a ...interface{}) {
	l.basicLog(levelWarn, format, "", a...)
}

func (l *Logger) Error(a ...interface{}) {
	l.basicLog(levelErr, "", "", a...)
}

func (l *Logger) Errorf(format string, a ...interface{}) {
	l.basicLog(levelErr, format, "", a...)
}

func (l *Logger) Fatal(a ...interface{}) {
	l.basicLog(levelFatal, "", "", a...)
}

func (l *Logger) FatalF(format string, a ...interface{}) {
	l.basicLog(levelFatal, format, "", a...)
}

func capitalize(str string) string {
	if len(str) == 0 {
		return ""
	}
	tmp := []rune(str)
	tmp[0] = unicode.ToUpper(tmp[0])
	return string(tmp)
}
