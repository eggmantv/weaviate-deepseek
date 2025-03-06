package main

import (
	"flag"
	"go-weaviate-deepseek/ext/weaviatelib"
	"log"
	"os"
	"strconv"

	"go-weaviate-deepseek/conf"
	"go-weaviate-deepseek/conn"
	"go-weaviate-deepseek/ext"
	"go-weaviate-deepseek/jobs/api"
)

func main() {
	ext.RunWithRecover(createParentProcess)
}

func createParentProcess() {
	e := flag.String("e", "development", "production | development")
	flag.Parse()

	// setup logrus
	defer Prepare(*e)()

	log.Println("env:", *e)
	log.Println("tika host:", conf.TIKA_HOST)

	// redis
	_ = conn.RedisConnect()

	createChildProcess()
}

func createChildProcess() {
	c := make(chan string)

	ws := []*ext.Worker{
		ext.NewWorker("WebAPI", c, api.RunWebAPI, 1000, false),
	}

	// halt
	ext.RunWithWorkers(ws, c)

	log.Println("main process exit!")
	os.Exit(0)
}

func writePid(pid int) {
	d := []byte(strconv.Itoa(pid))
	_ = os.WriteFile("run.pid", d, 0644)
}

func Prepare(env string) func() {
	file := ext.SetLog(env)

	weaviateLog := ext.LF("weaviatelib")
	weaviatelib.Setup(weaviateLog)

	conf.Parse(env)

	weaviatelib.VectorizerFunc = api.Vectorizer

	return func() {
		file.Close()
	}
}
