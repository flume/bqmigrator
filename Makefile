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

validate-tag-arg:
ifeq ("", "$(v)")
	@echo "version arg (v) must be used with the 'tag' target"
	@exit 1;
endif
ifneq ("v", "$(shell echo $(v) | head -c 1)")
	@echo "version arg (v) must begin with v"
	@exit 1;
endif

tag:
	@echo "creating tag $(v)"
	bash ./scripts/tag.sh $(v)
