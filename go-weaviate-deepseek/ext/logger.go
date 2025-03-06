package ext

import (
	"io"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var L *logrus.Logger

func SetLog(e string) *os.File {
	var logPath string
	if e == "development" {
		logPath = "run.log"
	} else {
		logPath = "/var/log/ais/run.log"
	}
	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		L.Fatal(err)
		return nil
	}
	dst := io.MultiWriter(f, os.Stdout)

	// set standard log
	log.SetOutput(dst)
	log.SetFlags(log.LstdFlags)
	log.SetPrefix("[STDLOG]")

	// set logrus
	L = logrus.New()
	L.SetNoLock()
	// webHook := newWebHooker(fn)
	// L.AddHook(webHook)
	if e == "development" {
		L.SetOutput(os.Stdout)
	} else {
		L.SetOutput(dst)
		// L.SetFormatter(&logrus.JSONFormatter{})
	}
	L.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	// L.SetReportCaller(true)

	// set Gin log
	gin.DefaultWriter = dst
	gin.DefaultErrorWriter = dst

	return f
}

func LF(serviceName string) *logrus.Entry {
	return L.WithField("service", serviceName)
}
