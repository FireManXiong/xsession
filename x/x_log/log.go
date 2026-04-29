package x_log

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const (
	DebugLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

var (
	currentLevel = InfoLevel
	logPath      = "./logs/log.log"
	logDir       = "./logs"
	maxSize      = 1024 * 1024 * 100 // 100MB 自动切割
	maxBackups   = 10                // 最大保留10个日志
	mu           sync.Mutex
	file         *os.File
	writer       *bufio.Writer
)

func Init(fileName string) {
	os.MkdirAll(logDir, 0755)
	logPath = fmt.Sprintf("./logs/%s", fileName)
	delayInit()
}

func delayInit() {
	os.MkdirAll(logDir, 0755)
	openLogFile()
}

func SetLogLevel(level int) {
	currentLevel = level
}

func openLogFile() {
	mu.Lock()
	defer mu.Unlock()

	var err error
	file, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	writer = bufio.NewWriter(file)
}

// Rotate 手动切割日志
func Rotate() {
	mu.Lock()
	defer mu.Unlock()

	stat, err := file.Stat()
	if err != nil {
		return
	}
	if stat.Size() < int64(maxSize) {
		return
	}

	_ = writer.Flush()
	_ = file.Close()

	backup := fmt.Sprintf("%s.%s", logPath, time.Now().Format("20060102-150405"))
	_ = os.Rename(logPath, backup)

	file, _ = os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	writer = bufio.NewWriter(file)
	cleanBackups()
}

func cleanBackups() {
	matches, _ := filepath.Glob(logPath + ".*")
	if len(matches) > maxBackups {
		for i := 0; i < len(matches)-maxBackups; i++ {
			_ = os.Remove(matches[i])
		}
	}
}

func logf(level int, format string, args ...any) {
	if writer == nil {
		delayInit()
	}
	if level < currentLevel {
		return
	}

	Rotate()

	levelStr := [...]string{"DEBUG", "INFO", "WARN", "ERROR"}[level]
	_, file, line, _ := runtime.Caller(2)
	file = filepath.Base(file)
	now := time.Now().Format("2006-01-02 15:04:05")

	msg := fmt.Sprintf(format, args...)
	logStr := fmt.Sprintf("[%s] %-5s %s:%d → %s\n", now, levelStr, file, line, msg)

	mu.Lock()
	_, _ = writer.WriteString(logStr)
	_ = writer.Flush()
	mu.Unlock()

	fmt.Print(logStr)
}

// SetLevel 设置日志级别 Debug/Info/Warn/Error
func SetLevel(level int) {
	currentLevel = level
}

// Debugf 调试日志
func Debugf(format string, args ...any) {
	logf(DebugLevel, format, args...)
}

// Infof 普通信息日志
func Infof(format string, args ...any) {
	logf(InfoLevel, format, args...)
}

// Warnf 警告日志
func Warnf(format string, args ...any) {
	logf(WarnLevel, format, args...)
}

// Errorf 错误日志
func Errorf(format string, args ...any) {
	logf(ErrorLevel, format, args...)
}
