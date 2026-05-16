VERSION ?= 1.1.1
DESTDIR ?= /usr/local/bin
LDFLAGS  = -ldflags "-X dwatch/cmd.Version=$(VERSION)"

build:
	go build $(LDFLAGS) -o dwatch .
	-codesign --sign - dwatch 2>/dev/null

install: build
	cp dwatch $(DESTDIR)/dwatch
	@mkdir -p $(HOME)/.dwatch
	@if [ ! -f $(HOME)/.dwatch/dwatch.conf ]; then \
		cp example.dwatch.conf $(HOME)/.dwatch/dwatch.conf; \
		echo "Installed default config to $(HOME)/.dwatch/dwatch.conf"; \
	else \
		echo "Config already exists at $(HOME)/.dwatch/dwatch.conf (skipping)"; \
	fi
	@echo "Installed dwatch $(VERSION) to $(DESTDIR)/dwatch"

uninstall:
	rm -f $(DESTDIR)/dwatch

clean:
	rm -f dwatch

release: clean build
	@echo "Built dwatch $(VERSION)"

.PHONY: build install uninstall clean release
