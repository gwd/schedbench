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
	"io/ioutil"
	"encoding/json"
	"math"
	"time"
)

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
	Kops int
	MaxDelta int
	Cputime time.Duration
}

type WorkerParams struct {
	Args []string
}

func (l *WorkerParams) SetkHZ(kHZ uint64) {
	if l.Args[0] == "kHZ" {
		l.Args[1] = fmt.Sprintf("%d", kHZ)
	} else {
		l.Args = append([]string{"kHZ", fmt.Sprintf("%d", kHZ)}, l.Args...)
	}
}

type WorkerConfig struct {
	Pool string
}

// Propagate unset values from a higher level
func (l *WorkerConfig) PropagateFrom(g WorkerConfig) {
	if l.Pool == "" {
		l.Pool = g.Pool
	}
}


type WorkerSet struct {
	Params WorkerParams
	Config WorkerConfig
	Count int
}

type Worker interface {
	SetId(WorkerId)
	Init(WorkerParams, WorkerConfig) error
	Shutdown()
	Process(chan WorkerReport, chan bool)
}

const (
	USEC = 1000
	MSEC = USEC * 1000
	SEC = MSEC * 1000
)

func Throughput(lt int, lm int, t int, m int) (tput float64) {
	time := float64(t - lt) / SEC
	kops := m - lm
	
	tput = float64(kops) / time
	return
}

func Utilization(lt int, lct time.Duration, t int, ct time.Duration) (util float64) {
	util = float64(ct - lct) / float64(t - lt)
	return
}

type MinMax struct {
	Min float64
	Max float64
}

func (mm *MinMax) Update(x float64) {
	if x > mm.Max {
		mm.Max = x
	}
	if x < mm.Min || mm.Min == 0 {
		mm.Min = x
	}
}

type WorkerSummary struct {
	Raw []WorkerReport
	MinMaxTput MinMax
	MinMaxUtil MinMax
	TotalTput int
	TotalTime time.Duration
	TotalCputime time.Duration
	AvgTput float64
	AvgUtil float64
}

type WorkerSetSummary struct {
	Workers    []WorkerSummary
	TotalTput     float64
	AvgAvgTput    float64
	MinMaxTput    MinMax
	MinMaxAvgTput MinMax
	AvgStdDevTput float64

	TotalUtil     float64
	MinMaxUtil    MinMax
	MinMaxAvgUtil MinMax
	AvgAvgUtil    float64
	AvgStdDevUtil float64
}

type BenchmarkRunData struct {
	Raw []WorkerReport       `json:",omitempty"`
	Summary []WorkerSetSummary  `json:",omitempty"`
}

type BenchmarkRun struct {
	Label string
	WorkerSets []WorkerSet
	WorkerConfig
	RuntimeSeconds int
	Completed bool
	Results BenchmarkRunData 
}

type BenchmarkPlan struct {
	filename string
	WorkerType int
	// Global options for workers that will be over-ridden by Run
	// and WorkerSet config options
	WorkerConfig
	Runs []BenchmarkRun
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

	run.Results.Summary = make([]WorkerSetSummary, len(run.WorkerSets))

	type Data struct{
		startTime int
		startCputime time.Duration
		lastTime int
		lastKops int
		lastCputime time.Duration
	}
	
	data := make(map[WorkerId]*Data)

	// FIXME: Filter out results which started before all have started
	for i := range run.Results.Raw {
		e := run.Results.Raw[i]

		if e.Id.Set > len(run.Results.Summary) {
			return fmt.Errorf("Internal error: e.Id.Set %d > len(Results.Summary) %d\n",
				e.Id.Set, len(run.Results.Summary))
		}
		
		if run.Results.Summary[e.Id.Set].Workers == nil {
			run.Results.Summary[e.Id.Set].Workers = make([]WorkerSummary,
				run.WorkerSets[e.Id.Set].Count)
		}

		ws := &run.Results.Summary[e.Id.Set]

		if e.Id.Id > len(ws.Workers) {
			return fmt.Errorf("Internal error: e.Id.Id %d > len(Results.Summary[].Workers) %d\n",
				e.Id.Id, len(ws.Workers))
		}

		s := &ws.Workers[e.Id.Id]

		s.Raw = append(s.Raw, e)
		
		d := data[e.Id]
		if d == nil {
			d = &Data{}
			data[e.Id] = d
		}
			
		if d.startTime == 0 {
			d.startTime = e.Now
			d.startCputime = e.Cputime
		} else {
			tput := Throughput(d.lastTime, d.lastKops, e.Now, e.Kops)
			util := Utilization(d.lastTime, d.lastCputime, e.Now, e.Cputime)

			s.MinMaxTput.Update(tput)
			s.MinMaxUtil.Update(util)
			ws.MinMaxTput.Update(tput)
			ws.MinMaxUtil.Update(util)
		}
		d.lastTime = e.Now
		d.lastKops = e.Kops
		d.lastCputime = e.Cputime
	}

	for Id, d := range data {
		ws := &run.Results.Summary[Id.Set]
		s := &ws.Workers[Id.Id]

		s.TotalTput = d.lastKops
		s.TotalTime = time.Duration(d.lastTime - d.startTime)
		s.TotalCputime = d.lastCputime - d.startCputime
		
		s.AvgTput = Throughput(d.startTime, 0, d.lastTime, d.lastKops)
		s.AvgUtil = Utilization(d.startTime, d.startCputime, d.lastTime, d.lastCputime)

		ws.MinMaxAvgTput.Update(s.AvgTput)
		ws.MinMaxAvgUtil.Update(s.AvgUtil)
	}

	// Calculate the average-of-averages for each set
	for set := range run.Results.Summary {
		ws := &run.Results.Summary[set]
		
		var totalTput float64
		var totalUtil float64
		var count int
		for id := range ws.Workers {
			totalTput += ws.Workers[id].AvgTput
			totalUtil += ws.Workers[id].AvgUtil
			count++
		}

		// FIXME -- Is this legit?
		ws.TotalTput = totalTput
		ws.TotalUtil = totalUtil
		ws.AvgAvgTput = totalTput / float64(count)
		ws.AvgAvgUtil = totalUtil / float64(count)
	}

	// Then calculate the standard deviation
	for set := range run.Results.Summary {
		ws := &run.Results.Summary[set]
		
		var totalAvgTput float64
		var totalAvgUtil float64
		var count int
		
		for id := range ws.Workers {
			d1 := ws.Workers[id].AvgTput - ws.AvgAvgTput
			d2 := ws.Workers[id].AvgUtil - ws.AvgAvgUtil
			totalAvgTput += d1 * d1
			totalAvgUtil += d2 * d2
			count++
		}
		v1 := totalAvgTput / float64(count)
		v2 := totalAvgUtil / float64(count)
		ws.AvgStdDevTput = math.Sqrt(v1)
		ws.AvgStdDevUtil = math.Sqrt(v2)
	}

	return
}

