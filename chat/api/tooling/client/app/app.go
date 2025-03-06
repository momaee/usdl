// Package app provides client app support.
package app

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type App struct {
	app      *tview.Application
	flex     *tview.Flex
	list     *tview.List
	textView *tview.TextView
	textArea *tview.TextArea
	button   *tview.Button
	client   *Client
	cfg      *Config
}

func New(client *Client, cfg *Config) *App {
	app := tview.NewApplication()

	// -------------------------------------------------------------------------

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle("Users")

	users := cfg.Contacts()
	for i, user := range users {
		id := rune(i + 49)
		list.AddItem(user.Name, user.ID, id, nil)
	}

	// -------------------------------------------------------------------------

	textView := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	textView.SetBorder(true)
	textView.SetTitle(fmt.Sprintf("*** %s ***", cfg.User().ID))

	// -------------------------------------------------------------------------

	button := tview.NewButton("SUBMIT")
	button.SetStyle(tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen).Bold(true))
	button.SetActivatedStyle(tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen).Bold(true))
	button.SetBorder(true)
	button.SetBorderColor(tcell.ColorGreen)

	// -------------------------------------------------------------------------

	textArea := tview.NewTextArea()
	textArea.SetWrap(false)
	textArea.SetPlaceholder("Enter message here...")
	textArea.SetBorder(true)
	textArea.SetBorderPadding(0, 0, 1, 0)

	// -------------------------------------------------------------------------

	flex := tview.NewFlex().
		AddItem(list, 20, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(textView, 0, 5, false).
			AddItem(tview.NewFlex().
				SetDirection(tview.FlexColumn).
				AddItem(textArea, 0, 90, false).
				AddItem(button, 0, 10, false),
				0, 1, false),
			0, 1, false)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyCtrlQ:
			app.Stop()
			return nil
		}

		return event
	})

	a := App{
		app:      app,
		flex:     flex,
		list:     list,
		textView: textView,
		textArea: textArea,
		button:   button,
		client:   client,
		cfg:      cfg,
	}

	button.SetSelectedFunc(a.ButtonHandler)

	textArea.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			a.ButtonHandler()
			return nil
		}
		return event
	})

	return &a
}

func (a *App) Run() error {
	return a.app.SetRoot(a.flex, true).EnableMouse(true).Run()
}

func (a *App) FindName(id string) string {
	for i := range a.list.GetItemCount() {
		name, toIDStr := a.list.GetItemText(i)
		if id == toIDStr {
			return name
		}
	}

	return ""
}

func (a *App) ButtonHandler() {
	_, to := a.list.GetItemText(a.list.GetCurrentItem())

	msg := a.textArea.GetText()
	if msg == "" {
		return
	}

	if err := a.client.Send(to, msg); err != nil {
		a.WriteText("system", fmt.Sprintf("Error sending message: %s", err))
		return
	}

	a.textArea.SetText("", false)
	a.WriteText("You", msg)
}

func (a *App) WriteText(name string, msg string) {
	a.textView.ScrollToEnd()
	fmt.Fprintln(a.textView, "-----")
	fmt.Fprintln(a.textView, name+": "+msg)
}
