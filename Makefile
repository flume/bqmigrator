.PHONY: help
# help:
#    Print this help message
help:
	@grep -o '^\#.*' Makefile | cut -d" " -f2-

.PHONY: fmt
# fmt:
#    Format go code
fmt:
	goimports -local github.com/flume -w ./

.PHONY: lint
# lint:
#    Lint the code
lint:
	golangci-lint run

.PHONY: test
# test:
#    Run the tests
test:
	go test ./...

tag:
	@echo "creating tag"
	bash ./scripts/tag.sh
