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

// Desktop represents an area where windows go.
type Desktop struct {
	*tview.Box
	wins           []*Window // in stack/draw order
	winMgr         WindowManager
	client         tview.Primitive
	autoWinPos     int
	init           bool
	clientFullSize bool
}

// NewDesktop creates a new desktop, it needs to be added to an Application.
func NewDesktop() *Desktop {
	d := &Desktop{
		Box:    tview.NewBox(),
		winMgr: DefaultWindowManager,
	}
	d.SetBackgroundColor(tcell.ColorValid + 234)
	return d
}

// AddWindow adds a window to the desktop.
// If the window already belongs to a desktop, it is first removed.
// The window is added to the top of the stack, but it is not activated.
func (d *Desktop) AddWindow(win *Window) *Desktop {
	if win.desktop != nil {
		if win.desktop == d {
			return d
		}
		win.desktop.RemoveWindow(win)
	}
	d.wins = append(d.wins, win)
	win.Desktop(d)
	if d.init {
		win.InitWindow()
		/*if win.autoActivate {
			app.SetFocus(win)
		}*/
	}
	d.winMgr.Added(win)
	return d
}

// RemoveWindow removes a window from the desktop.
// The window should be blurred before calling this.
func (d *Desktop) RemoveWindow(win *Window) *Desktop {
	for i, xw := range d.wins {
		if xw == win {
			//hasFocus := d.init && win.GetFocusable().HasFocus()
			copy(d.wins[i:], d.wins[i+1:])
			d.wins = d.wins[:len(d.wins)-1]
			/*if hasFocus {
				// Focus the one before it.
				if i == 0 {
					if len(d.wins) > 0 {
						app.SetFocus(d.wins[len(d.wins)-1])
					}
				} else {
					app.SetFocus(d.wins[i-1])
				}
			}*/
			d.winMgr.Removed(win)
			break
		}
	}
	return d
}

// TopWindow gets the top window, highest in z-order.
func (d *Desktop) TopWindow() *Window {
	if len(d.wins) > 0 {
		return d.wins[len(d.wins)-1]
	}
	return nil
}

// BottomWindow gets the bottom window, lowest in z-order.
func (d *Desktop) BottomWindow() *Window {
	if len(d.wins) > 0 {
		return d.wins[0]
	}
	return nil
}

// GetClient gets the client primitive previously set by SetClient, or nil.
func (d *Desktop) GetClient() tview.Primitive {
	return d.client
}

// SetClient sets a desktop client primitive.
// A desktop client can be a way to show desktop icons or widgets behind windows.
// The client can avoid drawing its background to inherit the desktop background.
func (d *Desktop) SetClient(client tview.Primitive, fullSize bool) {
	d.client = client
	d.clientFullSize = fullSize
	if client != nil && fullSize {
		client.SetRect(d.GetInnerRect())
	}
}

// SetWindowManager changes the WindowManager; see DefaultWindowManager
func (d *Desktop) SetWindowManager(wm WindowManager) {
	if d.winMgr == wm {
		return
	}

	for _, win := range d.wins {
		d.winMgr.Removed(win)
	}

	if wm == nil {
		wm = DefaultWindowManager
	}
	d.winMgr = wm

	for _, win := range d.wins {
		wm.Added(win)
	}
}

func (d *Desktop) SetRect(x, y, width, height int) {
	d.Box.SetRect(x, y, width, height)
	if d.client != nil && d.clientFullSize {
		d.client.SetRect(d.GetInnerRect())
	}
	d.winMgr.DesktopResized(d)
}

func (d *Desktop) SetBorder(show bool) *Desktop {
	d.Box.SetBorder(show)
	if d.client != nil && d.clientFullSize {
		d.client.SetRect(d.GetInnerRect())
	}
	d.winMgr.DesktopResized(d)
	return d
}

func (d *Desktop) Focus(delegate func(p tview.Primitive)) {
	if len(d.wins) > 0 {
		// Focus one on top.
		delegate(d.wins[len(d.wins)-1])
		return
	}
	d.Box.Focus(delegate)
}

func (d *Desktop) HasFocus() bool {
	for _, win := range d.wins {
		if win.HasFocus() {
			return true
		}
	}
	return d.Box.HasFocus()
}

func (d *Desktop) Draw(screen tcell.Screen) {
	//d.Box.Draw(screen)
	d.Box.DrawForSubclass(screen, d)
	d.winMgr.DesktopDraw(d, screen)
	if d.client != nil {
		d.client.Draw(screen)
	}
	init := d.init
	d.init = true
	if !init {
		d.winMgr.DesktopResized(d)
	}
	for _, win := range d.wins {
		if !init {
			win.InitWindow()
		}
		win.Draw(screen)
	}
}

func (d *Desktop) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return d.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		if d.client != nil && d.client.HasFocus() {
			if handler := d.client.InputHandler(); handler != nil {
				handler(event, setFocus)
				return
			}
		}
		for _, win := range d.wins {
			if win.HasFocus() {
				if handler := win.InputHandler(); handler != nil {
					handler(event, setFocus)
					return
				}
			}
		}
	})
}

func (d *Desktop) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return d.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
		atX, atY := event.Position()
		if !d.InRect(atX, atY) {
			return false, nil
		}

		// Propagate mouse events; needs to be reverse order, topmost first!
		for iwin := len(d.wins) - 1; iwin >= 0; iwin-- {
			win := d.wins[iwin]
			consumed, capture = win.MouseHandler()(action, event, setFocus)
			if consumed {
				return
			}
		}
		if d.client != nil {
			if handler := d.client.MouseHandler(); handler != nil {
				consumed, capture = handler(action, event, setFocus)
				if consumed {
					return
				}
			}
		}
		return
	})
}
