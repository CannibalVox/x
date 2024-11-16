package ansi

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi/parser"
	"github.com/rivo/uniseg"
)

// State represents the state of the ANSI escape sequence parser used by
// [DecodeSequence].
type State = byte

// ANSI escape sequence states used by [DecodeSequence].
const (
	NormalState State = iota
	MarkerState
	ParamsState
	IntermedState
	EscapeState
	StringState
)

// DecodeSequence decodes the first ANSI escape sequence or a printable
// grapheme from the given data. It returns the sequence slice, the number of
// bytes read, the cell width for each sequence, and the new state.
//
// The cell width will always be 0 for control and escape sequences, 1 for
// ASCII printable characters, and the number of cells other Unicode characters
// occupy. It uses the uniseg package to calculate the width of Unicode
// graphemes and characters. This means it will always do grapheme clustering
// (mode 2027).
//
// Passing a non-nil [*Parser] as the last argument will allow the decoder to
// collect sequence parameters, data, and commands. The parser cmd will have
// the packed command value that contains intermediate and marker characters.
// In the case of a OSC sequence, the cmd will be the OSC command number. Use
// [Command] and [Parameter] types to unpack command intermediates and markers as well
// as parameters.
//
// Zero [Command] means the CSI, DCS, or ESC sequence is invalid. Moreover, checking the
// validity of other data sequences, OSC, DCS, etc, will require checking for
// the returned sequence terminator bytes such as ST (ESC \\) and BEL).
//
// We store the command byte in [Command] in the most significant byte, the
// marker byte in the next byte, and the intermediate byte in the least
// significant byte. This is done to avoid using a struct to store the command
// and its intermediates and markers. The command byte is always the least
// significant byte i.e. [Cmd & 0xff]. Use the [Command] type to unpack the
// command, intermediate, and marker bytes. Note that we only collect the last
// marker character and intermediate byte.
//
// The [p.Params] slice will contain the parameters of the sequence. Any
// sub-parameter will have the [parser.HasMoreFlag] set. Use the [Parameter] type
// to unpack the parameters.
//
// Example:
//
//	var state byte // the initial state is always zero [NormalState]
//	p := NewParser(32, 1024) // create a new parser with a 32 params buffer and 1024 data buffer (optional)
//	input := []byte("\x1b[31mHello, World!\x1b[0m")
//	for len(input) > 0 {
//		seq, width, n, newState := DecodeSequence(input, state, p)
//		log.Printf("seq: %q, width: %d", seq, width)
//		state = newState
//		input = input[n:]
//	}
func DecodeSequence[T string | []byte](b T, state byte, p *Parser) (seq T, width int, n int, newState byte) {
	for i := 0; i < len(b); i++ {
		c := b[i]

		switch state {
		case NormalState:
			switch c {
			case ESC:
				if p != nil {
					if len(p.params) > 0 {
						p.params[0] = parser.MissingParam
					}
					p.cmd = 0
					p.paramsLen = 0
					p.dataLen = 0
				}
				state = EscapeState
				continue
			case CSI, DCS:
				if p != nil {
					if len(p.params) > 0 {
						p.params[0] = parser.MissingParam
					}
					p.cmd = 0
					p.paramsLen = 0
					p.dataLen = 0
				}
				state = MarkerState
				continue
			case OSC, APC, SOS, PM:
				if p != nil {
					p.cmd = parser.MissingCommand
					p.dataLen = 0
				}
				state = StringState
				continue
			}

			if p != nil {
				p.dataLen = 0
				p.paramsLen = 0
				p.cmd = 0
			}
			if c > US && c < DEL {
				// ASCII printable characters
				return b[i : i+1], 1, 1, NormalState
			}

			if c <= US || c == DEL || c < 0xC0 {
				// C0 & C1 control characters & DEL
				return b[i : i+1], 0, 1, NormalState
			}

			if utf8.RuneStart(c) {
				seq, _, width, _ = FirstGraphemeCluster(b, -1)
				i += len(seq)
				return b[:i], width, i, NormalState
			}

			// Invalid UTF-8 sequence
			return b[:i], 0, i, NormalState
		case MarkerState:
			if c >= '<' && c <= '?' {
				if p != nil {
					// We only collect the last marker character.
					p.cmd &^= 0xff << parser.MarkerShift
					p.cmd |= int(c) << parser.MarkerShift
				}
				break
			}

			state = ParamsState
			fallthrough
		case ParamsState:
			if c >= '0' && c <= '9' {
				if p != nil {
					if p.params[p.paramsLen] == parser.MissingParam {
						p.params[p.paramsLen] = 0
					}

					p.params[p.paramsLen] *= 10
					p.params[p.paramsLen] += int(c - '0')
				}
				break
			}

			if c == ':' {
				if p != nil {
					p.params[p.paramsLen] |= parser.HasMoreFlag
				}
			}

			if c == ';' || c == ':' {
				if p != nil {
					p.paramsLen++
					if p.paramsLen < len(p.params) {
						p.params[p.paramsLen] = parser.MissingParam
					}
				}
				break
			}

			state = IntermedState
			fallthrough
		case IntermedState:
			if c >= ' ' && c <= '/' {
				if p != nil {
					p.cmd &^= 0xff << parser.IntermedShift
					p.cmd |= int(c) << parser.IntermedShift
				}
				break
			}

			if p != nil {
				// Increment the last parameter
				if p.paramsLen > 0 && p.paramsLen < len(p.params)-1 ||
					p.paramsLen == 0 && len(p.params) > 0 && p.params[0] != parser.MissingParam {
					p.paramsLen++
				}
			}

			if c >= '@' && c <= '~' {
				if p != nil {
					p.cmd &^= 0xff
					p.cmd |= int(c)
				}

				if HasDcsPrefix(b) {
					// Continue to collect DCS data
					if p != nil {
						p.dataLen = 0
					}
					state = StringState
					continue
				}

				return b[:i+1], 0, i + 1, NormalState
			}

			// Invalid CSI/DCS sequence
			return b[:i], 0, i, NormalState
		case EscapeState:
			switch c {
			case '[', 'P':
				if p != nil {
					if len(p.params) > 0 {
						p.params[0] = parser.MissingParam
					}
					p.paramsLen = 0
					p.cmd = 0
				}
				state = MarkerState
				continue
			case ']', 'X', '^', '_':
				if p != nil {
					p.cmd = parser.MissingCommand
					p.dataLen = 0
				}
				state = StringState
				continue
			}

			if c >= ' ' && c <= '/' {
				if p != nil {
					p.cmd &^= 0xff << parser.IntermedShift
					p.cmd |= int(c) << parser.IntermedShift
				}
				continue
			} else if c >= '0' && c <= '~' {
				if p != nil {
					p.cmd &^= 0xff
					p.cmd |= int(c)
				}
				return b[:i+1], 0, i + 1, NormalState
			}

			// Invalid escape sequence
			return b[:i], 0, i, NormalState
		case StringState:
			switch c {
			case BEL:
				if HasOscPrefix(b) {
					parseOscCmd(p)
					return b[:i+1], 0, i + 1, NormalState
				}
			case CAN, SUB:
				if HasOscPrefix(b) {
					// Ensure we parse the OSC command number
					parseOscCmd(p)
				}

				// Cancel the sequence
				return b[:i], 0, i, NormalState
			case ST:
				if HasOscPrefix(b) {
					// Ensure we parse the OSC command number
					parseOscCmd(p)
				}

				return b[:i+1], 0, i + 1, NormalState
			case ESC:
				if HasStPrefix(b[i:]) {
					if HasOscPrefix(b) {
						// Ensure we parse the OSC command number
						parseOscCmd(p)
					}

					// End of string 7-bit (ST)
					return b[:i+2], 0, i + 2, NormalState
				}

				// Otherwise, cancel the sequence
				return b[:i], 0, i, NormalState
			}

			if p != nil && p.dataLen < len(p.data) {
				p.data[p.dataLen] = c
				p.dataLen++

				// Parse the OSC command number
				if c == ';' && HasOscPrefix(b) {
					parseOscCmd(p)
				}
			}
		}
	}

	return b, 0, len(b), state
}

