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
	NumaDisable []bool
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

	// Start by making a slice with baselines and each of the counts
	var a, b []BenchmarkRun
	
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

		run.Label = wn+" baseline"
		a = append(a, run)
	}


	for _, c := range plan.Input.SimpleMatrix.Count {
		run := BenchmarkRun{
			RuntimeSeconds:10,
		}
		
		for _, wn := range plan.Input.SimpleMatrix.Workers {
			wp := WorkerPresets[wn]
			
			if run.Label != "" {
				run.Label = run.Label+" + "
			}
			run.Label = fmt.Sprintf("%s%s %d", run.Label, wn, c)

			ws := WorkerSet{Params:wp, Count:c}
			run.WorkerSets = append(run.WorkerSets, ws)
		}

		a = append(a, run)
	}

	// ...then cross it by schedulers
	if len(schedulers) > 0 {
		for _, base := range a {
			for _, s := range schedulers {
				run := base
				run.RunConfig.Scheduler = s
				run.Label = run.Label+" "+s
				b = append(b, run)
			}
		}
		a = b
		b = nil
	}

	// ...and NumaDisable
	if len(plan.Input.SimpleMatrix.NumaDisable) > 0 {
		for _, base := range a {
			for _, d := range plan.Input.SimpleMatrix.NumaDisable {
				run := base
				// Need to make a copy of this so that
				// we have a pointer to use as a tristate
				run.RunConfig.NumaDisable = new(bool)
				*run.RunConfig.NumaDisable = d
				if d {
					run.Label = run.Label+" NumaOff"
				} else {
					run.Label = run.Label+" NumaOn "
				}
				b = append(b, run)
			}
		}
		a = b
		b = nil
	}

	for i := range a {
		fmt.Printf("%s\n", a[i].Label)
	}
	plan.Runs = a;
	return
}
