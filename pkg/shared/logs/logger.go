package logs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/sirupsen/logrus"
)

var Logger = logrus.New()
var logsLevel = map[string]logrus.Level{
	"debug": logrus.DebugLevel,
	"info":  logrus.InfoLevel,
	"warn":  logrus.WarnLevel,
	"error": logrus.ErrorLevel,
	"fatal": logrus.FatalLevel,
}

type CustomFormatter struct {
	logrus.TextFormatter
}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	levelColors := map[logrus.Level]string{
		logrus.DebugLevel: "\033[37m",
		logrus.InfoLevel:  "\033[32m",
		logrus.WarnLevel:  "\033[33m",
		logrus.ErrorLevel: "\033[31m",
		logrus.FatalLevel: "\033[35m",
	}

	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	level := strings.ToUpper(entry.Level.String())
	color := levelColors[entry.Level]

	baseMessage := fmt.Sprintf("%s %s [%s]\033[0m %s",
		color,
		timestamp,
		level,
		entry.Message,
	)

	var fields string
	if len(entry.Data) > 0 {
		fields = "\n	Details:\n"
		for k, v := range entry.Data {
			fields += fmt.Sprintf("	%-10s: %v\n", k, v)
		}
	}

	return []byte(fmt.Sprintf("%s%s\n", baseMessage, fields)), nil
}

func InitLogger(logLevel, logPath string) error {
	Logger = logrus.New()

	formatter := &CustomFormatter{
		TextFormatter: logrus.TextFormatter{
			DisableColors: false,
			FullTimestamp: true,
		},
	}

	Logger.SetFormatter(formatter)
	Logger.SetLevel(determineLogLevel(logLevel))
	Logger.SetOutput(os.Stdout)

	if logPath == "" {
		return nil
	}

	logDir := utils.FindProjectRoot()
	fullLogDir := filepath.Join(logDir, logPath)

	if err := os.MkdirAll(fullLogDir, 0755); err != nil {
		return nil
	}

	logFilePath := filepath.Join(fullLogDir, "dte_microservice.log")

	logFile, err := os.OpenFile(
		logFilePath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0666,
	)
	if err != nil {
		return nil
	}

	Logger.SetOutput(io.MultiWriter(os.Stdout, logFile))

	return nil
}

func logWithFields(level logrus.Level, msg string, fields ...map[string]interface{}) {
	if len(fields) > 0 && fields[0] != nil {
		Logger.WithFields(fields[0]).Log(level, msg)
	} else {
		Logger.Log(level, msg)
	}
}

func Debug(msg string, fields ...map[string]interface{}) {
	logWithFields(logrus.DebugLevel, msg, fields...)
}

func Info(msg string, fields ...map[string]interface{}) {
	logWithFields(logrus.InfoLevel, msg, fields...)
}

func Warn(msg string, fields ...map[string]interface{}) {
	logWithFields(logrus.WarnLevel, msg, fields...)
}

func Error(msg string, fields ...map[string]interface{}) {
	logWithFields(logrus.ErrorLevel, msg, fields...)
}

func Fatal(msg string, fields ...map[string]interface{}) {
	logWithFields(logrus.FatalLevel, msg, fields...)
}

func determineLogLevel(logLevel string) logrus.Level {
	if level, ok := logsLevel[logLevel]; ok {
		return level
	}
	return logrus.InfoLevel
}

type WriteHook struct {
	Writer    io.Writer
	Formatter logrus.Formatter
}

func (hook *WriteHook) Fire(entry *logrus.Entry) error {
	line, err := hook.Formatter.Format(entry)
	if err != nil {
		return err
	}
	_, err = hook.Writer.Write(line)
	return err
}

func (hook *WriteHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
