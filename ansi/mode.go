package ansi

import (
	"strconv"
	"strings"
)

// Mode represents an interface for terminal modes.
// Modes can be set, reset, and requested.
type Mode interface {
	Mode() int
}

// SetMode (SM) returns a sequence to set a mode.
// The mode arguments are a list of modes to set.
//
// If one of the modes is a [DECMode], the sequence will use the DEC format.
//
// ANSI format:
//
//	CSI Pd ; ... ; Pd h
//
// DEC format:
//
//	CSI ? Pd ; ... ; Pd h
//
// See: https://vt100.net/docs/vt510-rm/SM.html
func SetMode(modes ...Mode) string {
	return setMode(false, modes...)
}

// SM is an alias for [SetMode].
func SM(modes ...Mode) string {
	return SetMode(modes...)
}

// ResetMode (RM) returns a sequence to reset a mode.
// The mode arguments are a list of modes to reset.
//
// If one of the modes is a [DECMode], the sequence will use the DEC format.
//
// ANSI format:
//
//	CSI Pd ; ... ; Pd l
//
// DEC format:
//
//	CSI ? Pd ; ... ; Pd l
//
// See: https://vt100.net/docs/vt510-rm/RM.html
func ResetMode(modes ...Mode) string {
	return setMode(true, modes...)
}

// RM is an alias for [ResetMode].
func RM(modes ...Mode) string {
	return ResetMode(modes...)
}

func setMode(reset bool, modes ...Mode) string {
	if len(modes) == 0 {
		return ""
	}

	cmd := "h"
	if reset {
		cmd = "l"
	}

	seq := "\x1b["
	if len(modes) == 1 {
		switch modes[0].(type) {
		case DECMode:
			seq += "?"
		}
		return seq + strconv.Itoa(modes[0].Mode()) + cmd
	}

	var (
		dec  bool
		list []string
	)
	for _, m := range modes {
		list = append(list, strconv.Itoa(m.Mode()))
		switch m.(type) {
		case DECMode:
			dec = true
		}
	}

	if dec {
		seq += "?"
	}

	return seq + strings.Join(list, ";") + cmd
}

// RequestMode (DECRQM) returns a sequence to request a mode from the terminal.
// The terminal responds with a report mode function [DECRPM].
//
// ANSI format:
//
//	CSI Pa $ p
//
// DEC format:
//
//	CSI ? Pa $ p
//
// See: https://vt100.net/docs/vt510-rm/DECRQM.html
func RequestMode(m Mode) string {
	seq := "\x1b["
	switch m.(type) {
	case DECMode:
		seq += "?"
	}
	return seq + strconv.Itoa(m.Mode()) + "$p"
}

// DECRQM is an alias for [RequestMode].
func DECRQM(m Mode) string {
	return RequestMode(m)
}

// ReportMode (DECRPM) returns a sequence that the terminal sends to the host
// in response to a mode request [DECRQM].
//
// ANSI format:
//
//	CSI Pa ; Ps ; $ y
//
// DEC format:
//
//	CSI ? Pa ; Ps $ y
//
// Where Pa is the mode number, and Ps is the mode value.
//
//	0: Not recognized
//	1: Set
//	2: Reset
//	3: Permanent set
//	4: Permanent reset
//
// See: https://vt100.net/docs/vt510-rm/DECRPM.html
func ReportMode(mode, value int) string {
	if mode < 0 {
		mode = 0
	}
	if value < 0 {
		value = 0
	}
	return "\x1b[" + strconv.Itoa(mode) + ";" + strconv.Itoa(value) + "$y"
}

// DECRPM is an alias for [ReportMode].
func DECRPM(mode, value int) string {
	return ReportMode(mode, value)
}

// ANSIMode represents an ANSI terminal mode.
type ANSIMode int //nolint:revive

// Mode returns the ANSI mode as an integer.
func (m ANSIMode) Mode() int {
	return int(m)
}

// DECMode represents a private DEC terminal mode.
type DECMode int

// Mode returns the DEC mode as an integer.
func (m DECMode) Mode() int {
	return int(m)
}

// Cursor Keys Mode (DECCKM) is a mode that determines whether the cursor keys
// send ANSI cursor sequences or application sequences.
//
// See: https://vt100.net/docs/vt510-rm/DECCKM.html
const (
	CursorKeysMode = DECMode(1)
	DECCKM         = CursorKeysMode

	SetCursorKeysMode     = "\x1b[?1h"
	ResetCursorKeysMode   = "\x1b[?1l"
	RequestCursorKeysMode = "\x1b[?1$p"
)

