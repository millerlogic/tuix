// Copyright (C) 2019 Christopher E. Miller
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package tuix

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Window is a window.
type Window struct {
	*tview.Box
	desktop        *Desktop
	client         tview.Primitive // can be nil
	title          string
	moveX, moveY   int
	rx, ry, rw, rh int // Restored rect.
	state          WindowState
	clientFullSize bool
	border         bool
	noCaption      bool
	autoActivate   bool
	moving         bool
	autoPosition   bool
	resizable      bool
	resizing       byte // 1=horiz, 2=vert, 3=both
}

func NewWindow() *Window {
	win := &Window{
		Box:          tview.NewBox(),
		autoActivate: true,
	}
	return win
}

// GetClient gets the client primitive previously set by SetClient, or nil.
func (win *Window) GetClient() tview.Primitive {
	return win.client
}

// SetClient sets the window's client primitive.
func (win *Window) SetClient(client tview.Primitive, fullSize bool) {
	win.client = client
	win.clientFullSize = fullSize
	if client != nil && fullSize {
		client.SetRect(win.GetInnerRect())
	}
}

// Desktop is called by the Desktop, do not call it directly.
func (win *Window) Desktop(d *Desktop) {
	win.desktop = d
}

// GetDesktop gets the desktop, or nil.
func (win *Window) GetDesktop() *Desktop {
	return win.desktop
}

// SetAutoActivate determines if the window will automatically activate.
func (win *Window) SetAutoActivate(on bool) *Window {
	win.autoActivate = on
	return win
}

// SetAutoPosition sets whether or not the desktop decides where to put the window.
func (win *Window) SetAutoPosition(on bool) *Window {
	win.autoPosition = on
	return win
}

// SetResizable determines if the user is able to resize the window directly.
func (win *Window) SetResizable(on bool) *Window {
	win.resizable = on
	return win
}

// InitWindow is called by the Desktop to initialize the window.
// Do not call directly!
func (win *Window) InitWindow() {
	d := win.desktop
	if d == nil {
		return
	}
	theme := d.winMgr.GetTheme()
	win.SetTitleAlign(theme.TitleAlign)
	if win.autoPosition {
		inX, inY, inW, inH := d.GetInnerRect()
		_, _, winW, winH := win.GetRect()
		win.SetRect(inX+d.autoWinPos, inY+d.autoWinPos, winW, winH)
		d.autoWinPos += 2
		if d.autoWinPos >= inW-10 || d.autoWinPos >= inH-10 {
			// When there's not much screen space left, reset...
			d.autoWinPos = 0
		}
	}
	if win.client != nil && win.clientFullSize {
		win.client.SetRect(win.GetInnerRect())
	}
}

func (win *Window) SetRect(x, y, width, height int) {
	if win.border { // Same code in Set*Rect.
		if width < 12 {
			width = 12
		}
		if height < 2 {
			height = 2
		}
	}
	win.Box.SetRect(x, y, width, height)
	if win.state == Restored {
		win.rx, win.ry, win.rw, win.rh = win.GetRect()
	}
	if win.client != nil && win.clientFullSize {
		win.client.SetRect(win.GetInnerRect())
	}
	if win.desktop != nil {
		win.desktop.winMgr.Resized(win)
	}
}

// GetRestoredRect gets the rect of the window as if it were restored.
func (win *Window) GetRestoredRect() (int, int, int, int) {
	return win.rx, win.ry, win.rw, win.rh
}

// SetRestoredRect sets the rect of the restored window,
// if it is not restored, it will be the size when it is later restored.
func (win *Window) SetRestoredRect(x, y, width, height int) {
	if win.state == Restored {
		win.SetRect(x, y, width, height)
	} else {
		if win.border { // Same code in Set*Rect.
			if width < 12 {
				width = 12
			}
			if height < 2 {
				height = 2
			}
		}
		win.rx, win.ry, win.rw, win.rh = x, y, width, height
	}
}

func (win *Window) GetState() WindowState {
	return win.state
}

