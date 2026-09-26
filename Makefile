BINARY := coup
TARGETS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

.PHONY: build site release test verify clean

build: site
	CGO_ENABLED=0 go build -trimpath -o $(BINARY) ./cmd/coup

site:
	cd web && pnpm install --frozen-lockfile && pnpm build

release: site
	@for target in $(TARGETS); do \
		os=$${target%/*}; arch=$${target#*/}; \
		echo "release/$(BINARY)-$$os-$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags="-s -w" \
			-o release/$(BINARY)-$$os-$$arch ./cmd/coup || exit 1; \
	done

test:
	go test -race -count=1 ./...

verify: test
	go vet ./...
	test -z "$$(gofmt -l .)"
	cd web && pnpm build && pnpm lint

clean:
	rm -rf $(BINARY) release
