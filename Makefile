VERSION := $(shell grep 'var Version' main.go | sed 's/.*"\(.*\)"/\1/')

PLATFORMS := \
	darwin/arm64 \
	darwin/amd64 \
	linux/amd64 \
	linux/arm64 \
	windows/amd64

BINARIES := $(foreach p,$(PLATFORMS),dist/ai-trailer-$(subst /,-,$(p))$(if $(findstring windows,$(p)),.exe,))

.PHONY: build test release clean

build: $(BINARIES)

dist/ai-trailer-%:
	$(eval PARTS := $(subst -, ,$*))
	$(eval GOOS  := $(word 1,$(PARTS)))
	$(eval GOARCH := $(if $(findstring exe,$*),amd64,$(word 2,$(PARTS))))
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-s -w" -o $@ .

dist/ai-trailer-windows-amd64.exe:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $@ .

test:
	go test ./...

release: test build
	@echo "Releasing $(VERSION)..."
	gh release create $(VERSION) $(BINARIES) \
		--title "$(VERSION)" \
		--generate-notes
	git push origin master --tags
	@echo "Done: https://github.com/lucianopf/ai-trailer/releases/tag/$(VERSION)"

clean:
	rm -f $(BINARIES)
