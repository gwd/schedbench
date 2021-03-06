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

/*
 * Types: Builtins
 */

type Domid uint32

type MemKB uint64

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

type Context struct {
	ctx *C.libxl_ctx
}

type Uuid C.libxl_uuid

/*
 * Types: IDL
 * 
 * FIXME: Generate these automatically from the IDL
 */
type Dominfo struct {
	Uuid              Uuid
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

func (c C.libxl_dominfo) toGo() (g Dominfo) {
	g.Uuid = Uuid(c.uuid)
	g.Domid = Domid(c.domid)
	g.Running = bool(c.running)
	g.Blocked = bool(c.blocked)
	g.Paused = bool(c.paused)
	g.Shutdown = bool(c.shutdown)
	g.Dying = bool(c.dying)
	g.Never_stop = bool(c.never_stop)
	g.Shutdown_reason = int32(c.shutdown_reason)
	g.Outstanding_memkb = MemKB(c.outstanding_memkb)
	g.Current_memkb = MemKB(c.current_memkb)
	g.Shared_memkb = MemKB(c.shared_memkb)
	g.Paged_memkb = MemKB(c.paged_memkb)
	g.Max_memkb = MemKB(c.max_memkb)
	g.Cpu_time = time.Duration(c.cpu_time)
	g.Vcpu_max_id = uint32(c.vcpu_max_id)
	g.Vcpu_online = uint32(c.vcpu_online)
	g.Cpupool = uint32(c.cpupool)
	g.Domain_type = int32(c.domain_type)

	return
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
	SchedulerUnknown  Scheduler = C.LIBXL_SCHEDULER_UNKNOWN
	SchedulerSedf     Scheduler = C.LIBXL_SCHEDULER_SEDF
	SchedulerCredit   Scheduler = C.LIBXL_SCHEDULER_CREDIT
	SchedulerCredit2  Scheduler = C.LIBXL_SCHEDULER_CREDIT2
	SchedulerArinc653 Scheduler = C.LIBXL_SCHEDULER_ARINC653
	SchedulerRTDS     Scheduler = C.LIBXL_SCHEDULER_RTDS
)

// libxl_cpupoolinfo = Struct("cpupoolinfo", [
//     ("poolid",      uint32),
//     ("pool_name",   string),
//     ("sched",       libxl_scheduler),
//     ("n_dom",       uint32),
//     ("cpumap",      libxl_bitmap)
//     ], dir=DIR_OUT)

type CpupoolInfo struct {
	Poolid uint32
	PoolName string
	Scheduler Scheduler
	DomainCount int
	Cpumap Bitmap
}

func (c C.libxl_cpupoolinfo) toGo() (g CpupoolInfo) {
	g.Poolid = uint32(c.poolid)
	g.PoolName = C.GoString(c.pool_name)
	g.Scheduler = Scheduler(c.sched)
	g.DomainCount = int(c.n_dom)
	g.Cpumap = bitmapCToGo(c.cpumap)

	return
}

/*
 * Context
 */
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

func (Ctx *Context) CheckOpen() (err error) {
	if Ctx.ctx == nil {
		err = fmt.Errorf("Context not opened")
	}
	return
}

func (Ctx *Context) DomainInfo(Id Domid) (di Dominfo, err error) {
	err = Ctx.CheckOpen()
	if err != nil {
		return
	}

	var cdi C.libxl_dominfo

	ret := C.libxl_domain_info(Ctx.ctx, unsafe.Pointer(&cdi), C.uint32_t(Id))

	// FIXME: IsDomainNotPresentError
	if ret != 0 {
		err = fmt.Errorf("libxl_domain_info failed: %d", ret)
		return
	}

	di = cdi.toGo()

	return
}

func (Ctx *Context) DomainUnpause(Id Domid) (err error) {
	err = Ctx.CheckOpen()
	if err != nil {
		return
	}

	ret := C.libxl_domain_unpause(Ctx.ctx, C.uint32_t(Id))

	if ret != 0 {
		err = fmt.Errorf("libxl_domain_unpause failed: %d", ret)
	}
	return
}

/*
 * Bitmap operations
 */

// Return a Go bitmap which is a copy of the referred C bitmap.
func bitmapCToGo(cbm C.libxl_bitmap) (gbm Bitmap) {
	// Alloc a Go slice for the bytes
	size := int(cbm.size)
	gbm.bitmap = make([]C.uint8_t, size)

	// Make a slice pointing to the C array
	mapslice := (*[1 << 30]C.uint8_t)(unsafe.Pointer(cbm._map))[:size:size]

	// And copy the C array into the Go array
	copy(gbm.bitmap, mapslice)

	return
}

// Must be C.libxl_bitmap_dispose'd of afterwards
func bitmapGotoC(gbm Bitmap) (cbm C.libxl_bitmap) {
	C.libxl_bitmap_init(&cbm)

	size := len(gbm.bitmap)
	cbm._map = (*C.uint8_t)(C.malloc(C.size_t(size)))
	cbm.size = C.uint32_t(size)
	if cbm._map == nil {
		panic("C.calloc failed!")
	}

	// Make a slice pointing to the C array
	mapslice := (*[1 << 30]C.uint8_t)(unsafe.Pointer(cbm._map))[:size:size]

	// And copy the Go array into the C array
	copy(mapslice, gbm.bitmap)

	return
}

func (bm *Bitmap) Test(bit int) (bool) {
	ubit := uint(bit)
	if (bit > bm.Max() || bm.bitmap == nil) {
		return false
	}
	
	return (bm.bitmap[bit / 8] & (1 << (ubit & 7))) != 0
}

func (bm *Bitmap) Set(bit int) {
	ibit := bit / 8;
	if (ibit + 1 > len(bm.bitmap)) {
		bm.bitmap = append(bm.bitmap, make([]C.uint8_t, ibit+1-len(bm.bitmap))...)
	}
	
	bm.bitmap[ibit] |= 1 << (uint(bit) & 7)
}

func (bm *Bitmap) SetRange(start int, end int) {
	for i := start; i <= end; i++ {
		bm.Set(i)
	}
}

func (bm *Bitmap) Clear(bit int) {
	ubit := uint(bit)
	if (bit > bm.Max() || bm.bitmap == nil) {
		return
	}
	
	bm.bitmap[bit / 8] &= ^(1 << (ubit & 7))
}

func (bm *Bitmap) ClearRange(start int, end int) {
	for i := start; i <= end; i++ {
		bm.Clear(i)
	}
}

func (bm *Bitmap) Max() (int) {
	return len(bm.bitmap) * 8 - 1
}

func (bm *Bitmap) IsEmpty() (bool) {
	for i:=0; i<len(bm.bitmap); i++ {
		if bm.bitmap[i] != 0 {
			return false
		}
	}
	return true
}

func (a Bitmap) And(b Bitmap) (c Bitmap) {
	var max, min int
	if len(a.bitmap) > len(b.bitmap) {
		max = len(a.bitmap)
		min = len(b.bitmap)
	} else {
		max = len(b.bitmap)
		min = len(a.bitmap)
	}
	c.bitmap = make([]C.uint8_t, max)

	for i := 0; i < min; i++ {
		c.bitmap[i] = a.bitmap[i] & b.bitmap[i]
	}
	return
}

func (bm Bitmap) String() (s string) {
	lastOnline := false
	crange := false
	printed := false
	var i int
	/// --x-xxxxx-x -> 2,4-8,10
	/// --x-xxxxxxx -> 2,4-10
	for i = 0; i <= bm.Max(); i++ {
		if bm.Test(i) {
			if !lastOnline {
				// Switching offline -> online, print this cpu
				if printed {
					s += ","
				}
				s += fmt.Sprintf("%d", i)
				printed = true
			} else if !crange {
				// last was online, but we're not in a range; print -
				crange = true
				s += "-"
			} else {
				// last was online, we're in a range,  nothing else to do
			}
			lastOnline = true
		} else {
			if lastOnline {
				// Switching online->offline; do we need to end a range?
				if crange {
					s += fmt.Sprintf("%d", i-1)
				}
			}
			lastOnline = false
			crange = false
		}
	}
	if lastOnline {
		// Switching online->offline; do we need to end a range?
		if crange {
			s += fmt.Sprintf("%d", i-1)
		}
	}

	return
}

// const char *libxl_scheduler_to_string(libxl_scheduler p);
func (s Scheduler) String() (string) {
	cs := C.libxl_scheduler_to_string(C.libxl_scheduler(s))
	// No need to free const return value

	return C.GoString(cs)
}

// int libxl_scheduler_from_string(const char *s, libxl_scheduler *e);
func (s *Scheduler) FromString(gstr string) (err error) {
	cstr := C.CString(gstr)
	defer C.free(unsafe.Pointer(cstr))

	var cs C.libxl_scheduler
	ret := C.libxl_scheduler_from_string(cstr, &cs)
	if ret != 0 {
		err = fmt.Errorf("libxl_scheduler_from_string: %d\n", ret)
		return
	}

	*s = Scheduler(cs)
	return
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

// libxl_cpupoolinfo * libxl_list_cpupool(libxl_ctx*, int *nb_pool_out);
// void libxl_cpupoolinfo_list_free(libxl_cpupoolinfo *list, int nb_pool);
func (Ctx *Context) ListCpupool() (list []CpupoolInfo) {
	err := Ctx.CheckOpen()
	if err != nil {
		return
	}

	var nbPool C.int

	c_cpupool_list := C.libxl_list_cpupool(Ctx.ctx, &nbPool)

	defer C.libxl_cpupoolinfo_list_free(c_cpupool_list, nbPool)

	if int(nbPool) == 0 {
		return
	}

	// Magic
	cpupoolListSlice := (*[1 << 30]C.libxl_cpupoolinfo)(unsafe.Pointer(c_cpupool_list))[:nbPool:nbPool]

	for i := range cpupoolListSlice {
		info := cpupoolListSlice[i].toGo()
		
		list = append(list, info)
	}

	return
}

// int libxl_cpupool_info(libxl_ctx *ctx, libxl_cpupoolinfo *info, uint32_t poolid);
func (Ctx *Context) CpupoolInfo(Poolid uint32) (pool CpupoolInfo) {
	err := Ctx.CheckOpen()
	if err != nil {
		return
	}

	var c_cpupool C.libxl_cpupoolinfo
	
	ret := C.libxl_cpupool_info(Ctx.ctx, &c_cpupool, C.uint32_t(Poolid))
	if ret != 0 {
		err = fmt.Errorf("libxl_cpupool_info failed: %d", ret)
		return
	}
	defer C.libxl_cpupoolinfo_dispose(&c_cpupool)

	pool = c_cpupool.toGo()

	return
}



// int libxl_cpupool_create(libxl_ctx *ctx, const char *name,
//                          libxl_scheduler sched,
//                          libxl_bitmap cpumap, libxl_uuid *uuid,
//                          uint32_t *poolid);
// FIXME: uuid
// FIXME: Setting poolid
func (Ctx *Context) CpupoolCreate(Name string, Scheduler Scheduler, Cpumap Bitmap) (err error, Poolid uint32) {
	err = Ctx.CheckOpen()
	if err != nil {
		return
	}

	poolid := C.uint32_t(0)
	name := C.CString(Name)
	defer C.free(unsafe.Pointer(name))
	
	// For now, just do what xl does, and make a new uuid every time we create the pool
	var uuid C.libxl_uuid
	C.libxl_uuid_generate(&uuid)

	cbm := bitmapGotoC(Cpumap)
	defer C.libxl_bitmap_dispose(&cbm)
	
	ret := C.libxl_cpupool_create(Ctx.ctx, name, C.libxl_scheduler(Scheduler),
		cbm, &uuid, &poolid)
	// FIXME: Proper error
	if ret != 0 {
		err = fmt.Errorf("libxl_cpupool_create failed: %d", ret)
		return
	}

	Poolid = uint32(poolid)
	
	return
}

// int libxl_cpupool_destroy(libxl_ctx *ctx, uint32_t poolid);
func (Ctx *Context) CpupoolDestroy(Poolid uint32) (err error) {
	err = Ctx.CheckOpen()
	if err != nil {
		return
	}

	ret := C.libxl_cpupool_destroy(Ctx.ctx, C.uint32_t(Poolid))
	// FIXME: Proper error
	if ret != 0 {
		err = fmt.Errorf("libxl_cpupool_destroy failed: %d", ret)
		return
	}

	return
}

// int libxl_cpupool_cpuadd(libxl_ctx *ctx, uint32_t poolid, int cpu);
func (Ctx *Context) CpupoolCpuadd(Poolid uint32, Cpu int) (err error) {
	err = Ctx.CheckOpen()
	if err != nil {
		return
	}

	ret := C.libxl_cpupool_cpuadd(Ctx.ctx, C.uint32_t(Poolid), C.int(Cpu))
	// FIXME: Proper error
	if ret != 0 {
		err = fmt.Errorf("libxl_cpupool_cpuadd failed: %d", ret)
		return
	}

	return
}

// int libxl_cpupool_cpuadd_cpumap(libxl_ctx *ctx, uint32_t poolid,
//                                 const libxl_bitmap *cpumap);
func (Ctx *Context) CpupoolCpuaddCpumap(Poolid uint32, Cpumap Bitmap) (err error) {
	err = Ctx.CheckOpen()
	if err != nil {
		return
	}

	cbm := bitmapGotoC(Cpumap)
	defer C.libxl_bitmap_dispose(&cbm)
	
	ret := C.libxl_cpupool_cpuadd_cpumap(Ctx.ctx, C.uint32_t(Poolid), &cbm)
	// FIXME: Proper error
	if ret != 0 {
		err = fmt.Errorf("libxl_cpupool_cpuadd_cpumap failed: %d", ret)
		return
	}

	return
}

// int libxl_cpupool_cpuremove(libxl_ctx *ctx, uint32_t poolid, int cpu);
func (Ctx *Context) CpupoolCpuremove(Poolid uint32, Cpu int) (err error) {
	err = Ctx.CheckOpen()
	if err != nil {
		return
	}

	ret := C.libxl_cpupool_cpuremove(Ctx.ctx, C.uint32_t(Poolid), C.int(Cpu))
	// FIXME: Proper error
	if ret != 0 {
		err = fmt.Errorf("libxl_cpupool_cpuremove failed: %d", ret)
		return
	}

	return
}

// int libxl_cpupool_cpuremove_cpumap(libxl_ctx *ctx, uint32_t poolid,
//                                    const libxl_bitmap *cpumap);
func (Ctx *Context) CpupoolCpuremoveCpumap(Poolid uint32, Cpumap Bitmap) (err error) {
	err = Ctx.CheckOpen()
	if err != nil {
		return
	}

	cbm := bitmapGotoC(Cpumap)
	defer C.libxl_bitmap_dispose(&cbm)
	
	ret := C.libxl_cpupool_cpuremove_cpumap(Ctx.ctx, C.uint32_t(Poolid), &cbm)
	// FIXME: Proper error
	if ret != 0 {
		err = fmt.Errorf("libxl_cpupool_cpuremove_cpumap failed: %d", ret)
		return
	}

	return
}

// int libxl_cpupool_rename(libxl_ctx *ctx, const char *name, uint32_t poolid);
// int libxl_cpupool_cpuadd_node(libxl_ctx *ctx, uint32_t poolid, int node, int *cpus);
// int libxl_cpupool_cpuremove_node(libxl_ctx *ctx, uint32_t poolid, int node, int *cpus);
// int libxl_cpupool_movedomain(libxl_ctx *ctx, uint32_t poolid, uint32_t domid);

//
// Utility functions
//
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

func (Ctx *Context) CpupoolMakeFree(Cpumap Bitmap) (err error) {
	plist := Ctx.ListCpupool()

	for i := range plist {
		var Intersection Bitmap
		Intersection = Cpumap.And(plist[i].Cpumap)
		if ! Intersection.IsEmpty() {
			err = Ctx.CpupoolCpuremoveCpumap(plist[i].Poolid, Intersection)
			if err != nil {
				return
			}
		}
	}
	return
}

func XlTest(Args []string) {
	var Cpumap Bitmap

	Cpumap.Set(2)
	Cpumap.SetRange(4, 8)
	Cpumap.Set(10)

	fmt.Printf("Cpumap: %v\n", Cpumap)

	Cpumap.Set(9)

	fmt.Printf("Cpumap: %v\n", Cpumap)

	var Ctx Context

	err := Ctx.Open()
	if err != nil {
		fmt.Printf("Opening context: %v\n", err)
		return
	}

	pool, found := Ctx.CpupoolFindByName("schedbench")

	if found {
		fmt.Printf("Found schedbench, destroying\n")

		err = Ctx.CpupoolDestroy(pool.Poolid)
		if err != nil {
			fmt.Printf("Couldn't destroy pool: %v\n", err)
			return
		}

		fmt.Printf("Returning cpus to pool 0 for fun\n")
		err = Ctx.CpupoolCpuaddCpumap(0, pool.Cpumap)
		if err != nil {
			fmt.Printf("Couldn't add cpus to domain 0: %v\n", err)
			return
		}
	}

	Cpumap = Bitmap{}

	Cpumap.SetRange(12, 15)

	fmt.Printf("Freeing cpus\n")
	err = Ctx.CpupoolMakeFree(Cpumap)
	if err != nil {
		fmt.Printf("Couldn't free cpus: %v\n", err)
		return
	}


	fmt.Printf("Creating new pool\n")

	err, Poolid := Ctx.CpupoolCreate("schedbench", SchedulerCredit, Cpumap)
	if err != nil {
		fmt.Printf("Error creating cpupool: %v\n", err)
	} else {
		fmt.Printf("Pool id: %d\n", Poolid)
	}

	pool = Ctx.CpupoolInfo(0)
	fmt.Printf("Cpupool 0 info: %v\n", pool)
	
	Ctx.Close()
}
