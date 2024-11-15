package vt

import (
	"image/color"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/parser"
	"github.com/charmbracelet/x/cellbuf"
)

var spaceCell = cellbuf.Cell{
	Content: " ",
	Width:   1,
}

// handleCsi handles a CSI escape sequences.
func (t *Terminal) handleCsi(seq []byte) {
	// params := t.parser.Params[:t.parser.ParamsLen]
	cmd := t.parser.Cmd
	switch cmd { // cursor
	case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'X', 'd', 'e':
		t.handleCursor()
	case 'm': // SGR - Select Graphic Rendition
		t.handleSgr()
	case 'J':
		t.handleScreen()
	case 'K', 'L', 'M', 'S', 'T':
		t.handleLine()
	case 'l', 'h', 'l' | '?'<<parser.MarkerShift, 'h' | '?'<<parser.MarkerShift:
		t.handleMode()
	}
}

func (t *Terminal) handleMode() {
	if t.parser.ParamsLen == 0 {
		return
	}

	cmd := ansi.Cmd(t.parser.Cmd)
	mode := ansi.Param(t.parser.Params[0]).Param()
	setting := ModeReset
	if cmd.Command() == 'h' {
		setting = ModeSet
	}

	if cmd.Marker() == '?' {
		mode := ansi.DECMode(mode)
		t.pmodes[mode] = setting
		switch mode {
		case ansi.CursorEnableMode:
			t.scr.cur.Visible = setting.IsSet()
		case 1047: // Alternate Screen Buffer
			if setting == ModeSet {
				t.scr = &t.scrs[1]
			} else {
				t.scr = &t.scrs[0]
			}
		case ansi.AltScreenBufferMode:
			if setting == ModeSet {
				t.scr = &t.scrs[1]
				t.scr.Clear(nil)
			} else {
				t.scr = &t.scrs[0]
			}
		}
	} else {
		mode := ansi.ANSIMode(mode)
		t.modes[mode] = setting
	}
}

func (t *Terminal) handleScreen() {
	var count int
	if t.parser.ParamsLen > 0 {
		count = ansi.Param(t.parser.Params[0]).Param()
	}

	scr := t.scr
	cur := scr.Cursor()
	w, h := scr.Width(), scr.Height()
	_, y := cur.Pos.X, cur.Pos.Y

	cmd := ansi.Cmd(t.parser.Cmd)
	switch cmd.Command() {
	case 'J':
		switch count {
		case 0: // Erase screen below (including cursor)
			t.scr.Clear(&cellbuf.Rectangle{X: 0, Y: y, Width: w, Height: h - y})
		case 1: // Erase screen above (including cursor)
			t.scr.Clear(&cellbuf.Rectangle{X: 0, Y: 0, Width: w, Height: y + 1})
		case 2: // erase screen
			fallthrough
		case 3: // erase display
			// TODO: Scrollback buffer support?
			t.scr.Clear(&cellbuf.Rectangle{X: 0, Y: 0, Width: w, Height: h})
		}
	}
}

func (t *Terminal) handleLine() {
	var count int
	if t.parser.ParamsLen > 0 {
		count = ansi.Param(t.parser.Params[0]).Param()
	}

	cmd := ansi.Cmd(t.parser.Cmd)
	switch cmd.Command() {
	case 'K':
		// NOTE: Erase Line (EL) is a bit confusing. Erasing the line erases
		// the characters on the line while applying the cursor pen style
		// like background color and so on. The cursor position is not changed.
		cur := t.scr.Cursor()
		x, y := cur.Pos.X, cur.Pos.Y
		w := t.scr.Width()
		switch count {
		case 0: // Erase from cursor to end of line
			c := spaceCell
			c.Style = t.scr.cur.Pen
			t.scr.Fill(c, &cellbuf.Rectangle{X: x, Y: y, Width: w - x, Height: 1})
		case 1: // Erase from start of line to cursor
			c := spaceCell
			c.Style = t.scr.cur.Pen
			t.scr.Fill(c, &cellbuf.Rectangle{X: 0, Y: y, Width: x + 1, Height: 1})
		case 2: // Erase entire line
			c := spaceCell
			c.Style = t.scr.cur.Pen
			t.scr.Fill(c, &cellbuf.Rectangle{X: 0, Y: y, Width: w, Height: 1})
		}
	case 'L': // TODO: insert n blank lines
	case 'M': // TODO: delete n lines
	case 'S': // TODO: scroll up n lines
	case 'T': // TODO: scroll down n lines
	}
}

func (t *Terminal) handleCursor() {
	p := t.parser
	width, height := t.scr.Width(), t.scr.Height()
	cmd := ansi.Cmd(p.Cmd)
	n := 1
	if p.ParamsLen > 0 {
		n = p.Params[0]
	}

	x, y := t.scr.cur.Pos.X, t.scr.cur.Pos.Y
	switch cmd.Command() {
	case 'A':
		// CUU - Cursor Up
		y = max(0, y-n)
	case 'B':
		// CUD - Cursor Down
		y = min(height-1, y+n)
	case 'C':
		// CUF - Cursor Forward
		x = min(width-1, x+n)
	case 'D':
		// CUB - Cursor Back
		x = max(0, x-n)
	case 'E':
		// CNL - Cursor Next Line
		y = min(height-1, y+n)
		x = 0
	case 'F':
		// CPL - Cursor Previous Line
		y = max(0, y-n)
		x = 0
	case 'G':
		// CHA - Cursor Character Absolute
		x = min(width-1, max(0, n-1))
	case 'H', 'f':
		// CUP - Cursor Position
		// HVP - Horizontal and Vertical Position
		if p.ParamsLen >= 2 {
			row, col := ansi.Param(p.Params[0]).Param(), ansi.Param(p.Params[1]).Param()
			y = min(height-1, max(0, row-1))
			x = min(width-1, max(0, col-1))
		} else {
			x, y = 0, 0
		}
	case 'I':
		// CHT - Cursor Forward Tabulation
		for i := 0; i < n; i++ {
			x = t.scr.tabstops.Next(x)
		}
	case 'X':
		// ECH - Erase Character
		c := spaceCell
		c.Style = t.scr.cur.Pen
		t.scr.Fill(c, &cellbuf.Rectangle{X: x, Y: y, Width: n, Height: 1})
		x = min(width-1, x+n)
	case 'd':
		// VPA - Vertical Line Position Absolute
		y = min(height-1, max(0, n-1))
	case 'e':
		// VPR - Vertical Line Position Relative
		y = min(height-1, max(0, y+n))
	}
	t.scr.moveCursor(x, y)
}

