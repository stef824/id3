package id3

import (
	"bufio"
	"encoding/binary"
)

type codec23 struct {
}

func (c *codec23) Read(t *Tag, r *bufio.Reader) (int, error) {
	return 0, nil
}

func (c *codec23) Write(t *Tag, w *bufio.Writer) (int, error) {
	return 0, nil
}

func (h *FrameHeader) read23(r *bufio.Reader) (int, error) {
	buf := make([]byte, 10)
	n, err := r.Read(buf)
	if n < 10 || err != nil {
		return n, err
	}

	h.IDvalue = string(buf[0:4])
	h.Size = binary.BigEndian.Uint32(buf[4:8])
	h.Flags = 0

	flags := binary.BigEndian.Uint16(buf[8:10])
	if flags != 0 {
		if (flags & (1 << 15)) != 0 {
			h.Flags |= FrameFlagDiscardOnTagAlteration
		}
		if (flags & (1 << 14)) != 0 {
			h.Flags |= FrameFlagDiscardOnFileAlteration
		}
		if (flags & (1 << 13)) != 0 {
			h.Flags |= FrameFlagReadOnly
		}
		if (flags & (1 << 7)) != 0 {
			h.Flags |= FrameFlagCompressed
		}
		if (flags & (1 << 6)) != 0 {
			h.Flags |= FrameFlagEncrypted
		}
		if (flags & (1 << 5)) != 0 {
			h.Flags |= FrameFlagHasGroupInfo
		}
	}
	return n, nil
}

func (h *FrameHeader) write23(w *bufio.Writer) (int, error) {
	nn := 0

	n, err := w.WriteString(h.IDvalue)
	nn += n
	if err != nil {
		return nn, err
	}

	buf := make([]byte, 6)
	binary.BigEndian.PutUint32(buf[0:4], h.Size)

	var flags uint16
	if h.Flags != 0 {
		if (h.Flags & FrameFlagDiscardOnTagAlteration) != 0 {
			flags |= 1 << 15
		}
		if (h.Flags & FrameFlagDiscardOnFileAlteration) != 0 {
			flags |= 1 << 14
		}
		if (h.Flags & FrameFlagReadOnly) != 0 {
			flags |= 1 << 13
		}
		if (h.Flags & FrameFlagCompressed) != 0 {
			flags |= 1 << 7
		}
		if (h.Flags & FrameFlagEncrypted) != 0 {
			flags |= 1 << 6
		}
		if (h.Flags & FrameFlagHasGroupInfo) != 0 {
			flags |= 1 << 5
		}
	}
	buf[4] = uint8(flags >> 8)
	buf[5] = uint8(flags)

	n, err = w.Write(buf)
	nn += n
	return nn, err
}