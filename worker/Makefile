CFLAGS += -Werror -g -O2

.PHONY: all
all: worker

worker: worker.o
	$(CC) $(LDFLAGS) -o $@ $< 
