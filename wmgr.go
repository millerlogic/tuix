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

// WindowManager represents the management of windows on a desktop.
// Note that many Window calls call into the window manager,
// so if the window manager needs to make changes, it could call back recursively.
type WindowManager interface {
	Added(win *Window)        // window added to desktop
	Removed(win *Window)      // window removed from desktop
	Resized(win *Window)      // window resized
	TitleChanged(win *Window) // window title changed
	StateChanged(win *Window) // window state changed
	GetTheme() WindowTheme
	SetTheme(theme WindowTheme)
	DesktopResized(d *Desktop)
	DesktopDraw(d *Desktop, screen tcell.Screen)  // allows drawing a wallpaper, etc
	DefaultDraw(win *Window, screen tcell.Screen) // for a window
	DefaultInputHandler(win *Window, event *tcell.EventKey, setFocus func(p tview.Primitive)) (consumed bool)
	DefaultMouseHandler(win *Window, action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive)
}

type winMgr struct {
	theme WindowTheme
}

var _ WindowManager = &winMgr{}

func (wm *winMgr) Added(win *Window) {
}

func (wm *winMgr) Removed(win *Window) {
}

func (wm *winMgr) Resized(win *Window) {
}

func (wm *winMgr) TitleChanged(win *Window) {
}

func (wm *winMgr) StateChanged(win *Window) {
	switch win.state {
	case Restored:
		win.SetRect(win.rx, win.ry, win.rw, win.rh)
	case Minimized:
		win.SetRect(0, 0, 1, 1) // Let SetRect bound it.
	case Maximized:
		if win.desktop != nil {
			win.SetRect(win.desktop.GetInnerRect())
		}
	}
}

func (wm *winMgr) GetTheme() WindowTheme {
	return wm.theme
}

func (wm *winMgr) SetTheme(theme WindowTheme) {
	wm.theme = theme
}

func (wm *winMgr) DesktopResized(d *Desktop) {
	for _, win := range d.wins {
		if win.state == Maximized {
			win.SetRect(d.GetInnerRect())
		}
	}
}

func (wm *winMgr) DesktopDraw(d *Desktop, screen tcell.Screen) {
}

func (wm *winMgr) DefaultDraw(win *Window, screen tcell.Screen) {
	//win.Box.Draw(screen)
	win.Box.DrawForSubclass(screen, win)
	x, y, w, h := win.GetRect()
	focused := win.HasFocus()
	if !win.noCaption {
		style := tcell.StyleDefault
		if focused {
			style = style.Foreground(wm.theme.ActiveCaptionTextColor)
			style = style.Background(wm.theme.ActiveCaptionColor)
		} else {
			style = style.Foreground(wm.theme.InactiveCaptionTextColor)
			style = style.Background(wm.theme.InactiveCaptionColor)
		}
		for i := x; i < x+w; i++ {
			// Use whatever is there as the caption text.
			c, combc, _, _ := screen.GetContent(i, y)
			screen.SetContent(i, y, c, combc, style)
		}
	}
	if win.resizable && focused && screen.HasMouse() {
		c, combc, _, _ := screen.GetContent(x+w-1, y+h-1)
		screen.SetContent(x+w-1, y+h-1, c, combc,
			tcell.StyleDefault.Foreground(tcell.ColorValid+226))
	}
	if win.client != nil {
		win.client.Draw(screen)
	}
}

func (wm *winMgr) DefaultInputHandler(win *Window, event *tcell.EventKey, setFocus func(p tview.Primitive)) (consumed bool) {
	return
}

func (wm *winMgr) DefaultMouseHandler(win *Window, action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	if !win.InRect(event.Position()) && !win.moving && win.resizing == 0 {
		return
	}

	if action == tview.MouseLeftDown {
		x, y, w, h := win.GetRect()
		atX, atY := event.Position()
		if win.border && atY >= y && atY < y+1 { // mouse in caption
			win.moveX, win.moveY = atX-x, atY-y
			win.moving = event.Buttons() == tcell.Button1
		}
		if win.moving {
			consumed = true
		} else if !win.moving && win.resizable {
			if atX == x+w-1 {
				win.resizing |= 1
				consumed = true
			}
			if atY == y+h-1 {
				win.resizing |= 2
				consumed = true
			}
		}
	} else if action == tview.MouseLeftUp {
		if win.moving || win.resizing != 0 {
			win.moving = false
			win.resizing = 0
			// Move or resize is done, consume but don't capture mouse.
			return true, nil
		}
	} else if action == tview.MouseMove {
		x, y, w, h := win.GetRect()
		atX, atY := event.Position()
		if win.moving {
			moveX, moveY := atX-x, atY-y
			win.SetRect(x+(moveX-win.moveX), y+(moveY-win.moveY), w, h)
			consumed = true
		} else if win.resizing != 0 {
			neww := w
			if win.resizing&1 != 0 {
				neww = atX - x + 1
			}
			newh := h
			if win.resizing&2 != 0 {
				newh = atY - y + 1
			}
			win.SetRect(x, y, neww, newh)
			consumed = true
		}
	}
	if consumed {
		capture = win
		return
	}

	if action == tview.MouseLeftDoubleClick {
		if win.resizable && win.desktop != nil {
			_, y, _, _ := win.GetRect()
			_, atY := event.Position()
			if win.border && atY >= y && atY < y+1 { // mouse in caption
				switch win.GetState() {
				case Minimized, Maximized:
					win.SetState(Restored)
				case Restored:
					//if win.resizable {
					win.SetState(Maximized)
				}
				consumed = true
			}
		}
	}
	return
}

var defWinMgr = &winMgr{theme: DefaultWindowTheme}

// DefaultWindowManager is the default window manager.
// Most likely when making your own window manager, you'll want to embed this one.
var DefaultWindowManager = defWinMgr

// WindowState is a state of the window, managed by the window manager.
type WindowState byte

const (
	Restored WindowState = iota
	Minimized
	Maximized
)

type WindowTheme struct {
	TitleAlign               int
	ActiveCaptionTextColor   tcell.Color
	ActiveCaptionColor       tcell.Color
	InactiveCaptionTextColor tcell.Color
	InactiveCaptionColor     tcell.Color
}

// DefaultWindowTheme is the default desktop theme.
// These colors were chosen to look decent and readable in most color counts.
var DefaultWindowTheme = WindowTheme{
	TitleAlign:               tview.AlignLeft,
	ActiveCaptionTextColor:   tcell.ColorValid + 230,
	ActiveCaptionColor:       tcell.ColorValid + 26,
	InactiveCaptionTextColor: tcell.ColorValid + 15,
	InactiveCaptionColor:     tcell.ColorValid + 239,
}
