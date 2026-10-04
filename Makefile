APP := build/CmdKeySwitcher.app
GO := mise exec -- go
GOFMT := mise exec -- gofmt
CLANG_FORMAT := mise exec -- clang-format
GO_FILES := $(wildcard src/*.go)
NATIVE_FILES := src/native_darwin.m src/native.h

.PHONY: setup build test format check analyze install uninstall
setup:
	bash scripts/setup.sh

build:
	mkdir -p "$(APP)/Contents/MacOS"
	$(GO) build -o "$(APP)/Contents/MacOS/cmd-key-switcher" ./src
	cp src/Info.plist "$(APP)/Contents/Info.plist"
test:
	$(GO) test ./...
install: build
	bash scripts/install.sh
uninstall:
	bash scripts/uninstall.sh

format:
	$(GOFMT) -w $(GO_FILES)
	$(CLANG_FORMAT) -i $(NATIVE_FILES)

check:
	@unformatted="$$($(GOFMT) -l $(GO_FILES))" || exit $$?; if [ -n "$$unformatted" ]; then echo "Go files need formatting:"; echo "$$unformatted"; exit 1; fi
	$(CLANG_FORMAT) --dry-run --Werror $(NATIVE_FILES)
	$(GO) vet ./...
	bash -n scripts/setup.sh scripts/install.sh scripts/uninstall.sh
	plutil -lint src/Info.plist
	$(MAKE) analyze

analyze:
	mkdir -p build
	xcrun clang --analyze -x objective-c -fobjc-arc -Wall -Wextra -Werror -Wno-unused-parameter -Xanalyzer -analyzer-werror src/native_darwin.m -o build/native-analysis.plist
