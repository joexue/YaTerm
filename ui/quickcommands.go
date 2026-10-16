package ui

import (
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"yaterm/assert"
)

/*
 * ================================
 *  Quick Commands: saved commands typed into the terminal from the menu, for the
 *  long commands people use often and would copy and paste otherwise, but
 *  the clipboard changes in between
 * ================================
 */

// quickNameWidth is the width of the name column in the command editor
const quickNameWidth = 160

// prefQuickCommands is the preference key of the saved commands, kept as
// JSON. Its name is from when the feature was called Type Agent.
const prefQuickCommands = "typeAgent"

// quickCommand is a saved command, shown in the menu by its name, or a
// folder of commands, shown as a submenu. Folders are only at the top level.
type quickCommand struct {
	Name     string         `json:"name"`
	Command  string         `json:"command,omitempty"`
	Folder   bool           `json:"folder,omitempty"`
	Commands []quickCommand `json:"commands,omitempty"`
}

// loadQuickCommands returns the saved commands, none if there are none or
// they can not be read
func loadQuickCommands() []quickCommand {
	var cmds []quickCommand
	data := fyne.CurrentApp().Preferences().String(prefQuickCommands)
	if data == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(data), &cmds); err != nil {
		fyne.LogError("Quick Commands: can not read the saved commands", err)
		return nil
	}
	return cmds
}

// saveQuickCommands keeps the commands
func saveQuickCommands(cmds []quickCommand) {
	data, err := json.Marshal(cmds)
	if err != nil {
		fyne.LogError("Quick Commands: can not save the commands", err)
		return
	}
	fyne.CurrentApp().Preferences().SetString(prefQuickCommands, string(data))
}

// quickCommandsMenu is the Quick Commands submenu: the editor of the commands, then
// one item for each saved command, which types it into this pane's terminal,
// and a submenu for each folder
func (p *Pane) quickCommandsMenu() *fyne.Menu {
	items := []*fyne.MenuItem{
		fyne.NewMenuItemWithIcon("Manage commands...", theme.Icon(theme.IconNameSettings), func() {
			showQuickCommands(win)
		}),
	}

	cmds := loadQuickCommands()
	if len(cmds) > 0 {
		items = append(items, fyne.NewMenuItemSeparator())
	}
	return fyne.NewMenu("", append(items, p.quickItems(cmds)...)...)
}

// quickItems are the menu items of commands and folders
func (p *Pane) quickItems(cmds []quickCommand) []*fyne.MenuItem {
	var items []*fyne.MenuItem
	for _, c := range cmds {
		if !c.Folder {
			items = append(items, fyne.NewMenuItemWithIcon(c.Name, quickCommandIcon(), func() {
				p.typeText(c.Command)
			}))
			continue
		}

		sub := p.quickItems(c.Commands)
		if len(sub) == 0 {
			empty := fyne.NewMenuItem("No commands", nil)
			empty.Disabled = true
			sub = []*fyne.MenuItem{empty}
		}
		folder := fyne.NewMenuItemWithIcon(c.Name, theme.FolderIcon(), nil)
		folder.ChildMenu = fyne.NewMenu("", sub...)
		items = append(items, folder)
	}
	return items
}

// quickCommandIcon is the icon of a saved command, a shell prompt
func quickCommandIcon() fyne.Resource {
	return theme.NewThemedResource(assert.CommandIconRes)
}

// typeText types text into the terminal like a paste: new lines are sent as
// Enter, so a command saved with a new line at its end runs right away
func (p *Pane) typeText(text string) {
	if p.term == nil || text == "" || p.tmuxControl != nil {
		return
	}

	text = strings.ReplaceAll(text, "\r\n", "\r")
	text = strings.ReplaceAll(text, "\n", "\r")

	p.screen.ScrollToBottom()
	_, _ = p.term.Write([]byte(text))

	// The menu took the focus, give it back to type on
	if c := fyne.CurrentApp().Driver().CanvasForObject(p); c != nil {
		c.Focus(p)
	}
}

