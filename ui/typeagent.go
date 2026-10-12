package ui

import (
	"encoding/json"
	"errors"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

/*
 * ================================
 *  Type Agent: saved commands typed into the terminal from the menu, for the
 *  long commands people use often and would copy and paste otherwise, but
 *  the clipboard changes in between
 * ================================
 */

// agentNameWidth is the width of the name column in the command editor
const agentNameWidth = 160

// prefTypeAgent is the preference key of the saved commands, kept as JSON
const prefTypeAgent = "typeAgent"

// agentCommand is a saved command, shown in the menu by its name
type agentCommand struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// loadAgentCommands returns the saved commands, none if there are none or
// they can not be read
func loadAgentCommands() []agentCommand {
	var cmds []agentCommand
	data := fyne.CurrentApp().Preferences().String(prefTypeAgent)
	if data == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(data), &cmds); err != nil {
		fyne.LogError("Type Agent: can not read the saved commands", err)
		return nil
	}
	return cmds
}

// saveAgentCommands keeps the commands
func saveAgentCommands(cmds []agentCommand) {
	data, err := json.Marshal(cmds)
	if err != nil {
		fyne.LogError("Type Agent: can not save the commands", err)
		return
	}
	fyne.CurrentApp().Preferences().SetString(prefTypeAgent, string(data))
}

// typeAgentMenu is the Type Agent submenu: the editor of the commands, then
// one item for each saved command, which types it into this pane's terminal
func (p *Pane) typeAgentMenu() *fyne.Menu {
	items := []*fyne.MenuItem{
		fyne.NewMenuItemWithIcon("Manage commands...", theme.Icon(theme.IconNameSettings), func() {
			showAgentCommands(win)
		}),
	}

	cmds := loadAgentCommands()
	if len(cmds) > 0 {
		items = append(items, fyne.NewMenuItemSeparator())
	}
	for _, c := range cmds {
		items = append(items, fyne.NewMenuItem(c.Name, func() {
			p.typeText(c.Command)
		}))
	}

	return fyne.NewMenu("", items...)
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

// showAgentCommands opens the editor of the saved commands: a list of them
// with Add, Edit and Delete. Every change is saved right away.
func showAgentCommands(w fyne.Window) {
	cmds := loadAgentCommands()
	selected := -1

	list := widget.NewList(
		func() int { return len(cmds) },
		func() fyne.CanvasObject {
			name := widget.NewLabel("")
			name.TextStyle = fyne.TextStyle{Bold: true}
			name.Truncation = fyne.TextTruncateEllipsis
			command := widget.NewLabel("")
			command.TextStyle = fyne.TextStyle{Monospace: true}
			command.Truncation = fyne.TextTruncateEllipsis

			// The row is made empty, so the name gets a fixed width column,
			// sized to its empty text it would be drawn over the command
			nameCol := container.NewGridWrap(fyne.NewSize(agentNameWidth, name.MinSize().Height), name)
			return container.NewBorder(nil, nil, nameCol, nil, command)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			// NewBorder keeps the center object first, then the left one
			row := o.(*fyne.Container)
			command := row.Objects[0].(*widget.Label)
			name := row.Objects[1].(*fyne.Container).Objects[0].(*widget.Label)
			name.SetText(cmds[i].Name)
			command.SetText(firstLine(cmds[i].Command))
		},
	)

	empty := widget.NewLabel("No commands yet, add one to type it from the Type Agent menu.")
	empty.Importance = widget.LowImportance

	var edit, del *widget.Button
	update := func() {
		if selected >= 0 && selected < len(cmds) {
			edit.Enable()
			del.Enable()
		} else {
			edit.Disable()
			del.Disable()
		}
		if len(cmds) == 0 {
			empty.Show()
		} else {
			empty.Hide()
		}
	}
	list.OnSelected = func(id widget.ListItemID) {
		selected = id
		update()
	}
	list.OnUnselected = func(widget.ListItemID) {
		selected = -1
		update()
	}

	changed := func() {
		saveAgentCommands(cmds)
		list.UnselectAll()
		selected = -1
		list.Refresh()
		update()
	}

	add := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), func() {
		editAgentCommand(w, agentCommand{}, func(c agentCommand) {
			cmds = append(cmds, c)
			changed()
		})
	})
	edit = widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {
		i := selected
		editAgentCommand(w, cmds[i], func(c agentCommand) {
			cmds[i] = c
			changed()
		})
	})
	del = widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		i := selected
		dialog.ShowConfirm("Delete command", "Delete \""+cmds[i].Name+"\"?", func(ok bool) {
			if ok {
				cmds = append(cmds[:i], cmds[i+1:]...)
				changed()
			}
		}, w)
	})
	update()

	body := container.NewStack(list, container.NewCenter(empty))
	content := container.NewBorder(nil, container.NewHBox(add, edit, del), nil, nil, body)

	d := dialog.NewCustom("Type Agent", "Close", content, w)
	d.Resize(fyne.NewSize(600, 400))
	d.Show()
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

// editAgentCommand opens a form for a command, done gets it when saved
func editAgentCommand(w fyne.Window, c agentCommand, done func(agentCommand)) {
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

	// Own buttons, not a form dialog's, so they can be on the right
	var d dialog.Dialog
	buttons, save := saveCancelButtons(func() {
		if form.Validate() != nil {
			return
		}
		d.Hide()
		done(agentCommand{Name: strings.TrimSpace(name.Text), Command: command.Text})
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
