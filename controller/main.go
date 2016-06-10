package main

import (
	"fmt"
	"os"
)

func main() {
	filename := "test.bench"

	switch(os.Args[1]) {
	case "plan":
		plan :=  BenchmarkPlan{
			filename:filename,
			Runs:[]BenchmarkRun{
				{Label:"baseline-a",
					Workers:[]WorkerSet{
						{Params:WorkerParams{[]string{"burnwait", "20", "20000000"}},
							Count:1}},
					RuntimeSeconds:5,},
				{Label:"baseline-b",
					Workers:[]WorkerSet{
						{Params:WorkerParams{[]string{"burnwait", "10", "20000000"}},
							Count:1}},
					RuntimeSeconds:5,},
				{Label:"4a+4b",
					Workers:[]WorkerSet{
						{Params:WorkerParams{[]string{"burnwait", "20", "20000000"}},
							Count:4},
						{Params:WorkerParams{[]string{"burnwait", "10", "30000000"}},
							Count:4}},
					RuntimeSeconds:5,},
			}}
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
	default:
		fmt.Println("Unknown argument: ", os.Args[1])
	}
}

