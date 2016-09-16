package main

/*
#include <libxl.h>
#include <stdlib.h>
*/
import "C"

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
// void libxl_bitmap_init(libxl_bitmap *map);
// void libxl_bitmap_dispose(libxl_bitmap *map);

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
	// Punt on cpumap for now
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
	} else {
		fmt.Printf("schedbench not found")
	}

	pool, found = Ctx.CpupoolFindByName("schedbnch")

	if found {
		fmt.Printf("%v\n", pool)
	} else {
		fmt.Printf("schedbnch not found")
	}

	Ctx.Close()
}
