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
			filename:filename,
			Runs:[]BenchmarkRun{
				{Label:"baseline-a",
					Workers:[]WorkerSet{
						{Params:WorkerParams{workerA},
							Count:1}},
					RuntimeSeconds:10,},
				{Label:"baseline-b",
					Workers:[]WorkerSet{
						{Params:WorkerParams{workerB},
							Count:1}},
					RuntimeSeconds:10,},
			}}


		for i := 1; i <= 16 ; i *= 2 {
			label := fmt.Sprintf("%da+%db", i, i)
			run := BenchmarkRun{
				Label:label,
				Workers:[]WorkerSet{
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

