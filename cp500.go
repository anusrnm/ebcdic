package ebcdic

import (
	"bytes"
	"io"
	//"fmt"
	//"os"
	// "flag"
	//"strconv"
	//"encoding/hex"
)

/*
func main() {
	ebcdicarr := []byte{'\x00', '\x01', '\x02', '\x03', '\x04', '\x05', '\x06',
 '\x07', '\x08', '\x09', '\x0A', '\x0B', '\x0C', '\x0D', '\x0E', '\x0F'}
	asciiarr := Ebc2asc(ebcdicarr)
	again2ebc := Tocp500(asciiarr)
	//fmt.Println("In ASCII")
	//fmt.Println(string(asciiarr))
	fmt.Println("ASCII ARRAY")
	fmt.Println(hex.Dump(asciiarr))
	fmt.Println("EBCDIC ARRAY AGAIN")
	fmt.Println(hex.Dump(again2ebc))
	if bytes.Equal(ebcdicarr, again2ebc) {
		fmt.Println("Equal")
	} else {
		fmt.Println("Not Equal")
	}
}
*/

const hextable = "0123456789abcdef"

// Encode encodes src into EncodedLen(len(src))
// bytes of dst. As a convenience, it returns the number
// of bytes written to dst, but this value is always EncodedLen(len(src)).
// Encode implements hexadecimal encoding.
func Encode(dst, src []byte) int {
	for i, v := range src {
		dst[i*2] = hextable[v>>4]
		dst[i*2+1] = hextable[v&0x0f]
	}

	return len(src) * 2
}

// Dump returns a string that contains a hex dump of the given data. The format
// of the hex dump matches the output of `hexdump -C` on the command line.
func Dump(data []byte) string {
	var buf bytes.Buffer
	dumper := Dumper(&buf)
	dumper.Write(data)
	dumper.Close()
	return buf.String()
}

// Dumper returns a WriteCloser that writes a hex dump of all written data to
// w. The format of the dump matches the output of `hexdump -C` on the command
// line.
func Dumper(w io.Writer) io.WriteCloser {
	return &dumper{w: w}
}

type dumper struct {
	w          io.Writer
	rightChars [18]byte
	buf        [14]byte
	used       int  // number of bytes in the current line
	n          uint // number of bytes, total
}

//ToString returns ebcdic string from byte array
func ToString(b []byte) string {
	var buf bytes.Buffer
	for _, v := range b {
		if Cp500StringDecodingTable[v] != '.' || v == 75 {
			buf.WriteByte(Cp500StringDecodingTable[v])
		}
	}
	return buf.String()
}

func ToChar(b byte) byte {
	return Cp500SafeDecodingTable[b]
}

func (h *dumper) Write(data []byte) (n int, err error) {
	// Output lines look like:
	// 00000010  2e 2f 30 31 32 33 34 35  36 37 38 39 3a 3b 3c 3d  |./0123456789:;<=|
	// ^ offset                          ^ extra space              ^ ASCII of line.
	for i := range data {
		if h.used == 0 {
			// At the beginning of a line we print the current
			// offset in hex.
			h.buf[0] = byte(h.n >> 24)
			h.buf[1] = byte(h.n >> 16)
			h.buf[2] = byte(h.n >> 8)
			h.buf[3] = byte(h.n)
			Encode(h.buf[4:], h.buf[:4])
			h.buf[12] = ' '
			h.buf[13] = ' '
			_, err = h.w.Write(h.buf[4:])
			if err != nil {
				return
			}
		}
		Encode(h.buf[:], data[i:i+1])
		h.buf[2] = ' '
		l := 3
		if h.used == 7 {
			// There's an additional space after the 8th byte.
			h.buf[3] = ' '
			l = 4
		} else if h.used == 15 {
			// At the end of the line there's an extra space and
			// the bar for the right column.
			h.buf[3] = ' '
			h.buf[4] = '|'
			l = 5
		}
		_, err = h.w.Write(h.buf[:l])
		if err != nil {
			return
		}
		n++
		h.rightChars[h.used] = ToChar(data[i])
		h.used++
		h.n++
		if h.used == 16 {
			h.rightChars[16] = '|'
			h.rightChars[17] = '\n'
			_, err = h.w.Write(h.rightChars[:])
			if err != nil {
				return
			}
			h.used = 0
		}
	}
	return
}

func (h *dumper) Close() (err error) {
	// See the comments in Write() for the details of this format.
	if h.used == 0 {
		return
	}
	h.buf[0] = ' '
	h.buf[1] = ' '
	h.buf[2] = ' '
	h.buf[3] = ' '
	h.buf[4] = '|'
	nBytes := h.used
	for h.used < 16 {
		l := 3
		if h.used == 7 {
			l = 4
		} else if h.used == 15 {
			l = 5
		}
		_, err = h.w.Write(h.buf[:l])
		if err != nil {
			return
		}
		h.used++
	}
	h.rightChars[nBytes] = '|'
	h.rightChars[nBytes+1] = '\n'
	_, err = h.w.Write(h.rightChars[:nBytes+2])
	return
}

//Tocp500 Converts byte array to ebcdic byte array
func Tocp500(asc []byte) []byte {
	ebc := make([]byte, len(asc))
	for key, value := range asc {
		ebc[key] = Cp500EncodingTable[value]
	}
	return ebc
}

//Cp500toSafeASCII Converts ebcdic array to safe utf-8
//Non-printable bytes are set to dot (.)
func Cp500toSafeASCII(ebc []byte) []byte {
	asc := make([]byte, len(ebc))
	for key, value := range ebc {
		asc[key] = Cp500SafeDecodingTable[value]
	}
	return asc
}

//Cp500toASCII Converts ebcdic array to utf-8
func Cp500toASCII(ebc []byte) []byte {
	asc := make([]byte, len(ebc))
	for key, value := range ebc {
		asc[key] = Cp500DecodingTable[value]
	}
	return asc
}

