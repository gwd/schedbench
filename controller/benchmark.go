package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"
	"io/ioutil"
	"encoding/json"
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

type BenchmarkRunData struct {
	Raw []WorkerReport
}

type BenchmarkRun struct {
	Completed bool
	Label string
	Workers []WorkerSet
	RuntimeSeconds int
	Results BenchmarkRunData 
}

func (run *BenchmarkRun) Run() (err error) {
	Workers, err := NewWorkerList(run.Workers, WorkerProcess)
	if err != nil {
		fmt.Println("Error creating workers: %v", err)
		return
	}
	
	report := make(chan WorkerReport)
	done := make(chan bool)
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, os.Interrupt)
	
	i := Workers.Start(report, done)

	// FIXME:
	// 1. Make a zero timeout mean "never"
	// 2. Make the signals / timeout thing a bit more rational; signal then timeout shouldn't hard kill
	timeout := time.After(time.Duration(run.RuntimeSeconds) * time.Second);
	stopped := false
	for i > 0 {
		select {
		case r := <-report:
			run.Results.Raw = append(run.Results.Raw, r)
			Report(&Workers[r.Id], r)
		case <-done:
			i--;
			fmt.Println(i, "workers left");
		case <-timeout:
			if ! stopped {
				Workers.Stop()
				stopped = true
				run.Completed = true
			}
		case <-signals:
			if ! stopped {
				fmt.Println("SIGINT receieved, shutting down workers")
				Workers.Stop()
				stopped = true
				if run.RuntimeSeconds == 0 {
					run.Completed = true
				}
				err = fmt.Errorf("Interrupted")
			} else {
				err = fmt.Errorf("Interrupted")
				fmt.Println("SIGINT received after stop, exiting without cleaning up")
				return
			}
		}
	}
	return
}

type BenchmarkPlan struct {
	filename string
	Runs []BenchmarkRun
}

func (plan *BenchmarkPlan) Run() (err error) {
	for i := range plan.Runs {
		if ! plan.Runs[i].Completed {
			fmt.Printf("Running test [%d] %s\n", i, plan.Runs[i].Label)
			err = plan.Runs[i].Run()
			if err != nil {
				return
			}
		}
		fmt.Printf("Test [%d] %s completed\n", i, plan.Runs[i].Label)
		err = plan.Save()
		if err != nil {
			fmt.Println("Error saving: ", err)
			return
		}
	}
	return
}

func LoadBenchmark(filename string) (plan BenchmarkPlan, err error) {
	plan.filename = filename
	
	var b []byte
	b, err = ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	
	err = json.Unmarshal(b, &plan)
	if err != nil {
		return
	}

	return
}

func (plan *BenchmarkPlan) Save() (err error) {
	if plan.filename == "" {
		err = fmt.Errorf("Invalid filename")
		return
	}
	
	var b []byte
	b, err = json.Marshal(*plan)
	if err != nil {
		return
	}

	backupFilename := fmt.Sprintf(".%s.tmp", plan.filename)
	err = os.Rename(plan.filename, backupFilename)
	if err != nil {
		if os.IsNotExist(err) {
			backupFilename = ""
		} else {
			return
		}
	}

	err = ioutil.WriteFile(plan.filename, b, 0666)
	if err != nil {
		if backupFilename != "" {
			os.Rename(backupFilename, plan.filename)
		}
		return
	}

	if backupFilename != "" {
		os.Remove(backupFilename)
	}
	return
}
