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
	"io"
	"encoding/json"
)

type OptionAxis struct {
	Title string `json:"title"`
	MinValue float64 `json:"minValue"`
	MaxValue float64 `json:"maxValue"`
}

type Options struct {
	Title string     `json:"title"`
	HAxis OptionAxis `json:"hAxis"`
	VAxis OptionAxis `json:"vAxis"`
	Legend string    `json:"legend"`
}

type Point struct {
	x float64
	y float64
}

type RunRaw struct {
	Label string
	Points [][]Point
}

func (options *Options) OutputJavascript(w io.Writer, id int) (err error) {
	var optionsJson []byte
	optionsJson, err = json.Marshal(options)
	if err != nil {
		return
	}

	fmt.Fprintf(w, "        var sp%dopt = ", id)
	fmt.Fprint(w, string(optionsJson))
	fmt.Fprintln(w, ";")

	return
}

func (p *Point) OutputJson(w io.Writer, id int, max int) (err error) {
	fmt.Fprintf(w, "            [%f", p.x)
	for i := 0; i < max; i++ {
		if i == id {
			fmt.Fprintf(w, ", %f", p.y)
		} else {
			fmt.Fprintf(w, ", null")
		}
	}
	fmt.Fprint(w, "],\n")
	return
}

func (d *RunRaw) OutputHTML(w io.Writer, run int) (err error) {
	fmt.Fprintf(w, "    <div class='scatterplot' id='scatterplot%d'></div>\n", run)
	return
}

func (d *RunRaw) OutputJavascript(w io.Writer, run int) (err error) {
	var options Options

	options.Title = fmt.Sprintf("Run %s (%d) Individual Throughput", d.Label, run)
	options.HAxis.Title = "Time"
	options.VAxis.Title = "Throughput"

	var xmm MinMax
	var ymm MinMax
	for i := range d.Points {
		for j := range d.Points[i] {
			xmm.Update(d.Points[i][j].x)
			ymm.Update(d.Points[i][j].y)
		}
	}

	options.HAxis.MaxValue = xmm.Max
	options.VAxis.MaxValue = ymm.Max

	err = options.OutputJavascript(w, run)
	if err != nil {
		return
	}

	fmt.Printf("        var sp%ddata = new google.visualization.DataTable();\n", run)
	fmt.Printf("        sp%ddata.addColumn('number', 'Time');\n", run)
	for i := range d.Points {
		fmt.Printf("        sp%ddata.addColumn('number', 'Worker %d');\n", run, i)
	}
	fmt.Printf("        sp%ddata.addRows([\n", run)

	// Can't use json here because we need to be able to use 'null' for non-existent values
	for i := range d.Points {
		for j := range d.Points[i] {
			err = d.Points[i][j].OutputJson(w, i, len(d.Points))
			if err != nil {
				return
			}
		}
	}
	fmt.Print("          ]);\n")
	
	fmt.Printf("        var sp%dchart = new google.visualization.ScatterChart(document.getElementById('scatterplot%d'));\n", run, run);
	fmt.Printf("        sp%dchart.draw(sp%ddata, sp%dopt);\n\n", run, run, run)
	
	return
}

type HTMLReport struct {
	Raw []RunRaw
}


func (rpt *HTMLReport) Output(w io.Writer) (err error) {
	// Print start -> json charts
	fmt.Fprint(w,
		`<html>
  <head>
    <style>
      .scatterplot {
      margin:auto;
      width: 100vw;
      height: 60vw;
      }

      .empty {
      margin: auto;
      }
    </style>
    <script type="text/javascript" src="https://www.gstatic.com/charts/loader.js"></script>
    <script type="text/javascript">
      google.charts.load('current', {'packages':['corechart']});
      google.charts.setOnLoadCallback(drawCharts);
      function drawCharts() {
`);
	// Print json chart code
	for i := range rpt.Raw {
		err = rpt.Raw[i].OutputJavascript(w, i)
		if err != nil {
			return
		}
	}
	// Print json -> html
	fmt.Fprint(w,
		`      }
    </script>
  </head>
  <body>
`);
	// Print html
	for i := range rpt.Raw {
		err = rpt.Raw[i].OutputHTML(w, i)
		if err != nil {
			return
		}
	}
	// Print html -> end
	fmt.Fprint(w,
		`  </body>
</html>
`);
	return
}

func (rpt *HTMLReport) AddRun(run *BenchmarkRun) (err error) {
	var d RunRaw

	d.Label = run.Label
	for set := range run.Results.Summary {
		var idPoints []Point
		for id := range run.Results.Summary[set].Workers {
			var le WorkerReport
			for _, e := range run.Results.Summary[set].Workers[id].Raw {
				if e.Now > le.Now {
					time := float64(e.Now) / SEC
					tput := Throughput(e.Now, e.Kops, le.Now, le.Kops)
					idPoints = append(idPoints, Point{x:time, y:tput})
				}
				le = e
			}
		}
		d.Points = append(d.Points, idPoints)
	}
	rpt.Raw = append(rpt.Raw, d)
	return
}

func (plan *BenchmarkPlan) HTMLReport() (err error) {
	rpt := HTMLReport{}

	for i := range plan.Runs {
		r := &plan.Runs[i]
		if ! r.Completed {
			fmt.Printf("Test [%d] %s not run\n", i, r.Label)
		}

		err = r.Process()
		if err != nil {
			fmt.Printf("Error processing [%d] %s: %v\n", i, r.Label, err)
			return
		}

		err = rpt.AddRun(r)
		if err != nil {
			return
		}
	}
	err = rpt.Output(os.Stdout)

	return
}
