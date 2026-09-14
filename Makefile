.phony: all windows linux macos

CROSS = /home/joe/go/bin/fyne-cross

all: windows

windows:
	$(CROSS) windows -app-id github.com/joexue/YaTerm -icon assert/Resource/icons/yaterm.png -output yaterm
