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
)

type PlanSimpleMatrix struct {
	Schedulers []string
	Workers []string
	Count []int
}

type PlanInput struct {
	WorkerPresets map[string]WorkerParams
	SimpleMatrix *PlanSimpleMatrix
}

var WorkerPresets = map[string]WorkerParams{
	"P001":WorkerParams{[]string{"burnwait", "70", "200000"}},
}

func (plan *BenchmarkPlan) ClearRuns() (err error) {
	plan.Runs = nil

	return
}

func (plan *BenchmarkPlan) ExpandInput() (err error) {
	if plan.Runs != nil {
		err = fmt.Errorf("Runs non-empty, not doing anything\n");
		return
	}

	if plan.Input == nil {
		err = fmt.Errorf("Input nil, nothing to do")
		return
	}
	
	if plan.Input.SimpleMatrix == nil {
		err = fmt.Errorf("Input.SimpleMatrix nil, nothing to do\n");
		return
	}

	for k := range plan.Input.WorkerPresets {
		WorkerPresets[k] = plan.Input.WorkerPresets[k];
	}

	// Use named schedulers, or default to "" (which will use the
	// current one)
	var schedulers []string
	if plan.Input.SimpleMatrix.Schedulers != nil {
		schedulers = plan.Input.SimpleMatrix.Schedulers
	} else {
		schedulers = append(schedulers, "")
	}

	// Always do the baselines
	for _, wn := range plan.Input.SimpleMatrix.Workers {
		wp := WorkerPresets[wn]
		
		if wp.Args == nil {
			err = fmt.Errorf("Invalid worker preset: %s", wn)
			return
		}
		
		run := BenchmarkRun{
			WorkerSets:[]WorkerSet{{Params:wp, Count:1}},
			RuntimeSeconds:10,
		}
		
		for _, s := range schedulers {
			fmt.Printf("Making baseline %s run, sched %s\n", wn, s)
			run.RunConfig.Scheduler = s
			run.Label = wn+" baseline "+s
			plan.Runs = append(plan.Runs, run)
		}
	}
		
	for _, c := range plan.Input.SimpleMatrix.Count {
		run := BenchmarkRun{
			RuntimeSeconds:10,
		}
		
		var label string
		for _, wn := range plan.Input.SimpleMatrix.Workers {
			wp := WorkerPresets[wn]
			
			if label != "" {
				label = label+" + "
			}
			label = fmt.Sprintf("%s%s %d ", label, wn, c)
			
			ws := WorkerSet{Params:wp, Count:c}
			run.WorkerSets = append(run.WorkerSets, ws)
		}
		for _, s := range schedulers {
			fmt.Printf("Making count %d run, sched %s\n", c, s)
			run.RunConfig.Scheduler = s
			run.Label = label+s
			plan.Runs = append(plan.Runs, run)
		}
	}

	return
}
