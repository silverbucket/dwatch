VERSION ?= 1.0.0
LDFLAGS  = -ldflags "-X dwatch/cmd.Version=$(VERSION)"

build:
	go build $(LDFLAGS) -o dwatch .

install: build
	cp dwatch /usr/local/bin/dwatch
	@echo "Installed dwatch $(VERSION) to /usr/local/bin/dwatch"

uninstall:
	rm -f /usr/local/bin/dwatch

clean:
	rm -f dwatch

release: clean build
	@echo "Built dwatch $(VERSION)"

.PHONY: build install uninstall clean release
