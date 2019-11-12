// Copyright (C) 2019 Christopher E. Miller
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package tuix

import (
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

// Desktop represents an area where windows go.
type Desktop struct {
	*tview.Box
	wins            []*Window // in stack/draw order
	winMgr          WindowManager
	client          tview.Primitive
	lastClickTimeMs int64
	lastClickPosxy  int
	autoWinPos      int
	init            bool
	clientFullSize  bool
	clickCount      byte
	dclickDelay20ms byte
}

// NewDesktop creates a new desktop, it needs to be added to an Application.
func NewDesktop() *Desktop {
	return &Desktop{
		Box:             tview.NewBox(),
		winMgr:          DefaultWindowManager,
		dclickDelay20ms: 500 / 20, /// 500ms is common double-click delay.
	}
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

// GetClickCount can be used from a mouse event to determine how many clicks,
// such as 1 for single click, or 2 for double click.
func (d *Desktop) GetClickCount() int {
	return int(d.clickCount)
}

func (d *Desktop) GetDoubleClickDelay() time.Duration {
	return time.Duration(d.dclickDelay20ms) * 20 * time.Millisecond
}

func (d *Desktop) SetDoubleClickDelay(dur time.Duration) *Desktop {
	ms := dur.Milliseconds()
	dur20ms := ms / 20
	if ms%20 >= 10 {
		dur20ms++
	}
	if dur20ms >= 255 {
		dur20ms = 255
	} else {
		d.dclickDelay20ms = byte(dur20ms)
	}
	return d
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
		if win.GetFocusable().HasFocus() {
			return true
		}
	}
	return d.Box.HasFocus()
}

func (d *Desktop) GetFocusable() tview.Focusable {
	return d
}

func (d *Desktop) Draw(screen tcell.Screen) {
	d.Box.Draw(screen)
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

func (d *Desktop) lastClickTime() time.Time {
	return time.Unix(0, d.lastClickTimeMs*1000000)
}

func (d *Desktop) withinClickDelay() bool {
	return time.Now().Before(d.lastClickTime().Add(d.GetDoubleClickDelay()))
}

func (d *Desktop) ObserveMouseEvent(event *tview.EventMouse) {
	if event.Action()&tview.MouseClick != 0 {
		atX, atY := event.Position()
		clickPosxy := atX ^ atY<<16
		if clickPosxy != d.lastClickPosxy || d.clickCount == 0 {
			d.clickCount = 1
		} else if d.withinClickDelay() {
			d.clickCount++
		} else {
			d.clickCount = 1
		}
		d.lastClickTimeMs = time.Now().UnixNano() / 1000000
		d.lastClickPosxy = clickPosxy
	}
}

func (d *Desktop) GetChildren() []tview.Primitive {
	clen := len(d.wins)
	if d.client != nil {
		clen++
	}
	children := make([]tview.Primitive, 0, clen)
	if d.client != nil {
		children = append(children, d.client)
	}
	for _, win := range d.wins {
		children = append(children, win)
	}
	return children
}

func findChild(pp, p tview.Primitive) bool {
	for _, px := range pp.GetChildren() {
		if px == p {
			return true
		}
		if findChild(px, p) {
			return true
		}
	}
	return false
}

// FindPrimitive looks for the primitive within the desktop, returns true if found.
// If it returns (nil, true) then the child is part of the desktop client.
func (d *Desktop) FindPrimitive(p tview.Primitive) (*Window, bool) {
	if p == d {
		return nil, true
	}
	if d.client != nil && (d.client == p || findChild(d.client, p)) {
		return nil, true
	}
	for _, win := range d.wins {
		if win == p || findChild(win, p) {
			return win, true
		}
	}
	return nil, false
}
