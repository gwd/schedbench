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
	Title string `json:"title,omitempty"`
	// Always include this one so that we can start graphs at 0
	MinValue float64 `json:"minValue"`
	MaxValue float64 `json:"maxValue,omitempty"`
}

type Options struct {
	Title string     `json:"title,omitempty"`
	HAxis OptionAxis `json:"hAxis"`
	VAxis OptionAxis `json:"vAxis"`
	Legend string    `json:"legend,omitempty"`
}

type Point struct {
	x float64
	y float64
}

type RunRaw struct {
	Tag string
	Title string
	hTitle string
	vTitle string
	Points [][]Point
}

func (options *Options) OutputJavascript(w io.Writer, tag string) (err error) {
	var optionsJson []byte
	optionsJson, err = json.Marshal(options)
	if err != nil {
		return
	}

	fmt.Fprintf(w, "        var %sopt = ", tag)
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

func (d *RunRaw) OutputHTML(w io.Writer) (err error) {
	fmt.Fprintf(w, "    <div class='scatterplot' id='scatterplot%s'></div>\n", d.Tag)
	return
}

func (d *RunRaw) OutputJavascript(w io.Writer) (err error) {
	var options Options

	options.Title = d.Title
	options.HAxis.Title = d.hTitle
	options.VAxis.Title = d.vTitle

	err = options.OutputJavascript(w, d.Tag)
	if err != nil {
		return
	}

	fmt.Printf("        var %sdata = new google.visualization.DataTable();\n", d.Tag)
	fmt.Printf("        %sdata.addColumn('number', 'Time');\n", d.Tag)
	for i := range d.Points {
		fmt.Printf("        %sdata.addColumn('number', 'Worker %d');\n", d.Tag, i)
	}
	fmt.Printf("        %sdata.addRows([\n", d.Tag)

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
	
	fmt.Printf("        var %schart = new google.visualization.ScatterChart(document.getElementById('scatterplot%s'));\n", d.Tag, d.Tag);
	fmt.Printf("        %schart.draw(%sdata, %sopt);\n\n", d.Tag, d.Tag, d.Tag)
	
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
		err = rpt.Raw[i].OutputJavascript(w)
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
		err = rpt.Raw[i].OutputHTML(w)
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
	var tPut RunRaw
	var Util RunRaw

	tPut.Title = fmt.Sprintf("Run %s Individual Throughput", run.Label)
	tPut.hTitle = "Time (s)"
	tPut.vTitle = "Throughput (kOps)"
	Util.Title = fmt.Sprintf("Run %s Individual Utilization", run.Label)
	Util.hTitle = "Time (s)"
	Util.vTitle = "Utilization"
	for set := range run.Results.Summary {
		var idTput []Point
		var idUtil []Point
		for id := range run.Results.Summary[set].Workers {
			var le WorkerReport
			for _, e := range run.Results.Summary[set].Workers[id].Raw {
				if e.Kops > 0 {
					time := float64(e.Now) / SEC
					tput := Throughput(e.Now, e.Kops, le.Now, le.Kops)
					util := Utilization(e.Now, e.Cputime, le.Now, le.Cputime)
					idTput = append(idTput, Point{x:time, y:tput})
					idUtil = append(idUtil, Point{x:time, y:util})
				}
				le = e
			}
		}
		tPut.Points = append(tPut.Points, idTput)
		Util.Points = append(Util.Points, idUtil)
	}
	tPut.Tag = fmt.Sprintf("raw%d", len(rpt.Raw))
	rpt.Raw = append(rpt.Raw, tPut)
	Util.Tag = fmt.Sprintf("raw%d", len(rpt.Raw))
	rpt.Raw = append(rpt.Raw, Util)
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
