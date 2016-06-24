/*
 * Copyright (C) 2016 George W. Dunlap, Citrix Systems UK Ltd
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU General Public License as
 * published by the Free Software Foundation; either version 2 of the
 * License only.
 *
 * This program is distributed in the hope that it will be useful, but
 * WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * General Public License for more details.
 * 
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA
 * 02110-1301, USA.
 */
package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"
	"io/ioutil"
	"encoding/json"
)

type WorkerSummary struct {
	MaxTput float64
	AvgTput float64
	MinTput float64
}

type WorkerId struct {
	Set int
	Id int
}

func (wid WorkerId) String() (string) {
	return fmt.Sprintf("%d:%d", wid.Set, wid.Id)
}

type WorkerReport struct {
	Id WorkerId
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
	SetId(WorkerId)
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

func Throughput(lt int, lm int, t int, m int) (tput float64) {
	time := float64(t - lt) / SEC
	mops := m - lm
	
	tput = float64(mops) / time
	return
}

func Report(ws *WorkerState, r WorkerReport) {
	//fmt.Println(r)

	lr := ws.LastReport

	if (lr.Now > 0) {
		time := float64(r.Now - lr.Now) / SEC
		mops := r.Mops - lr.Mops

		tput := Throughput(lr.Now, lr.Mops, r.Now, r.Mops)
		
		fmt.Printf("%v Time: %2.3f Mops: %d Tput: %4.2f\n", r.Id, time, mops, tput);
	}

	ws.LastReport = r
}

type WorkerList map[WorkerId]*WorkerState

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

func NewWorkerList(workers []WorkerSet, workerType int) (wl WorkerList, err error) {
	wl = WorkerList(make(map[WorkerId]*WorkerState))

	for wsi := range workers {
		for i := 0; i < workers[wsi].Count; i = i+1 {
			Id := WorkerId{Set:wsi,Id:i}

			ws := wl[Id]

			if ws != nil {
				panic("Duplicate worker for id!")
			}
			
			ws = &WorkerState{}
			
			switch workerType {
			case WorkerProcess:
				ws.w = &ProcessWorker{}
			case WorkerXen:
				ws.w = &XenWorker{}
			default:
				err = fmt.Errorf("Unknown type: %d", workerType)
				return
			}
			
			ws.w.SetId(Id)
		
			ws.w.Init(workers[wsi].Params)

			wl[Id] = ws
		}
	}
	return
}

type BenchmarkRunData struct {
	Raw []WorkerReport       `json:",omitempty"`
	Summary map[WorkerId]*WorkerSummary  `json:",omitempty"`
}

type BenchmarkRun struct {
	Label string
	Workers []WorkerSet
	RuntimeSeconds int
	Completed bool
	Results BenchmarkRunData 
}

func (run *BenchmarkRun) Run() (err error) {
	Workers, err := NewWorkerList(run.Workers, WorkerXen)
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
			Report(Workers[r.Id], r)
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

func (run *BenchmarkRun) checkSummary() (done bool, err error) {
	if run.Results.Summary != nil {
		done = true
		return 
	}
	
	return
}

func (run *BenchmarkRun) Process() (err error) {
	done, err := run.checkSummary()
	if done || err != nil {
		return
	}

	run.Results.Summary = make(map[WorkerId]*WorkerSummary)

	type Data struct{
		startTime int
		lastTime int
		lastMops int
	}
	
	data := make(map[WorkerId]*Data)

	// FIXME: Filter out results which started before all have started
	for i := range run.Results.Raw {
		e := run.Results.Raw[i]
		
		d := data[e.Id]
		if d == nil {
			d = &Data{}
			data[e.Id] = d
		}
		s := run.Results.Summary[e.Id]
		if s == nil {
			s = &WorkerSummary{}
			run.Results.Summary[e.Id] = s
		}

		if d.startTime == 0 {
			d.startTime = e.Now
		} else {
			tput := Throughput(d.lastTime, d.lastMops, e.Now, e.Mops)
		
			if tput > s.MaxTput {
				s.MaxTput = tput
			}
			if tput < s.MinTput || s.MinTput == 0 {
				s.MinTput = tput
			}
		}
		d.lastTime = e.Now
		d.lastMops = e.Mops
	}

	for Id := range data {
		run.Results.Summary[Id].AvgTput = Throughput(data[Id].startTime,
			0, data[Id].lastTime, data[Id].lastMops)
	}
	
	return
}

func (run *BenchmarkRun) TextReport() (err error) {
	var done bool
	done, err = run.checkSummary()
	if err != nil {
		return
	}
	if ! done {
		err = fmt.Errorf("Run not yet processed")
		return
	}

	fmt.Printf("== RUN %s ==", run.Label)

	wStart := 0
	for i := range run.Workers {
		ws := &run.Workers[i]
		n := ws.Count
		params := ""
		for _, s := range ws.Params.Args {
			params = fmt.Sprintf("%s %s", params, s)
		}
		fmt.Printf("[%d-%d]: %s\n", wStart, wStart+n-1, params)
		wStart += n
	}

	fmt.Printf("\n%8s %8s %8s %8s\n", "id", "avg", "min", "max")
	for id, s := range run.Results.Summary {
		fmt.Printf("%8v %8.2f %8.2f %8.2f\n",
			id, s.AvgTput, s.MinTput, s.MaxTput)
	}

	return
}

type BenchmarkPlan struct {
	filename string
	WorkerType int
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

func (plan *BenchmarkPlan) TextReport() (err error) {
	for i := range plan.Runs {
		r := &plan.Runs[i]
		if ! r.Completed {
			fmt.Printf("Test [%d] %s not run\n", i, r.Label)
		}

		err = r.Process()
		if err != nil {
			fmt.Printf("Error processing [%d] %s: %v\n", i, r.Label, err)
			return
		}

		err = r.TextReport()
		if err != nil {
			return
		}
	}

	return
}
