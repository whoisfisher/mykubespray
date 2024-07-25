package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	globalLogger *logrus.Logger
	once         sync.Once
)

// Init initializes the global logger based on configuration
func Init() error {
	// Load configuration from file
	//if err := loadConfig(); err != nil {
	//	return err
	//}

	// Initialize logger once
	once.Do(func() {
		globalLogger = logrus.New()
		globalLogger.SetLevel(getLogLevel())

		// Set formatter
		globalLogger.SetFormatter(&ColoredTextFormatter{})

		// Initialize log hook based on output type
		switch getOutputType() {
		case "file":
			initFileLogging(getLogFile())
		case "stdout":
			initStdoutLogging()
		case "file_and_stdout":
			initFileAndStdoutLogging(getLogFile())
		default:
			fmt.Println("Unknown output type, defaulting to stdout")
			initStdoutLogging()
		}
	})

	return nil
}

func loadConfig() error {
	viper.SetConfigName("config")
	viper.AddConfigPath("./")
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}
	return nil
}

func getLogLevel() logrus.Level {
	levelStr := viper.GetString("log.level")
	level, err := logrus.ParseLevel(levelStr)
	if err != nil {
		fmt.Printf("Failed to parse log level: %s, defaulting to Debug\n", err)
		return logrus.DebugLevel
	}
	return level
}

func getOutputType() string {
	return viper.GetString("log.output_type")
}

func getLogFile() string {
	return viper.GetString("log.logfile")
}

func initFileLogging(logfile string) {
	fileWriter, err := rotatelogs.New(
		logfile+".%Y%m%d%H%M",
		rotatelogs.WithLinkName(logfile),
		rotatelogs.WithMaxAge(7*24*time.Hour),
		rotatelogs.WithRotationTime(24*time.Hour),
	)
	if err != nil {
		fmt.Printf("Failed to create rotatelogs: %s\n", err)
		return
	}

	// Create log hook
	lfHook := lfshook.NewHook(lfshook.WriterMap{
		logrus.DebugLevel: fileWriter,
		logrus.InfoLevel:  fileWriter,
		logrus.WarnLevel:  fileWriter,
		logrus.ErrorLevel: fileWriter,
		logrus.FatalLevel: fileWriter,
		logrus.PanicLevel: fileWriter,
	}, &ColoredTextFormatter{})

	// Add hook to logger
	globalLogger.AddHook(lfHook)
}

func initStdoutLogging() {
	// Standard output, no additional setup needed
}

func initFileAndStdoutLogging(logfile string) {
	// Output to both file and standard output
	fileWriter, err := rotatelogs.New(
		logfile+".%Y%m%d%H%M",
		rotatelogs.WithLinkName(logfile),
		rotatelogs.WithMaxAge(7*24*time.Hour),
		rotatelogs.WithRotationTime(24*time.Hour),
	)
	if err != nil {
		fmt.Printf("Failed to create rotatelogs: %s\n", err)
		return
	}

	// Create log hook
	lfHook := lfshook.NewHook(lfshook.WriterMap{
		logrus.DebugLevel: io.MultiWriter(fileWriter, os.Stdout),
		logrus.InfoLevel:  io.MultiWriter(fileWriter, os.Stdout),
		logrus.WarnLevel:  io.MultiWriter(fileWriter, os.Stdout),
		logrus.ErrorLevel: io.MultiWriter(fileWriter, os.Stdout),
		logrus.FatalLevel: io.MultiWriter(fileWriter, os.Stdout),
		logrus.PanicLevel: io.MultiWriter(fileWriter, os.Stdout),
	}, &ColoredTextFormatter{})

	// Add hook to logger
	globalLogger.AddHook(lfHook)
}

// GetLogger returns the global logger instance
func GetLogger() *logrus.Logger {
	return globalLogger
}

type ColoredTextFormatter struct{}

func (f *ColoredTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var colorFn func(...interface{}) string

	switch entry.Level {
	case logrus.DebugLevel:
		colorFn = color.New(color.FgWhite).SprintFunc()
	case logrus.InfoLevel:
		colorFn = color.New(color.FgGreen).SprintFunc()
	case logrus.WarnLevel:
		colorFn = color.New(color.FgYellow).SprintFunc()
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		colorFn = color.New(color.FgRed).SprintFunc()
	default:
		colorFn = color.New(color.Reset).SprintFunc()
	}

	// Format log message
	msg := fmt.Sprintf("[%s] %s %s\n", entry.Time.Format("2006-01-02 15:04:05"), strings.ToUpper(entry.Level.String()), entry.Message)
	return []byte(colorFn(msg)), nil
}