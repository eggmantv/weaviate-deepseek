package ext

import (
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"
)

// Process ...
type Process struct {
	Workers []*Worker
	Running bool
}

// TrapSignal ...
func (mp *Process) TrapSignal() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT,
		syscall.SIGUSR1, syscall.SIGUSR2)
	go mp.HandleSignal(c)
}

// HandleSignal ...
func (mp *Process) HandleSignal(c chan os.Signal) {
	s := <-c
	switch s {
	case syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT:
		for _, w := range mp.Workers {
			// TODO, why need goroutine here???
			go w.Stop()
		}
	case syscall.SIGUSR1:
		L.Println("USR1 signal received, continue")
		mp.HandleSignal(c)
	case syscall.SIGUSR2:
		L.Println("USR2 signal received, continue")
		mp.HandleSignal(c)
	default:
		L.Println("unhandled signal:", s)
		mp.HandleSignal(c)
	}
}

type jobFunc func(c chan string)

// Worker ...
type Worker struct {
	Chan     chan string   // parent between child
	Sleep    time.Duration // milliseconds
	Name     string
	Running  bool
	Job      jobFunc
	Cycling  bool
	QuitChan chan string
}

// Stop quit worker
func (w *Worker) Stop() {
	w.Running = false
	w.QuitChan <- "stop"
}

// Run ...
func (w *Worker) Run() {
	if w.Cycling {
		for {
			if w.Running {
				// FIX
				// defer capturePanic(w)
				runWorkerWithRecover(w)
				time.Sleep(time.Millisecond * w.Sleep)
			} else {
				w.Chan <- w.Name + " exited gracefully"
				break
			}
		}
	} else {
		// non-cyclic worker will halt
		runWorkerWithRecover(w)
		w.Chan <- w.Name + " exited gracefully"
	}
}

// NewWorker ...
func NewWorker(name string, c chan string, job jobFunc, sleep time.Duration, cyclic bool) *Worker {
	L.Println(name, "registered")

	quitChan := make(chan string)
	return &Worker{
		Name:     name,
		Sleep:    sleep,
		Chan:     c,
		Running:  true,
		Job:      job,
		Cycling:  cyclic,
		QuitChan: quitChan,
	}
}

// RunWithWorkers run works
func RunWithWorkers(ws []*Worker, c chan string) {
	mp := Process{
		Workers: ws,
		Running: true,
	}
	mp.TrapSignal()

	for _, w := range mp.Workers {
		go w.Run()
	}

	// wait for routines to finish
	for i := 0; i < len(ws); i++ {
		L.Println(<-c)
	}
}

func runWorkerWithRecover(w *Worker) {
	defer func() {
		if e := recover(); e != nil {
			L.Errorf("FATAL in worker: %s\n%s", e, debug.Stack())
			L.Errorln(w.Name, "recovered, will try next time")

			// retry, will exit if failed again
			// time.Sleep(3 * time.Second)
			// w.Job(w.QuitChan)

			// 发生panic不停止线程
			// w.Chan <- fmt.Sprintf("%s quit w exception: %s", w.Name, e)
		}
	}()

	w.Job(w.QuitChan)
}

// RunWithRecover recover不能抓获routine中的panic
//
//	usage: go RunWithRecover(func(){})
func RunWithRecover(worker func()) {
	defer func() {
		if err := recover(); err != nil {
			L.Printf("FATAL in routine: %s\n%s", err, debug.Stack())
		}
	}()
	worker()
}