// Deprecated: use [SetCursorKeysMode] and [ResetCursorKeysMode] instead.
const (
	EnableCursorKeys  = "\x1b[?1h"
	DisableCursorKeys = "\x1b[?1l"
)

// Origin Mode (DECOM) is a mode that determines whether the cursor moves to the
// home position or the margin position.
//
// See: https://vt100.net/docs/vt510-rm/DECOM.html
const (
	OriginMode = DECMode(6)

	SetOriginMode     = "\x1b[?6h"
	ResetOriginMode   = "\x1b[?6l"
	RequestOriginMode = "\x1b[?6$p"
)

// Autowrap Mode (DECAWM) is a mode that determines whether the cursor wraps
// to the next line when it reaches the right margin.
//
// See: https://vt100.net/docs/vt510-rm/DECAWM.html
const (
	AutowrapMode = DECMode(7)
	DECAWM       = AutowrapMode

	SetAutowrapMode     = "\x1b[?7h"
	ResetAutowrapMode   = "\x1b[?7l"
	RequestAutowrapMode = "\x1b[?7$p"
)

// X10 Mouse Mode is a mode that determines whether the mouse reports on button
// presses.
//
// The terminal responds with the following encoding:
//
//	CSI M CbCxCy
//
// Where Cb is the button-1, where it can be 1, 2, or 3.
// Cx and Cy are the x and y coordinates of the mouse event.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Mouse-Tracking
const (
	X10MouseMode = DECMode(9)

	SetX10MouseMode     = "\x1b[?9h"
	ResetX10MouseMode   = "\x1b[?9l"
	RequestX10MouseMode = "\x1b[?9$p"
)

// Text Cursor Enable Mode (DECTCEM) is a mode that shows/hides the cursor.
//
// See: https://vt100.net/docs/vt510-rm/DECTCEM.html
const (
	TextCursorEnableMode = DECMode(25)
	DECTCEM              = TextCursorEnableMode

	SetTextCursorEnableMode     = "\x1b[?25h"
	ResetTextCursorEnableMode   = "\x1b[?25l"
	RequestTextCursorEnableMode = "\x1b[?25$p"
)

// These are aliases for [SetTextCursorEnableMode] and [ResetTextCursorEnableMode].
const (
	ShowCursor = SetTextCursorEnableMode
	HideCursor = ResetTextCursorEnableMode
)

// Text Cursor Enable Mode (DECTCEM) is a mode that shows/hides the cursor.
//
// See: https://vt100.net/docs/vt510-rm/DECTCEM.html
// Deprecated: use [SetTextCursorEnableMode] and [ResetTextCursorEnableMode] instead.
const (
	CursorEnableMode        = DECMode(25)
	RequestCursorVisibility = "\x1b[?25$p"
)

// Numeric Keypad Mode (DECNKM) is a mode that determines whether the keypad
// sends application sequences or numeric sequences.
//
// This works like [DECKPAM] and [DECKPNM], but uses different sequences.
//
// See: https://vt100.net/docs/vt510-rm/DECNKM.html
const (
	NumericKeypadMode = DECMode(66)
	DECNKM            = NumericKeypadMode

	SetNumericKeypadMode     = "\x1b[?66h"
	ResetNumericKeypadMode   = "\x1b[?66l"
	RequestNumericKeypadMode = "\x1b[?66$p"
)

// Normal Mouse Mode is a mode that determines whether the mouse reports on
// button presses and releases. It will also report modifier keys, wheel
// events, and extra buttons.
//
// It uses the same encoding as [X10MouseMode] with a few differences:
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Mouse-Tracking
const (
	NormalMouseMode = DECMode(1000)

	SetNormalMouseMode     = "\x1b[?1000h"
	ResetNormalMouseMode   = "\x1b[?1000l"
	RequestNormalMouseMode = "\x1b[?1000$p"
)

// VT Mouse Tracking is a mode that determines whether the mouse reports on
// button press and release.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Mouse-Tracking
// Deprecated: use [NormalMouseMode] instead.
const (
	MouseMode = DECMode(1000)

	EnableMouse  = "\x1b[?1000h"
	DisableMouse = "\x1b[?1000l"
	RequestMouse = "\x1b[?1000$p"
)