func parseOscCmd(p *Parser) {
	if p == nil || p.cmd != parser.MissingCommand {
		return
	}
	for j := 0; j < p.dataLen; j++ {
		d := p.data[j]
		if d < '0' || d > '9' {
			break
		}
		if p.cmd == parser.MissingCommand {
			p.cmd = 0
		}
		p.cmd *= 10
		p.cmd += int(d - '0')
	}
}

// Index returns the index of the first occurrence of the given byte slice in
// the data. It returns -1 if the byte slice is not found.
func Index[T string | []byte](data, b T) int {
	switch data := any(data).(type) {
	case string:
		return strings.Index(data, string(b))
	case []byte:
		return bytes.Index(data, []byte(b))
	}
	panic("unreachable")
}

// Equal returns true if the given byte slices are equal.
func Equal[T string | []byte](a, b T) bool {
	return string(a) == string(b)
}

// HasPrefix returns true if the given byte slice has prefix.
func HasPrefix[T string | []byte](b, prefix T) bool {
	return len(b) >= len(prefix) && Equal(b[0:len(prefix)], prefix)
}

// HasSuffix returns true if the given byte slice has suffix.
func HasSuffix[T string | []byte](b, suffix T) bool {
	return len(b) >= len(suffix) && Equal(b[len(b)-len(suffix):], suffix)
}

// HasCsiPrefix returns true if the given byte slice has a CSI prefix.
func HasCsiPrefix[T string | []byte](b T) bool {
	return (len(b) > 0 && b[0] == CSI) ||
		(len(b) > 1 && b[0] == ESC && b[1] == '[')
}

