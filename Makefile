.PHONY: *

test:
	go test -tags mock -race -cover ./...

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4.0 run --color always ${args}

lint-fix:
	make lint args=--fix

ui-dev:
	cd ui && npm run dev

ui-build:
	cd ui && npm run build

ui-lint:
	cd ui && npm run lint

ui-lint-fix:
	cd ui && npm run lint:fix

build: ui-build
	go build

