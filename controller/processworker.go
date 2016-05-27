package main

import (
	"fmt"
	"os/exec"
	"encoding/json"
	"bufio"
	"io"
	
)

type ProcessWorker struct {
	id int
	c *exec.Cmd
	stdout io.ReadCloser
	jsonStarted bool
}

func (w *ProcessWorker) SetId(i int) {
	w.id = i
}

func (w *ProcessWorker) Init(p WorkerParams) (err error) {
	w.c = exec.Command("../worker/worker-proc", p.Args...)

	w.stdout, err = w.c.StdoutPipe()
	if err != nil {
		fmt.Print("Conneting to stdout: ", err)
		return
	}

	return
}

func (w *ProcessWorker) Shutdown() {
	w.c.Process.Kill()
}

func (w *ProcessWorker) Process(report chan WorkerReport, done chan bool) {
	w.c.Start()

	scanner := bufio.NewScanner(w.stdout)

	for scanner.Scan() {
		s := scanner.Text()
		
		//fmt.Println("Got these bytes: ", s);

		if w.jsonStarted {
			var r WorkerReport
			json.Unmarshal([]byte(s), &r)
			r.Id = w.id
			report <- r
		} else {
			if s == "START JSON" {
				//fmt.Println("Got token to start parsing json")
				w.jsonStarted = true
			}
		}
	}

	done <- true

	w.c.Wait()
}

