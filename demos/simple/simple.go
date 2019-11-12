// This demo is public domain, or MPLv2 if you prefer.

package main

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell"
	"github.com/millerlogic/tuix"
	"github.com/rivo/tview"
)

func run() error {
	app := tview.NewApplication()

	//app.EnableMouse() // Use this instead of creating a new screen.
	// Explicitly creating screen here to test for features.
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	err = screen.Init()
	if err != nil {
		return err
	}
	screen.EnableMouse()
	app.SetScreen(screen)

	win := tuix.NewWindow().SetAutoPosition(true)
	win.SetTitle("Window 1")
	win.SetBorder(true).SetRect(0, 0, 30, 15)
	wform := tview.NewForm()
	wform.AddButton("Restore", func() {
		win.SetState(tuix.Restored)
	})
	wform.AddButton("Maximize", func() {
		win.SetState(tuix.Maximized)
	})
	win.SetClient(wform, true)

	win2 := tuix.NewWindow().SetAutoPosition(true).SetResizable(true)
	win2.SetTitle("Window 2")
	win2.SetBorder(true).SetRect(0, 0, 30, 15)
	tv := tview.NewTextView()
	mouseComment := ""
	if !screen.HasMouse() {
		mouseComment = "   [red]<<No mouse!>>[-]"
	}
	colorComment := " [green]Good![-]"
	if screen.Colors() < 256 {
		colorComment = " [red]Not good![-]"
	}
	fmt.Fprintf(tv,
		"[orange]Hello![-]%s\n\n"+
			"Your terminal supports [::b]%d[::-] colors.%s\n\n"+
			"Click and drag the window captions to move around.\n\n"+
			"You can resize this window and double click the caption.",
		mouseComment, screen.Colors(), colorComment)
	tv.SetWordWrap(true).SetDynamicColors(true).SetScrollable(true)
	tv.SetBorderPadding(1, 1, 1, 1)
	win2.SetClient(tv, true)

	desktop := tuix.NewDesktop()
	desktop.SetBackgroundColor(234)
	//desktop.SetTitle("Desktop").SetBorder(true)
	desktop.AddWindow(win).AddWindow(win2)

	app.SetRoot(desktop, true)

	if err := app.Run(); err != nil {
		return err
	}

	return nil
}

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
