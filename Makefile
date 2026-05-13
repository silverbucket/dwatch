VERSION ?= 1.0.0
DESTDIR ?= /usr/local/bin
LDFLAGS  = -ldflags "-X dwatch/cmd.Version=$(VERSION)"

build:
	go build $(LDFLAGS) -o dwatch .
	-codesign --sign - dwatch 2>/dev/null

install: build
	cp dwatch $(DESTDIR)/dwatch
	@echo "Installed dwatch $(VERSION) to $(DESTDIR)/dwatch"

uninstall:
	rm -f $(DESTDIR)/dwatch

clean:
	rm -f dwatch

release: clean build
	@echo "Built dwatch $(VERSION)"

.PHONY: build install uninstall clean release