// handleSgr handles SGR escape sequences.
// handleSgr handles Select Graphic Rendition (SGR) escape sequences.
func (t *Terminal) handleSgr() {
	p, pen := t.parser, &t.scr.cur.Pen
	if p.ParamsLen == 0 {
		pen.Reset()
		return
	}

	params := p.Params[:p.ParamsLen]
	for i := 0; i < len(params); i++ {
		r := ansi.Param(params[i])
		param, hasMore := r.Param(), r.HasMore() // Are there more subparameters i.e. separated by ":"?
		switch param {
		case 0: // Reset
			pen.Reset()
		case 1: // Bold
			pen.Bold(true)
		case 2: // Dim/Faint
			pen.Faint(true)
		case 3: // Italic
			pen.Italic(true)
		case 4: // Underline
			if hasMore { // Only accept subparameters i.e. separated by ":"
				nextParam := ansi.Param(params[i+1]).Param()
				switch nextParam {
				case 0: // No Underline
					pen.UnderlineStyle(cellbuf.NoUnderline)
				case 1: // Single Underline
					pen.UnderlineStyle(cellbuf.SingleUnderline)
				case 2: // Double Underline
					pen.UnderlineStyle(cellbuf.DoubleUnderline)
				case 3: // Curly Underline
					pen.UnderlineStyle(cellbuf.CurlyUnderline)
				case 4: // Dotted Underline
					pen.UnderlineStyle(cellbuf.DottedUnderline)
				case 5: // Dashed Underline
					pen.UnderlineStyle(cellbuf.DashedUnderline)
				}
			} else {
				// Single Underline
				pen.Underline(true)
			}
		case 5: // Slow Blink
			pen.SlowBlink(true)
		case 6: // Rapid Blink
			pen.RapidBlink(true)
		case 7: // Reverse
			pen.Reverse(true)
		case 8: // Conceal
			pen.Conceal(true)
		case 9: // Crossed-out/Strikethrough
			pen.Strikethrough(true)
		case 22: // Normal Intensity (not bold or faint)
			pen.Bold(false).Faint(false)
		case 23: // Not italic, not Fraktur
			pen.Italic(false)
		case 24: // Not underlined
			pen.Underline(false)
		case 25: // Blink off
			pen.SlowBlink(false).RapidBlink(false)
		case 27: // Positive (not reverse)
			pen.Reverse(false)
		case 28: // Reveal
			pen.Conceal(false)
		case 29: // Not crossed out
			pen.Strikethrough(false)
		case 30, 31, 32, 33, 34, 35, 36, 37: // Set foreground
			pen.Foreground(ansi.Black + ansi.BasicColor(param-30)) //nolint:gosec
		case 38: // Set foreground 256 or truecolor
			if c := readColor(&i, params); c != nil {
				pen.Foreground(c)
			}
		case 39: // Default foreground
			pen.Foreground(nil)
		case 40, 41, 42, 43, 44, 45, 46, 47: // Set background
			pen.Background(ansi.Black + ansi.BasicColor(param-40)) //nolint:gosec
		case 48: // Set background 256 or truecolor
			if c := readColor(&i, params); c != nil {
				pen.Background(c)
			}
		case 49: // Default Background
			pen.Background(nil)
		case 58: // Set underline color
			if c := readColor(&i, params); c != nil {
				pen.UnderlineColor(c)
			}
		case 59: // Default underline color
			pen.UnderlineColor(nil)
		case 90, 91, 92, 93, 94, 95, 96, 97: // Set bright foreground
			pen.Foreground(ansi.BrightBlack + ansi.BasicColor(param-90)) //nolint:gosec
		case 100, 101, 102, 103, 104, 105, 106, 107: // Set bright background
			pen.Background(ansi.BrightBlack + ansi.BasicColor(param-100)) //nolint:gosec
		}
	}
}

func readColor(idxp *int, params []int) (c ansi.Color) {
	i := *idxp
	paramsLen := len(params)
	if i > paramsLen-1 {
		return
	}
	// Note: we accept both main and subparams here
	switch param := ansi.Param(params[i+1]); param {
	case 2: // RGB
		if i > paramsLen-4 {
			return
		}
		c = color.RGBA{
			R: uint8(ansi.Param(params[i+2])), //nolint:gosec
			G: uint8(ansi.Param(params[i+3])), //nolint:gosec
			B: uint8(ansi.Param(params[i+4])), //nolint:gosec
			A: 0xff,
		}
		*idxp += 4
	case 5: // 256 colors
		if i > paramsLen-2 {
			return
		}
		c = ansi.ExtendedColor(ansi.Param(params[i+2])) //nolint:gosec
		*idxp += 2
	}
	return
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

func clamp(v, low, high int) int {
	if high < low {
		low, high = high, low
	}
	return min(high, max(low, v))
}
