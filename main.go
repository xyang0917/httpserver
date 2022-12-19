package main

import (
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var sugarLogger *zap.SugaredLogger
var atomicLogLevel = zap.NewAtomicLevel()

// https://www.liwenzhou.com/posts/Go/zap/
func InitLogger() {
	writeSyncer := getLogWriter()
	encoder := getEncoder()
	// 根据环境变量，设置日志级别
	atomicLogLevel.SetLevel(zapcore.DebugLevel)
	profile := os.Getenv("PROFILE")
	if profile == "PROD" {
		atomicLogLevel.SetLevel(zapcore.InfoLevel)
	}

	core := zapcore.NewCore(encoder, writeSyncer, atomicLogLevel)

	logger := zap.New(core, zap.AddCaller())
	sugarLogger = logger.Sugar()
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getLogWriter() zapcore.WriteSyncer {
	logPath := os.Getenv("LOG_PATH")
	logFileName := "httpsvc.log"
	if logPath == "" {
		logPath = "./" + logFileName
	} else {
		logPath += "/" + logFileName
	}

	lumberJackLogger := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    1,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   false,
	}

	ws := io.MultiWriter(lumberJackLogger, os.Stdout)
	return zapcore.AddSync(ws)
}

func index(w http.ResponseWriter, r *http.Request) {
	// 读取Request Header, 并写入Response Header
	for key, value := range r.Header {
		for _, v := range value {
			sugarLogger.Debugf("Request header: %s = %s", key, v)
			w.Header().Add(key, v)
		}
	}

	// 读取系统变量VERSION，并写入Response Header
	sysVersion := os.Getenv("VERSION")
	if sysVersion == "" {
		sysVersion = "0.0.1"
	}
	w.Header().Add("Version", sysVersion)

	w.WriteHeader(http.StatusOK)

	// 记录日志（客户端和响应码）
	clientIP := getClientIP(r)

	// Response Body
	body := "<h1>Hello World</h1>"
	body += "<p><h1>" + clientIP + "</h1></p>"
	w.Write([]byte(body))

	sugarLogger.Infof("Request client ip: %s", clientIP)
	sugarLogger.Infof("Response status code: %d", 200)

}

// 获取客户端IP
func getClientIP(r *http.Request) string {
	// X-Real-IP
	clientIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if clientIP != "" {
		return clientIP
	}

	// X-Forwarded-For，格式：client, proxy1, proxy2
	clientIP = strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if clientIP != "" {
		clientIP = strings.Split(clientIP, ",")[0]
		return clientIP
	}

	// ip:port
	clientIP, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return clientIP
	}

	return ""
}

// 健康检查
func healthz(w http.ResponseWriter, r *http.Request) {
	sugarLogger.Debugf("Request healthz: %s", getClientIP(r))
	w.WriteHeader(http.StatusOK)
	status := "success"
	w.Write([]byte(status))
}

func main() {
	InitLogger()
	defer sugarLogger.Sync()

	httpServer := http.NewServeMux()
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	// debug
	httpServer.HandleFunc("/debug/pprof/", pprof.Index)
	httpServer.HandleFunc("/debug/pprof/trace", pprof.Trace)
	httpServer.HandleFunc("/debug/pprof/profile", pprof.Profile)
	httpServer.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

	// 修改日志级别
	/*
		var levelMap = map[string]zapcore.Level{
			"debug":  zapcore.DebugLevel,
			"info":   zapcore.InfoLevel,
			"warn":   zapcore.WarnLevel,
			"error":  zapcore.ErrorLevel,
			"dpanic": zapcore.DPanicLevel,
			"panic":  zapcore.PanicLevel,
			"fatal":  zapcore.FatalLevel,
		}
		// 示例
		// curl -X PUT localhost:8080/log/level -H 'Content-Type: application/json' -d '{"level": "info"}'
	*/
	httpServer.HandleFunc("/log/level", atomicLogLevel.ServeHTTP)

	// biz
	httpServer.HandleFunc("/", index)
	httpServer.HandleFunc("/healthz", healthz)
	err := http.ListenAndServe(":"+serverPort, httpServer)
	sugarLogger.Infof("HttpServer start success, listen port: %s", serverPort)
	if err != nil {
		sugarLogger.Errorf("Start server error: %s", err.Error())
	}

}
