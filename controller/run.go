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

// If the pool is specified, use that pool; otherwise assume pool 0.
//
// Unspecified schedulers match any pool; unspecifiend cpu lists match
// any pool.
//
// If the pool exists and the scheduler and cpu lists match the pool,
// carry on.  (This is running the VMs in a pre-configured pool.)
//
// If the pool exists and either the scheduler or the cpus don't match
// the pool, and this is pool 0, skip.
//
// TODO: If the scheduler matches but the cpus don't, modify the pool
// by adding or removing cpus.  (This can be done for Pool-0 as well.)
//
// If the pool is not Pool-0, and the scheduler doesn't match or the
// pool doesn't exist, but there are no cpus, skip (because we don't
// have enough information to create the pool).
//
// If the pool is not Pool-0, and either the scheduler or the cpus
// don't match, and the cpus are specified, create the pool.
func (run *BenchmarkRun) Prep() (ready bool, why string) {
	var pool CpupoolInfo
	poolPresent := false
	
	// Generate the requested cpumap
	var Cpumap Bitmap
	if run.RunConfig.Cpus != nil {
		fmt.Print("Run.Prep: Cpus: ")
		printed := false
		for _, i := range run.RunConfig.Cpus {
			if printed {
				fmt.Printf(",%d", i)
			} else {
				printed = true
				fmt.Printf("%d", i)
			}				
			Cpumap.Set(i)
		}
		fmt.Print("\n")
		if Cpumap.IsEmpty() {
			why = "Invalid (empty) cpumap"
			return
		}
	}
	

	if run.RunConfig.Pool == "" {
		fmt.Printf("Run.Prep: No pool set, using 0\n")
		pool = Ctx.CpupoolInfo(0)
		poolPresent = true
	} else {
		pool, poolPresent = Ctx.CpupoolFindByName(run.RunConfig.Pool)
		if poolPresent {
			fmt.Printf("Run.Prep: Pool %s found, Poolid %d\n",
				run.RunConfig.Pool, pool.Poolid)
		} else {
			fmt.Printf("Run.Prep: Pool %s not found\n")
		}
	}

	schedMatches := true
	if run.RunConfig.Scheduler != "" &&
		poolPresent &&
		pool.Scheduler.String() != run.RunConfig.Scheduler {
			schedMatches = false;
	}

	cpuMatches := true
	if run.RunConfig.Cpus != nil {
		if !poolPresent {
			cpuMatches = false
		} else {
			for i := 0; i <= pool.Cpumap.Max(); i++ {
				if pool.Cpumap.Test(i) != Cpumap.Test(i) {
					fmt.Printf("Prep: cpu %d: pool %v, want %v, bailing\n",
						i, pool.Cpumap.Test(i), Cpumap.Test(i))
					cpuMatches = false
					break
				}
			}
		}
	}
		

	// If we're using pool 0, and the scheduler or cpus don't
	// match, bail; otherwise say we're ready.
	if poolPresent && pool.Poolid == 0 {
		if ! schedMatches {
			why = "scheduler != "+run.RunConfig.Scheduler+", can't change"
			return
		}

		// TODO: Actually, we can modify pool 0; leave this until we want it.
		if ! cpuMatches {
			why = "Cpumap mismatch"
			return
		}

		fmt.Printf("Prep: Poolid 0, sched and cpumap matches\n")
		ready = true
		return
	}

	// OK, we got here it
	if run.RunConfig.Cpus == nil {
		// No construction information; is the cpupool ready without it?
		if !poolPresent {
			why = "Pool not present, no pool construction information"
			return
		} else if !schedMatches {
			why = "scheduler != "+run.RunConfig.Scheduler+", no pool construction information"
			return
		}

		// Scheduler matches, pool present, cpus not
		// specified, just go with it
		ready = true
		return
	}

	// OK, we have all the information we need to create the pool we want.
	Scheduler := SchedulerCredit
	err := Scheduler.FromString(run.RunConfig.Scheduler)
	if err != nil {
		why = "Invalid scheduler: "+run.RunConfig.Scheduler
		return
	}

	// Destroy the pool if it's present;
	if poolPresent {
		err := Ctx.CpupoolDestroy(pool.Poolid)
		if err != nil {
			fmt.Printf("Trying to destroy pool: %v\n", err)
			why = "Couldn't destroy cpupool"
			return
		}
	}

	// Free the cpus we need;
	err = Ctx.CpupoolMakeFree(Cpumap)
	if err != nil {
		why = "Couldn't free cpus"
		return
	}

	// And create the pool.
	err, _ = Ctx.CpupoolCreate("schedbench", Scheduler, Cpumap)
	if err != nil {
		why = "Couldn't create cpupool"
		return
	}

	ready = true
	return 
}

func (run *BenchmarkRun) GetCpumap() (Cpumap Bitmap) {
	if run.RunConfig.Pool == "" {
		fmt.Printf("Run.Prep: No pool set, using 0\n")
		pool := Ctx.CpupoolInfo(0)
		Cpumap = pool.Cpumap
	} else {
		pool, poolPresent := Ctx.CpupoolFindByName(run.RunConfig.Pool)
		if poolPresent {
			Cpumap = pool.Cpumap
		} else {
			panic("run.GetCpumap(): Pool "+run.RunConfig.Pool+" not found!")
		}
	}
	return
}

func (run *BenchmarkRun) Run() (err error) {
	for wsi := range run.WorkerSets {
		conf := &run.WorkerSets[wsi].Config
		
		conf.PropagateFrom(run.WorkerConfig)
		if conf.Pool == "" {
			conf.Pool = run.RunConfig.Pool
		}
		run.WorkerSets[wsi].Params.SetkHZ(CpukHZ)
		
		if *run.RunConfig.NumaDisable {
			if conf.SoftAffinity != "" {
				err = fmt.Errorf("Cannot disable Numa if SoftAffinity is set!")
				return
			}
			// Disable libxl NUMA by setting the soft
			// affinity to the set of cpus in the cpupool
		 	conf.SoftAffinity = run.GetCpumap().String()
			fmt.Printf("Setting SoftAffinity to %s to disable NUMA placement\n",
				conf.SoftAffinity)
		}
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
			r.RunConfig.PropagateFrom(plan.RunConfig)
			ready, why := r.Prep()
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

