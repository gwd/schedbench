BIN = controller
BINALL = $(BIN)

.PHONY: all
all: $(BIN)

controller: main.go
	go build -o $@ $<

.PHONY: clean
clean:
	rm -f $(BINALL)