var Cp500SafeDecodingTable = []byte{
	'.',  // 0x00 -> NULL
	'.',  // 0x01 -> START OF HEADING
	'.',  // 0x02 -> START OF TEXT
	'.',  // 0x03 -> END OF TEXT
	'.',  // 0x04 -> CONTROL
	'.',  // 0x05 -> HORIZONTAL TABULATION
	'.',  // 0x06 -> CONTROL
	'.',  // 0x07 -> DELETE
	'.',  // 0x08 -> CONTROL
	'.',  // 0x09 -> CONTROL
	'.',  // 0x0A -> CONTROL
	'.',  // 0x0B -> VERTICAL TABULATION
	'.',  // 0x0C -> FORM FEED
	'.',  // 0x0D -> CARRIAGE RETURN
	'.',  // 0x0E -> SHIFT OUT
	'.',  // 0x0F -> SHIFT IN
	'.',  // 0x10 -> DATA LINK ESCAPE
	'.',  // 0x11 -> DEVICE CONTROL ONE
	'.',  // 0x12 -> DEVICE CONTROL TWO
	'.',  // 0x13 -> DEVICE CONTROL THREE
	'.',  // 0x14 -> CONTROL
	'.',  // 0x15 -> CONTROL
	'.',  // 0x16 -> BACKSPACE
	'.',  // 0x17 -> CONTROL
	'.',  // 0x18 -> CANCEL
	'.',  // 0x19 -> END OF MEDIUM
	'.',  // 0x1A -> CONTROL
	'.',  // 0x1B -> CONTROL
	'.',  // 0x1C -> FILE SEPARATOR
	'.',  // 0x1D -> GROUP SEPARATOR
	'.',  // 0x1E -> RECORD SEPARATOR
	'.',  // 0x1F -> UNIT SEPARATOR
	'.',  // 0x20 -> CONTROL
	'.',  // 0x21 -> CONTROL
	'.',  // 0x22 -> CONTROL
	'.',  // 0x23 -> CONTROL
	'.',  // 0x24 -> CONTROL
	'.',  // 0x25 -> LINE FEED
	'.',  // 0x26 -> END OF TRANSMISSION BLOCK
	'.',  // 0x27 -> ESCAPE
	'.',  // 0x28 -> CONTROL
	'.',  // 0x29 -> CONTROL
	'.',  // 0x2A -> CONTROL
	'.',  // 0x2B -> CONTROL
	'.',  // 0x2C -> CONTROL
	'.',  // 0x2D -> ENQUIRY
	'.',  // 0x2E -> ACKNOWLEDGE
	'.',  // 0x2F -> BELL
	'.',  // 0x30 -> CONTROL
	'.',  // 0x31 -> CONTROL
	'.',  // 0x32 -> SYNCHRONOUS IDLE
	'.',  // 0x33 -> CONTROL
	'.',  // 0x34 -> CONTROL
	'.',  // 0x35 -> CONTROL
	'.',  // 0x36 -> CONTROL
	'.',  // 0x37 -> END OF TRANSMISSION
	'.',  // 0x38 -> CONTROL
	'.',  // 0x39 -> CONTROL
	'.',  // 0x3A -> CONTROL
	'.',  // 0x3B -> CONTROL
	'.',  // 0x3C -> DEVICE CONTROL FOUR
	'.',  // 0x3D -> NEGATIVE ACKNOWLEDGE
	'.',  // 0x3E -> CONTROL
	'.',  // 0x3F -> SUBSTITUTE
	' ',  // 0x40 -> SPACE
	'.',  // 0x41 -> NO-BREAK SPACE
	'.',  // 0x42 -> LATIN SMALL LETTER A WITH CIRCUMFLEX
	'.',  // 0x43 -> LATIN SMALL LETTER A WITH DIAERESIS
	'.',  // 0x44 -> LATIN SMALL LETTER A WITH GRAVE
	'.',  // 0x45 -> LATIN SMALL LETTER A WITH ACUTE
	'.',  // 0x46 -> LATIN SMALL LETTER A WITH TILDE
	'.',  // 0x47 -> LATIN SMALL LETTER A WITH RING ABOVE
	'.',  // 0x48 -> LATIN SMALL LETTER C WITH CEDILLA
	'.',  // 0x49 -> LATIN SMALL LETTER N WITH TILDE
	'[',  // 0x4A -> LEFT SQUARE BRACKET
	'.',  // 0x4B -> FULL STOP
	'<',  // 0x4C -> LESS-THAN SIGN
	'(',  // 0x4D -> LEFT PARENTHESIS
	'+',  // 0x4E -> PLUS SIGN
	'!',  // 0x4F -> EXCLAMATION MARK
	'&',  // 0x50 -> AMPERSAND
	'.',  // 0x51 -> LATIN SMALL LETTER E WITH ACUTE
	'.',  // 0x52 -> LATIN SMALL LETTER E WITH CIRCUMFLEX
	'.',  // 0x53 -> LATIN SMALL LETTER E WITH DIAERESIS
	'.',  // 0x54 -> LATIN SMALL LETTER E WITH GRAVE
	'.',  // 0x55 -> LATIN SMALL LETTER I WITH ACUTE
	'.',  // 0x56 -> LATIN SMALL LETTER I WITH CIRCUMFLEX
	'.',  // 0x57 -> LATIN SMALL LETTER I WITH DIAERESIS
	'.',  // 0x58 -> LATIN SMALL LETTER I WITH GRAVE
	'.',  // 0x59 -> LATIN SMALL LETTER SHARP S (GERMAN)
	']',  // 0x5A -> RIGHT SQUARE BRACKET
	'$',  // 0x5B -> DOLLAR SIGN
	'*',  // 0x5C -> ASTERISK
	')',  // 0x5D -> RIGHT PARENTHESIS
	';',  // 0x5E -> SEMICOLON
	'^',  // 0x5F -> CIRCUMFLEX ACCENT
	'-',  // 0x60 -> HYPHEN-MINUS
	'/',  // 0x61 -> SOLIDUS
	'.',  // 0x62 -> LATIN CAPITAL LETTER A WITH CIRCUMFLEX
	'.',  // 0x63 -> LATIN CAPITAL LETTER A WITH DIAERESIS
	'.',  // 0x64 -> LATIN CAPITAL LETTER A WITH GRAVE
	'.',  // 0x65 -> LATIN CAPITAL LETTER A WITH ACUTE
	'.',  // 0x66 -> LATIN CAPITAL LETTER A WITH TILDE
	'.',  // 0x67 -> LATIN CAPITAL LETTER A WITH RING ABOVE
	'.',  // 0x68 -> LATIN CAPITAL LETTER C WITH CEDILLA
	'.',  // 0x69 -> LATIN CAPITAL LETTER N WITH TILDE
	'.',  // 0x6A -> BROKEN BAR
	',',  // 0x6B -> COMMA
	'%',  // 0x6C -> PERCENT SIGN
	'_',  // 0x6D -> LOW LINE
	'>',  // 0x6E -> GREATER-THAN SIGN
	'?',  // 0x6F -> QUESTION MARK
	'.',  // 0x70 -> LATIN SMALL LETTER O WITH STROKE
	'.',  // 0x71 -> LATIN CAPITAL LETTER E WITH ACUTE
	'.',  // 0x72 -> LATIN CAPITAL LETTER E WITH CIRCUMFLEX
	'.',  // 0x73 -> LATIN CAPITAL LETTER E WITH DIAERESIS
	'.',  // 0x74 -> LATIN CAPITAL LETTER E WITH GRAVE
	'.',  // 0x75 -> LATIN CAPITAL LETTER I WITH ACUTE
	'.',  // 0x76 -> LATIN CAPITAL LETTER I WITH CIRCUMFLEX
	'.',  // 0x77 -> LATIN CAPITAL LETTER I WITH DIAERESIS
	'.',  // 0x78 -> LATIN CAPITAL LETTER I WITH GRAVE
	'`',  // 0x79 -> GRAVE ACCENT
	':',  // 0x7A -> COLON
	'#',  // 0x7B -> NUMBER SIGN
	'@',  // 0x7C -> COMMERCIAL AT
	'\'', // 0x7D -> APOSTROPHE
	'=',  // 0x7E -> EQUALS SIGN
	'"',  // 0x7F -> QUOTATION MARK
	'.',  // 0x80 -> LATIN CAPITAL LETTER O WITH STROKE
	'a',  // 0x81 -> LATIN SMALL LETTER A
	'b',  // 0x82 -> LATIN SMALL LETTER B
	'c',  // 0x83 -> LATIN SMALL LETTER C
	'd',  // 0x84 -> LATIN SMALL LETTER D
	'e',  // 0x85 -> LATIN SMALL LETTER E
	'f',  // 0x86 -> LATIN SMALL LETTER F
	'g',  // 0x87 -> LATIN SMALL LETTER G
	'h',  // 0x88 -> LATIN SMALL LETTER H
	'i',  // 0x89 -> LATIN SMALL LETTER I
	'.',  // 0x8A -> LEFT-POINTING DOUBLE ANGLE QUOTATION MARK
	'.',  // 0x8B -> RIGHT-POINTING DOUBLE ANGLE QUOTATION MARK
	'.',  // 0x8C -> LATIN SMALL LETTER ETH (ICELANDIC)
	'.',  // 0x8D -> LATIN SMALL LETTER Y WITH ACUTE
	'.',  // 0x8E -> LATIN SMALL LETTER THORN (ICELANDIC)
	'.',  // 0x8F -> PLUS-MINUS SIGN
	'.',  // 0x90 -> DEGREE SIGN
	'j',  // 0x91 -> LATIN SMALL LETTER J
	'k',  // 0x92 -> LATIN SMALL LETTER K
	'l',  // 0x93 -> LATIN SMALL LETTER L
	'm',  // 0x94 -> LATIN SMALL LETTER M
	'n',  // 0x95 -> LATIN SMALL LETTER N
	'o',  // 0x96 -> LATIN SMALL LETTER O
	'p',  // 0x97 -> LATIN SMALL LETTER P
	'q',  // 0x98 -> LATIN SMALL LETTER Q
	'r',  // 0x99 -> LATIN SMALL LETTER R
	'.',  // 0x9A -> FEMININE ORDINAL INDICATOR
	'.',  // 0x9B -> MASCULINE ORDINAL INDICATOR
	'.',  // 0x9C -> LATIN SMALL LIGATURE AE
	'.',  // 0x9D -> CEDILLA
	'.',  // 0x9E -> LATIN CAPITAL LIGATURE AE
	'.',  // 0x9F -> CURRENCY SIGN
	'.',  // 0xA0 -> MICRO SIGN
	'~',  // 0xA1 -> TILDE
	's',  // 0xA2 -> LATIN SMALL LETTER S
	't',  // 0xA3 -> LATIN SMALL LETTER T
	'u',  // 0xA4 -> LATIN SMALL LETTER U
	'v',  // 0xA5 -> LATIN SMALL LETTER V
	'w',  // 0xA6 -> LATIN SMALL LETTER W
	'x',  // 0xA7 -> LATIN SMALL LETTER X
	'y',  // 0xA8 -> LATIN SMALL LETTER Y
	'z',  // 0xA9 -> LATIN SMALL LETTER Z
	'.',  // 0xAA -> INVERTED EXCLAMATION MARK
	'.',  // 0xAB -> INVERTED QUESTION MARK
	'.',  // 0xAC -> LATIN CAPITAL LETTER ETH (ICELANDIC)
	'.',  // 0xAD -> LATIN CAPITAL LETTER Y WITH ACUTE
	'.',  // 0xAE -> LATIN CAPITAL LETTER THORN (ICELANDIC)
	'.',  // 0xAF -> REGISTERED SIGN
	'.',  // 0xB0 -> CENT SIGN
	'.',  // 0xB1 -> POUND SIGN
	'.',  // 0xB2 -> YEN SIGN
	'.',  // 0xB3 -> MIDDLE DOT
	'.',  // 0xB4 -> COPYRIGHT SIGN
	'.',  // 0xB5 -> SECTION SIGN
	'.',  // 0xB6 -> PILCROW SIGN
	'.',  // 0xB7 -> VULGAR FRACTION ONE QUARTER
	'.',  // 0xB8 -> VULGAR FRACTION ONE HALF
	'.',  // 0xB9 -> VULGAR FRACTION THREE QUARTERS
	'.',  // 0xBA -> NOT SIGN
	'|',  // 0xBB -> VERTICAL LINE
	'.',  // 0xBC -> MACRON
	'.',  // 0xBD -> DIAERESIS
	'.',  // 0xBE -> ACUTE ACCENT
	'.',  // 0xBF -> MULTIPLICATION SIGN
	'{',  // 0xC0 -> LEFT CURLY BRACKET
	'A',  // 0xC1 -> LATIN CAPITAL LETTER A
	'B',  // 0xC2 -> LATIN CAPITAL LETTER B
	'C',  // 0xC3 -> LATIN CAPITAL LETTER C
	'D',  // 0xC4 -> LATIN CAPITAL LETTER D
	'E',  // 0xC5 -> LATIN CAPITAL LETTER E
	'F',  // 0xC6 -> LATIN CAPITAL LETTER F
	'G',  // 0xC7 -> LATIN CAPITAL LETTER G
	'H',  // 0xC8 -> LATIN CAPITAL LETTER H
	'I',  // 0xC9 -> LATIN CAPITAL LETTER I
	'.',  // 0xCA -> SOFT HYPHEN
	'.',  // 0xCB -> LATIN SMALL LETTER O WITH CIRCUMFLEX
	'.',  // 0xCC -> LATIN SMALL LETTER O WITH DIAERESIS
	'.',  // 0xCD -> LATIN SMALL LETTER O WITH GRAVE
	'.',  // 0xCE -> LATIN SMALL LETTER O WITH ACUTE
	'.',  // 0xCF -> LATIN SMALL LETTER O WITH TILDE
	'}',  // 0xD0 -> RIGHT CURLY BRACKET
	'J',  // 0xD1 -> LATIN CAPITAL LETTER J
	'K',  // 0xD2 -> LATIN CAPITAL LETTER K
	'L',  // 0xD3 -> LATIN CAPITAL LETTER L
	'M',  // 0xD4 -> LATIN CAPITAL LETTER M
	'N',  // 0xD5 -> LATIN CAPITAL LETTER N
	'O',  // 0xD6 -> LATIN CAPITAL LETTER O
	'P',  // 0xD7 -> LATIN CAPITAL LETTER P
	'Q',  // 0xD8 -> LATIN CAPITAL LETTER Q
	'R',  // 0xD9 -> LATIN CAPITAL LETTER R
	'.',  // 0xDA -> SUPERSCRIPT ONE
	'.',  // 0xDB -> LATIN SMALL LETTER U WITH CIRCUMFLEX
	'.',  // 0xDC -> LATIN SMALL LETTER U WITH DIAERESIS
	'.',  // 0xDD -> LATIN SMALL LETTER U WITH GRAVE
	'.',  // 0xDE -> LATIN SMALL LETTER U WITH ACUTE
	'.',  // 0xDF -> LATIN SMALL LETTER Y WITH DIAERESIS
	'\\', // 0xE0 -> REVERSE SOLIDUS
	'.',  // 0xE1 -> DIVISION SIGN
	'S',  // 0xE2 -> LATIN CAPITAL LETTER S
	'T',  // 0xE3 -> LATIN CAPITAL LETTER T
	'U',  // 0xE4 -> LATIN CAPITAL LETTER U
	'V',  // 0xE5 -> LATIN CAPITAL LETTER V
	'W',  // 0xE6 -> LATIN CAPITAL LETTER W
	'X',  // 0xE7 -> LATIN CAPITAL LETTER X
	'Y',  // 0xE8 -> LATIN CAPITAL LETTER Y
	'Z',  // 0xE9 -> LATIN CAPITAL LETTER Z
	'.',  // 0xEA -> SUPERSCRIPT TWO
	'.',  // 0xEB -> LATIN CAPITAL LETTER O WITH CIRCUMFLEX
	'.',  // 0xEC -> LATIN CAPITAL LETTER O WITH DIAERESIS
	'.',  // 0xED -> LATIN CAPITAL LETTER O WITH GRAVE
	'.',  // 0xEE -> LATIN CAPITAL LETTER O WITH ACUTE
	'.',  // 0xEF -> LATIN CAPITAL LETTER O WITH TILDE
	'0',  // 0xF0 -> DIGIT ZERO
	'1',  // 0xF1 -> DIGIT ONE
	'2',  // 0xF2 -> DIGIT TWO
	'3',  // 0xF3 -> DIGIT THREE
	'4',  // 0xF4 -> DIGIT FOUR
	'5',  // 0xF5 -> DIGIT FIVE
	'6',  // 0xF6 -> DIGIT SIX
	'7',  // 0xF7 -> DIGIT SEVEN
	'8',  // 0xF8 -> DIGIT EIGHT
	'9',  // 0xF9 -> DIGIT NINE
	'.',  // 0xFA -> SUPERSCRIPT THREE
	'.',  // 0xFB -> LATIN CAPITAL LETTER U WITH CIRCUMFLEX
	'.',  // 0xFC -> LATIN CAPITAL LETTER U WITH DIAERESIS
	'.',  // 0xFD -> LATIN CAPITAL LETTER U WITH GRAVE
	'.',  // 0xFE -> LATIN CAPITAL LETTER U WITH ACUTE
	'.',  // 0xFF -> CONTROL
}