// Highlight Mouse Tracking is a mode that determines whether the mouse reports
// on button presses, releases, and highlighted cells.
//
// It uses the same encoding as [NormalMouseMode] with a few differences:
//
// On highlight events, the terminal responds with the following encoding:
//
//	CSI t CxCy
//	CSI T CxCyCxCyCxCy
//
// Where the parameters are startx, starty, endx, endy, mousex, and mousey.
const (
	HighlightMouseMode = DECMode(1001)

	SetHighlightMouseMode     = "\x1b[?1001h"
	ResetHighlightMouseMode   = "\x1b[?1001l"
	RequestHighlightMouseMode = "\x1b[?1001$p"
)

// VT Hilite Mouse Tracking is a mode that determines whether the mouse reports on
// button presses, releases, and highlighted cells.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Mouse-Tracking
// Deprecated: use [HighlightMouseMode] instead.
const (
	MouseHiliteMode = DECMode(1001)

	EnableMouseHilite  = "\x1b[?1001h"
	DisableMouseHilite = "\x1b[?1001l"
	RequestMouseHilite = "\x1b[?1001$p"
)

// Button Event Mouse Tracking is essentially the same as [NormalMouseMode],
// but it also reports button-motion events when a button is pressed.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Mouse-Tracking
const (
	ButtonEventMouseMode = DECMode(1002)

	SetButtonEventMouseMode     = "\x1b[?1002h"
	ResetButtonEventMouseMode   = "\x1b[?1002l"
	RequestButtonEventMouseMode = "\x1b[?1002$p"
)

// Cell Motion Mouse Tracking is a mode that determines whether the mouse
// reports on button press, release, and motion events.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Mouse-Tracking
// Deprecated: use [ButtonEventMouseMode] instead.
const (
	MouseCellMotionMode = DECMode(1002)

	EnableMouseCellMotion  = "\x1b[?1002h"
	DisableMouseCellMotion = "\x1b[?1002l"
	RequestMouseCellMotion = "\x1b[?1002$p"
)

// Any Event Mouse Tracking is the same as [ButtonEventMouseMode], except that
// all motion events are reported even if no mouse buttons are pressed.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Mouse-Tracking
const (
	AnyEventMouseMode = DECMode(1003)

	SetAnyEventMouseMode     = "\x1b[?1003h"
	ResetAnyEventMouseMode   = "\x1b[?1003l"
	RequestAnyEventMouseMode = "\x1b[?1003$p"
)

// All Mouse Tracking is a mode that determines whether the mouse reports on
// button press, release, motion, and highlight events.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Mouse-Tracking
// Deprecated: use [AnyEventMouseMode] instead.
const (
	MouseAllMotionMode = DECMode(1003)

	EnableMouseAllMotion  = "\x1b[?1003h"
	DisableMouseAllMotion = "\x1b[?1003l"
	RequestMouseAllMotion = "\x1b[?1003$p"
)

// Focus Event Mode is a mode that determines whether the terminal reports focus
// and blur events.
//
// The terminal sends the following encoding:
//
//	CSI I // Focus In
//	CSI O // Focus Out
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Focus-Tracking
const (
	FocusEventMode = DECMode(1004)

	SetFocusEventMode     = "\x1b[?1004h"
	ResetFocusEventMode   = "\x1b[?1004l"
	RequestFocusEventMode = "\x1b[?1004$p"
)

// Deprecated: use [SetFocusEventMode], [ResetFocusEventMode], and
// [RequestFocusEventMode] instead.
const (
	ReportFocusMode = DECMode(1004)

	EnableReportFocus  = "\x1b[?1004h"
	DisableReportFocus = "\x1b[?1004l"
	RequestReportFocus = "\x1b[?1004$p"
)

// Mouse SGR Extended Mode is a mode that changes the mouse tracking encoding
// to use SGR parameters.
//
// The terminal responds with the following encoding:
//
//	CSI < Cb ; Cx ; Cy M
//
// Where Cb is the same as [NormalMouseMode], and Cx and Cy are the x and y.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Mouse-Tracking
const (
	MouseSgrExtMode = DECMode(1006)

	SetSgrExtMouseMode     = "\x1b[?1006h"
	ResetSgrExtMouseMode   = "\x1b[?1006l"
	RequestSgrExtMouseMode = "\x1b[?1006$p"
)

