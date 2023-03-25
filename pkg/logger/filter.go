package logger

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

type filterSpec struct {
	filters []filter
}

// filter holds filename and level to match logs against log messages.
type filter struct {
	Pattern string
	Level   int
}

// fromString initializes filterSpec from string.
//
// Use the isTraceLevel flag to indicate whether the levels are numeric (for
// trace messages) or are level strings (for log messages).
//
// Format "<filter>,<filter>,[<filter>]..."
//
//	filter:
//	  <pattern=level> | <level>
//	pattern:
//	  shell glob to match caller file name
//	level:
//	  log or trace level of the logs to enable in matched files.
//
//	Example:
//	- "RLOG_TRACE_LEVEL=3"
//	  Just a global trace level of 3 for all files and modules.
//	- "RLOG_TRACE_LEVEL=client.go=1,ip*=5,3"
//	  This enables trace level 1 in client.go, level 5 in all files whose
//	  names start with 'ip', and level 3 for everyone else.
//	- "RLOG_LOG_LEVEL=DEBUG"
//	  Global log level DEBUG for all files and modules.
//	- "RLOG_LOG_LEVEL=client.go=ERROR,INFO,ip*=WARN"
//	  ERROR and higher for client.go, WARN or higher for all files whose
//	  name starts with 'ip', INFO for everyone else.
func (spec *filterSpec) fromString(s string, isTraceLevels bool, globalLevelDefault int) error {
	var globalLevel int = globalLevelDefault
	var levelToken string
	var matchToken string

	fields := strings.Split(s, ",")

	for _, f := range fields {
		var filterLevel int
		var err error
		var ok bool

		// Tokens should contain two elements: The filename and the trace
		// level. If there is only one token then we have to assume that this
		// is the 'global' filter (without filename component).
		tokens := strings.Split(f, "=")
		if len(tokens) == 1 {
			// Global level. We'll store this one for the end, since it needs
			// to sit last in the list of filters (during evaluation in gets
			// checked last).
			matchToken = ""
			levelToken = tokens[0]
		} else if len(tokens) == 2 {
			matchToken = tokens[0]
			levelToken = tokens[1]
		} else {
			// Skip anything else that's malformed
			return fmt.Errorf("Malformed log filter expression: '%s'", f)
		}
		if isTraceLevels {
			// The level token should contain a numeric value
			if filterLevel, err = strconv.Atoi(levelToken); err != nil {
				if levelToken != "" {
					return fmt.Errorf("Trace level '%s' is not a number.", levelToken)
				}
				continue
			}
		} else {
			// The level token should contain the name of a log level
			filterLevel, ok = levelNumbers[levelToken]
			if !ok || filterLevel == levelTrace {
				// User not allowed to set trace log levels, so if that or
				// not a known log level then this specification will be
				// ignored.
				if levelToken != "" {
					return fmt.Errorf("Illegal log level '%s'.", levelToken)
				}
				continue
			}

		}

		if matchToken == "" {
			// Global level just remembered for now, not yet added
			globalLevel = filterLevel
		} else {
			spec.filters = append(spec.filters, filter{matchToken, filterLevel})
		}
	}

	// Now add the global level, so that later it will be evaluated last.
	// For trace levels we do something extra: There are possibly many trace
	// messages, but most often trace level debugging is fully disabled. We
	// want to optimize this. Therefore, a globalLevel of -1 (no trace levels)
	// isn't stored in the filter chain. If no other trace filters were defined
	// then this means the filter chain is empty, which can be tested very
	// efficiently in the top-level trace functions for an early exit.
	if !isTraceLevels || globalLevel != noTraceOutput {
		spec.filters = append(spec.filters, filter{"", globalLevel})
	}

	return nil
}

// matchfilters checks if given filename and trace level are accepted
// by any of the filters
func (spec *filterSpec) matchfilters(filename string, level int) bool {
	// If there are no filters then we don't match anything.
	if len(spec.filters) == 0 {
		return false
	}

	// If at least one filter matches.
	for _, filter := range spec.filters {
		if matched, loggit := filter.match(filename, level); matched {
			return loggit
		}
	}

	return false
}

// match checks if given filename and level are matched by
// this filter. Returns two bools: One to indicate whether a filename match was
// made, and the second to indicate whether the message should be logged
// (matched the level).
func (f filter) match(filename string, level int) (bool, bool) {
	var match bool
	if f.Pattern != "" {
		match, _ = filepath.Match(f.Pattern, filepath.Base(filename))
	} else {
		match = true
	}
	if match {
		return true, level <= f.Level
	}

	return false, false
}