var Cp500DecodingTable = []byte{
	'\x00', // 0x00 -> NULL
	'\x01', // 0x01 -> START OF HEADING
	'\x02', // 0x02 -> START OF TEXT
	'\x03', // 0x03 -> END OF TEXT
	'\x9c', // 0x04 -> CONTROL
	'\t',   // 0x05 -> HORIZONTAL TABULATION
	'\x86', // 0x06 -> CONTROL
	'\x7f', // 0x07 -> DELETE
	'\x97', // 0x08 -> CONTROL
	'\x8d', // 0x09 -> CONTROL
	'\x8e', // 0x0A -> CONTROL
	'\x0b', // 0x0B -> VERTICAL TABULATION
	'\x0c', // 0x0C -> FORM FEED
	'\r',   // 0x0D -> CARRIAGE RETURN
	'\x0e', // 0x0E -> SHIFT OUT
	'\x0f', // 0x0F -> SHIFT IN
	'\x10', // 0x10 -> DATA LINK ESCAPE
	'\x11', // 0x11 -> DEVICE CONTROL ONE
	'\x12', // 0x12 -> DEVICE CONTROL TWO
	'\x13', // 0x13 -> DEVICE CONTROL THREE
	'\x9d', // 0x14 -> CONTROL
	'\x85', // 0x15 -> CONTROL
	'\x08', // 0x16 -> BACKSPACE
	'\x87', // 0x17 -> CONTROL
	'\x18', // 0x18 -> CANCEL
	'\x19', // 0x19 -> END OF MEDIUM
	'\x92', // 0x1A -> CONTROL
	'\x8f', // 0x1B -> CONTROL
	'\x1c', // 0x1C -> FILE SEPARATOR
	'\x1d', // 0x1D -> GROUP SEPARATOR
	'\x1e', // 0x1E -> RECORD SEPARATOR
	'\x1f', // 0x1F -> UNIT SEPARATOR
	'\x80', // 0x20 -> CONTROL
	'\x81', // 0x21 -> CONTROL
	'\x82', // 0x22 -> CONTROL
	'\x83', // 0x23 -> CONTROL
	'\x84', // 0x24 -> CONTROL
	'\n',   // 0x25 -> LINE FEED
	'\x17', // 0x26 -> END OF TRANSMISSION BLOCK
	'\x1b', // 0x27 -> ESCAPE
	'\x88', // 0x28 -> CONTROL
	'\x89', // 0x29 -> CONTROL
	'\x8a', // 0x2A -> CONTROL
	'\x8b', // 0x2B -> CONTROL
	'\x8c', // 0x2C -> CONTROL
	'\x05', // 0x2D -> ENQUIRY
	'\x06', // 0x2E -> ACKNOWLEDGE
	'\x07', // 0x2F -> BELL
	'\x90', // 0x30 -> CONTROL
	'\x91', // 0x31 -> CONTROL
	'\x16', // 0x32 -> SYNCHRONOUS IDLE
	'\x93', // 0x33 -> CONTROL
	'\x94', // 0x34 -> CONTROL
	'\x95', // 0x35 -> CONTROL
	'\x96', // 0x36 -> CONTROL
	'\x04', // 0x37 -> END OF TRANSMISSION
	'\x98', // 0x38 -> CONTROL
	'\x99', // 0x39 -> CONTROL
	'\x9a', // 0x3A -> CONTROL
	'\x9b', // 0x3B -> CONTROL
	'\x14', // 0x3C -> DEVICE CONTROL FOUR
	'\x15', // 0x3D -> NEGATIVE ACKNOWLEDGE
	'\x9e', // 0x3E -> CONTROL
	'\x1a', // 0x3F -> SUBSTITUTE
	' ',    // 0x40 -> SPACE
	'\xa0', // 0x41 -> NO-BREAK SPACE
	'\xe2', // 0x42 -> LATIN SMALL LETTER A WITH CIRCUMFLEX
	'\xe4', // 0x43 -> LATIN SMALL LETTER A WITH DIAERESIS
	'\xe0', // 0x44 -> LATIN SMALL LETTER A WITH GRAVE
	'\xe1', // 0x45 -> LATIN SMALL LETTER A WITH ACUTE
	'\xe3', // 0x46 -> LATIN SMALL LETTER A WITH TILDE
	'\xe5', // 0x47 -> LATIN SMALL LETTER A WITH RING ABOVE
	'\xe7', // 0x48 -> LATIN SMALL LETTER C WITH CEDILLA
	'\xf1', // 0x49 -> LATIN SMALL LETTER N WITH TILDE
	'[',    // 0x4A -> LEFT SQUARE BRACKET
	'.',    // 0x4B -> FULL STOP
	'<',    // 0x4C -> LESS-THAN SIGN
	'(',    // 0x4D -> LEFT PARENTHESIS
	'+',    // 0x4E -> PLUS SIGN
	'!',    // 0x4F -> EXCLAMATION MARK
	'&',    // 0x50 -> AMPERSAND
	'\xe9', // 0x51 -> LATIN SMALL LETTER E WITH ACUTE
	'\xea', // 0x52 -> LATIN SMALL LETTER E WITH CIRCUMFLEX
	'\xeb', // 0x53 -> LATIN SMALL LETTER E WITH DIAERESIS
	'\xe8', // 0x54 -> LATIN SMALL LETTER E WITH GRAVE
	'\xed', // 0x55 -> LATIN SMALL LETTER I WITH ACUTE
	'\xee', // 0x56 -> LATIN SMALL LETTER I WITH CIRCUMFLEX
	'\xef', // 0x57 -> LATIN SMALL LETTER I WITH DIAERESIS
	'\xec', // 0x58 -> LATIN SMALL LETTER I WITH GRAVE
	'\xdf', // 0x59 -> LATIN SMALL LETTER SHARP S (GERMAN)
	']',    // 0x5A -> RIGHT SQUARE BRACKET
	'$',    // 0x5B -> DOLLAR SIGN
	'*',    // 0x5C -> ASTERISK
	')',    // 0x5D -> RIGHT PARENTHESIS
	';',    // 0x5E -> SEMICOLON
	'^',    // 0x5F -> CIRCUMFLEX ACCENT
	'-',    // 0x60 -> HYPHEN-MINUS
	'/',    // 0x61 -> SOLIDUS
	'\xc2', // 0x62 -> LATIN CAPITAL LETTER A WITH CIRCUMFLEX
	'\xc4', // 0x63 -> LATIN CAPITAL LETTER A WITH DIAERESIS
	'\xc0', // 0x64 -> LATIN CAPITAL LETTER A WITH GRAVE
	'\xc1', // 0x65 -> LATIN CAPITAL LETTER A WITH ACUTE
	'\xc3', // 0x66 -> LATIN CAPITAL LETTER A WITH TILDE
	'\xc5', // 0x67 -> LATIN CAPITAL LETTER A WITH RING ABOVE
	'\xc7', // 0x68 -> LATIN CAPITAL LETTER C WITH CEDILLA
	'\xd1', // 0x69 -> LATIN CAPITAL LETTER N WITH TILDE
	'\xa6', // 0x6A -> BROKEN BAR
	',',    // 0x6B -> COMMA
	'%',    // 0x6C -> PERCENT SIGN
	'_',    // 0x6D -> LOW LINE
	'>',    // 0x6E -> GREATER-THAN SIGN
	'?',    // 0x6F -> QUESTION MARK
	'\xf8', // 0x70 -> LATIN SMALL LETTER O WITH STROKE
	'\xc9', // 0x71 -> LATIN CAPITAL LETTER E WITH ACUTE
	'\xca', // 0x72 -> LATIN CAPITAL LETTER E WITH CIRCUMFLEX
	'\xcb', // 0x73 -> LATIN CAPITAL LETTER E WITH DIAERESIS
	'\xc8', // 0x74 -> LATIN CAPITAL LETTER E WITH GRAVE
	'\xcd', // 0x75 -> LATIN CAPITAL LETTER I WITH ACUTE
	'\xce', // 0x76 -> LATIN CAPITAL LETTER I WITH CIRCUMFLEX
	'\xcf', // 0x77 -> LATIN CAPITAL LETTER I WITH DIAERESIS
	'\xcc', // 0x78 -> LATIN CAPITAL LETTER I WITH GRAVE
	'`',    // 0x79 -> GRAVE ACCENT
	':',    // 0x7A -> COLON
	'#',    // 0x7B -> NUMBER SIGN
	'@',    // 0x7C -> COMMERCIAL AT
	'\'',   // 0x7D -> APOSTROPHE
	'=',    // 0x7E -> EQUALS SIGN
	'"',    // 0x7F -> QUOTATION MARK
	'\xd8', // 0x80 -> LATIN CAPITAL LETTER O WITH STROKE
	'a',    // 0x81 -> LATIN SMALL LETTER A
	'b',    // 0x82 -> LATIN SMALL LETTER B
	'c',    // 0x83 -> LATIN SMALL LETTER C
	'd',    // 0x84 -> LATIN SMALL LETTER D
	'e',    // 0x85 -> LATIN SMALL LETTER E
	'f',    // 0x86 -> LATIN SMALL LETTER F
	'g',    // 0x87 -> LATIN SMALL LETTER G
	'h',    // 0x88 -> LATIN SMALL LETTER H
	'i',    // 0x89 -> LATIN SMALL LETTER I
	'\xab', // 0x8A -> LEFT-POINTING DOUBLE ANGLE QUOTATION MARK
	'\xbb', // 0x8B -> RIGHT-POINTING DOUBLE ANGLE QUOTATION MARK
	'\xf0', // 0x8C -> LATIN SMALL LETTER ETH (ICELANDIC)
	'\xfd', // 0x8D -> LATIN SMALL LETTER Y WITH ACUTE
	'\xfe', // 0x8E -> LATIN SMALL LETTER THORN (ICELANDIC)
	'\xb1', // 0x8F -> PLUS-MINUS SIGN
	'\xb0', // 0x90 -> DEGREE SIGN
	'j',    // 0x91 -> LATIN SMALL LETTER J
	'k',    // 0x92 -> LATIN SMALL LETTER K
	'l',    // 0x93 -> LATIN SMALL LETTER L
	'm',    // 0x94 -> LATIN SMALL LETTER M
	'n',    // 0x95 -> LATIN SMALL LETTER N
	'o',    // 0x96 -> LATIN SMALL LETTER O
	'p',    // 0x97 -> LATIN SMALL LETTER P
	'q',    // 0x98 -> LATIN SMALL LETTER Q
	'r',    // 0x99 -> LATIN SMALL LETTER R
	'\xaa', // 0x9A -> FEMININE ORDINAL INDICATOR
	'\xba', // 0x9B -> MASCULINE ORDINAL INDICATOR
	'\xe6', // 0x9C -> LATIN SMALL LIGATURE AE
	'\xb8', // 0x9D -> CEDILLA
	'\xc6', // 0x9E -> LATIN CAPITAL LIGATURE AE
	'\xa4', // 0x9F -> CURRENCY SIGN
	'\xb5', // 0xA0 -> MICRO SIGN
	'~',    // 0xA1 -> TILDE
	's',    // 0xA2 -> LATIN SMALL LETTER S
	't',    // 0xA3 -> LATIN SMALL LETTER T
	'u',    // 0xA4 -> LATIN SMALL LETTER U
	'v',    // 0xA5 -> LATIN SMALL LETTER V
	'w',    // 0xA6 -> LATIN SMALL LETTER W
	'x',    // 0xA7 -> LATIN SMALL LETTER X
	'y',    // 0xA8 -> LATIN SMALL LETTER Y
	'z',    // 0xA9 -> LATIN SMALL LETTER Z
	'\xa1', // 0xAA -> INVERTED EXCLAMATION MARK
	'\xbf', // 0xAB -> INVERTED QUESTION MARK
	'\xd0', // 0xAC -> LATIN CAPITAL LETTER ETH (ICELANDIC)
	'\xdd', // 0xAD -> LATIN CAPITAL LETTER Y WITH ACUTE
	'\xde', // 0xAE -> LATIN CAPITAL LETTER THORN (ICELANDIC)
	'\xae', // 0xAF -> REGISTERED SIGN
	'\xa2', // 0xB0 -> CENT SIGN
	'\xa3', // 0xB1 -> POUND SIGN
	'\xa5', // 0xB2 -> YEN SIGN
	'\xb7', // 0xB3 -> MIDDLE DOT
	'\xa9', // 0xB4 -> COPYRIGHT SIGN
	'\xa7', // 0xB5 -> SECTION SIGN
	'\xb6', // 0xB6 -> PILCROW SIGN
	'\xbc', // 0xB7 -> VULGAR FRACTION ONE QUARTER
	'\xbd', // 0xB8 -> VULGAR FRACTION ONE HALF
	'\xbe', // 0xB9 -> VULGAR FRACTION THREE QUARTERS
	'\xac', // 0xBA -> NOT SIGN
	'|',    // 0xBB -> VERTICAL LINE
	'\xaf', // 0xBC -> MACRON
	'\xa8', // 0xBD -> DIAERESIS
	'\xb4', // 0xBE -> ACUTE ACCENT
	'\xd7', // 0xBF -> MULTIPLICATION SIGN
	'{',    // 0xC0 -> LEFT CURLY BRACKET
	'A',    // 0xC1 -> LATIN CAPITAL LETTER A
	'B',    // 0xC2 -> LATIN CAPITAL LETTER B
	'C',    // 0xC3 -> LATIN CAPITAL LETTER C
	'D',    // 0xC4 -> LATIN CAPITAL LETTER D
	'E',    // 0xC5 -> LATIN CAPITAL LETTER E
	'F',    // 0xC6 -> LATIN CAPITAL LETTER F
	'G',    // 0xC7 -> LATIN CAPITAL LETTER G
	'H',    // 0xC8 -> LATIN CAPITAL LETTER H
	'I',    // 0xC9 -> LATIN CAPITAL LETTER I
	'\xad', // 0xCA -> SOFT HYPHEN
	'\xf4', // 0xCB -> LATIN SMALL LETTER O WITH CIRCUMFLEX
	'\xf6', // 0xCC -> LATIN SMALL LETTER O WITH DIAERESIS
	'\xf2', // 0xCD -> LATIN SMALL LETTER O WITH GRAVE
	'\xf3', // 0xCE -> LATIN SMALL LETTER O WITH ACUTE
	'\xf5', // 0xCF -> LATIN SMALL LETTER O WITH TILDE
	'}',    // 0xD0 -> RIGHT CURLY BRACKET
	'J',    // 0xD1 -> LATIN CAPITAL LETTER J
	'K',    // 0xD2 -> LATIN CAPITAL LETTER K
	'L',    // 0xD3 -> LATIN CAPITAL LETTER L
	'M',    // 0xD4 -> LATIN CAPITAL LETTER M
	'N',    // 0xD5 -> LATIN CAPITAL LETTER N
	'O',    // 0xD6 -> LATIN CAPITAL LETTER O
	'P',    // 0xD7 -> LATIN CAPITAL LETTER P
	'Q',    // 0xD8 -> LATIN CAPITAL LETTER Q
	'R',    // 0xD9 -> LATIN CAPITAL LETTER R
	'\xb9', // 0xDA -> SUPERSCRIPT ONE
	'\xfb', // 0xDB -> LATIN SMALL LETTER U WITH CIRCUMFLEX
	'\xfc', // 0xDC -> LATIN SMALL LETTER U WITH DIAERESIS
	'\xf9', // 0xDD -> LATIN SMALL LETTER U WITH GRAVE
	'\xfa', // 0xDE -> LATIN SMALL LETTER U WITH ACUTE
	'\xff', // 0xDF -> LATIN SMALL LETTER Y WITH DIAERESIS
	'\\',   // 0xE0 -> REVERSE SOLIDUS
	'\xf7', // 0xE1 -> DIVISION SIGN
	'S',    // 0xE2 -> LATIN CAPITAL LETTER S
	'T',    // 0xE3 -> LATIN CAPITAL LETTER T
	'U',    // 0xE4 -> LATIN CAPITAL LETTER U
	'V',    // 0xE5 -> LATIN CAPITAL LETTER V
	'W',    // 0xE6 -> LATIN CAPITAL LETTER W
	'X',    // 0xE7 -> LATIN CAPITAL LETTER X
	'Y',    // 0xE8 -> LATIN CAPITAL LETTER Y
	'Z',    // 0xE9 -> LATIN CAPITAL LETTER Z
	'\xb2', // 0xEA -> SUPERSCRIPT TWO
	'\xd4', // 0xEB -> LATIN CAPITAL LETTER O WITH CIRCUMFLEX
	'\xd6', // 0xEC -> LATIN CAPITAL LETTER O WITH DIAERESIS
	'\xd2', // 0xED -> LATIN CAPITAL LETTER O WITH GRAVE
	'\xd3', // 0xEE -> LATIN CAPITAL LETTER O WITH ACUTE
	'\xd5', // 0xEF -> LATIN CAPITAL LETTER O WITH TILDE
	'0',    // 0xF0 -> DIGIT ZERO
	'1',    // 0xF1 -> DIGIT ONE
	'2',    // 0xF2 -> DIGIT TWO
	'3',    // 0xF3 -> DIGIT THREE
	'4',    // 0xF4 -> DIGIT FOUR
	'5',    // 0xF5 -> DIGIT FIVE
	'6',    // 0xF6 -> DIGIT SIX
	'7',    // 0xF7 -> DIGIT SEVEN
	'8',    // 0xF8 -> DIGIT EIGHT
	'9',    // 0xF9 -> DIGIT NINE
	'\xb3', // 0xFA -> SUPERSCRIPT THREE
	'\xdb', // 0xFB -> LATIN CAPITAL LETTER U WITH CIRCUMFLEX
	'\xdc', // 0xFC -> LATIN CAPITAL LETTER U WITH DIAERESIS
	'\xd9', // 0xFD -> LATIN CAPITAL LETTER U WITH GRAVE
	'\xda', // 0xFE -> LATIN CAPITAL LETTER U WITH ACUTE
	'\x9f', // 0xFF -> CONTROL
}

