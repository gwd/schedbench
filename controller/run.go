/*
 * Copyright (C) 2016 George W. Dunlap, Citrix Systems UK Ltd
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU General Public License as
 * published by the Free Software Foundation; version 2 of the
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
	"regexp"
	"strconv"
	"bufio"
	"io"
)

type WorkerState struct {
	w Worker
	LastReport WorkerReport
}

type Worker interface {
	SetId(WorkerId)
	Init(WorkerParams, WorkerConfig) error
	Shutdown()
	Process(chan WorkerReport, chan WorkerId)
	DumpLog(io.Writer) error
}

func Report(ws *WorkerState, r WorkerReport) {
	//fmt.Println(r)

	lr := ws.LastReport

	if (lr.Now > 0) {
		time := float64(r.Now) / SEC
		dtime := float64(r.Now - lr.Now) / SEC
		kops := r.Kops - lr.Kops

		tput := Throughput(lr.Now, lr.Kops, r.Now, r.Kops)

		util := Utilization(lr.Now, lr.Cputime, r.Now, r.Cputime)
		
		fmt.Printf("%v %8.3f [%8.3f] cpu %8.3f [%8.3f] Kops: %8d [%8d] Tput: %4.2f Util: %4.2f\n",
			r.Id, time, dtime, r.Cputime.Seconds(), r.Cputime.Seconds() - lr.Cputime.Seconds(),
			r.Kops, kops, tput, util);
	}

	ws.LastReport = r
}

type WorkerList map[WorkerId]*WorkerState

func (ws *WorkerList) Start(report chan WorkerReport, done chan WorkerId) (i int) {
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
		
			ws.w.Init(WorkerSets[wsi].Params, WorkerSets[wsi].Config)

			wl[Id] = ws
		}
	}
	return
}

var CpukHZ uint64

func getCpuHz() (err error) {
	if CpukHZ == 0 {
		var cpuinfo *os.File
		cpuinfo, err = os.Open("/proc/cpuinfo")
		if err != nil {
			return
		}
		re := regexp.MustCompile("^cpu MHz\\s*: ([0-9.]+)$")
		scanner := bufio.NewScanner(cpuinfo)
		for scanner.Scan() {
			s := scanner.Text()
			m := re.FindStringSubmatch(s)
			if m != nil {
				var MHZ float64
				MHZ, err = strconv.ParseFloat(m[1], 64)
				if err != nil {
					return
				}
				CpukHZ = uint64(MHZ*1000)
				break
			}
		}
		if CpukHZ == 0 {
			err = fmt.Errorf("Couldn't find cpu MHz")
			return
		} else {
			fmt.Println("CpukHZ: ", CpukHZ)
			
		}
	}
	return
}

func (run *BenchmarkRun) Ready() (ready bool, why string) {
	// FIXME: Check WorkerType
	// Skip this run if it's not the scheduler we want
	if run.RunConfig.Scheduler != "" {
		var pool CpupoolInfo
		if run.WorkerConfig.Pool != "" {
			var found bool
			pool, found = Ctx.CpupoolFindByName(run.WorkerConfig.Pool)
			if !found {
				why = "cpupool error"
				return
			}
		} else {
			// xl defaults to cpupool 0
			plist := Ctx.ListCpupool()
			if len(plist) > 0 {
				pool = plist[0]
			} else {
				why = "cpupool error"
				return
			}
		}

		if pool.Scheduler.String() != run.RunConfig.Scheduler {
			why = "scheduler != "+run.RunConfig.Scheduler
			return 
		}
	}
	ready = true
	return 
}

func (run *BenchmarkRun) Run() (err error) {
	for wsi := range run.WorkerSets {
		run.WorkerSets[wsi].Config.PropagateFrom(run.WorkerConfig)
		run.WorkerSets[wsi].Params.SetkHZ(CpukHZ)
	}
	
	Workers, err := NewWorkerList(run.WorkerSets, WorkerXen)
	if err != nil {
		fmt.Println("Error creating workers: %v", err)
		return
	}
	
	report := make(chan WorkerReport)
	done := make(chan WorkerId)
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
			if ! stopped {
				run.Results.Raw = append(run.Results.Raw, r)
				Report(Workers[r.Id], r)
			}
		case did := <-done:
			if ! stopped {
				fmt.Println("WARNING: Worker", did, "left early, shutting down workers")
				Workers.Stop()
				stopped = true
				err = fmt.Errorf("Worker %v exited early", did)
				Workers[did].w.DumpLog(os.Stdout)
			}
			i--;
			fmt.Printf("Worker %v exited; %d workers left\n", did, i);
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
				fmt.Println("SIGINT received after stop, exiting without cleaning up")
				return
			}
		}
	}
	return
}

func (plan *BenchmarkPlan) Run() (err error) {

	err = getCpuHz()
	if err != nil {
		return
	}

	if plan.WorkerType == WorkerXen {
		err = Ctx.Open()
		if err != nil {
			return
		}
	}
	
	for i := range plan.Runs {
		r := &plan.Runs[i];
		if ! r.Completed { 
			r.WorkerConfig.PropagateFrom(plan.WorkerConfig)
			ready, why := r.Ready()
			if ready {
				fmt.Printf("Running test [%d] %s\n", i, r.Label)
				err = r.Run()
				if err != nil {
					return
				}
			} else {
				fmt.Printf("Test [%d]: %s skipped (%s)\n", i, r.Label, why)
			}
		}
		if r.Completed {
			fmt.Printf("Test [%d] %s completed\n", i, r.Label)
			err = plan.Save()
			if err != nil {
				fmt.Println("Error saving: ", err)
				return
			}
		}
	}
	return
}

