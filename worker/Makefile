RUMPCC=x86_64-rumprun-netbsd-gcc
CFLAGS += -Werror -g -O2

BIN = worker-proc worker-xen.img
BINALL = $(BIN) worker-xen

.PHONY: all
all: $(BIN)

worker-proc: worker.o
	$(CC) $(LDFLAGS) -o $@ $< 

worker-xen: worker.c
	$(RUMPCC) $(LDFLAGS) -o $@ $< 

worker-xen.img: worker-xen
	rumprun-bake xen_pv worker-xen.img worker-xen

.PHONY: clean
clean:
	rm -f *.o $(BINALL)
