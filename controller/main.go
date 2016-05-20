package main

import (
	"fmt"
	"os/exec"
	"encoding/json"
	"bufio"
	"io"
)

type Worker struct {
	c *exec.Cmd

	stdout io.ReadCloser
	
	jsonStarted bool
}

type WorkerReport struct {
	Now int
	Mops int
	MaxDelta int
}

func (w *Worker) Start() (err error) {
	w.c = exec.Command("../worker/worker-proc", "burnwait", "20", "20000000")
	

	w.stdout, err = w.c.StdoutPipe()
	if err != nil {
		fmt.Print("Conneting to stdout: ", err)
		return
	}

	w.c.Start()

	b, err := json.Marshal(WorkerReport{5,6,7})
	fmt.Print("Example json: ", string(b))
	
	return
}

func (w *Worker) Wait() {
	w.c.Wait()
}

func (w *Worker) Process() {
	scanner := bufio.NewScanner(w.stdout)

	for scanner.Scan() {
		s := scanner.Text()
		
		fmt.Println("Got these bytes: ", s);

		if w.jsonStarted {
			var r WorkerReport
			
			json.Unmarshal([]byte(s), &r)
			fmt.Println(r)
		} else {
			if s == "START JSON" {
				fmt.Println("Got token to start parsing json")
				w.jsonStarted = true
			}
		}
	}
}

func main() {

	w:=Worker{}
	
	w.Start()

	w.Process()

	w.Wait()
}
