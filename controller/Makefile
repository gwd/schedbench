BIN = schedbench
BINALL = $(BIN)

.PHONY: all
all: $(BIN)



CGO_CFLAGS = -I/build/hg/xen.git/dist/install/usr/local/include

# FIXME
XENLIB_PATH ?= /build/hg/xen.git/dist/install/usr/local/lib/
CGO_LDFLAGS = -L$(XENLIB_PATH) -Wl,-rpath-link=$(XENLIB_PATH) 

schedbench: main.go processworker.go xenworker.go benchmark.go run.go libxl.go htmlreport.go plan.go
	CGO_LDFLAGS="$(CGO_LDFLAGS)" CGO_CFLAGS="$(CGO_CFLAGS)" go build -ldflags '-linkmode external -extldflags "-static"' -o $@ $^

# If we use a statically linked binary we don't need this; the same
# binary can be used on any system.  Keep this version (without any
# run support) support) around for now in case we want to go back to
# it.
schedbench-report: main.go benchmark.go stubs.go htmlreport.go plan.go
	go build -o $@ $^

.PHONY: clean
clean:
	rm -f $(BINALL)

.PHONY: dist
dist:
	cp $(BIN) $(DISTDIR)