// quickPath is where a command or folder is: its index at the top level,
// and its index in that folder for a command in one (-1 otherwise)
type quickPath struct {
	top, sub int
}

// noQuickPath is a command not saved yet
var noQuickPath = quickPath{-1, -1}

// The tree of the editor has a node ID for each path: "3" at the top level,
// "3/1" in a folder, the root is ""
func (a quickPath) id() widget.TreeNodeID {
	if a.sub < 0 {
		return strconv.Itoa(a.top)
	}
	return strconv.Itoa(a.top) + "/" + strconv.Itoa(a.sub)
}

func parseQuickPath(id widget.TreeNodeID) quickPath {
	top, sub, inFolder := strings.Cut(id, "/")
	a := noQuickPath
	a.top, _ = strconv.Atoi(top)
	if inFolder {
		a.sub, _ = strconv.Atoi(sub)
	}
	return a
}

// at is the command or folder at a path
func quickAt(cmds []quickCommand, a quickPath) *quickCommand {
	if a.sub < 0 {
		return &cmds[a.top]
	}
	return &cmds[a.top].Commands[a.sub]
}

// setQuickCommand puts c at its path, in folder (a top level index, -1 for
// none). One that changes folder, or is new, goes to the end of the folder.
// Returns the commands and where the folder is now, it moves up when the
// command leaves the top level before it.
func setQuickCommand(cmds []quickCommand, at quickPath, c quickCommand, folder int) ([]quickCommand, int) {
	if at != noQuickPath {
		cur := -1
		if at.sub >= 0 {
			cur = at.top
		}
		if cur == folder {
			*quickAt(cmds, at) = c
			return cmds, folder
		}

		// Out of its folder or the top level, the folders after it move up
		if at.sub >= 0 {
			f := &cmds[at.top]
			f.Commands = slices.Delete(f.Commands, at.sub, at.sub+1)
		} else {
			cmds = slices.Delete(cmds, at.top, at.top+1)
			if folder > at.top {
				folder--
			}
		}
	}

	if folder >= 0 {
		cmds[folder].Commands = append(cmds[folder].Commands, c)
	} else {
		cmds = append(cmds, c)
	}
	return cmds, folder
}

