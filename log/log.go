package log

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"sync"
)

const (
	// DebugLevel 平时自己打印用, 尽量不要提交到线上, 并且线上会屏蔽
	DebugLevel int8 = iota - 1
	// InfoLevel 正常的日志输出
	InfoLevel
	// WarnLevel 在可控范围内的错误, 可以预知会报错的地方
	WarnLevel
	// ErrorLevel 不可控的范围内的错误, 需要专门去查看的地方
	ErrorLevel
	// DPanicLevel 记录.恐慌
	DPanicLevel
	// PanicLevel 记录.恐慌
	PanicLevel
	// FatalLevel 启动大加载时才允许使用的错误,  通常是核心缺失, 使用以后就必须终止进程, 严禁在
	FatalLevel
)

var RequestId string = "x-request-id"

type RequestIdKey struct{}

// Logger 定义了项目的日志接口. 该接口只包含了支持的日志记录方法.
type Logger interface {
	Debugw(msg string, keysAndValues ...interface{})
	Infow(msg string, keysAndValues ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	Panicw(msg string, keysAndValues ...interface{})
	Fatalw(msg string, keysAndValues ...interface{})
	Sync()
}

// zapLogger 是 Logger 接口的具体实现. 它底层封装了 zap.Logger.
type zapLogger struct {
	z *zap.Logger
}

// 确保 zapLogger 实现了 Logger 接口. 以下变量赋值，可以使错误在编译期被发现.
var _ Logger = &zapLogger{}

var (
	mu sync.Mutex

	// std 定义了默认的全局 Logger.
	std = NewLogger(NewOptions())
)

// Init 使用指定的选项初始化 Logger.
func Init(opts *Options) {
	mu.Lock()
	defer mu.Unlock()

	std = NewLogger(opts)
}

// NewLogger 根据传入的 opts 创建 Logger.
func NewLogger(opts *Options) *zapLogger {
	if opts == nil {
		opts = NewOptions()
	}

	// 将文本格式的日志级别，例如 info 转换为 zapcore.Level 类型以供后面使用
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(opts.Level)); err != nil {
		// 如果指定了非法的日志级别，则默认使用 info 级别
		zapLevel = zapcore.InfoLevel
	}

	var cores []zapcore.Core
	if len(opts.OutputPaths) > 0 {
		for _, file := range opts.OutputPaths {
			if err := createDirIfNotExists(file); err != nil {
				panic(err)
			}
			// 获取日志写入位置
			writeSyncer := getLogWriter(file, opts.Maxsize, opts.MaxBackup, opts.MaxAge)
			// 获取日志编码格式
			encoder := getEncoder(opts)
			// 创建一个将日志写入 WriteSyncer 的核心。
			fileCore := zapcore.NewCore(encoder, writeSyncer, zapLevel)
			cores = append(cores, fileCore)
		}

	}

	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), // 控制台格式
		zapcore.AddSync(os.Stdout),
		zap.DebugLevel,
	)
	cores = append(cores, consoleCore)

	// 合并 Core
	combinedCore := zapcore.NewTee(cores...)
	skip := zap.AddCallerSkip(1)
	if opts.CallerSkip > 0 {
		skip = zap.AddCallerSkip(opts.CallerSkip)
	}

	z := zap.New(combinedCore, zap.AddCaller(), skip)
	logger := &zapLogger{z: z}

	// 把标准库的 log.Logger 的 info 级别的输出重定向到 zap.Logger
	zap.RedirectStdLog(z)

	return logger
}

// Sync 调用底层 zap.Logger 的 Sync 方法，将缓存中的日志刷新到磁盘文件中. 主程序需要在退出前调用 Sync.
func Sync() { std.Sync() }

func (l *zapLogger) Sync() {
	_ = l.z.Sync()
}

// Debugw 输出 debug 级别的日志.
func Debugw(msg string, keysAndValues ...interface{}) {
	std.z.Sugar().Debugw(msg, keysAndValues...)
}

func (l *zapLogger) Debugw(msg string, keysAndValues ...interface{}) {
	l.z.Sugar().Debugw(msg, keysAndValues...)
}

// Infow 输出 info 级别的日志.
func Infow(msg string, keysAndValues ...interface{}) {
	std.z.Sugar().Infow(msg, keysAndValues...)
}

func (l *zapLogger) Infow(msg string, keysAndValues ...interface{}) {
	l.z.Sugar().Infow(msg, keysAndValues...)
}

// Warnw 输出 warning 级别的日志.
func Warnw(msg string, keysAndValues ...interface{}) {
	std.z.Sugar().Warnw(msg, keysAndValues...)
}

func (l *zapLogger) Warnw(msg string, keysAndValues ...interface{}) {
	l.z.Sugar().Warnw(msg, keysAndValues...)
}

// Errorw 输出 error 级别的日志.
func Errorw(msg string, keysAndValues ...interface{}) {
	std.z.Sugar().Errorw(msg, keysAndValues...)
}

func (l *zapLogger) Errorw(msg string, keysAndValues ...interface{}) {
	l.z.Sugar().Errorw(msg, keysAndValues...)
}

// Panicw 输出 panic 级别的日志.
func Panicw(msg string, keysAndValues ...interface{}) {
	std.z.Sugar().Panicw(msg, keysAndValues...)
}

func (l *zapLogger) Panicw(msg string, keysAndValues ...interface{}) {
	l.z.Sugar().Panicw(msg, keysAndValues...)
}

// Fatalw 输出 fatal 级别的日志.
func Fatalw(msg string, keysAndValues ...interface{}) {
	std.z.Sugar().Fatalw(msg, keysAndValues...)
}

func (l *zapLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.z.Sugar().Fatalw(msg, keysAndValues...)
}

func DebugfWithContext(c context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	C(c).Debugw(msg) // 强制转为结构化日志
}

func InfofWithContext(c context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	C(c).Infow(msg) // 强制转为结构化日志
}

func WarnfWithContext(c context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	C(c).Warnw(msg) // 强制转为结构化日志
}

func ErrorfWithContext(c context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	C(c).Errorw(msg) // 强制转为结构化日志
}

// PanicfWithContext...
func PanicfWithContext(c context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	C(c).Panicw(msg) // 强制转为结构化日志
}

func FatalfWithContext(c context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	C(c).Fatalw(msg) // 强制转为结构化日志
}

// C 解析传入的 context，尝试提取关注的键值，并添加到 zap.Logger 结构化日志中.
func C(ctx context.Context) *zapLogger {
	return std.C(ctx)
}

func (l *zapLogger) C(ctx context.Context) *zapLogger {
	lc := l.clone()

	if requestID := ctx.Value(RequestIdKey{}); requestID != nil {
		lc.z = lc.z.With(zap.Any(RequestId, requestID))
	}

	return lc
}

// clone 深度拷贝 zapLogger.
func (l *zapLogger) clone() *zapLogger {
	lc := *l
	return &lc
}
