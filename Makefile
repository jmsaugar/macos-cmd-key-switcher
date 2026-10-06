APP := build/CmdKeySwitcher.app
SIGN_IDENTITY ?= -
GO := mise exec -- go
GOFMT := mise exec -- gofmt
CLANG_FORMAT := mise exec -- clang-format
GO_FILES := $(wildcard src/*.go)
NATIVE_SOURCES := $(wildcard src/*_darwin.m)
NATIVE_FILES := $(NATIVE_SOURCES) src/native.h

.PHONY: setup build test format check analyze
setup:
	bash scripts/setup.sh

build:
	mkdir -p "$(APP)/Contents/MacOS"
	$(GO) build -o "$(APP)/Contents/MacOS/cmd-key-switcher" ./src
	cp src/Info.plist "$(APP)/Contents/Info.plist"
	bash scripts/build-icon.sh src/assets/AppIcon.png "$(APP)/Contents/Resources/AppIcon.icns"
	codesign --force --sign "$(SIGN_IDENTITY)" "$(APP)"
test:
	$(GO) test ./...

format:
	$(GOFMT) -w $(GO_FILES)
	$(CLANG_FORMAT) -i $(NATIVE_FILES)

check:
	@unformatted="$$($(GOFMT) -l $(GO_FILES))" || exit $$?; if [ -n "$$unformatted" ]; then echo "Go files need formatting:"; echo "$$unformatted"; exit 1; fi
	$(CLANG_FORMAT) --dry-run --Werror $(NATIVE_FILES)
	$(GO) vet ./...
	bash -n scripts/setup.sh scripts/build-icon.sh
	plutil -lint src/Info.plist
	$(MAKE) analyze

analyze:
	mkdir -p build
	@set -e; for source in $(NATIVE_SOURCES); do \
		xcrun clang --analyze -x objective-c -fobjc-arc -mmacosx-version-min=13.0 -Wall -Wextra -Werror -Wno-unused-parameter -Xanalyzer -analyzer-werror "$$source" -o "build/$$(basename "$$source" .m)-analysis.plist"; \
	done
