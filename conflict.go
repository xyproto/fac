package main

import (
	"bytes"
	"fmt"

	"github.com/jroimartin/gocui"
)

// Conflict represents a single conflict that may have occured
type Conflict struct {
	Choice       int
	FileName     string
	AbsolutePath string
	Start        int
	Middle       int
	End          int

	CurrentLines        []string
	ForeignLines        []string
	ColoredCurrentLines []string
	ColoredForeignLines []string

	CurrentName string
	ForeignName string

	topPeek     int
	bottomPeek  int
	displayDiff bool
}

type ErrNoConflict struct {
	message string
}

func NewErrNoConflict(message string) *ErrNoConflict {
	return &ErrNoConflict{
		message: message,
	}
}

func (e *ErrNoConflict) Error() string {
	return e.message
}

const (
	Local    = 1
	Incoming = 2
	Up       = 3
	Down     = 4
)

func (c *Conflict) isEqual(c2 *Conflict) bool {
	return c.FileName == c2.FileName && c.Start == c2.Start
}

func (c *Conflict) toggleDiff() {
	c.displayDiff = !(c.displayDiff)
}

func (c *Conflict) Select(g *gocui.Gui, showHelp bool) error {
	g.Update(func(g *gocui.Gui) error {
		v, err := g.View(Panel)
		if err != nil {
			return err
		}
		v.Clear()

		for idx, conflict := range conflicts {
			var out string
			if conflict.Choice != 0 {
				out = Green(Regular, "✅  %s:%d", conflict.FileName, conflict.Start)
			} else {
				out = Red(Regular, "%d. %s:%d", idx+1, conflict.FileName, conflict.Start)
			}

			if conflict.isEqual(c) {
				fmt.Fprintf(v, "%s <-\n", out)
			} else {
				fmt.Fprintf(v, "%s\n", out)
			}
		}

		if showHelp {
			printHelp(v)
		}
		return nil
	})

	g.Update(func(g *gocui.Gui) error {
		v, err := g.View(Current)
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		buf.WriteString(c.CurrentName)
		buf.WriteString(" (Current Change) ")
		v.Title = buf.String()

		top, bottom := c.getPaddingLines()
		v.Clear()
		printLines(v, top)
		if c.displayDiff {
			printLines(v, c.diff())
		} else {
			printLines(v, c.ColoredCurrentLines)
		}
		printLines(v, bottom)

		v, err = g.View(Foreign)
		if err != nil {
			return err
		}
		buf.Reset()
		buf.WriteString(c.ForeignName)
		buf.WriteString(" (Incoming Change) ")
		v.Title = buf.String()

		top, bottom = c.getPaddingLines()
		v.Clear()
		printLines(v, top)
		printLines(v, c.ColoredForeignLines)
		printLines(v, bottom)
		return nil
	})
	return nil
}

func (c *Conflict) getPaddingLines() (topPadding, bottomPadding []string) {
	lines := allFileLines[c.AbsolutePath]
	start, end := c.Start-1, c.End

	if c.topPeek >= start {
		c.topPeek = start
	} else if c.topPeek < 0 {
		c.topPeek = 0
	}

	for _, l := range lines[start-c.topPeek : start] {
		topPadding = append(topPadding, Black(Regular, l))
	}

	if c.bottomPeek >= len(lines)-c.End {
		c.bottomPeek = len(lines) - c.End
	} else if c.bottomPeek < 0 {
		c.bottomPeek = 0
	}

	for _, l := range lines[end : end+c.bottomPeek] {
		bottomPadding = append(bottomPadding, Black(Regular, l))
	}
	return
}

func (c *Conflict) Resolve(g *gocui.Gui, v *gocui.View, version int) error {
	g.Update(func(g *gocui.Gui) error {
		c.Choice = version
		nextConflict(g, v)
		return nil
	})
	return nil
}

func nextConflict(g *gocui.Gui, v *gocui.View) error {
	originalCur := cur

	for originalCur != cur {
		cur++
		if cur >= conflictCount {
			cur = 0
		}
	}

	if originalCur == cur {
		globalQuit(g)
	}

	conflicts[cur].Select(g, false)
	return nil
}

func scroll(g *gocui.Gui, c *Conflict, direction int) {
	if direction == Up {
		c.topPeek--
		c.bottomPeek++
	} else if direction == Down {
		c.topPeek++
	} else {
		return
	}

	c.Select(g, false)
}