// HasOscPrefix returns true if the given byte slice has an OSC prefix.
func HasOscPrefix[T string | []byte](b T) bool {
	return (len(b) > 0 && b[0] == OSC) ||
		(len(b) > 1 && b[0] == ESC && b[1] == ']')
}

// HasApcPrefix returns true if the given byte slice has an APC prefix.
func HasApcPrefix[T string | []byte](b T) bool {
	return (len(b) > 0 && b[0] == APC) ||
		(len(b) > 1 && b[0] == ESC && b[1] == '_')
}

// HasDcsPrefix returns true if the given byte slice has a DCS prefix.
func HasDcsPrefix[T string | []byte](b T) bool {
	return (len(b) > 0 && b[0] == DCS) ||
		(len(b) > 1 && b[0] == ESC && b[1] == 'P')
}

// HasSosPrefix returns true if the given byte slice has a SOS prefix.
func HasSosPrefix[T string | []byte](b T) bool {
	return (len(b) > 0 && b[0] == SOS) ||
		(len(b) > 1 && b[0] == ESC && b[1] == 'X')
}

// HasPmPrefix returns true if the given byte slice has a PM prefix.
func HasPmPrefix[T string | []byte](b T) bool {
	return (len(b) > 0 && b[0] == PM) ||
		(len(b) > 1 && b[0] == ESC && b[1] == '^')
}

// HasStPrefix returns true if the given byte slice has a ST prefix.
func HasStPrefix[T string | []byte](b T) bool {
	return (len(b) > 0 && b[0] == ST) ||
		(len(b) > 1 && b[0] == ESC && b[1] == '\\')
}

// HasEscPrefix returns true if the given byte slice has an ESC prefix.
func HasEscPrefix[T string | []byte](b T) bool {
	return len(b) > 0 && b[0] == ESC
}

// FirstGraphemeCluster returns the first grapheme cluster in the given string or byte slice.
// This is a syntactic sugar function that wraps
// uniseg.FirstGraphemeClusterInString and uniseg.FirstGraphemeCluster.
func FirstGraphemeCluster[T string | []byte](b T, state int) (T, T, int, int) {
	switch b := any(b).(type) {
	case string:
		cluster, rest, width, newState := uniseg.FirstGraphemeClusterInString(b, state)
		return T(cluster), T(rest), width, newState
	case []byte:
		cluster, rest, width, newState := uniseg.FirstGraphemeCluster(b, state)
		return T(cluster), T(rest), width, newState
	}
	panic("unreachable")
}

// Command represents a sequence command. This is used to pack/unpack a sequence
// command with its intermediate and marker characters. Those are commonly
// found in CSI and DCS sequences.
type Command int

// Marker returns the unpacked marker byte of the CSI sequence.
// This is always gonna be one of the following '<' '=' '>' '?' and in the
// range of 0x3C-0x3F.
// Zero is returned if the sequence does not have a marker.
func (c Command) Marker() int {
	return parser.Marker(int(c))
}

// Intermediate returns the unpacked intermediate byte of the CSI sequence.
// An intermediate byte is in the range of 0x20-0x2F. This includes these
// characters from ' ', '!', '"', '#', '$', '%', '&', ”', '(', ')', '*', '+',
// ',', '-', '.', '/'.
// Zero is returned if the sequence does not have an intermediate byte.
func (c Command) Intermediate() int {
	return parser.Intermediate(int(c))
}

// Command returns the unpacked command byte of the CSI sequence.
func (c Command) Command() int {
	return parser.Command(int(c))
}

// Cmd returns a packed [Command] with the given command, marker, and
// intermediate.
// The first byte is the command, the next shift is the marker, and the next
// shift is the intermediate.
//
// Even though this function takes integers, it only uses the lower 8 bits of
// each integer.
func Cmd(marker, inter, cmd int) (c Command) {
	c = Command(cmd & parser.CommandMask)
	c |= Command(marker&parser.CommandMask) << parser.MarkerShift
	c |= Command(inter&parser.CommandMask) << parser.IntermedShift
	return
}

// Parameter represents a sequence parameter. Sequence parameters with
// sub-parameters are packed with the HasMoreFlag set. This is used to unpack
// the parameters from a CSI and DCS sequences.
type Parameter int

// Param returns the unpacked parameter at the given index.
// It returns the default value if the parameter is missing.
func (s Parameter) Param(def int) int {
	p := int(s) & parser.ParamMask
	if p == parser.MissingParam {
		return def
	}
	return p
}

// HasMore unpacks the HasMoreFlag from the parameter.
func (s Parameter) HasMore() bool {
	return s&parser.HasMoreFlag != 0
}

// Param returns a packed [Parameter] with the given parameter and whether this
// parameter has following sub-parameters.
func Param(p int, hasMore bool) (s Parameter) {
	s = Parameter(p & parser.ParamMask)
	if hasMore {
		s |= Parameter(parser.HasMoreFlag)
	}
	return
}
