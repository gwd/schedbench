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

/*
#cgo LDFLAGS: -lxenlight -lyajl_s -lxengnttab -lxenstore -lxenguest -lxentoollog -lxenevtchn -lxenctrl -lblktapctl -lxenforeignmemory -lxencall -lz -luuid -lutil
#include <stdlib.h>
#include <libxl.h>
*/
import "C"

/*
 * Other flags that may be needed at some point: 
 *  -lnl-route-3 -lnl-3
 *
 * To get back to simple dynamic linking:
#cgo LDFLAGS: -lxenlight -lyajl
*/

import (
	"unsafe"
	"fmt"
	"time"
)

type Context struct {
	ctx *C.libxl_ctx
}

var Ctx Context

func (Ctx *Context) IsOpen() bool {
	return Ctx.ctx != nil
}

func (Ctx *Context) Open() (err error) {
	if Ctx.ctx != nil {
		return
	}
	
	ret := C.libxl_ctx_alloc(unsafe.Pointer(&Ctx.ctx), C.LIBXL_VERSION, 0, nil)

	if ret != 0 {
		err = fmt.Errorf("Allocating libxl context: %d", ret)
	}
	return
}

func (Ctx *Context) Close() (err error) {
	ret := C.libxl_ctx_free(unsafe.Pointer(Ctx.ctx))
	Ctx.ctx = nil

	if ret != 0 {
		err = fmt.Errorf("Freeing libxl context: %d", ret)
	}
	return
}

// Builtins
type Domid uint32

type MemKB uint64

// FIXME: Use the idl to generate types
type Dominfo struct {
	// FIXME: uuid
	Domid             Domid
	Running           bool
	Blocked           bool
	Paused            bool
	Shutdown          bool
	Dying             bool
	Never_stop        bool
	
	Shutdown_reason   int32 // FIXME shutdown_reason enumeration
	Outstanding_memkb MemKB
	Current_memkb     MemKB
	Shared_memkb      MemKB
	Paged_memkb       MemKB
	Max_memkb         MemKB
	Cpu_time          time.Duration
	Vcpu_max_id       uint32
	Vcpu_online       uint32
	Cpupool           uint32
	Domain_type       int32 //FIXME libxl_domain_type enumeration

}

func (Ctx *Context) DomainInfo(Id Domid) (di *Dominfo, err error) {
	if Ctx.ctx == nil {
		err = fmt.Errorf("Context not opened")
		return
	}

		
	var cdi C.libxl_dominfo

	ret := C.libxl_domain_info(Ctx.ctx, unsafe.Pointer(&cdi), C.uint32_t(Id))

	// FIXME: IsDomainNotPresentError
	if ret != 0 {
		err = fmt.Errorf("libxl_domain_info failed: %d", ret)
		return
	}

	// FIXME -- use introspection to make this more robust
	di = &Dominfo{}
	di.Domid = Domid(cdi.domid)
	di.Running = bool(cdi.running)
	di.Blocked = bool(cdi.blocked)
	di.Paused = bool(cdi.paused)
	di.Shutdown = bool(cdi.shutdown)
	di.Dying = bool(cdi.dying)
	di.Never_stop = bool(cdi.never_stop)
	di.Shutdown_reason = int32(cdi.shutdown_reason)
	di.Outstanding_memkb = MemKB(cdi.outstanding_memkb)
	di.Current_memkb = MemKB(cdi.current_memkb)
	di.Shared_memkb = MemKB(cdi.shared_memkb)
	di.Paged_memkb = MemKB(cdi.paged_memkb)
	di.Max_memkb = MemKB(cdi.max_memkb)
	di.Cpu_time = time.Duration(cdi.cpu_time)
	di.Vcpu_max_id = uint32(cdi.vcpu_max_id)
	di.Vcpu_online = uint32(cdi.vcpu_online)
	di.Cpupool = uint32(cdi.cpupool)
	di.Domain_type = int32(cdi.domain_type)
	return
}

