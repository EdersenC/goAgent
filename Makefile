GO ?= go

.PHONY: test vet fmt validate

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

validate: test vet