// showQuickCommands opens the editor of the saved commands: a tree of the
// folders and commands with Add, Edit and Delete. Every change is saved right
// away.
func showQuickCommands(w fyne.Window) {
	cmds := loadQuickCommands()
	selected := noQuickPath

	tree := widget.NewTree(
		func(id widget.TreeNodeID) []widget.TreeNodeID {
			var ids []widget.TreeNodeID
			if id == "" {
				for i := range cmds {
					ids = append(ids, quickPath{i, -1}.id())
				}
				return ids
			}
			a := parseQuickPath(id)
			for j := range cmds[a.top].Commands {
				ids = append(ids, quickPath{a.top, j}.id())
			}
			return ids
		},
		func(id widget.TreeNodeID) bool {
			if id == "" {
				return true
			}
			a := parseQuickPath(id)
			return a.sub < 0 && cmds[a.top].Folder
		},
		func(bool) fyne.CanvasObject {
			icon := widget.NewIcon(nil)
			name := widget.NewLabel("")
			name.TextStyle = fyne.TextStyle{Bold: true}
			name.Truncation = fyne.TextTruncateEllipsis
			command := widget.NewLabel("")
			command.TextStyle = fyne.TextStyle{Monospace: true}
			command.Truncation = fyne.TextTruncateEllipsis

			// The row is made empty, so the name gets a fixed width column,
			// sized to its empty text it would be drawn over the command
			nameCol := container.NewGridWrap(fyne.NewSize(quickNameWidth, name.MinSize().Height), name)
			return container.NewBorder(nil, nil, container.NewHBox(icon, nameCol), nil, command)
		},
		func(id widget.TreeNodeID, branch bool, o fyne.CanvasObject) {
			// NewBorder keeps the center object first, then the left one
			row := o.(*fyne.Container)
			command := row.Objects[0].(*widget.Label)
			left := row.Objects[1].(*fyne.Container)
			icon := left.Objects[0].(*widget.Icon)
			name := left.Objects[1].(*fyne.Container).Objects[0].(*widget.Label)

			c := quickAt(cmds, parseQuickPath(id))
			name.SetText(c.Name)
			if c.Folder {
				icon.SetResource(theme.FolderIcon())
				command.Importance = widget.LowImportance
				command.TextStyle = fyne.TextStyle{}
				command.SetText(countCommands(len(c.Commands)))
			} else {
				icon.SetResource(quickCommandIcon())
				command.Importance = widget.MediumImportance
				command.TextStyle = fyne.TextStyle{Monospace: true}
				command.SetText(firstLine(c.Command))
			}
		},
	)

	empty := widget.NewLabel("No commands yet, add one to type it from the Quick Commands menu.")
	empty.Importance = widget.LowImportance

	var edit, del, up, down *widget.Button
	update := func() {
		if selected != noQuickPath {
			edit.Enable()
			del.Enable()
		} else {
			edit.Disable()
			del.Disable()
		}
		// Up and down within the list the selection is in
		up.Disable()
		down.Disable()
		if selected != noQuickPath {
			i, n := quickSiblings(cmds, selected)
			if i > 0 {
				up.Enable()
			}
			if i < n-1 {
				down.Enable()
			}
		}
		if len(cmds) == 0 {
			empty.Show()
		} else {
			empty.Hide()
		}
	}
	tree.OnSelected = func(id widget.TreeNodeID) {
		selected = parseQuickPath(id)
		update()
	}
	tree.OnUnselected = func(widget.TreeNodeID) {
		selected = noQuickPath
		update()
	}

	// changed saves the commands and shows them. removed is the top level
	// index of what left the top level, -1 for none: the tree knows the open
	// folders by ID, which is the index, so the ones after it move up.
	changed := func(removed int) {
		saveQuickCommands(cmds)

		var open []int
		for i := 0; i <= len(cmds); i++ {
			if !tree.IsBranchOpen(quickPath{i, -1}.id()) || i == removed {
				continue
			}
			if removed >= 0 && i > removed {
				open = append(open, i-1)
			} else {
				open = append(open, i)
			}
		}
		tree.CloseAllBranches()
		for _, i := range open {
			if i < len(cmds) && cmds[i].Folder {
				tree.OpenBranch(quickPath{i, -1}.id())
			}
		}

		tree.UnselectAll()
		selected = noQuickPath
		tree.Refresh()
		update()
	}

	// folders are the names of the folders for the command form, with their
	// top level indexes
	folders := func() ([]string, []int) {
		var names []string
		var idx []int
		for i, c := range cmds {
			if c.Folder {
				names = append(names, c.Name)
				idx = append(idx, i)
			}
		}
		return names, idx
	}

	// editCommand opens the form of the command at a path, in folder
	editCommand := func(at quickPath, c quickCommand, folder int) {
		names, idx := folders()
		choice := slices.Index(idx, folder)
		editQuickCommand(w, c, names, choice, func(c quickCommand, choice int) {
			folder := -1
			if choice >= 0 {
				folder = idx[choice]
			}
			// A command at the top level that goes into a folder leaves it
			removed := -1
			if at != noQuickPath && at.sub < 0 && folder >= 0 {
				removed = at.top
			}
			cmds, folder = setQuickCommand(cmds, at, c, folder)
			changed(removed)
			// Show the command in its folder
			if folder >= 0 {
				tree.OpenBranch(quickPath{folder, -1}.id())
			}
		})
	}

	// A new command goes into the folder of the selection, if any
	addCmd := widget.NewButtonWithIcon("Add command", theme.ContentAddIcon(), func() {
		folder := -1
		if selected != noQuickPath && (selected.sub >= 0 || cmds[selected.top].Folder) {
			folder = selected.top
		}
		editCommand(noQuickPath, quickCommand{}, folder)
	})
	addFolder := widget.NewButtonWithIcon("Add folder", theme.FolderNewIcon(), func() {
		editQuickFolder(w, "", func(name string) {
			cmds = append(cmds, quickCommand{Name: name, Folder: true})
			changed(-1)
		})
	})
	edit = widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {
		at := selected
		c := quickAt(cmds, at)
		if c.Folder {
			editQuickFolder(w, c.Name, func(name string) {
				quickAt(cmds, at).Name = name
				changed(-1)
			})
			return
		}
		folder := -1
		if at.sub >= 0 {
			folder = at.top
		}
		editCommand(at, *c, folder)
	})
	del = widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		at := selected
		c := quickAt(cmds, at)
		title, msg := "Delete command", "Delete \""+c.Name+"\"?"
		if c.Folder {
			title = "Delete folder"
			if n := len(c.Commands); n > 0 {
				msg = "Delete \"" + c.Name + "\" and its " + countCommands(n) + "?"
			}
		}
		dialog.ShowConfirm(title, msg, func(ok bool) {
			if !ok {
				return
			}
			removed := -1
			if at.sub >= 0 {
				f := &cmds[at.top]
				f.Commands = slices.Delete(f.Commands, at.sub, at.sub+1)
			} else {
				cmds = slices.Delete(cmds, at.top, at.top+1)
				removed = at.top
			}
			changed(removed)
		}, w)
	})
	// move swaps the selection with the one before (-1) or after (1) it in
	// its list, it stays selected to move again
	move := func(d int) {
		at := selected
		i, n := quickSiblings(cmds, at)
		j := i + d
		if at == noQuickPath || j < 0 || j >= n {
			return
		}

		to := at
		if at.sub >= 0 {
			f := cmds[at.top].Commands
			f[i], f[j] = f[j], f[i]
			to.sub = j
		} else {
			cmds[i], cmds[j] = cmds[j], cmds[i]
			to.top = j
			// The tree knows the open folders by index, they swap too
			a, b := quickPath{i, -1}.id(), quickPath{j, -1}.id()
			aOpen, bOpen := tree.IsBranchOpen(a), tree.IsBranchOpen(b)
			setBranchOpen(tree, a, bOpen)
			setBranchOpen(tree, b, aOpen)
		}
		saveQuickCommands(cmds)
		tree.Refresh()
		tree.Select(to.id())
		tree.ScrollTo(to.id())
	}
	up = widget.NewButtonWithIcon("", theme.MoveUpIcon(), func() { move(-1) })
	down = widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() { move(1) })
	update()

	body := container.NewStack(tree, container.NewCenter(empty))
	buttons := container.NewBorder(nil, nil, container.NewHBox(addCmd, addFolder, edit, del), container.NewHBox(up, down))
	content := container.NewBorder(nil, buttons, nil, nil, body)

	d := dialog.NewCustom("Quick Commands", "Close", content, w)
	d.Resize(fyne.NewSize(600, 400))
	d.Show()
}

