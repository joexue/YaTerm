package assert

import (
	_ "embed"
    "fyne.io/fyne/v2"
)

//go:embed Resource/icons/yaterm.png
var yatermIcon []byte

var YatermIconRes = &fyne.StaticResource{
	StaticName:    "yaterm.png",
	StaticContent: yatermIcon,
}

//go:embed Resource/icons/hsplit.svg
var hsplitIcon []byte

var HSplitIconRes = &fyne.StaticResource{
	StaticName:    "hsplit.svg",
	StaticContent: hsplitIcon,
}

//go:embed Resource/icons/vsplit.svg
var vsplitIcon []byte

var VSplitIconRes = &fyne.StaticResource{
	StaticName:    "hsplit.svg",
	StaticContent: vsplitIcon,
}
