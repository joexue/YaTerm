.phony: all windows linux macos

CROSS = /home/joe/go/bin/fyne-cross

all: windows macos

windows:
	$(CROSS) windows -app-id github.com/joexue/YaTerm -icon assert/Resource/icons/yaterm.png -output yaterm

macos:
	$(CROSS) darwin -arch=arm64 --macosx-sdk-path=/home/joe/SDKs/MacOSX26.sdk -app-id github.com/joexue/YaTerm -icon assert/Resource/icons/yaterm.png -output yaterm