// quickSiblings is the index of a path in its list, the top level or its
// folder, and the length of that list
func quickSiblings(cmds []quickCommand, a quickPath) (int, int) {
	if a.sub >= 0 {
		return a.sub, len(cmds[a.top].Commands)
	}
	return a.top, len(cmds)
}

func setBranchOpen(tree *widget.Tree, id widget.TreeNodeID, open bool) {
	if open {
		tree.OpenBranch(id)
	} else {
		tree.CloseBranch(id)
	}
}

// countCommands is "1 command" or "n commands"
func countCommands(n int) string {
	if n == 1 {
		return "1 command"
	}
	return strconv.Itoa(n) + " commands"
}

// firstLine is the first line of a command for the list, with "..." when
// more lines follow
func firstLine(s string) string {
	line, rest, more := strings.Cut(strings.TrimRight(s, "\n"), "\n")
	if more && rest != "" {
		return line + " ..."
	}
	return line
}

// editQuickCommand opens a form for a command, done gets it when saved
// with the folder it is in, an index of folders or -1 for none
func editQuickCommand(w fyne.Window, c quickCommand, folders []string, folder int, done func(quickCommand, int)) {
	name := widget.NewEntry()
	name.SetText(c.Name)
	name.SetPlaceHolder("e.g. deploy")
	name.Validator = func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("a name is needed")
		}
		return nil
	}

	command := widget.NewMultiLineEntry()
	command.SetText(c.Command)
	command.SetPlaceHolder("The command, end it with a new line to run it right away")
	command.TextStyle = fyne.TextStyle{Monospace: true}
	command.SetMinRowsVisible(5)
	command.Validator = func(s string) error {
		if s == "" {
			return errors.New("a command is needed")
		}
		return nil
	}

	title := "Add command"
	if c.Name != "" {
		title = "Edit command"
	}

	form := widget.NewForm(
		widget.NewFormItem("Name", name),
		widget.NewFormItem("Command", command),
	)

	// The folder, also to move the command to another one. Picked by index,
	// two folders may have the same name.
	var folderSel *widget.Select
	if len(folders) > 0 {
		folderSel = widget.NewSelect(append([]string{"None"}, folders...), nil)
		folderSel.SetSelectedIndex(folder + 1)
		form.Append("Folder", folderSel)
	}

	// Own buttons, not a form dialog's, so they can be on the right
	var d dialog.Dialog
	buttons, save := saveCancelButtons(func() {
		if form.Validate() != nil {
			return
		}
		d.Hide()
		folder := -1
		if folderSel != nil {
			folder = folderSel.SelectedIndex() - 1
		}
		done(quickCommand{Name: strings.TrimSpace(name.Text), Command: command.Text}, folder)
	}, func() {
		d.Hide()
	})

	// Save is only possible with a name and a command
	form.SetOnValidationChanged(func(err error) {
		if err != nil {
			save.Disable()
		} else {
			save.Enable()
		}
	})
	if form.Validate() != nil {
		save.Disable()
	}

	content := container.NewBorder(nil, container.NewBorder(nil, nil, nil, buttons), nil, nil, form)
	d = dialog.NewCustomWithoutButtons(title, content, w)
	d.Resize(fyne.NewSize(520, 300))
	d.Show()
}

