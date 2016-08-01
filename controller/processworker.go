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
	"os/exec"
	"encoding/json"
	"bufio"
	"io"
	
)

type ProcessWorker struct {
	id WorkerId
	c *exec.Cmd
	stdout io.ReadCloser
	jsonStarted bool
	Log []string
}

func (w *ProcessWorker) SetId(i WorkerId) {
	w.id = i
}

func (w *ProcessWorker) Init(p WorkerParams, g WorkerConfig) (err error) {
	w.c = exec.Command("./worker-proc", p.Args...)

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

func (w *ProcessWorker) DumpLog(f io.Writer) (err error) {
	b := bufio.NewWriter(f)
	defer b.Flush()
	for _, line := range w.Log {
		_, err = fmt.Println(b, line)
		if err != nil {
			return
		}
	}
	return
}

func (w *ProcessWorker) Process(report chan WorkerReport, done chan WorkerId) {
	w.c.Start()

	scanner := bufio.NewScanner(w.stdout)

	for scanner.Scan() {
		s := scanner.Text()
		
		//fmt.Println("Got these bytes: ", s);
		w.Log = append(w.Log, s)

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

	done <- w.id

	w.c.Wait()
}

