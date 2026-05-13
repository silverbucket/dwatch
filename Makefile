build:
	go build -o dwatch .

install: build
	cp dwatch /usr/local/bin/dwatch
	@echo "Installed to /usr/local/bin/dwatch"

uninstall:
	rm -f /usr/local/bin/dwatch

clean:
	rm -f dwatch

.PHONY: build install uninstall clean
