package main

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	nocolor = 0
	red     = 31
	green   = 32
	yellow  = 33
	blue    = 36
	gray    = 37
)

const defaultTimestampFormat = time.RFC3339

var (
	baseTimestamp time.Time
	emptyFieldMap logrus.FieldMap
)

func init() {
	baseTimestamp = time.Now()
}

// TextFormatter formats logs into text
type TextFormatter struct {
	// Set to true to bypass checking for a TTY before outputting colors.
	ForceColors bool

	// Force disabling colors.
	DisableColors bool

	// Override coloring based on CLICOLOR and CLICOLOR_FORCE. - https://bixense.com/clicolors/
	EnvironmentOverrideColors bool

	// Disable timestamp logging. useful when output is redirected to logging
	// system that already adds timestamps.
	DisableTimestamp bool

	// Enable logging the full timestamp when a TTY is attached instead of just
	// the time passed since beginning of execution.
	FullTimestamp bool

	// TimestampFormat to use for display when a full timestamp is printed
	TimestampFormat string

	// The fields are sorted by default for a consistent output. For applications
	// that log extremely frequently and don't use the JSON formatter this may not
	// be desired.
	DisableSorting bool

	// Disables the truncation of the level text to 4 characters.
	DisableLevelTruncation bool

	// QuoteEmptyFields will wrap empty fields in quotes if true
	QuoteEmptyFields bool

	// Whether the logger's out is to a terminal
	isTerminal bool

	// FieldMap allows users to customize the names of keys for default fields.
	// As an example:
	// formatter := &TextFormatter{
	//     FieldMap: FieldMap{
	//         FieldKeyTime:  "@timestamp",
	//         FieldKeyLevel: "@level",
	//         FieldKeyMsg:   "@message"}}
	FieldMap logrus.FieldMap

	sync.Once
}

func (f *TextFormatter) init(entry *logrus.Entry) {
	if entry.Logger != nil {
		f.isTerminal = checkIfTerminal(entry.Logger.Out)
	}
}

func (f *TextFormatter) isColored() bool {
	isColored := f.ForceColors || f.isTerminal

	if f.EnvironmentOverrideColors {
		if force, ok := os.LookupEnv("CLICOLOR_FORCE"); ok && force != "0" {
			isColored = true
		} else if ok && force == "0" {
			isColored = false
		} else if os.Getenv("CLICOLOR") == "0" {
			isColored = false
		}
	}

	return isColored && !f.DisableColors
}

// Format renders a single log entry
func (f *TextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// prefixFieldClashes(entry.Data, f.FieldMap)

	keys := make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}

	if !f.DisableSorting {
		sort.Strings(keys)
	}

	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	f.Do(func() { f.init(entry) })

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = defaultTimestampFormat
	}
	if f.isColored() {
		f.printColored(b, entry, keys, timestampFormat)
	} else {
		f.print(b, entry, keys, timestampFormat)
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

func (f *TextFormatter) print(b *bytes.Buffer, entry *logrus.Entry, keys []string, timestampFormat string) {

	levelText := strings.ToUpper(entry.Level.String())
	if !f.DisableLevelTruncation {
		levelText = levelText[0:4]
	}

	// Remove a single newline if it already exists in the message to keep
	// the behavior of logrus text_formatter the same as the stdlib log package
	entry.Message = strings.TrimSuffix(entry.Message, "\n")

	if f.DisableTimestamp {
		fmt.Fprintf(b, "%s %-44s ", levelText, entry.Message)
	} else if !f.FullTimestamp {
		fmt.Fprintf(b, "%s[%04d] %-44s ", levelText, int(entry.Time.Sub(baseTimestamp)/time.Second), entry.Message)
	} else {
		fmt.Fprintf(b, "%s[%s] %-44s ", levelText, entry.Time.Format(timestampFormat), entry.Message)
	}
	for _, k := range keys {
		v := entry.Data[k]
		fmt.Fprintf(b, " %s=", k)
		f.appendValue(b, v)
	}

}

func (f *TextFormatter) printColored(b *bytes.Buffer, entry *logrus.Entry, keys []string, timestampFormat string) {
	var levelColor int
	switch entry.Level {
	case logrus.DebugLevel:
		levelColor = gray
	case logrus.WarnLevel:
		levelColor = yellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelColor = red
	default:
		levelColor = blue
	}

	levelText := strings.ToUpper(entry.Level.String())
	if !f.DisableLevelTruncation {
		levelText = levelText[0:4]
	}

	// Remove a single newline if it already exists in the message to keep
	// the behavior of logrus text_formatter the same as the stdlib log package
	entry.Message = strings.TrimSuffix(entry.Message, "\n")

	if f.DisableTimestamp {
		fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m %-44s ", levelColor, levelText, entry.Message)
	} else if !f.FullTimestamp {
		fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m[%04d] %-44s ", levelColor, levelText, int(entry.Time.Sub(baseTimestamp)/time.Second), entry.Message)
	} else {
		fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m[%s] %-44s ", levelColor, levelText, entry.Time.Format(timestampFormat), entry.Message)
	}
	for _, k := range keys {
		v := entry.Data[k]
		fmt.Fprintf(b, " \x1b[%dm%s\x1b[0m=", levelColor, k)
		f.appendValue(b, v)
	}
}

func (f *TextFormatter) needsQuoting(text string) bool {
	if f.QuoteEmptyFields && len(text) == 0 {
		return true
	}
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.' || ch == '_' || ch == '/' || ch == '@' || ch == '^' || ch == '+') {
			return true
		}
	}
	return false
}

func (f *TextFormatter) appendKeyValue(b *bytes.Buffer, key string, value interface{}) {
	if b.Len() > 0 {
		b.WriteByte(' ')
	}
	b.WriteString(key)
	b.WriteByte('=')
	f.appendValue(b, value)
}

func (f *TextFormatter) appendValue(b *bytes.Buffer, value interface{}) {
	stringVal, ok := value.(string)
	if !ok {
		stringVal = fmt.Sprint(value)
	}

	if !f.needsQuoting(stringVal) {
		b.WriteString(stringVal)
	} else {
		b.WriteString(fmt.Sprintf("%q", stringVal))
	}
}
