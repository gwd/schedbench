CFLAGS += -Werror -g -O2
LDFLAGS += -static
#LDFLAGS += -lyajl_s

RUMPCFLAGS = $(CFLAGS)
#RUMPCFLAGS += -I/build/hg/rumprun-packages.git/pkgs/include/
RUMPLDFLAGS = $(LDFLAGS) 
#RUMPLDFLAGS += -L/build/hg/rumprun-packages.git/pkgs/lib

BIN = worker-proc worker-xen.img
BINALL = $(BIN) worker-xen

.PHONY: all
all: $(BIN)

worker-proc: worker.c
	$(CC) -o $@ $< $(LDFLAGS) 

worker-xen: worker.c
	$(RUMPCC) $(RUMPCFLAGS) -o $@ $< $(RUMPLDFLAGS) 

worker-xen.img: worker-xen
	rumprun-bake xen_pv worker-xen.img worker-xen

.PHONY: clean
clean:
	rm -f *.o $(BINALL)

.PHONY: dist
dist:
	cp $(BIN) $(DISTDIR)
