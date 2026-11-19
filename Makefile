.PHONY: all windows linux macos macos-sign darwin-image

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
	$(call stamped,$(CROSS) windows -app-id com.github.joexue.yaterm -icon assert/icons/yaterm.png -output yaterm)

linux:
	$(call stamped,$(CROSS) linux -app-id com.github.joexue.yaterm -icon assert/icons/yaterm.png -output yaterm)
# The macOS SDK must be 12 or later: Go 1.26 needs macOS 12, its crypto/x509
# links SecTrustCopyCertificateChain, which older SDKs do not have
MACOS_SDK = /home/joe/SDKs/MacOSX26.5.sdk

MACOS_MIN = 12.0

macos: darwin-image
	$(call stamped,$(CROSS) darwin -image $(DARWIN_IMAGE) -arch=arm64 --macosx-sdk-path $(MACOS_SDK) -macosx-version-min $(MACOS_MIN) -app-id com.github.joexue.yaterm -icon assert/icons/yaterm.png)
	$(MAKE) macos-sign

# The app from fyne-cross only has the linker's ad-hoc signature on its
# executable, which does not seal the bundle (Info.plist, resources), so
# macOS calls a downloaded copy "damaged". rcodesign signs the whole bundle
# ad-hoc, with no Apple account: a downloaded copy then only asks once to be
# opened (right click, Open). Without warnings it would need a Developer ID
# and notarization. fyne-cross makes its files as root, so the bundle is
# copied first. The Info.plist gets the real minimum macOS, fyne-cross
# leaves 10.11 there, it must change before signing as the signature
# covers it.
RCODESIGN    = /home/joe/go/bin/rcodesign
MACOS_DIST   = fyne-cross/dist/darwin-arm64
MACOS_SIGNED = fyne-cross/dist/darwin-arm64-signed

macos-sign:
	rm -rf $(MACOS_SIGNED)
	mkdir -p $(MACOS_SIGNED)
	cp -R $(MACOS_DIST)/YaTerm.app $(MACOS_SIGNED)/
	perl -0pi -e 's#(<key>LSMinimumSystemVersion</key>\s*<string>)[^<]*#$${1}$(MACOS_MIN)#' $(MACOS_SIGNED)/YaTerm.app/Contents/Info.plist
	$(RCODESIGN) sign $(MACOS_SIGNED)/YaTerm.app
	tar -C $(MACOS_SIGNED) -czf $(MACOS_SIGNED)/YaTerm-macos-arm64.tar.gz YaTerm.app

# The darwin image of fyne-cross with the frameworks of the macOS SDK where
# its linker looks for them. fyne-cross mounts the SDK at /sdk and links with
# "zig cc --sysroot /sdk -F/System/Library/Frameworks", but zig does not apply
# the sysroot to -F, so it looks in /System/Library/Frameworks of the
# container and finds no framework ("unable to find framework 'Security'").
# The link points there. The image has no /System, the rm is in case a later
# one does. The Dockerfile comes on stdin, the build needs no context.
DARWIN_IMAGE = yaterm/fyne-cross-darwin

define DARWIN_DOCKERFILE
FROM fyneio/fyne-cross-images:darwin
RUN rm -rf /System/Library/Frameworks && mkdir -p /System/Library && ln -s /sdk/System/Library/Frameworks /System/Library/Frameworks
endef
export DARWIN_DOCKERFILE

darwin-image:
	printf '%s\n' "$$DARWIN_DOCKERFILE" | docker build -t $(DARWIN_IMAGE) -

