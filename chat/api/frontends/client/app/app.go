// Package app provides client app support.
package app

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AppStorage interface {
	QueryContactByID(id common.Address) (User, error)
	MyAccount() User
	Contacts() []User
}

type App struct {
	app      *tview.Application
	flex     *tview.Flex
	list     *tview.List
	textView *tview.TextView
	textArea *tview.TextArea
	button   *tview.Button
	client   *Client
	db       AppStorage
}

func New(client *Client, db AppStorage) *App {
	app := tview.NewApplication()

	// -------------------------------------------------------------------------

	textView := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	textView.SetBorder(true)
	textView.SetTitle(fmt.Sprintf("*** %s ***", db.MyAccount().ID))

	// -------------------------------------------------------------------------

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle("Users")
	list.SetChangedFunc(func(idx int, name string, id string, shortcut rune) {
		textView.Clear()

		addrID := common.HexToAddress(id)

		user, err := db.QueryContactByID(addrID)
		if err != nil {
			textView.ScrollToEnd()
			fmt.Fprintln(textView, "-----")
			fmt.Fprintln(textView, err.Error())
			return
		}

		for i, msg := range user.Messages {
			fmt.Fprintln(textView, msg)
			if i < len(user.Messages)-1 {
				fmt.Fprintln(textView, "-----")
			}
		}

		list.SetItemText(idx, user.Name, user.ID.Hex())
	})

	users := db.Contacts()
	for i, user := range users {
		shortcut := rune(i + 49)
		list.AddItem(user.Name, user.ID.Hex(), shortcut, nil)
	}

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
		db:       db,
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

func (a *App) ButtonHandler() {
	_, to := a.list.GetItemText(a.list.GetCurrentItem())

	msg := a.textArea.GetText()
	if msg == "" {
		return
	}

	if err := a.client.Send(common.HexToAddress(to), msg); err != nil {
		a.WriteText("system", fmt.Sprintf("Error sending message: %s", err))
		return
	}

	a.textArea.SetText("", false)
}

func (a *App) WriteText(id string, msg string) {
	a.textView.ScrollToEnd()

	switch id {
	case "system":
		fmt.Fprintln(a.textView, "-----")
		fmt.Fprintln(a.textView, msg)

	default:
		idx := a.list.GetCurrentItem()

		_, currentID := a.list.GetItemText(idx)
		if currentID == "" {
			fmt.Fprintln(a.textView, "-----")
			fmt.Fprintln(a.textView, "id not found: "+id)
			return
		}

		if id == currentID {
			fmt.Fprintln(a.textView, "-----")
			fmt.Fprintln(a.textView, msg)
			return
		}

		for i := range a.list.GetItemCount() {
			name, idStr := a.list.GetItemText(i)
			if id == idStr {
				a.list.SetItemText(i, "* "+name, idStr)
				a.app.Draw()
				return
			}
		}
	}
}

func (a *App) UpdateContact(id string, name string) {
	shortcut := rune(a.list.GetItemCount() + 49)
	a.list.AddItem(name, id, shortcut, nil)
}
