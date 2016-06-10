BIN = schedbench
BINALL = $(BIN)

.PHONY: all
all: $(BIN)

schedbench: main.go processworker.go xenworker.go benchmark.go
	go build -o $@ $^

.PHONY: clean
clean:
	rm -f $(BINALL)

.PHONY: dist
dist:
	cp $(BIN) $(DISTDIR)
