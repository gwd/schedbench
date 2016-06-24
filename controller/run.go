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

func NewWorkerList(WorkerSets []WorkerSet, workerType int) (wl WorkerList, err error) {
	wl = WorkerList(make(map[WorkerId]*WorkerState))

	for wsi := range WorkerSets {
		for i := 0; i < WorkerSets[wsi].Count; i = i+1 {
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
		
			ws.w.Init(WorkerSets[wsi].Params)

			wl[Id] = ws
		}
	}
	return
}

func (run *BenchmarkRun) Run() (err error) {
	Workers, err := NewWorkerList(run.WorkerSets, WorkerXen)
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

