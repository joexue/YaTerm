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
	StaticName:    "vsplit.svg",
	StaticContent: vsplitIcon,
}

//go:embed Resource/icons/markdown.svg
var markdownIcon []byte

var MarkdownIconRes = &fyne.StaticResource{
	StaticName:    "markdown.svg",
	StaticContent: markdownIcon,
}

//go:embed fonts/JetBrainsMono-Regular.ttf
var resourceJetBrainsMonoRegular []byte
var ResourceJetBrainsMonoRegularTtf = &fyne.StaticResource{
	StaticName:    "fonts/JetBrainsMono-Regular.ttf",
	StaticContent: resourceJetBrainsMonoRegular,
}

//go:embed fonts/JetBrainsMono-Bold.ttf
var resourceJetBrainsMonoBold []byte
var ResourceJetBrainsMonoBoldTtf = &fyne.StaticResource{
	StaticName:    "fonts/JetBrainsMono-Bold.ttf",
	StaticContent: resourceJetBrainsMonoBold,
}

//go:embed fonts/JetBrainsMono-Italic.ttf
var resourceJetBrainsMonoItalic []byte
var ResourceJetBrainsMonoItalicTtf = &fyne.StaticResource{
	StaticName:    "fonts/JetBrainsMono-Italic.ttf",
	StaticContent: resourceJetBrainsMonoItalic,
}