var Cp500EncodingTable = []byte{
	'\x00', // 0x00  ->
	'\x01', // 0x01  ->
	'\x02', // 0x02  ->
	'\x03', // 0x03  ->
	'\x37', // 0x04  ->
	'\x2D', // 0x05  ->
	'\x2E', // 0x06  ->
	'\x2F', // 0x07  ->
	'\x16', // 0x08  ->
	'\x05', // 0x09  ->
	'\x25', // 0x0A  ->
	'\x0B', // 0x0B  ->
	'\x0C', // 0x0C  ->
	'\x0D', // 0x0D  ->
	'\x0E', // 0x0E  ->
	'\x0F', // 0x0F  ->
	'\x10', // 0x10  ->
	'\x11', // 0x11  ->
	'\x12', // 0x12  ->
	'\x13', // 0x13  ->
	'\x3c', // 0x14  ->
	'\x3d', // 0x15  ->
	'\x32', // 0x16  ->
	'\x26', // 0x17  ->
	'\x18', // 0x18  ->
	'\x19', // 0x19  ->
	'\x3F', // 0x1A  ->
	'\x27', // 0x1B  ->
	'\x1C', // 0x1C  ->
	'\x1D', // 0x1D  ->
	'\x1E', // 0x1E  ->
	'\x1F', // 0x1F  ->
	'\x40', // 0x20  ->
	'\x4F', // 0x21  ->
	'\x7F', // 0x22  ->
	'\x7B', // 0x23  ->
	'\x5B', // 0x24  ->
	'\x6C', // 0x25  ->
	'\x50', // 0x26  ->
	'\x7D', // 0x27  ->
	'\x4D', // 0x28  ->
	'\x5D', // 0x29  ->
	'\x5C', // 0x2A  ->
	'\x4E', // 0x2B  ->
	'\x6B', // 0x2C  ->
	'\x60', // 0x2D  ->
	'\x4B', // 0x2E  ->
	'\x61', // 0x2F  ->
	'\xF0', // 0x30  ->
	'\xF1', // 0x31  ->
	'\xF2', // 0x32  ->
	'\xF3', // 0x33  ->
	'\xF4', // 0x34  ->
	'\xF5', // 0x35  ->
	'\xF6', // 0x36  ->
	'\xF7', // 0x37  ->
	'\xF8', // 0x38  ->
	'\xF9', // 0x39  ->
	'\x7A', // 0x3A  ->
	'\x5E', // 0x3B  ->
	'\x4C', // 0x3C  ->
	'\x7E', // 0x3D  ->
	'\x6E', // 0x3E  ->
	'\x6F', // 0x3F  ->
	'\x7C', // 0x40  ->
	'\xC1', // 0x41  ->
	'\xC2', // 0x42  ->
	'\xC3', // 0x43  ->
	'\xC4', // 0x44  ->
	'\xC5', // 0x45  ->
	'\xC6', // 0x46  ->
	'\xC7', // 0x47  ->
	'\xC8', // 0x48  ->
	'\xC9', // 0x49  ->
	'\xD1', // 0x4A  ->
	'\xD2', // 0x4B  ->
	'\xD3', // 0x4C  ->
	'\xD4', // 0x4D  ->
	'\xD5', // 0x4E  ->
	'\xD6', // 0x4F  ->
	'\xD7', // 0x50  ->
	'\xD8', // 0x51  ->
	'\xD9', // 0x52  ->
	'\xE2', // 0x53  ->
	'\xE3', // 0x54  ->
	'\xE4', // 0x55  ->
	'\xE5', // 0x56  ->
	'\xE6', // 0x57  ->
	'\xE7', // 0x58  ->
	'\xE8', // 0x59  ->
	'\xE9', // 0x5A  ->
	'\x4A', // 0x5B  ->
	'\xE0', // 0x5C  ->
	'\x5A', // 0x5D  ->
	'\x5F', // 0x5E  ->
	'\x6D', // 0x5F  ->
	'\x79', // 0x60  ->
	'\x81', // 0x61  ->
	'\x82', // 0x62  ->
	'\x83', // 0x63  ->
	'\x84', // 0x64  ->
	'\x85', // 0x65  ->
	'\x86', // 0x66  ->
	'\x87', // 0x67  ->
	'\x88', // 0x68  ->
	'\x89', // 0x69  ->
	'\x91', // 0x6A  ->
	'\x92', // 0x6B  ->
	'\x93', // 0x6C  ->
	'\x94', // 0x6D  ->
	'\x95', // 0x6E  ->
	'\x96', // 0x6F  ->
	'\x97', // 0x70  ->
	'\x98', // 0x71  ->
	'\x99', // 0x72  ->
	'\xA2', // 0x73  ->
	'\xA3', // 0x74  ->
	'\xA4', // 0x75  ->
	'\xA5', // 0x76  ->
	'\xA6', // 0x77  ->
	'\xA7', // 0x78  ->
	'\xA8', // 0x79  ->
	'\xA9', // 0x7A  ->
	'\xC0', // 0x7B  ->
	'\xBB', // 0x7C  ->
	'\xD0', // 0x7D  ->
	'\xA1', // 0x7E  ->
	'\x07', // 0x7F  ->
	'\x20', // 0x80  ->
	'\x21', // 0x81  ->
	'\x22', // 0x82  ->
	'\x23', // 0x83  ->
	'\x24', // 0x84  ->
	'\x15', // 0x85  ->
	'\x06', // 0x86  ->
	'\x17', // 0x87  ->
	'\x28', // 0x88  ->
	'\x29', // 0x89  ->
	'\x2A', // 0x8A  ->
	'\x2B', // 0x8B  ->
	'\x2C', // 0x8C  ->
	'\x09', // 0x8D  ->
	'\x0A', // 0x8E  ->
	'\x1B', // 0x8F  ->
	'\x30', // 0x90  ->
	'\x31', // 0x91  ->
	'\x1A', // 0x92  ->
	'\x33', // 0x93  ->
	'\x34', // 0x94  ->
	'\x35', // 0x95  ->
	'\x36', // 0x96  ->
	'\x08', // 0x97  ->
	'\x38', // 0x98  ->
	'\x39', // 0x99  ->
	'\x3A', // 0x9A  ->
	'\x3B', // 0x9B  ->
	'\x04', // 0x9C  ->
	'\x14', // 0x9D  ->
	'\x3E', // 0x9E  ->
	'\xFF', // 0x9F  ->
	'\x41', // 0xA0  ->
	'\xAA', // 0xA1  ->
	'\xB0', // 0xA2  ->
	'\xB1', // 0xA3  ->
	'\x9F', // 0xA4  ->
	'\xB2', // 0xA5  ->
	'\x6A', // 0xA6  ->
	'\xB5', // 0xA7  ->
	'\xBD', // 0xA8  ->
	'\xB4', // 0xA9  ->
	'\x9A', // 0xAA  ->
	'\x8A', // 0xAB  ->
	'\xBA', // 0xAC  ->
	'\xCA', // 0xAD  ->
	'\xAF', // 0xAE  ->
	'\xBC', // 0xAF  ->
	'\x90', // 0xB0  ->
	'\x8F', // 0xB1  ->
	'\xEA', // 0xB2  ->
	'\xFA', // 0xB3  ->
	'\xBE', // 0xB4  ->
	'\xA0', // 0xB5  ->
	'\xB6', // 0xB6  ->
	'\xB3', // 0xB7  ->
	'\x9D', // 0xB8  ->
	'\xDA', // 0xB9  ->
	'\x9B', // 0xBA  ->
	'\x8B', // 0xBB  ->
	'\xB7', // 0xBC  ->
	'\xB8', // 0xBD  ->
	'\xB9', // 0xBE  ->
	'\xAB', // 0xBF  ->
	'\x64', // 0xC0  ->
	'\x65', // 0xC1  ->
	'\x62', // 0xC2  ->
	'\x66', // 0xC3  ->
	'\x63', // 0xC4  ->
	'\x67', // 0xC5  ->
	'\x9E', // 0xC6  ->
	'\x68', // 0xC7  ->
	'\x74', // 0xC8  ->
	'\x71', // 0xC9  ->
	'\x72', // 0xCA  ->
	'\x73', // 0xCB  ->
	'\x78', // 0xCC  ->
	'\x75', // 0xCD  ->
	'\x76', // 0xCE  ->
	'\x77', // 0xCF  ->
	'\xAC', // 0xD0  ->
	'\x69', // 0xD1  ->
	'\xED', // 0xD2  ->
	'\xEE', // 0xD3  ->
	'\xEB', // 0xD4  ->
	'\xEF', // 0xD5  ->
	'\xEC', // 0xD6  ->
	'\xBF', // 0xD7  ->
	'\x80', // 0xD8  ->
	'\xFD', // 0xD9  ->
	'\xFE', // 0xDA  ->
	'\xFB', // 0xDB  ->
	'\xFC', // 0xDC  ->
	'\xAD', // 0xDD  ->
	'\xAE', // 0xDE  ->
	'\x59', // 0xDF  ->
	'\x44', // 0xE0  ->
	'\x45', // 0xE1  ->
	'\x42', // 0xE2  ->
	'\x46', // 0xE3  ->
	'\x43', // 0xE4  ->
	'\x47', // 0xE5  ->
	'\x9C', // 0xE6  ->
	'\x48', // 0xE7  ->
	'\x54', // 0xE8  ->
	'\x51', // 0xE9  ->
	'\x52', // 0xEA  ->
	'\x53', // 0xEB  ->
	'\x58', // 0xEC  ->
	'\x55', // 0xED  ->
	'\x56', // 0xEE  ->
	'\x57', // 0xEF  ->
	'\x8C', // 0xF0  ->
	'\x49', // 0xF1  ->
	'\xCD', // 0xF2  ->
	'\xCE', // 0xF3  ->
	'\xCB', // 0xF4  ->
	'\xCF', // 0xF5  ->
	'\xCC', // 0xF6  ->
	'\xE1', // 0xF7  ->
	'\x70', // 0xF8  ->
	'\xDD', // 0xF9  ->
	'\xDE', // 0xFA  ->
	'\xDB', // 0xFB  ->
	'\xDC', // 0xFC  ->
	'\x8D', // 0xFD  ->
	'\x8E', // 0xFE  ->
	'\xDF', // 0xFF  ->
}

