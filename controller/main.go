package main

import (
	"fmt"
	"os"
	"os/signal"
	
)

type WorkerReport struct {
	Id int
	Now int
	Mops int
	MaxDelta int
}

type Worker interface {
	SetId(int)
	Init() error
	Shutdown()
	Process(chan WorkerReport, chan bool)
}

const (
	USEC = 1000
	MSEC = USEC * 1000
	SEC = MSEC * 1000
)

type WorkerState struct {
	w Worker
	LastReport WorkerReport
}

func Report(ws *WorkerState, r WorkerReport) {
	//fmt.Println(r)

	lr := ws.LastReport

	if (lr.Now > 0) {
		time := float64(r.Now - lr.Now) / SEC
		mops := r.Mops - lr.Mops

		tput := float64(mops) / time

		fmt.Printf("%d Time: %2.3f Mops: %d Tput: %4.2f\n", r.Id, time, mops, tput);
	}

	ws.LastReport = r
}

func main() {
	count := 2
	
	report := make(chan WorkerReport)
	done := make(chan bool)
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, os.Interrupt)
	
	i := 0

	Workers := make([]WorkerState, count)
	
	for i = 0; i< count; i++ {
		Workers[i].w = &ProcessWorker{}
		Workers[i].w.SetId(i)
		
		Workers[i].w.Init()
		
		go Workers[i].w.Process(report, done)
	}

	for i > 0 {
		select {
		case r := <-report:
			Report(&Workers[r.Id], r)
		case <-done:
			i--;
			fmt.Println(i, "workers left");
		case <-signals:
			fmt.Println("SIGINT receieved, shutting down workers")
			for j := range Workers {
				Workers[j].w.Shutdown()
			}
		}
	}
}
