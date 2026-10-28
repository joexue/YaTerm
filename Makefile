.PHONY: all windows linux macos darwin-image

CROSS = /home/joe/go/bin/fyne-cross

# The version and revision the settings show, from git. They go into a Go
# file made for the build and removed after it, not into -ldflags -X:
# fyne-cross quotes an env value with "=", so fyne package does not find the
# -ldflags in GOFLAGS, and its own -ldflags replace them.
VERSION    = $(shell git describe --tags --abbrev=0 2>/dev/null)
REVISION   = $(shell git rev-parse --short HEAD 2>/dev/null)
STAMP_FILE = ui/zz_buildstamp.go
STAMP      = printf 'package ui\n\n// Made by make for a build, see Makefile\n\nfunc init() { buildStamp = "%s,%s" }\n' "$(VERSION)" "$(REVISION)" > $(STAMP_FILE)

# stamped runs a build command with the stamp file, removed whether the build
# works or not
stamped = $(STAMP) && { $(1); rc=$$?; rm -f $(STAMP_FILE); exit $$rc; }

all: windows macos linux

windows:
	$(call stamped,$(CROSS) windows -app-id com.github.joexue.yaterm -icon assert/Resource/icons/yaterm.png -output yaterm)

linux:
	$(call stamped,$(CROSS) linux -app-id com.github.joexue.yaterm -icon assert/Resource/icons/yaterm.png -output yaterm)
# The macOS SDK must be 12 or later: Go 1.26 needs macOS 12, its crypto/x509
# links SecTrustCopyCertificateChain, which older SDKs do not have
MACOS_SDK = /home/joe/SDKs/MacOSX26.5.sdk

macos: darwin-image
	$(call stamped,$(CROSS) darwin -image $(DARWIN_IMAGE) -arch=arm64 --macosx-sdk-path $(MACOS_SDK) -macosx-version-min 12.0 -app-id com.github.joexue.yaterm -icon assert/Resource/icons/yaterm.png)

# The darwin image of fyne-cross with the SDK frameworks where zig looks for
# them, see build/darwin/Dockerfile
DARWIN_IMAGE = yaterm/fyne-cross-darwin
darwin-image:
	docker build -t $(DARWIN_IMAGE) build/darwin

