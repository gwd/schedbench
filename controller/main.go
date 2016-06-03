package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"
)

type WorkerReport struct {
	Id int
	Now int
	Mops int
	MaxDelta int
}

type WorkerParams struct {
	Args []string
}

type WorkerSet struct {
	Params WorkerParams
	Count int
}

type BenchmarkParams struct {
	Workers []WorkerSet
	RuntimeSeconds int
}

type Worker interface {
	SetId(int)
	Init(WorkerParams) error
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

type WorkerList []WorkerState

func (ws *WorkerList) Start(report chan WorkerReport, done chan bool) (i int) {
	i = 0
	for j := range *ws {
		go (*ws)[j].w.Process(report, done)
		i++
	}
	return
}

func (ws *WorkerList) Stop() {
	for i := range *ws {
		(*ws)[i].w.Shutdown()
	}
}

const (
	WorkerProcess = iota
	WorkerXen = iota
)

func NewWorkerList(workers []WorkerSet, workerType int) (ws WorkerList, err error) {
	count := 0

	// wsi: WorkerSet index
	for wsi := range workers {
		count += workers[wsi].Count
	}

	fmt.Println("Making ", count, " total workers")
	ws = WorkerList(make([]WorkerState, count))

	// wli: WorkerList index
	wli := 0
	for wsi := range workers {
		for i := 0; i < workers[wsi].Count; i, wli = i+1, wli+1 {
			switch workerType {
			case WorkerProcess:
				ws[wli].w = &ProcessWorker{}
			case WorkerXen:
				ws[wli].w = &XenWorker{}
			default:
				err = fmt.Errorf("Unknown type: %d", workerType)
			}
			ws[wli].w.SetId(wli)
		
			ws[wli].w.Init(workers[wsi].Params)
		}
	}
	return
}

func main() {
	bp :=  BenchmarkParams{
		Workers:[]WorkerSet{
			{Params:WorkerParams{[]string{"burnwait", "20", "20000000"}},
				Count:2},
			{Params:WorkerParams{[]string{"burnwait", "10", "30000000"}},
			 	Count:3},
		},
		RuntimeSeconds:5,
	}

	Workers, err := NewWorkerList(bp.Workers, WorkerProcess)
	if err != nil {
		fmt.Println("Error creating workers: %v", err)
		return
	}
	
	report := make(chan WorkerReport)
	done := make(chan bool)
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, os.Interrupt)
	
	i := Workers.Start(report, done)

	timeout := time.After(time.Duration(bp.RuntimeSeconds) * time.Second);
	killed := false
	for i > 0 {
		select {
		case r := <-report:
			Report(&Workers[r.Id], r)
		case <-done:
			i--;
			fmt.Println(i, "workers left");
		case <-signals:
		case <-timeout:
			if ! killed {
				fmt.Println("SIGINT receieved, shutting down workers")
				Workers.Stop()
				killed = true
			} else {
				fmt.Println("Second SIGINT received, exiting without cleaning up")
				os.Exit(1)
			}
		}
	}
}
