CHS_ENV_HOME ?= $(HOME)/.chs_env
TESTS        ?= ./...

bin          := payments.api.ch.gov.uk
chs_envs     := $(CHS_ENV_HOME)/global_env $(CHS_ENV_HOME)/payments.api.ch.gov.uk/env
source_env   := for chs_env in $(chs_envs); do test -f $$chs_env && . $$chs_env; done
xunit_output := test.xml
lint_output  := lint.txt

commit       := $(shell git rev-parse --short HEAD)
tag          := $(shell git tag -l 'v*-rc*' --points-at HEAD)
version      := $(shell if [[ -n "$(tag)" ]]; then echo $(tag) | sed 's/^v//'; else echo $(commit); fi)

.PHONY: all
all: build

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: deps
deps:
	go get ./...

.PHONY: build
build: deps fmt $(bin)

$(bin):
	go build -o ./$(bin)

.PHONY: test-deps
test-deps: deps
	go get -t ./...

.PHONY: test
test: test-unit test-integration

.PHONY: test-unit
test-unit: test-deps
	set -a; go test $(TESTS) -run 'Unit'

.PHONY: test-util
test-util:
	set -a; go test $(TESTS) -run 'Util'

.PHONY: test-integration
test-integration: test-deps
	$(source_env); go test $(TESTS) -run 'Integration'

.PHONY: clean
clean:
	rm -f ./$(bin) ./$(bin)-*.zip $(test_path) build.log

.PHONY: package
package:
	$(eval tmpdir := $(shell mktemp -d build-XXXXXXXXXX))
	cp ./$(bin) $(tmpdir)
	cp ./start.sh $(tmpdir)
	cd $(tmpdir) && zip -r ../$(bin)-$(version).zip $(bin) start.sh
	rm -rf $(tmpdir)

.PHONY: dist
dist: clean build package

.PHONY: xunit-tests
xunit-tests: test-deps
	go get github.com/tebeka/go2xunit
	@set -a; $(test_unit_env); go test -v $(TESTS) -run 'Unit' | go2xunit -output $(xunit_output)

.PHONY: lint
lint:
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install
	gometalinter ./... > $(lint_output); true