// editQuickFolder opens a form for the name of a folder, "" for a new one,
// done gets it when saved
func editQuickFolder(w fyne.Window, current string, done func(string)) {
	name := widget.NewEntry()
	name.SetText(current)
	name.SetPlaceHolder("e.g. servers")
	name.Validator = func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("a name is needed")
		}
		return nil
	}

	title := "Add folder"
	if current != "" {
		title = "Rename folder"
	}

	var d dialog.Dialog
	submit := func() {
		if name.Validate() != nil {
			return
		}
		d.Hide()
		done(strings.TrimSpace(name.Text))
	}
	buttons, save := saveCancelButtons(submit, func() {
		d.Hide()
	})
	name.OnSubmitted = func(string) { submit() }
	// Save only with a name. From the text, not the validation callback:
	// the form below takes that one over.
	enable := func(text string) {
		if strings.TrimSpace(text) == "" {
			save.Disable()
		} else {
			save.Enable()
		}
	}
	name.OnChanged = enable
	enable(name.Text)

	form := widget.NewForm(widget.NewFormItem("Name", name))
	content := container.NewBorder(nil, container.NewBorder(nil, nil, nil, buttons), nil, nil, form)
	d = dialog.NewCustomWithoutButtons(title, content, w)
	d.Resize(fyne.NewSize(400, 0))
	d.Show()
	w.Canvas().Focus(name)
}
