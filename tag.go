package id3

import (
	"bytes"
	"hash/crc32"
	"io"
)

// A Tag represents an entire ID3 tag, including zero or more frames.
type Tag struct {
	Version      Version  // ID3 codec version (2.2, 2.3, or 2.4)
	Flags        TagFlags // Flags
	Size         int      // Size not including the header
	Padding      int      // Number of bytes of padding
	CRC          uint32   // Optional CRC code
	Restrictions uint8    // ID3 restrictions (v2.4 only)
	Frames       []Frame  // All ID3 frames included in the tag
}

// TagFlags describe flags that may appear within an ID3 tag. Not all
// flags are supported by all versions of the ID3 codec.
type TagFlags uint32

// All possible TagFlags.
const (
	TagFlagUnsync TagFlags = 1 << iota
	TagFlagExtended
	TagFlagExperimental
	TagFlagFooter
	TagFlagIsUpdate
	TagFlagHasCRC
	TagFlagHasRestrictions
)

func newCodec(v Version) (codec, error) {
	switch v {
	case Version2_2:
		return newCodec22(), nil
	case Version2_3:
		return newCodec23(), nil
	case Version2_4:
		return newCodec24(), nil
	default:
		return nil, ErrInvalidVersion
	}
}

// ReadFrom reads from a stream into an ID3 tag. It returns the number of
// bytes read and any error encountered during decoding.
func (t *Tag) ReadFrom(r io.Reader) (int64, error) {
	rr := newReader(r)

	// Read 3 bytes to check for the ID3 file id.
	if rr.Load(3); rr.err != nil {
		return int64(rr.n), rr.err
	}

	// Make sure the tag id is "ID3".
	fileID := rr.ConsumeAll()
	if fileID[0] != 'I' || fileID[1] != 'D' || fileID[2] != '3' {
		return int64(rr.n), ErrInvalidTag
	}

	// Process the version number (2.2, 2.3, or 2.4).
	if rr.Load(1); rr.err != nil {
		return int64(rr.n), rr.err
	}
	t.Version = Version(rr.ConsumeByte())
	c, err := newCodec(t.Version)
	if err != nil {
		return int64(rr.n), err
	}

	// Decode the rest of the tag.
	_, err = c.Decode(t, rr)
	return int64(rr.n), err
}

// WriteTo writes an ID3 tag to an output stream. It returns the number of
// bytes written and any error encountered during encoding.
func (t *Tag) WriteTo(w io.Writer) (int64, error) {
	codec, err := newCodec(t.Version)
	if err != nil {
		return 0, err
	}

	// Create a buffer to hold everything but the 10-byte header.
	buf := bytes.NewBuffer(make([]byte, 0, 1024))

	// Encode the tag's frames into the buffer.
	for _, f := range t.Frames {
		_, err := codec.EncodeFrame(t, f, buf)
		if err != nil {
			return 0, err
		}
	}

	// Add padding.
	if t.Padding > 0 {
		pad := make([]byte, t.Padding)
		buf.Write(pad)
	}

	// Calculate CRC if requested.
	if (t.Flags & TagFlagHasCRC) != 0 {
		t.CRC = uint32(crc32.ChecksumIEEE(buf.Bytes()))
	}

	// Mark the extended flag if necessary.
	if (t.Flags & (TagFlagHasCRC | TagFlagHasRestrictions | TagFlagIsUpdate)) != 0 {
		t.Flags |= TagFlagExtended
	}

	// Unsynchronize the tag if requested.
	b := buf.Bytes()
	if (t.Flags & TagFlagUnsync) != 0 {
		b = addUnsyncCodes(b)
	}

	// Encode the extended header into a buffer.
	exBuf := bytes.NewBuffer([]byte{})
	codec.EncodeExtendedHeader(t, exBuf)
	t.Size = len(b) + exBuf.Len()

	var n int64

	// Encode and write the tag header.
	nn, err := codec.EncodeHeader(t, w)
	n += int64(nn)
	if err != nil {
		return 0, err
	}

	// Write the extended header.
	nn, err = w.Write(exBuf.Bytes())
	n += int64(nn)
	if err != nil {
		return n, err
	}

	// Write the frames to the output stream.
	nn, err = w.Write(b)
	n += int64(nn)

	return n, err
}
