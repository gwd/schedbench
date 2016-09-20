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
	"os"
	"strconv"
)

func main() {
	Args := os.Args

	Args = Args[1:]
	filename := "test.bench"
	verbosity := 0

	for len(Args) > 0 {
		switch(Args[0]) {
		case "-f":
			if len(Args) < 2 {
				fmt.Println("Need arg for -f")
				os.Exit(1)
			}
			filename = Args[1]
			Args = Args[2:]
		case "-v":
			if len(Args) < 2 {
				fmt.Println("Need arg for -v")
				os.Exit(1)
			}
			verbosity, _ = strconv.Atoi(Args[1])
			Args = Args[2:]
		case "plan":
			plan, err := LoadBenchmark(filename)
			if err != nil {
				fmt.Println("Loading benchmark ", filename, " ", err)
				os.Exit(1)
			}
			
			plan.ExpandInput()

			err = plan.Save()
			if err != nil {
				fmt.Println("Saving plan ", filename, " ", err)
				os.Exit(1)
			}
			fmt.Println("Created plan in ", filename)
			Args = Args[1:]
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
			Args = Args[1:]
			
		case "report":
			Args = Args[1:]
			plan, err := LoadBenchmark(filename)
			if err != nil {
				fmt.Println("Loading benchmark ", filename, " ", err)
				os.Exit(1)
			}
			
			err = plan.TextReport(verbosity)
			if err != nil {
				fmt.Println("Running benchmark run:", err)
				os.Exit(1)
			}
		case "htmlreport":
			plan, err := LoadBenchmark(filename)
			if err != nil {
				fmt.Println("Loading benchmark ", filename, " ", err)
				os.Exit(1)
			}
			
			err = plan.HTMLReport()
			if err != nil {
				fmt.Println("Running benchmark run:", err)
				os.Exit(1)
			}
			Args = Args[1:]

		case "xltest":
			XlTest(Args)
			Args = nil
			
		default:
			fmt.Println("Unknown argument: ", Args[0])
			os.Exit(1)
		}
	}
}

