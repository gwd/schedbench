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
	"os/exec"
	"encoding/json"
	"bufio"
	"io"
)

type xenGlobal struct {
	Ctx Context
	count int
}

var xg xenGlobal

type XenWorker struct {
	id WorkerId
	Ctx Context
	vmname string
	domid int
	consoleCmd *exec.Cmd
	console io.ReadCloser
	jsonStarted bool
}

// We have to capitalize the element names so that the json class can
// get access to it; so annotate the elements so they come out lower
// case
type RumpRunConfigBlk struct {
	Source string     `json:"source"`
	Path string       `json:"path"`
	Fstype string     `json:"fstype"` 
	Mountpoint string `json:"mountpoint"`
}

type RumpRunConfig struct {
	Blk RumpRunConfigBlk `json:"blk"`
	Cmdline string       `json:"cmdline"`
	Hostname string      `json:"hostname"`
}

func (w *XenWorker) SetId(i WorkerId) {
	w.id = i
	w.vmname = fmt.Sprintf("worker-%v", i)
	w.domid = -1 // INVALID DOMID
}

func (w *XenWorker) Init(p WorkerParams, g WorkerConfig) (err error) {
	if xg.count == 0 {
		err = xg.Ctx.Open()
		if err != nil {
			return
		}
	}
	xg.count++
	
	mock := false
	
	// Make xl config file
	//  name=worker-$(id)

	cfgName := os.TempDir()+"/schedbench-"+w.vmname+".cfg"

	cfg, err := os.Create(cfgName)
	//defer os.Remove(cfgName)

	if err != nil {
		fmt.Printf("Error creating configfile %s: %v\n", cfgName, err)
		return
	}

	fmt.Fprintf(cfg, "name = '%s'\n", w.vmname)
	fmt.Fprintf(cfg, "kernel = 'worker-xen.img'\n")
	fmt.Fprintf(cfg, "memory = 32\n")
	fmt.Fprintf(cfg, "vcpus = 1\n")
	fmt.Fprintf(cfg, "on_crash = 'destroy'\n")

	if g.Pool != "" {
		fmt.Fprintf(cfg, "pool = '%s'\n", g.Pool)
	}

	
	// xl create -p [filename]
	{
		args := []string{"xl", "create", "-p", cfgName}
		if mock {
			args = append([]string{"echo"}, args...)
		}
		e := exec.Command(args[0], args[1:]...)
		
		e.Stdout = os.Stdout
		e.Stderr = os.Stderr

		err = e.Run()
		if err != nil {
			fmt.Printf("Error creating domain: %v\n", err)
			return
		}
	}

	// Get domid
	{
		var domidString []byte
		var args []string
		
		if mock {
			args = []string{"echo", "232"}
		} else {
			args = []string{"xl", "domid", w.vmname}
		}
		e := exec.Command(args[0], args[1:]...)

		domidString, err = e.Output()
		if err != nil {
			fmt.Printf("Error getting domid: %v\n", err)
			return
		}

		_, err = fmt.Sscanf(string(domidString), "%d\n", &w.domid)
		if err != nil {
			fmt.Printf("Error converting domid: %v\n", err)
			return
		}

		//fmt.Printf(" %s domid %d\n", w.vmname, w.domid)
	}
	
	// Set xenstore config
	{
		rcfg := RumpRunConfig{
			Blk:RumpRunConfigBlk{Source:"dev",
				Path:"virtual",
				Fstype:"kernfs",
				Mountpoint:"/kern"},
			Hostname:w.vmname}
		
		rcfg.Cmdline = "worker-xen.img"
		for _, a := range p.Args {
			rcfg.Cmdline += fmt.Sprintf(" %s", a)
		}

		var rcfgBytes []byte
	
		rcfgBytes, err = json.Marshal(rcfg)
		if err != nil {
			fmt.Printf("Error marshalling rumprun json: %v\n", err)
			return
		}

		//fmt.Printf("json:\n%s\n", string(rcfgBytes))
		rcfgPath := fmt.Sprintf("/local/domain/%d/rumprun/cfg", w.domid)

		//fmt.Printf("Writing to %s, json config %s\n", rcfgPath, rcfgBytes)
		
		args := []string{"xenstore-write", rcfgPath, string(rcfgBytes)}
		if mock {
			args = append([]string{"echo"}, args...)
		}
		e := exec.Command(args[0], args[1:]...)
		
		e.Stdout = os.Stdout
		e.Stderr = os.Stderr

		err = e.Run()
		if err != nil {
			fmt.Printf("Error writing json into xenstore: %v\n", err)
			return
		}
	}
	

	// Run console command, attach to w.console
	{
		args := []string{"xl", "console", w.vmname}
		if mock {
			args = append([]string{"echo"}, args...)
		}
		w.consoleCmd = exec.Command(args[0], args[1:]...)

		w.console, err = w.consoleCmd.StdoutPipe()
		if err != nil {
			fmt.Print("Conneting to stdout: ", err)
			return
		}

		w.consoleCmd.Start()
	}
	
	return
}

// FIXME: Return an error
func (w *XenWorker) Shutdown() {
	// xl destroy [vmname]
	e := exec.Command("xl", "destroy", w.vmname)

	e.Stdout = os.Stdout
	e.Stderr = os.Stderr

	xg.count--
	if xg.count == 0 {
		defer xg.Ctx.Close()
	}

	err := e.Run()
	if err != nil {
		fmt.Printf("Error destroying domain: %v\n", err)
		return
	}
}

// FIXME: Return an error
func (w *XenWorker) Process(report chan WorkerReport, done chan bool) {
	mock := false
	
	// xl unpause [vmname]
	args := []string{"xl", "unpause", w.vmname}
	if mock {
		args = append([]string{"echo"}, args...)
	}
	e := exec.Command(args[0], args[1:]...)

	err := e.Run()
	if err != nil {
		fmt.Printf("Error unpausing domain: %v\n", err)
		return
	}

	scanner := bufio.NewScanner(w.console)

	for scanner.Scan() {
		s := scanner.Text()
		
		//fmt.Println("Got these bytes: ", s);

		if w.jsonStarted {
			var r WorkerReport
			json.Unmarshal([]byte(s), &r)
			r.Id = w.id
			di, err := xg.Ctx.DomainInfo(Domid(w.domid))
			// Ignore errors for now
			if err == nil {
				r.Cputime = di.Cpu_time
			}
			report <- r
		} else {
			if s == "START JSON" {
				//fmt.Println("Got token to start parsing json")
				w.jsonStarted = true
			}
		}
	}

	done <- true

	w.consoleCmd.Wait()
}

