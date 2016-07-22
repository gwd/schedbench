package main

/*
#include <libxl.h>
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

func NewContext() (Ctx *Context, err error) {
	Ctx = &Context{}
	
	err = Ctx.Open()

	return
}

func (Ctx *Context) IsOpen() bool {
	return Ctx.ctx != nil
}

func (Ctx *Context) Open() (err error) {
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