// Deprecated: use [SetSgrExtMouseMode], [ResetSgrExtMouseMode], and
// [RequestSgrExtMouseMode] instead.
const (
	EnableMouseSgrExt  = "\x1b[?1006h"
	DisableMouseSgrExt = "\x1b[?1006l"
	RequestMouseSgrExt = "\x1b[?1006$p"
)

// Alternate Screen Buffer is a mode that determines whether the alternate screen
// buffer is active.
//
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-The-Alternate-Screen-Buffer
const (
	AltScreenBufferMode = DECMode(1049)

	SetAltScreenBufferMode     = "\x1b[?1049h"
	ResetAltScreenBufferMode   = "\x1b[?1049l"
	RequestAltScreenBufferMode = "\x1b[?1049$p"
)

// Deprecated: use [SetAltScreenBufferMode], [ResetAltScreenBufferMode], and
// [RequestAltScreenBufferMode] instead.
const (
	EnableAltScreenBuffer  = "\x1b[?1049h"
	DisableAltScreenBuffer = "\x1b[?1049l"
	RequestAltScreenBuffer = "\x1b[?1049$p"
)

// Bracketed Paste Mode is a mode that determines whether pasted text is
// bracketed with escape sequences.
//
// See: https://cirw.in/blog/bracketed-paste
// See: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Bracketed-Paste-Mode
const (
	BracketedPasteMode = DECMode(2004)

	SetBracketedPasteMode     = "\x1b[?2004h"
	ResetBracketedPasteMode   = "\x1b[?2004l"
	RequestBracketedPasteMode = "\x1b[?2004$p"
)

// Deprecated: use [SetBracketedPasteMode], [ResetBracketedPasteMode], and
// [RequestBracketedPasteMode] instead.
const (
	EnableBracketedPaste  = "\x1b[?2004h"
	DisableBracketedPaste = "\x1b[?2004l"
	RequestBracketedPaste = "\x1b[?2004$p"
)

// Synchronized Output Mode is a mode that determines whether output is
// synchronized with the terminal.
//
// See: https://gist.github.com/christianparpart/d8a62cc1ab659194337d73e399004036
const (
	SynchronizedOutputMode = DECMode(2026)

	SetSynchronizedOutputMode     = "\x1b[?2026h"
	ResetSynchronizedOutputMode   = "\x1b[?2026l"
	RequestSynchronizedOutputMode = "\x1b[?2026$p"
)

// Deprecated: use [SynchronizedOutputMode], [SetSynchronizedOutputMode], and
// [ResetSynchronizedOutputMode], and [RequestSynchronizedOutputMode] instead.
const (
	SyncdOutputMode = DECMode(2026)

	EnableSyncdOutput  = "\x1b[?2026h"
	DisableSyncdOutput = "\x1b[?2026l"
	RequestSyncdOutput = "\x1b[?2026$p"
)

// Grapheme Clustering Mode is a mode that determines whether the terminal
// should look for grapheme clusters instead of single runes in the rendered
// text. This makes the terminal properly render combining characters such as
// emojis.
//
// See: https://github.com/contour-terminal/terminal-unicode-core
const (
	GraphemeClusteringMode = DECMode(2027)

	SetGraphemeClusteringMode     = "\x1b[?2027h"
	ResetGraphemeClusteringMode   = "\x1b[?2027l"
	RequestGraphemeClusteringMode = "\x1b[?2027$p"
)

// Deprecated: use [SetGraphemeClusteringMode], [ResetGraphemeClusteringMode], and
// [RequestGraphemeClusteringMode] instead.
const (
	EnableGraphemeClustering  = "\x1b[?2027h"
	DisableGraphemeClustering = "\x1b[?2027l"
	RequestGraphemeClustering = "\x1b[?2027$p"
)

// Win32Input is a mode that determines whether input is processed by the
// Win32 console and Conpty.
//
// See: https://github.com/microsoft/terminal/blob/main/doc/specs/%234999%20-%20Improved%20keyboard%20handling%20in%20Conpty.md
const (
	Win32InputMode = DECMode(9001)

	SetWin32InputMode     = "\x1b[?9001h"
	ResetWin32InputMode   = "\x1b[?9001l"
	RequestWin32InputMode = "\x1b[?9001$p"
)

// Deprecated: use [SetWin32InputMode], [ResetWin32InputMode], and
// [RequestWin32InputMode] instead.
const (
	EnableWin32Input  = "\x1b[?9001h"
	DisableWin32Input = "\x1b[?9001l"
	RequestWin32Input = "\x1b[?9001$p"
)