var Cp500StringDecodingTable = []byte{
	'.',    // 0x00 -> NULL
	'.',    // 0x01 -> START OF HEADING
	'.',    // 0x02 -> START OF TEXT
	'.',    // 0x03 -> END OF TEXT
	'.',    // 0x04 -> CONTROL
	'\t',   // 0x05 -> HORIZONTAL TABULATION
	'.',    // 0x06 -> CONTROL
	'.',    // 0x07 -> DELETE
	'.',    // 0x08 -> CONTROL
	'.',    // 0x09 -> CONTROL
	'.',    // 0x0A -> CONTROL
	'\x0b', // 0x0B -> VERTICAL TABULATION
	'\x0c', // 0x0C -> FORM FEED
	'\r',   // 0x0D -> CARRIAGE RETURN
	'.',    // 0x0E -> SHIFT OUT
	'.',    // 0x0F -> SHIFT IN
	'.',    // 0x10 -> DATA LINK ESCAPE
	'.',    // 0x11 -> DEVICE CONTROL ONE
	'.',    // 0x12 -> DEVICE CONTROL TWO
	'.',    // 0x13 -> DEVICE CONTROL THREE
	'.',    // 0x14 -> CONTROL
	'.',    // 0x15 -> CONTROL
	'.',    // 0x16 -> BACKSPACE
	'.',    // 0x17 -> CONTROL
	'.',    // 0x18 -> CANCEL
	'.',    // 0x19 -> END OF MEDIUM
	'.',    // 0x1A -> CONTROL
	'.',    // 0x1B -> CONTROL
	'.',    // 0x1C -> FILE SEPARATOR
	'.',    // 0x1D -> GROUP SEPARATOR
	'.',    // 0x1E -> RECORD SEPARATOR
	'.',    // 0x1F -> UNIT SEPARATOR
	'.',    // 0x20 -> CONTROL
	'.',    // 0x21 -> CONTROL
	'.',    // 0x22 -> CONTROL
	'.',    // 0x23 -> CONTROL
	'.',    // 0x24 -> CONTROL
	'\n',   // 0x25 -> LINE FEED
	'.',    // 0x26 -> END OF TRANSMISSION BLOCK
	'.',    // 0x27 -> ESCAPE
	'.',    // 0x28 -> CONTROL
	'.',    // 0x29 -> CONTROL
	'.',    // 0x2A -> CONTROL
	'.',    // 0x2B -> CONTROL
	'.',    // 0x2C -> CONTROL
	'.',    // 0x2D -> ENQUIRY
	'.',    // 0x2E -> ACKNOWLEDGE
	'.',    // 0x2F -> BELL
	'.',    // 0x30 -> CONTROL
	'.',    // 0x31 -> CONTROL
	'.',    // 0x32 -> SYNCHRONOUS IDLE
	'.',    // 0x33 -> CONTROL
	'.',    // 0x34 -> CONTROL
	'.',    // 0x35 -> CONTROL
	'.',    // 0x36 -> CONTROL
	'.',    // 0x37 -> END OF TRANSMISSION
	'.',    // 0x38 -> CONTROL
	'.',    // 0x39 -> CONTROL
	'.',    // 0x3A -> CONTROL
	'.',    // 0x3B -> CONTROL
	'.',    // 0x3C -> DEVICE CONTROL FOUR
	'.',    // 0x3D -> NEGATIVE ACKNOWLEDGE
	'.',    // 0x3E -> CONTROL
	'.',    // 0x3F -> SUBSTITUTE
	' ',    // 0x40 -> SPACE
	'.',    // 0x41 -> NO-BREAK SPACE
	'.',    // 0x42 -> LATIN SMALL LETTER A WITH CIRCUMFLEX
	'.',    // 0x43 -> LATIN SMALL LETTER A WITH DIAERESIS
	'.',    // 0x44 -> LATIN SMALL LETTER A WITH GRAVE
	'.',    // 0x45 -> LATIN SMALL LETTER A WITH ACUTE
	'.',    // 0x46 -> LATIN SMALL LETTER A WITH TILDE
	'.',    // 0x47 -> LATIN SMALL LETTER A WITH RING ABOVE
	'.',    // 0x48 -> LATIN SMALL LETTER C WITH CEDILLA
	'.',    // 0x49 -> LATIN SMALL LETTER N WITH TILDE
	'[',    // 0x4A -> LEFT SQUARE BRACKET
	'.',    // 0x4B -> FULL STOP
	'<',    // 0x4C -> LESS-THAN SIGN
	'(',    // 0x4D -> LEFT PARENTHESIS
	'+',    // 0x4E -> PLUS SIGN
	'!',    // 0x4F -> EXCLAMATION MARK
	'&',    // 0x50 -> AMPERSAND
	'.',    // 0x51 -> LATIN SMALL LETTER E WITH ACUTE
	'.',    // 0x52 -> LATIN SMALL LETTER E WITH CIRCUMFLEX
	'.',    // 0x53 -> LATIN SMALL LETTER E WITH DIAERESIS
	'.',    // 0x54 -> LATIN SMALL LETTER E WITH GRAVE
	'.',    // 0x55 -> LATIN SMALL LETTER I WITH ACUTE
	'.',    // 0x56 -> LATIN SMALL LETTER I WITH CIRCUMFLEX
	'.',    // 0x57 -> LATIN SMALL LETTER I WITH DIAERESIS
	'.',    // 0x58 -> LATIN SMALL LETTER I WITH GRAVE
	'.',    // 0x59 -> LATIN SMALL LETTER SHARP S (GERMAN)
	']',    // 0x5A -> RIGHT SQUARE BRACKET
	'$',    // 0x5B -> DOLLAR SIGN
	'*',    // 0x5C -> ASTERISK
	')',    // 0x5D -> RIGHT PARENTHESIS
	';',    // 0x5E -> SEMICOLON
	'^',    // 0x5F -> CIRCUMFLEX ACCENT
	'-',    // 0x60 -> HYPHEN-MINUS
	'/',    // 0x61 -> SOLIDUS
	'.',    // 0x62 -> LATIN CAPITAL LETTER A WITH CIRCUMFLEX
	'.',    // 0x63 -> LATIN CAPITAL LETTER A WITH DIAERESIS
	'.',    // 0x64 -> LATIN CAPITAL LETTER A WITH GRAVE
	'.',    // 0x65 -> LATIN CAPITAL LETTER A WITH ACUTE
	'.',    // 0x66 -> LATIN CAPITAL LETTER A WITH TILDE
	'.',    // 0x67 -> LATIN CAPITAL LETTER A WITH RING ABOVE
	'.',    // 0x68 -> LATIN CAPITAL LETTER C WITH CEDILLA
	'.',    // 0x69 -> LATIN CAPITAL LETTER N WITH TILDE
	'.',    // 0x6A -> BROKEN BAR
	',',    // 0x6B -> COMMA
	'%',    // 0x6C -> PERCENT SIGN
	'_',    // 0x6D -> LOW LINE
	'>',    // 0x6E -> GREATER-THAN SIGN
	'?',    // 0x6F -> QUESTION MARK
	'.',    // 0x70 -> LATIN SMALL LETTER O WITH STROKE
	'.',    // 0x71 -> LATIN CAPITAL LETTER E WITH ACUTE
	'.',    // 0x72 -> LATIN CAPITAL LETTER E WITH CIRCUMFLEX
	'.',    // 0x73 -> LATIN CAPITAL LETTER E WITH DIAERESIS
	'.',    // 0x74 -> LATIN CAPITAL LETTER E WITH GRAVE
	'.',    // 0x75 -> LATIN CAPITAL LETTER I WITH ACUTE
	'.',    // 0x76 -> LATIN CAPITAL LETTER I WITH CIRCUMFLEX
	'.',    // 0x77 -> LATIN CAPITAL LETTER I WITH DIAERESIS
	'.',    // 0x78 -> LATIN CAPITAL LETTER I WITH GRAVE
	'`',    // 0x79 -> GRAVE ACCENT
	':',    // 0x7A -> COLON
	'#',    // 0x7B -> NUMBER SIGN
	'@',    // 0x7C -> COMMERCIAL AT
	'\'',   // 0x7D -> APOSTROPHE
	'=',    // 0x7E -> EQUALS SIGN
	'"',    // 0x7F -> QUOTATION MARK
	'.',    // 0x80 -> LATIN CAPITAL LETTER O WITH STROKE
	'a',    // 0x81 -> LATIN SMALL LETTER A
	'b',    // 0x82 -> LATIN SMALL LETTER B
	'c',    // 0x83 -> LATIN SMALL LETTER C
	'd',    // 0x84 -> LATIN SMALL LETTER D
	'e',    // 0x85 -> LATIN SMALL LETTER E
	'f',    // 0x86 -> LATIN SMALL LETTER F
	'g',    // 0x87 -> LATIN SMALL LETTER G
	'h',    // 0x88 -> LATIN SMALL LETTER H
	'i',    // 0x89 -> LATIN SMALL LETTER I
	'.',    // 0x8A -> LEFT-POINTING DOUBLE ANGLE QUOTATION MARK
	'.',    // 0x8B -> RIGHT-POINTING DOUBLE ANGLE QUOTATION MARK
	'.',    // 0x8C -> LATIN SMALL LETTER ETH (ICELANDIC)
	'.',    // 0x8D -> LATIN SMALL LETTER Y WITH ACUTE
	'.',    // 0x8E -> LATIN SMALL LETTER THORN (ICELANDIC)
	'.',    // 0x8F -> PLUS-MINUS SIGN
	'.',    // 0x90 -> DEGREE SIGN
	'j',    // 0x91 -> LATIN SMALL LETTER J
	'k',    // 0x92 -> LATIN SMALL LETTER K
	'l',    // 0x93 -> LATIN SMALL LETTER L
	'm',    // 0x94 -> LATIN SMALL LETTER M
	'n',    // 0x95 -> LATIN SMALL LETTER N
	'o',    // 0x96 -> LATIN SMALL LETTER O
	'p',    // 0x97 -> LATIN SMALL LETTER P
	'q',    // 0x98 -> LATIN SMALL LETTER Q
	'r',    // 0x99 -> LATIN SMALL LETTER R
	'.',    // 0x9A -> FEMININE ORDINAL INDICATOR
	'.',    // 0x9B -> MASCULINE ORDINAL INDICATOR
	'.',    // 0x9C -> LATIN SMALL LIGATURE AE
	'.',    // 0x9D -> CEDILLA
	'.',    // 0x9E -> LATIN CAPITAL LIGATURE AE
	'.',    // 0x9F -> CURRENCY SIGN
	'.',    // 0xA0 -> MICRO SIGN
	'~',    // 0xA1 -> TILDE
	's',    // 0xA2 -> LATIN SMALL LETTER S
	't',    // 0xA3 -> LATIN SMALL LETTER T
	'u',    // 0xA4 -> LATIN SMALL LETTER U
	'v',    // 0xA5 -> LATIN SMALL LETTER V
	'w',    // 0xA6 -> LATIN SMALL LETTER W
	'x',    // 0xA7 -> LATIN SMALL LETTER X
	'y',    // 0xA8 -> LATIN SMALL LETTER Y
	'z',    // 0xA9 -> LATIN SMALL LETTER Z
	'.',    // 0xAA -> INVERTED EXCLAMATION MARK
	'.',    // 0xAB -> INVERTED QUESTION MARK
	'.',    // 0xAC -> LATIN CAPITAL LETTER ETH (ICELANDIC)
	'.',    // 0xAD -> LATIN CAPITAL LETTER Y WITH ACUTE
	'.',    // 0xAE -> LATIN CAPITAL LETTER THORN (ICELANDIC)
	'.',    // 0xAF -> REGISTERED SIGN
	'.',    // 0xB0 -> CENT SIGN
	'.',    // 0xB1 -> POUND SIGN
	'.',    // 0xB2 -> YEN SIGN
	'.',    // 0xB3 -> MIDDLE DOT
	'.',    // 0xB4 -> COPYRIGHT SIGN
	'.',    // 0xB5 -> SECTION SIGN
	'.',    // 0xB6 -> PILCROW SIGN
	'.',    // 0xB7 -> VULGAR FRACTION ONE QUARTER
	'.',    // 0xB8 -> VULGAR FRACTION ONE HALF
	'.',    // 0xB9 -> VULGAR FRACTION THREE QUARTERS
	'.',    // 0xBA -> NOT SIGN
	'|',    // 0xBB -> VERTICAL LINE
	'.',    // 0xBC -> MACRON
	'.',    // 0xBD -> DIAERESIS
	'.',    // 0xBE -> ACUTE ACCENT
	'.',    // 0xBF -> MULTIPLICATION SIGN
	'{',    // 0xC0 -> LEFT CURLY BRACKET
	'A',    // 0xC1 -> LATIN CAPITAL LETTER A
	'B',    // 0xC2 -> LATIN CAPITAL LETTER B
	'C',    // 0xC3 -> LATIN CAPITAL LETTER C
	'D',    // 0xC4 -> LATIN CAPITAL LETTER D
	'E',    // 0xC5 -> LATIN CAPITAL LETTER E
	'F',    // 0xC6 -> LATIN CAPITAL LETTER F
	'G',    // 0xC7 -> LATIN CAPITAL LETTER G
	'H',    // 0xC8 -> LATIN CAPITAL LETTER H
	'I',    // 0xC9 -> LATIN CAPITAL LETTER I
	'.',    // 0xCA -> SOFT HYPHEN
	'.',    // 0xCB -> LATIN SMALL LETTER O WITH CIRCUMFLEX
	'.',    // 0xCC -> LATIN SMALL LETTER O WITH DIAERESIS
	'.',    // 0xCD -> LATIN SMALL LETTER O WITH GRAVE
	'.',    // 0xCE -> LATIN SMALL LETTER O WITH ACUTE
	'.',    // 0xCF -> LATIN SMALL LETTER O WITH TILDE
	'}',    // 0xD0 -> RIGHT CURLY BRACKET
	'J',    // 0xD1 -> LATIN CAPITAL LETTER J
	'K',    // 0xD2 -> LATIN CAPITAL LETTER K
	'L',    // 0xD3 -> LATIN CAPITAL LETTER L
	'M',    // 0xD4 -> LATIN CAPITAL LETTER M
	'N',    // 0xD5 -> LATIN CAPITAL LETTER N
	'O',    // 0xD6 -> LATIN CAPITAL LETTER O
	'P',    // 0xD7 -> LATIN CAPITAL LETTER P
	'Q',    // 0xD8 -> LATIN CAPITAL LETTER Q
	'R',    // 0xD9 -> LATIN CAPITAL LETTER R
	'.',    // 0xDA -> SUPERSCRIPT ONE
	'.',    // 0xDB -> LATIN SMALL LETTER U WITH CIRCUMFLEX
	'.',    // 0xDC -> LATIN SMALL LETTER U WITH DIAERESIS
	'.',    // 0xDD -> LATIN SMALL LETTER U WITH GRAVE
	'.',    // 0xDE -> LATIN SMALL LETTER U WITH ACUTE
	'.',    // 0xDF -> LATIN SMALL LETTER Y WITH DIAERESIS
	'\\',   // 0xE0 -> REVERSE SOLIDUS
	'.',    // 0xE1 -> DIVISION SIGN
	'S',    // 0xE2 -> LATIN CAPITAL LETTER S
	'T',    // 0xE3 -> LATIN CAPITAL LETTER T
	'U',    // 0xE4 -> LATIN CAPITAL LETTER U
	'V',    // 0xE5 -> LATIN CAPITAL LETTER V
	'W',    // 0xE6 -> LATIN CAPITAL LETTER W
	'X',    // 0xE7 -> LATIN CAPITAL LETTER X
	'Y',    // 0xE8 -> LATIN CAPITAL LETTER Y
	'Z',    // 0xE9 -> LATIN CAPITAL LETTER Z
	'.',    // 0xEA -> SUPERSCRIPT TWO
	'.',    // 0xEB -> LATIN CAPITAL LETTER O WITH CIRCUMFLEX
	'.',    // 0xEC -> LATIN CAPITAL LETTER O WITH DIAERESIS
	'.',    // 0xED -> LATIN CAPITAL LETTER O WITH GRAVE
	'.',    // 0xEE -> LATIN CAPITAL LETTER O WITH ACUTE
	'.',    // 0xEF -> LATIN CAPITAL LETTER O WITH TILDE
	'0',    // 0xF0 -> DIGIT ZERO
	'1',    // 0xF1 -> DIGIT ONE
	'2',    // 0xF2 -> DIGIT TWO
	'3',    // 0xF3 -> DIGIT THREE
	'4',    // 0xF4 -> DIGIT FOUR
	'5',    // 0xF5 -> DIGIT FIVE
	'6',    // 0xF6 -> DIGIT SIX
	'7',    // 0xF7 -> DIGIT SEVEN
	'8',    // 0xF8 -> DIGIT EIGHT
	'9',    // 0xF9 -> DIGIT NINE
	'.',    // 0xFA -> SUPERSCRIPT THREE
	'.',    // 0xFB -> LATIN CAPITAL LETTER U WITH CIRCUMFLEX
	'.',    // 0xFC -> LATIN CAPITAL LETTER U WITH DIAERESIS
	'.',    // 0xFD -> LATIN CAPITAL LETTER U WITH GRAVE
	'.',    // 0xFE -> LATIN CAPITAL LETTER U WITH ACUTE
	'.',    // 0xFF -> CONTROL
}