func (win *Window) SetState(state WindowState) *Window {
	if win.desktop != nil {
		win.desktop.winMgr.SetState(win, state)
	} else {
		win.state = state
	}
	return win
}

func (win *Window) GetTitle() string {
	return win.title
}

func (win *Window) SetTitle(title string) *Window {
	win.Box.SetTitle(title)
	win.title = title
	if win.desktop != nil {
		win.desktop.winMgr.TitleChanged(win)
	}
	return win
}

func (win *Window) SetBorder(show bool) *Window {
	win.Box.SetBorder(show)
	win.border = show
	if win.client != nil && win.clientFullSize {
		win.client.SetRect(win.GetInnerRect())
	}
	if win.desktop != nil {
		win.desktop.winMgr.Resized(win)
	}
	return win
}

func (win *Window) Focus(delegate func(p tview.Primitive)) {
	if win.client != nil {
		delegate(win.client)
		return
	}
	win.Box.Focus(delegate)
}

func (win *Window) HasFocus() bool {
	if win.client != nil {
		if win.client.HasFocus() {
			return true
		}
	}
	return win.Box.HasFocus()
}

func (win *Window) BringToFront() *Window {
	if win.desktop != nil && len(win.desktop.wins) > 0 {
		wins := win.desktop.wins
		if win != wins[len(wins)-1] { // Only if it's not already in front.
			for i, xwin := range wins {
				if win == xwin {
					copy(wins[i:], wins[i+1:])
					wins[len(wins)-1] = win
					win.desktop.wins = wins
					break
				}
			}
		}
	}
	return win
}

func (win *Window) Activate(setFocus func(p tview.Primitive)) *Window {
	win.BringToFront()
	if !win.HasFocus() {
		setFocus(win)
	}
	return win
}

func (win *Window) Draw(screen tcell.Screen) {
	if win.desktop != nil {
		win.desktop.winMgr.DefaultDraw(win, screen)
	} else {
		//win.Box.Draw(screen)
		win.Box.DrawForSubclass(screen, win)
	}
}

func (win *Window) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return win.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		if win.desktop != nil && win.HasFocus() {
			if win.desktop.winMgr.DefaultInputHandler(win, event, setFocus) {
				return // consumed
			}
		}
		if win.client != nil && win.client.HasFocus() {
			if handler := win.client.InputHandler(); handler != nil {
				handler(event, setFocus)
				return
			}
		}
	})
}

func (win *Window) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return win.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
		mouseInWin := win.InRect(event.Position())

		activated := false
		if action == tview.MouseLeftDown && mouseInWin {
			if win.autoActivate {
				win.Activate(setFocus)
				activated = true
			}
		}

		if win.desktop != nil {
			consumed, capture = win.desktop.winMgr.DefaultMouseHandler(win, action, event, setFocus)
			if consumed {
				return
			}
		}

		if !mouseInWin {
			return false, nil
		}

		if win.client != nil {
			if handler := win.client.MouseHandler(); handler != nil {
				consumed, capture = handler(action, event, setFocus)
				if consumed {
					return
				}
			}
		}

		if activated {
			consumed = true
		}
		return
	})
}

func (win *Window) NextWindow() *Window {
	if win.desktop != nil {
		wins := win.desktop.wins
		if len(wins) <= 1 {
			return nil
		}
		for i, wx := range wins {
			if wx == win {
				if i == 0 {
					return wins[len(wins)-1]
				}
				return wins[i-1]
			}
		}
	}
	return nil
}

func (win *Window) PrevWindow() *Window {
	if win.desktop != nil {
		wins := win.desktop.wins
		if len(wins) <= 1 {
			return nil
		}
		for i, wx := range wins {
			if wx == win {
				if i == len(wins)-1 {
					return wins[0]
				}
				return wins[i+1]
			}
		}
	}
	return nil
}

func (win *Window) GetChildren() []tview.Primitive {
	if win.client != nil {
		return []tview.Primitive{win.client}
	}
	return nil
}
