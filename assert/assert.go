// Package assert embeds the files YaTerm ships inside its binary: the
// JetBrains Mono fonts and the icons.
package assert

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed fonts/JetBrainsMono-Regular.ttf
var resourceJetBrainsMonoRegular []byte

// ResourceJetBrainsMonoRegularTtf is the regular JetBrains Mono font, the
// text of the terminals
var ResourceJetBrainsMonoRegularTtf = &fyne.StaticResource{
	StaticName:    "fonts/JetBrainsMono-Regular.ttf",
	StaticContent: resourceJetBrainsMonoRegular,
}

//go:embed fonts/JetBrainsMono-Bold.ttf
var resourceJetBrainsMonoBold []byte

// ResourceJetBrainsMonoBoldTtf is the bold JetBrains Mono font
var ResourceJetBrainsMonoBoldTtf = &fyne.StaticResource{
	StaticName:    "fonts/JetBrainsMono-Bold.ttf",
	StaticContent: resourceJetBrainsMonoBold,
}

//go:embed fonts/JetBrainsMono-Italic.ttf
var resourceJetBrainsMonoItalic []byte

// ResourceJetBrainsMonoItalicTtf is the italic JetBrains Mono font
var ResourceJetBrainsMonoItalicTtf = &fyne.StaticResource{
	StaticName:    "fonts/JetBrainsMono-Italic.ttf",
	StaticContent: resourceJetBrainsMonoItalic,
}

//go:embed icons/yaterm.png
var yatermIcon []byte

// YatermIconRes is the icon of the app and its window
var YatermIconRes = &fyne.StaticResource{
	StaticName:    "yaterm.png",
	StaticContent: yatermIcon,
}

//go:embed icons/hsplit.svg
var hsplitIcon []byte

// HSplitIconRes is the icon of "Split Right" in the pane menu
var HSplitIconRes = &fyne.StaticResource{
	StaticName:    "hsplit.svg",
	StaticContent: hsplitIcon,
}

//go:embed icons/vsplit.svg
var vsplitIcon []byte

// VSplitIconRes is the icon of "Split Down" in the pane menu
var VSplitIconRes = &fyne.StaticResource{
	StaticName:    "vsplit.svg",
	StaticContent: vsplitIcon,
}

//go:embed icons/markdown_color.svg
var markdownIconColor []byte

// MarkdownIconColorRes is the colored Markdown icon, on the Markdown tabs
var MarkdownIconColorRes = &fyne.StaticResource{
	StaticName:    "markdown_color.svg",
	StaticContent: markdownIconColor,
}

//go:embed icons/markdown.svg
var markdownIcon []byte

// MarkdownIconRes is the Markdown icon of "Show as Markdown" in the pane menu
var MarkdownIconRes = &fyne.StaticResource{
	StaticName:    "markdown.svg",
	StaticContent: markdownIcon,
}

//go:embed icons/tmux.svg
var tmuxIcon []byte

// TmuxIconRes is the icon of the tabs of tmux windows
var TmuxIconRes = &fyne.StaticResource{
	StaticName:    "tmux.svg",
	StaticContent: tmuxIcon,
}

//go:embed icons/local.svg
var localIcon []byte

// LocalIconRes is the icon of the local tabs, shown while there are tmux tabs
var LocalIconRes = &fyne.StaticResource{
	StaticName:    "local.svg",
	StaticContent: localIcon,
}

//go:embed icons/command.svg
var commandIcon []byte

// CommandIconRes is the icon of the quick commands
var CommandIconRes = &fyne.StaticResource{
	StaticName:    "command.svg",
	StaticContent: commandIcon,
}