func (run *BenchmarkRun) TextReport(level int) (err error) {
	var done bool
	done, err = run.checkSummary()
	if err != nil {
		return
	}
	if ! done {
		err = fmt.Errorf("Run not yet processed")
		return
	}

	fmt.Printf("== RUN %s ==\n", run.Label)

	for set := range run.WorkerSets {
		ws := &run.WorkerSets[set]
		params := ""
		for _, s := range ws.Params.Args {
			params = fmt.Sprintf("%s %s", params, s)
		}
		fmt.Printf("Set %d: %s\n", set, params)
	}

	fmt.Printf("\n%8s %8s %8s %8s %8s %8s %8s %8s %8s %8s %8s %8s %8s %8s %8s\n", "set", "ttotal", "tavgavg", "tstdev", "tavgmax", "tavgmin", "ttotmax", "ttotmin", "utotal", "uavgavg", "ustdev", "uavgmax", "uavgmin", "utotmax", "utotmin")
	for set := range run.WorkerSets {
		ws := &run.Results.Summary[set]
		fmt.Printf("%8d %8.2f %8.2f %8.2f %8.2f %8.2f %8.2f %8.2f %8.2f %8.2f %8.2f %8.2f %8.2f %8.2f %8.2f\n",
			set,
			ws.TotalTput, ws.AvgAvgTput, ws.AvgStdDevTput, ws.MinMaxAvgTput.Max,
			ws.MinMaxAvgTput.Min, ws.MinMaxTput.Max, ws.MinMaxTput.Min,
			ws.TotalUtil, ws.AvgAvgUtil, ws.AvgStdDevUtil, ws.MinMaxAvgUtil.Max,
			ws.MinMaxAvgUtil.Min, ws.MinMaxUtil.Max, ws.MinMaxUtil.Min)
	}

	if level >= 1 {
 		fmt.Printf("\n%8s %8s %8s %8s %8s %8s %8s %8s %8s %8s\n", "workerid", "toput", "time", "cpu", "tavg", "tmin", "tmax", "uavg", "umin", "umax")
		for set := range run.Results.Summary {
			for id := range run.Results.Summary[set].Workers {
				s := run.Results.Summary[set].Workers[id]
				fmt.Printf("%2d:%2d    %10d %8.2f %8.2f %8.2f %8.2f %8.2f %8.2f %8.2f %8.2f\n",
					set, id,
					s.TotalTput, s.TotalTime.Seconds(), s.TotalCputime.Seconds(),
					s.AvgTput, s.MinMaxTput.Min, s.MinMaxTput.Max,
					s.AvgUtil, s.MinMaxUtil.Min, s.MinMaxUtil.Max)

				if level >= 2 {
					for _, e := range s.Raw {
						time := float64(e.Now) / SEC
						fmt.Printf ("   [%8.3f] %8.3f %8d %12d\n", time,
							e.Cputime.Seconds(), e.Kops, e.MaxDelta)
					}
				}

			}
		}
	}

	fmt.Printf("\n\n")

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

func (plan *BenchmarkPlan) TextReport(level int) (err error) {
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

		err = r.TextReport(level)
		if err != nil {
			return
		}
	}

	return
}
