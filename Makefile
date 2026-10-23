.PHONY: all windows linux macos darwin-image

CROSS = /home/joe/go/bin/fyne-cross

all: windows macos

windows:
	$(CROSS) windows -app-id com.github.joexue.yaterm -icon assert/Resource/icons/yaterm.png -output yaterm

# The macOS SDK must be 12 or later: Go 1.26 needs macOS 12, its crypto/x509
# links SecTrustCopyCertificateChain, which older SDKs do not have
MACOS_SDK = /home/joe/SDKs/MacOSX26.5.sdk

macos: darwin-image
	$(CROSS) darwin -image $(DARWIN_IMAGE) -arch=arm64 --macosx-sdk-path $(MACOS_SDK) -macosx-version-min 12.0 -app-id com.github.joexue.yaterm -icon assert/Resource/icons/yaterm.png

# The darwin image of fyne-cross with the SDK frameworks where zig looks for
# them, see build/darwin/Dockerfile
DARWIN_IMAGE = yaterm/fyne-cross-darwin
darwin-image:
	docker build -t $(DARWIN_IMAGE) build/darwin

