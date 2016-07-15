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
)

func main() {
	filename := "test.bench"

	switch(os.Args[1]) {
	case "plan":
		workerA := []string{"burnwait", "20", "20000000"}
		//workerB := []string{"burnwait", "10", "20000000"}
		workerB := []string{"burnwait", "1", "20000000",
			"burnwait", "2", "20000000",
			"burnwait", "1", "20000000",
			"burnwait", "1", "20000000",
			"burnwait", "1", "20000000",
			"burnwait", "1", "20000000",
			"burnwait", "3", "20000000",
		}


		plan :=  BenchmarkPlan{
			WorkerType:WorkerXen,
			WorkerConfig:WorkerConfig{Pool:"schedbench"},
			filename:filename,
			Runs:[]BenchmarkRun{
				{Label:"baseline-a",
					WorkerSets:[]WorkerSet{
						{Params:WorkerParams{workerA},
							Count:1}},
					RuntimeSeconds:10,},
				{Label:"baseline-b",
					WorkerSets:[]WorkerSet{
						{Params:WorkerParams{workerB},
							Count:1}},
					RuntimeSeconds:10,},
			}}


		for i := 1; i <= 16 ; i *= 2 {
			label := fmt.Sprintf("%da+%db", i, i)
			run := BenchmarkRun{
				Label:label,
				WorkerSets:[]WorkerSet{
					{Params:WorkerParams{workerA},
						Count:i},
					{Params:WorkerParams{workerB},
						Count:i}},
				RuntimeSeconds:10}
			plan.Runs = append(plan.Runs, run)
		}
		
		err := plan.Save()
		if err != nil {
			fmt.Println("Saving plan ", filename, " ", err)
			os.Exit(1)
		}
		fmt.Println("Created plan in ", filename)
	case "run":
		plan, err := LoadBenchmark(filename)
		if err != nil {
			fmt.Println("Loading benchmark ", filename, " ", err)
			os.Exit(1)
		}
	
		err = plan.Run()
		if err != nil {
			fmt.Println("Running benchmark run:", err)
			os.Exit(1)
		}
		
	case "report":
		plan, err := LoadBenchmark(filename)
		if err != nil {
			fmt.Println("Loading benchmark ", filename, " ", err)
			os.Exit(1)
		}
	
		err = plan.TextReport()
		if err != nil {
			fmt.Println("Running benchmark run:", err)
			os.Exit(1)
		}
	default:
		fmt.Println("Unknown argument: ", os.Args[1])
	}
}