func (Ctx *Context) DomainUnpause(Id Domid) (err error) {
	if Ctx.ctx == nil {
		err = fmt.Errorf("Context not opened")
		return
	}

	ret := C.libxl_domain_unpause(Ctx.ctx, C.uint32_t(Id))

	if ret != 0 {
		err = fmt.Errorf("libxl_domain_unpause failed: %d", ret)
	}
	return
}


// typedef struct {
//     uint32_t size;          /* number of bytes in map */
//     uint8_t *map;
// } libxl_bitmap;

// Implement the Go bitmap type such that the underlying data can
// easily be copied in and out.  NB that we still have to do copies
// both directions, because cgo runtime restrictions forbid passing to
// a C function a pointer to a Go-allocated structure which contains a
// pointer.
type Bitmap struct {
	bitmap []C.uint8_t
}

func (bm *Bitmap) Alloc(max int) {
	bm.bitmap = make([]C.uint8_t, (max + 7) / 8)
}

// Return a Go bitmap which is a copy of the referred C bitmap.
func bitmapCToGo(cbm *C.libxl_bitmap) (bm Bitmap) {
	// Alloc a Go slice for the bytes
	size := int(cbm.size)
	bm.Alloc(size*8)

	// Make a slice pointing to the C array
	mapslice := (*[1 << 30]C.uint8_t)(unsafe.Pointer(cbm._map))[:size:size]

	// And copy the C array into the Go array
	copy(bm.bitmap, mapslice)

	return
}

func (bm *Bitmap) Test(bit int) (bool) {
	ubit := uint(bit)
	if (bit > bm.Max()) {
		return false
	}
	
	return (bm.bitmap[bit / 8] & (1 << (ubit & 7))) != 0
}

// FIXME: Do we really just want to silently fail here?
func (bm *Bitmap) Set(bit int) {
	ubit := uint(bit)
	if (bit > bm.Max()) {
		return
	}
	
	bm.bitmap[bit / 8] |= 1 << (ubit & 7)
}

func (bm *Bitmap) Clear(bit int) {
	ubit := uint(bit)
	if (bit > bm.Max()) {
		return
	}
	
	bm.bitmap[bit / 8] &= ^(1 << (ubit & 7))
}

func (bm *Bitmap) Max() (int) {
	return len(bm.bitmap) * 8
}

// # Consistent with values defined in domctl.h
// # Except unknown which we have made up
// libxl_scheduler = Enumeration("scheduler", [
//     (0, "unknown"),
//     (4, "sedf"),
//     (5, "credit"),
//     (6, "credit2"),
//     (7, "arinc653"),
//     (8, "rtds"),
//     ])
type Scheduler int
var (
	SchedulerUnknown  Scheduler = 0
	SchedulerSedf     Scheduler = 4
	SchedulerCredit   Scheduler = 5
	SchedulerCredit2  Scheduler = 6
	SchedulerArinc653 Scheduler = 7
	SchedulerRTDS     Scheduler = 8
)

// const char *libxl_scheduler_to_string(libxl_scheduler p);
// int libxl_scheduler_from_string(const char *s, libxl_scheduler *e);
func (s Scheduler) String() (string) {
	cs := C.libxl_scheduler_to_string(C.libxl_scheduler(s))
	// No need to free const return value

	return C.GoString(cs)
}

func SchedulerFromString(name string) (s Scheduler, err error) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))

	var cs C.libxl_scheduler

	ret := C.libxl_scheduler_from_string(cname, &cs)
	if ret != 0 {
		err = fmt.Errorf("libxl_scheduler_from_string failed: %d", ret)
		return
	}

	s = Scheduler(cs)

	return
}

// libxl_cpupoolinfo = Struct("cpupoolinfo", [
//     ("poolid",      uint32),
//     ("pool_name",   string),
//     ("sched",       libxl_scheduler),
//     ("n_dom",       uint32),
//     ("cpumap",      libxl_bitmap)
//     ], dir=DIR_OUT)

type CpupoolInfo struct {
	PoolId uint32
	PoolName string
	Scheduler Scheduler
	DomainCount int
	CpuMap Bitmap
}

