# Makefile for YaTerm
#
# Copyright (c) 2026 Joe Xue (lgxue@hotmail.com)
#
# Builds the YaTerm packages with fyne-cross: make windows | linux | macos,
# or all of them.
#
# Needs:
#  - Docker, and the user in the docker group, so make runs without sudo:
#        sudo usermod -aG docker $USER
#  - fyne-cross (CROSS):  go install github.com/fyne-io/fyne-cross@latest
#  - git, for the version and revision: the version is the latest tag
#  - zip, for the Windows package
#  - msix only: makeappx (MAKEAPPX) and the identity of Partner Center, see
#    there
#  - macos only: the macOS SDK (MACOS_SDK, 12 or later, see there), perl and
#    rcodesign (RCODESIGN, of apple-codesign)
#
# The packages go to fyne-cross/dist as Yaterm-<os>-<arch>-<version>.

CROSS      = ~/go/bin/fyne-cross
APPID      = com.github.joexue.yaterm
ICON       = assert/icons/yaterm.png

VERSION    = $(shell git describe --tags --abbrev=0 2>/dev/null)
REVISION   = $(shell git rev-parse --short HEAD 2>/dev/null)

STAMP_FILE = ui/zz_buildstamp.go
STAMP      = printf 'package ui\n\n// Made by make for a build, see Makefile\n\nfunc init() { buildStamp = "%s,%s" }\n' "$(VERSION)" "$(REVISION)" > $(STAMP_FILE)

# stamped runs a build command with the stamp file, removed whether the build
# works or not
stamped = $(STAMP) && { $(1); rc=$$?; rm -f $(STAMP_FILE); exit $$rc; }

# The packages to release, named Yaterm-<os>-<arch>-<version>, next to the
# folders fyne-cross makes in its dist
DIST       = fyne-cross/dist
package    = $(DIST)/Yaterm-$(1)-$(2)-$(VERSION)

# The Windows package has the newer ConPTY next to yaterm.exe, for tmux -CC,
# see pty/pty_windows.go. They are added to the zip of fyne-cross.
WIN_ZIP    = $(call package,windows,x86_64).zip
WIN_EXTRA  = assert/windows/conpty.dll assert/windows/OpenConsole.exe

.PHONY: all windows linux macos macos-sign darwin-image
all: windows macos linux

windows:
	$(call stamped,$(CROSS) windows -app-id $(APPID) -icon $(ICON) -output yaterm)
	cp $(DIST)/windows-amd64/yaterm.zip $(WIN_ZIP)
	zip -q -j $(WIN_ZIP) $(WIN_EXTRA)

linux:
	$(call stamped,$(CROSS) linux -app-id $(APPID) -icon $(ICON) -output yaterm)
	cp $(DIST)/linux-amd64/yaterm.tar.xz $(call package,linux,x86_64).tar.xz

# The macOS SDK must be 12 or later: Go 1.26 needs macOS 12, its crypto/x509
# links SecTrustCopyCertificateChain, which older SDKs do not have
MACOS_SDK = ~/SDKs/MacOSX26.5.sdk
MACOS_MIN = 12.0

macos: darwin-image
	$(call stamped,$(CROSS) darwin -image $(DARWIN_IMAGE) -arch=arm64 --macosx-sdk-path $(MACOS_SDK) -macosx-version-min $(MACOS_MIN) -app-id $(APPID) -icon $(ICON))
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
RCODESIGN    = ~/go/bin/rcodesign
MACOS_DIST   = fyne-cross/dist/darwin-arm64
MACOS_SIGNED = fyne-cross/dist/darwin-arm64-signed

macos-sign:
	rm -rf $(MACOS_SIGNED)
	mkdir -p $(MACOS_SIGNED)
	cp -R $(MACOS_DIST)/YaTerm.app $(MACOS_SIGNED)/
	perl -0pi -e 's#(<key>LSMinimumSystemVersion</key>\s*<string>)[^<]*#$${1}$(MACOS_MIN)#' $(MACOS_SIGNED)/YaTerm.app/Contents/Info.plist
	$(RCODESIGN) sign $(MACOS_SIGNED)/YaTerm.app
	tar -C $(MACOS_SIGNED) -czf $(call package,macos,arm64).tar.gz YaTerm.app

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

# The MSIX package for the Microsoft Store: the Windows build with the newer
# ConPTY, the logos and assert/msix/AppxManifest.xml. It is not signed, the
# Store signs it. The identity is the one Partner Center shows under Product
# identity, the version the latest tag with a 0 revision (the Store needs it).
# makeappx is of the NuGet package Microsoft.Windows.SDK.BuildTools, it runs
# from the Windows drive, so it gets Windows paths (wslpath -w).
MAKEAPPX                    = /mnt/c/Users/lgxue/tools/winsdk-buildtools/makeappx.exe
MSIX_IDENTITY_NAME          = yaterm
MSIX_PUBLISHER              = CN=Joe Xue, O=Joe Xue, C=CA
MSIX_PUBLISHER_DISPLAY_NAME = Joe Xue
MSIX_DIR                    = fyne-cross/msix
MSIX                        = $(call package,windows,x86_64).msix

msix: windows
	@test -n "$(MSIX_IDENTITY_NAME)" -a -n "$(MSIX_PUBLISHER)" -a -n "$(MSIX_PUBLISHER_DISPLAY_NAME)" || \
		{ echo "Set MSIX_IDENTITY_NAME, MSIX_PUBLISHER and MSIX_PUBLISHER_DISPLAY_NAME from Partner Center"; exit 1; }
	rm -rf $(MSIX_DIR) $(MSIX)
	mkdir -p $(MSIX_DIR)/Assets
	unzip -q $(DIST)/windows-amd64/yaterm.zip -d $(MSIX_DIR)
	cp $(WIN_EXTRA) $(MSIX_DIR)/
	cp assert/msix/*.png $(MSIX_DIR)/Assets/
	sed -e 's|@IDENTITY_NAME@|$(MSIX_IDENTITY_NAME)|' \
	    -e 's|@PUBLISHER@|$(MSIX_PUBLISHER)|' \
	    -e 's|@PUBLISHER_DISPLAY_NAME@|$(MSIX_PUBLISHER_DISPLAY_NAME)|' \
	    -e 's|@VERSION@|$(VERSION).0|' \
	    assert/msix/AppxManifest.xml > $(MSIX_DIR)/AppxManifest.xml
	$(MAKEAPPX) pack /o /d "$$(wslpath -w $(MSIX_DIR))" /p "$$(wslpath -w $(DIST))\$(notdir $(MSIX))"
