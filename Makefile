APP := build/CmdKeySwitcher.app

.PHONY: build test preview install uninstall
build:
	mkdir -p "$(APP)/Contents/MacOS"
	CGO_ENABLED=1 go build -o "$(APP)/Contents/MacOS/cmd-key-switcher" .
	cp Info.plist "$(APP)/Contents/Info.plist"
test:
	go test ./...
preview: build
	"$(APP)/Contents/MacOS/cmd-key-switcher" --dry-run
install: build
	bash scripts/install.sh
uninstall:
	bash scripts/uninstall.sh