// libxl_cpupoolinfo * libxl_list_cpupool(libxl_ctx*, int *nb_pool_out);
// void libxl_cpupoolinfo_list_free(libxl_cpupoolinfo *list, int nb_pool);
func (Ctx *Context) ListCpupool() (list []CpupoolInfo) {
	var nbPool C.int

	c_cpupool_list := C.libxl_list_cpupool(Ctx.ctx, &nbPool)

	defer C.libxl_cpupoolinfo_list_free(c_cpupool_list, nbPool)

	if int(nbPool) == 0 {
		return
	}

	// Magic
	cpupoolListSlice := (*[1 << 30]C.libxl_cpupoolinfo)(unsafe.Pointer(c_cpupool_list))[:nbPool:nbPool]

	for i := range cpupoolListSlice {
		var info CpupoolInfo
		
		info.PoolId = uint32(cpupoolListSlice[i].poolid)
		info.PoolName = C.GoString(cpupoolListSlice[i].pool_name)
		info.Scheduler = Scheduler(cpupoolListSlice[i].sched)
		info.DomainCount = int(cpupoolListSlice[i].n_dom)
		info.CpuMap = bitmapCToGo(&cpupoolListSlice[i].cpumap)

		list = append(list, info)
	}

	return
}

func (Ctx *Context) CpupoolFindByName(name string) (info CpupoolInfo, found bool) {
	plist := Ctx.ListCpupool()

	for i := range plist {
		if plist[i].PoolName == name {
			found = true
			info = plist[i]
			return
		}
	}
	return
}

// int libxl_cpupool_create(libxl_ctx *ctx, const char *name,
//                          libxl_scheduler sched,
//                          libxl_bitmap cpumap, libxl_uuid *uuid,
//                          uint32_t *poolid);
// int libxl_cpupool_destroy(libxl_ctx *ctx, uint32_t poolid);
// int libxl_cpupool_rename(libxl_ctx *ctx, const char *name, uint32_t poolid);
// int libxl_cpupool_cpuadd(libxl_ctx *ctx, uint32_t poolid, int cpu);
// int libxl_cpupool_cpuadd_node(libxl_ctx *ctx, uint32_t poolid, int node, int *cpus);
// int libxl_cpupool_cpuadd_cpumap(libxl_ctx *ctx, uint32_t poolid,
//                                 const libxl_bitmap *cpumap);
// int libxl_cpupool_cpuremove(libxl_ctx *ctx, uint32_t poolid, int cpu);
// int libxl_cpupool_cpuremove_node(libxl_ctx *ctx, uint32_t poolid, int node, int *cpus);
// int libxl_cpupool_cpuremove_cpumap(libxl_ctx *ctx, uint32_t poolid,
//                                    const libxl_bitmap *cpumap);
// int libxl_cpupool_movedomain(libxl_ctx *ctx, uint32_t poolid, uint32_t domid);
// int libxl_cpupool_info(libxl_ctx *ctx, libxl_cpupoolinfo *info, uint32_t poolid);

	
func XlTest(Args []string) {
	var Ctx Context

	err := Ctx.Open()
	if err != nil {
		fmt.Printf("Opening context: %v\n", err)
		return
	}

	pool, found := Ctx.CpupoolFindByName("schedbench")

	if found {
		fmt.Printf("%v\n", pool)

		a := int(pool.Scheduler)
		b := pool.Scheduler.String()
		c, err  := SchedulerFromString(b)

		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}

		fmt.Printf("a: %d b: %s c: %d\n", a, b, int(c))

		pool.CpuMap.Set(1)
		pool.CpuMap.Set(2)
		pool.CpuMap.Clear(2)
		
		fmt.Printf("cpumap: ")
		for i := 0; i < pool.CpuMap.Max() ; i++ {
			if pool.CpuMap.Test(i) {
				fmt.Printf("x")
			} else {
				fmt.Printf("-")
			}
		}
		fmt.Printf("\n")
	} else {
		fmt.Printf("schedbench not found")
	}

	pool, found = Ctx.CpupoolFindByName("schedbnch")

	if found {
		fmt.Printf("%v\n", pool)
	} else {
		fmt.Printf("schedbnch not found\n")
	}

	Ctx.Close()
}
