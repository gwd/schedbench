BIN = schedbench
BINALL = $(BIN)

.PHONY: all
all: $(BIN)



CGO_CFLAGS = -I/build/hg/xen.git/dist/install/usr/local/include
CGO_LIBS = -lyajl -lxenlight

# FIXME
XENLIB_PATH ?= /build/hg/xen.git/dist/install/usr/local/lib/
CGO_LDFLAGS = -L$(XENLIB_PATH) -Wl,-rpath-link=$(XENLIB_PATH) $(CGO_LIBS)

schedbench: main.go processworker.go xenworker.go benchmark.go run.go libxl.go htmlreport.go
	CGO_LDFLAGS="$(CGO_LDFLAGS)" CGO_CFLAGS="$(CGO_CFLAGS)" go build -o $@ $^

.PHONY: clean
clean:
	rm -f $(BINALL)

.PHONY: dist
dist:
	cp $(BIN) $(DISTDIR)
