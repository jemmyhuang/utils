// Copyright 2022 Innkeeper Belm(孔令飞) <nosbelm@qq.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/marmotedu/miniblog.

package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
	"time"
)

var logFilePath = "./app.log"

// Options 包含与日志相关的配置项.
type Options struct {
	// 是否开启 caller，如果开启会在日志中显示调用日志所在的文件和行号
	DisableCaller bool
	// 是否禁止在 panic 及以上级别打印堆栈信息
	DisableStacktrace bool
	// 指定日志级别，可选值：debug, info, warn, error, dpanic, panic, fatal
	Level string
	// 指定日志显示格式，可选值：console, json
	Format string
	// 指定日志输出位置
	OutputPaths []string
	// 文件的最大大小，超哥切片新文件
	Maxsize int
	// 旧日志保持的最大个数
	MaxBackup int
	// 文件保持的天数
	MaxAge int
	// 是否按照天保留
	LocalTime bool
	// 是否压缩/归档旧文件
	Compress bool
}

// NewOptions 创建一个带有默认参数的 Options 对象.
func NewOptions() *Options {
	return &Options{
		DisableCaller:     false,
		DisableStacktrace: false,
		Level:             zapcore.InfoLevel.String(),
		Format:            "console",
		OutputPaths:       []string{logFilePath},
		Maxsize:           100,
		MaxBackup:         30,
		MaxAge:            7,
		LocalTime:         true,
		Compress:          true,
	}
}

// 负责设置 encoding 的日志格式
func getEncoder() zapcore.Encoder {
	// 获取一个指定的的EncoderConfig，进行自定义
	encodeConfig := zap.NewProductionEncoderConfig()

	// 序列化时间。eg: 2022-09-01T19:11:35.921+0800
	encodeConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Local().Format("2006-01-02 15:04:05.000"))
	}
	// 自定义 TimeKey 为 timestamp，timestamp 语义更明确
	encodeConfig.TimeKey = "timestamp"
	// 指定 time.Duration 序列化函数，将 time.Duration 序列化为经过的毫秒数的浮点数
	// 毫秒数比默认的秒数更精确
	encodeConfig.EncodeDuration = func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendFloat64(float64(d) / float64(time.Millisecond))
	}
	// 将Level序列化为全大写字符串。例如，将info level序列化为INFO。
	encodeConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	// 以 package/file:行 的格式 序列化调用程序，从完整路径中删除除最后一个目录外的所有目录。
	encodeConfig.EncodeCaller = zapcore.ShortCallerEncoder

	return zapcore.NewJSONEncoder(encodeConfig)
}

// 负责日志写入的位置
func getLogWriter(filename string, maxsize, maxBackup, maxAge int) zapcore.WriteSyncer {
	if len(filename) == 0 {
		filename = filepath.Clean(filepath.Dir(logFilePath)) + string(filepath.Separator) + filepath.Base(logFilePath)
	}
	if maxsize == 0 {
		maxsize = 100 * 1024 * 1024
	}
	if maxBackup == 0 {
		maxBackup = 30
	}
	if maxAge == 0 {
		maxAge = 7
	}

	lumberJackLogger := &lumberjack.Logger{
		Filename:   filename,  // 文件位置
		MaxSize:    maxsize,   // 进行切割之前,日志文件的最大大小(MB为单位)
		MaxAge:     maxAge,    // 保留旧文件的最大天数
		MaxBackups: maxBackup, // 保留旧文件的最大个数
		LocalTime:  true,      // 是否按照天保留
		Compress:   false,     // 是否压缩/归档旧文件
	}

	return zapcore.AddSync(lumberJackLogger)
}

func createDirIfNotExists(logFilePath string) error {
	dir := filepath.Dir(logFilePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}
