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
	"io"
)

type XenWorker struct {
}

func XlTest(Args []string) {
	return
}

func (w *XenWorker) SetId(i WorkerId) {
}

func (w *XenWorker) Init(p WorkerParams, g WorkerConfig) (err error) {
	err = fmt.Errorf("Xen functionality not implemented");
	return
}

// FIXME: Return an error
func (w *XenWorker) Shutdown() {
	
	return
}

func (w *XenWorker) DumpLog(f io.Writer) (err error) {
	err = fmt.Errorf("Xen functionality not implemented");
	return
}



// FIXME: Return an error
func (w *XenWorker) Process(report chan WorkerReport, done chan WorkerId) {
	return;
}

