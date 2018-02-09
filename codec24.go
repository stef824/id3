package id3

import (
	"io"
	"reflect"
)

//
// codec24
//

type codec24 struct {
	headerFlags flagMap
	frameFlags  flagMap
	bounds      boundsMap
	frameTypes  *frameTypeMap
}

func newCodec24() *codec24 {
	return &codec24{
		headerFlags: flagMap{
			{1 << 7, uint32(TagFlagUnsync)},
			{1 << 6, uint32(TagFlagExtended)},
			{1 << 5, uint32(TagFlagExperimental)},
			{1 << 4, uint32(TagFlagFooter)},
		},
		frameFlags: flagMap{
			{1 << 14, uint32(FrameFlagDiscardOnTagAlteration)},
			{1 << 13, uint32(FrameFlagDiscardOnFileAlteration)},
			{1 << 12, uint32(FrameFlagReadOnly)},
			{1 << 6, uint32(FrameFlagHasGroupInfo)},
			{1 << 3, uint32(FrameFlagCompressed)},
			{1 << 2, uint32(FrameFlagEncrypted)},
			{1 << 1, uint32(FrameFlagUnsynchronized)},
			{1 << 0, uint32(FrameFlagHasDataLength)},
		},
		bounds: boundsMap{
			"Encoding":         {0, 3},
			"GroupSymbol":      {0x80, 0xf0},
			"PictureType":      {0, 20},
			"TimeStampFormat":  {1, 2},
			"LyricContentType": {0, 8},
		},
		frameTypes: newFrameTypeMap(map[FrameType]string{
			FrameTypeTextGroupDescription:    "TIT1",
			FrameTypeTextSongTitle:           "TIT2",
			FrameTypeTextSongSubtitle:        "TIT3",
			FrameTypeTextAlbumName:           "TALB",
			FrameTypeTextOriginalAlbum:       "TOAL",
			FrameTypeTextTrackNumber:         "TRCK",
			FrameTypeTextPartOfSet:           "TPOS",
			FrameTypeTextSetSubtitle:         "TSST",
			FrameTypeTextISRC:                "TSRC",
			FrameTypeTextArtist:              "TPE1",
			FrameTypeTextAlbumArtist:         "TPE2",
			FrameTypeTextConductor:           "TPE3",
			FrameTypeTextRemixer:             "TPE4",
			FrameTypeTextOriginalPerformer:   "TOPE",
			FrameTypeTextLyricist:            "TEXT",
			FrameTypeTextOriginalLyricist:    "TOLY",
			FrameTypeTextComposer:            "TCOM",
			FrameTypeTextMusicians:           "TMCL",
			FrameTypeTextInvolvedPeople:      "TIPL",
			FrameTypeTextEncodedBy:           "TENC",
			FrameTypeTextBPM:                 "TBPM",
			FrameTypeTextLengthInMs:          "TLEN",
			FrameTypeTextMusicalKey:          "TKEY",
			FrameTypeTextLanguage:            "TLAN",
			FrameTypeTextGenre:               "TCON",
			FrameTypeTextFileType:            "TFLT",
			FrameTypeTextMediaType:           "TMED",
			FrameTypeTextMood:                "TMOO",
			FrameTypeTextCopyright:           "TCOP",
			FrameTypeTextProducedNotice:      "TPRO",
			FrameTypeTextPublisher:           "TPUB",
			FrameTypeTextOwner:               "TOWN",
			FrameTypeTextRadioStation:        "TRSN",
			FrameTypeTextRadioStationOwner:   "TRSO",
			FrameTypeTextOriginalFileName:    "TOFN",
			FrameTypeTextPlaylistDelay:       "TDLY",
			FrameTypeTextEncodingTime:        "TDEN",
			FrameTypeTextOriginalReleaseTime: "TDOR",
			FrameTypeTextRecordingTime:       "TDRC",
			FrameTypeTextReleaseTime:         "TDRL",
			FrameTypeTextTaggingTime:         "TDTG",
			FrameTypeTextEncodingSoftware:    "TSSE",
			FrameTypeTextAlbumSortOrder:      "TSOA",
			FrameTypeTextTitleSortOrder:      "TSOT",
			FrameTypeURLCommercial:           "WCOM",
			FrameTypeURLCopyright:            "WCOP",
			FrameTypeURLAudioFile:            "WOAF",
			FrameTypeURLArtist:               "WOAR",
			FrameTypeURLAudioSource:          "WOAS",
			FrameTypeURLRadioStation:         "WORS",
			FrameTypeURLPayment:              "WPAY",
			FrameTypeURLPublisher:            "WPUB",
			FrameTypeURLCustom:               "WXXX",
			FrameTypeComment:                 "COMM",
			FrameTypeAttachedPicture:         "APIC",
			FrameTypeUniqueFileID:            "UFID",
			FrameTypeTermsOfUse:              "USER",
			FrameTypeLyricsUnsync:            "USLT",
			FrameTypeLyricsSync:              "SYLT",
			FrameTypeSyncTempoCodes:          "SYTC",
			FrameTypeGroupID:                 "GRID",
			FrameTypePrivate:                 "PRIV",
			FrameTypePlayCount:               "PCNT",
			FrameTypePopularimeter:           "POPM",
			FrameTypeUnknown:                 "ZZZZ",
		}),
	}
}

// A property holds the reflection data necessary to update a property's
// value. Usually the property is a struct field.
type property struct {
	typ   reflect.Type
	tags  tagList
	value reflect.Value
}

// The state structure keeps track of persistent state required while
// decoding a single frame.
type state struct {
	frameID  string
	encoding Encoding
}

func (c *codec24) HeaderFlags() flagMap {
	return c.headerFlags
}

func (c *codec24) DecodeExtendedHeader(t *Tag, r io.Reader) (int, error) {
	// Read the first 6 bytes of the extended header so we can see how big
	// the addition extended data is.
	var s scanner
	if s.Read(r, 6); s.err != nil {
		return s.n, s.err
	}

	// Read the size of the extended data.
	size, err := decodeSyncSafeUint32(s.ConsumeBytes(4))
	if err != nil {
		return s.n, err
	}

	// The number of extended flag bytes must be 1.
	if s.ConsumeByte() != 1 {
		return s.n, ErrInvalidHeader
	}

	// Read the extended flags field.
	exFlags := s.ConsumeByte()
	if s.err != nil {
		return s.n, s.err
	}

	// Read the rest of the extended header into the buffer.
	if s.Read(r, int(size)-6); s.err != nil {
		return s.n, s.err
	}

	if (exFlags & (1 << 6)) != 0 {
		t.Flags |= TagFlagIsUpdate
		if s.ConsumeByte() != 0 || s.err != nil {
			return s.n, ErrInvalidHeader
		}
	}

	if (exFlags & (1 << 5)) != 0 {
		t.Flags |= TagFlagHasCRC
		data := s.ConsumeBytes(6)
		if s.err != nil || data[0] != 5 {
			return s.n, ErrInvalidHeader
		}
		t.CRC, err = decodeSyncSafeUint32(data[1:])
		if err != nil {
			return s.n, ErrInvalidHeader
		}
	}

	if (exFlags & (1 << 4)) != 0 {
		t.Flags |= TagFlagHasRestrictions
		data := s.ConsumeBytes(2)
		if s.err != nil || data[0] != 1 {
			return s.n, ErrInvalidHeader
		}
		t.Restrictions = uint16(data[0])<<8 | uint16(data[1])
	}

	return s.n, s.err
}

func (c *codec24) DecodeFrame(t *Tag, f *Frame, r io.Reader) (int, error) {
	// Read the first four bytes of the frame header to see if it's padding.
	var s scanner
	if s.Read(r, 4); s.err != nil {
		return s.n, s.err
	}

	var header FrameHeader

	hdr := s.ConsumeAll()
	if hdr[0] == 0 && hdr[1] == 0 && hdr[2] == 0 && hdr[3] == 0 {
		return s.n, errPaddingEncountered
	}
	header.FrameID = string(hdr[0:4])

	// Read the rest of the header.
	if s.Read(r, 6); s.err != nil {
		return s.n, s.err
	}
	hdr = append(hdr, s.ConsumeAll()...)

	// Process the frame's size.
	size, err := decodeSyncSafeUint32(hdr[4:8])
	if err != nil {
		return s.n, err
	}
	if size < 1 {
		return s.n, ErrInvalidFrameHeader
	}
	header.Size = int(size)

	// Process the flags.
	flags := uint32(hdr[8])<<8 | uint32(hdr[9])
	header.Flags = FrameFlags(c.frameFlags.Decode(flags))

	// Read the rest of the frame into a buffer.
	if s.Read(r, header.Size); s.err != nil {
		return s.n, s.err
	}

	// Strip unsync codes if the frame is unsynchronized but the tag isn't.
	if (header.Flags&FrameFlagUnsynchronized) != 0 && (t.Flags&TagFlagUnsync) == 0 {
		s.Replace(removeUnsyncCodes(s.buf))
	}

	// Scan extra header data indicated by flags.
	if header.Flags != 0 {
		c.scanExtraHeaderData(&s, &header)
		if s.err != nil {
			return s.n, s.err
		}
	}

	// Initialize the frame payload scan state.
	state := state{
		frameID:  header.FrameID,
		encoding: EncodingISO88591,
	}

	// Select a frame payload type and scan its structure.
	typ := c.frameTypes.LookupReflectType(header.FrameID)
	p := property{
		typ:   typ,
		tags:  emptyTagList,
		value: reflect.New(typ),
	}
	c.scanStruct(&s, p, &state)

	// Retrieve the interface.
	if s.err == nil {
		*f = p.value.Interface().(Frame)

		// Copy the header into the frame's first field, which must be the
		// header.
		ht := reflect.ValueOf(*f).Elem()
		ht.Field(0).Set(reflect.ValueOf(header))
	}

	return s.n, s.err
}

func (c *codec24) scanExtraHeaderData(s *scanner, h *FrameHeader) {
	// If the frame is compressed, it must include a data length indicator.
	if (h.Flags&FrameFlagCompressed) != 0 && (h.Flags&FrameFlagHasDataLength) == 0 {
		s.err = ErrInvalidFrameFlags
		return
	}

	if (h.Flags & FrameFlagHasGroupInfo) != 0 {
		gid := s.ConsumeByte()
		if s.err != nil || gid < 0x80 || gid > 0xf0 {
			s.err = ErrInvalidFrameHeader
			return
		}
		h.GroupID = GroupSymbol(gid)
	}

	if (h.Flags & FrameFlagEncrypted) != 0 {
		em := s.ConsumeByte()
		if s.err != nil || em < 0x80 || em > 0xf0 {
			s.err = ErrInvalidFrameHeader
			return
		}
		h.EncryptMethod = em
	}

	if (h.Flags & FrameFlagHasDataLength) != 0 {
		b := s.ConsumeBytes(4)
		if s.err != nil {
			s.err = ErrInvalidFrameHeader
		}
		h.DataLength, s.err = decodeSyncSafeUint32(b)
	}
}

func (c *codec24) scanStruct(s *scanner, p property, state *state) {
	if p.typ.Name() == "FrameHeader" {
		return
	}

	for i, n := 0, p.typ.NumField(); i < n; i++ {
		field := p.typ.Field(i)

		fp := property{
			typ:   field.Type,
			tags:  getTags(field.Tag, "id3"),
			value: p.value.Elem().Field(i),
		}

		switch field.Type.Kind() {
		case reflect.Uint8:
			c.scanUint8(s, fp, state)

		case reflect.Uint16:
			c.scanUint16(s, fp, state)

		case reflect.Uint32:
			c.scanUint32(s, fp, state)

		case reflect.Uint64:
			c.scanUint64(s, fp, state)

		case reflect.Slice:
			switch field.Type.Elem().Kind() {
			case reflect.Uint8:
				c.scanByteSlice(s, fp, state)
			case reflect.String:
				c.scanStringSlice(s, fp, state)
			case reflect.Struct:
				c.scanStructSlice(s, fp, state)
			default:
				panic(errUnknownFieldType)
			}

		case reflect.String:
			c.scanString(s, fp, state)

		case reflect.Struct:
			c.scanStruct(s, fp, state)

		default:
			panic(errUnknownFieldType)
		}
	}
}

func (c *codec24) scanString(s *scanner, p property, state *state) {
	if s.err != nil {
		return
	}

	if p.typ.Name() == "FrameID" {
		p.value.SetString(state.frameID)
		return
	}

	enc := state.encoding
	if p.tags.Lookup("iso88519") {
		enc = EncodingISO88591
	}

	var str string
	if p.tags.Lookup("lang") {
		str = s.ConsumeFixedLengthString(3, EncodingISO88591)
	} else {
		str = s.ConsumeNextString(enc)
	}
	if s.err != nil {
		return
	}

	p.value.SetString(str)
}

func (c *codec24) scanByteSlice(s *scanner, p property, state *state) {
	if s.err != nil {
		return
	}

	b := s.ConsumeAll()
	p.value.Set(reflect.ValueOf(b))
}

func (c *codec24) scanStringSlice(s *scanner, p property, state *state) {
	if s.err != nil {
		return
	}

	ss := s.ConsumeStrings(state.encoding)
	if s.err != nil {
		return
	}
	p.value.Set(reflect.ValueOf(ss))
}

func (c *codec24) scanStructSlice(s *scanner, p property, state *state) {
	if s.err != nil {
		return
	}

	elems := make([]reflect.Value, 0)
	for s.Len() > 0 {
		etyp := p.typ.Elem()
		ep := property{
			typ:   etyp,
			tags:  emptyTagList,
			value: reflect.New(etyp),
		}

		c.scanStruct(s, ep, state)
		if s.err != nil {
			return
		}

		elems = append(elems, ep.value)
	}

	slice := reflect.MakeSlice(p.typ, len(elems), len(elems))
	for i := range elems {
		slice.Index(i).Set(elems[i].Elem())
	}
	p.value.Set(slice)
}

func (c *codec24) scanUint8(s *scanner, p property, state *state) {
	if s.err != nil {
		return
	}

	if p.typ.Name() == "FrameType" {
		typ := c.frameTypes.LookupFrameType(state.frameID)
		p.value.SetUint(uint64(typ))
		return
	}

	b, hasBounds := c.bounds[p.typ.Name()]

	value := s.ConsumeByte()
	if s.err != nil {
		return
	}

	if hasBounds && (value < uint8(b.min) || value > uint8(b.max)) {
		s.err = ErrInvalidFrame
		return
	}

	if p.typ.Name() == "Encoding" {
		state.encoding = Encoding(value)
	}

	p.value.SetUint(uint64(value))
}

func (c *codec24) scanUint16(s *scanner, p property, state *state) {
	if s.err != nil {
		return
	}

	var value uint16
	if p.tags.Lookup("tempo") {
		value = uint16(s.ConsumeByte())
		if value == 0xff {
			value += uint16(s.ConsumeByte())
		}
	} else {
		b := s.ConsumeBytes(2)
		value = uint16(b[0])<<8 | uint16(b[1])
	}
	if s.err != nil {
		return
	}

	p.value.SetUint(uint64(value))
}

func (c *codec24) scanUint32(s *scanner, p property, state *state) {
	if s.err != nil {
		return
	}

	buf := s.ConsumeBytes(4)

	var value uint64
	for _, b := range buf {
		value = (value << 8) | uint64(b)
	}

	p.value.SetUint(value)
}

func (c *codec24) scanUint64(s *scanner, p property, state *state) {
	if s.err != nil {
		return
	}

	var buf []byte
	if p.tags.Lookup("counter") {
		buf = s.ConsumeAll()
	} else {
		buf = s.ConsumeBytes(8)
	}

	if s.err != nil {
		s.err = ErrInvalidFrame
		return
	}

	var value uint64
	for _, b := range buf {
		value = (value << 8) | uint64(b)
	}

	p.value.SetUint(value)
}

func (c *codec24) EncodeFrame(t *Tag, f *Frame, w io.Writer) (int, error) {
	return 0, ErrUnimplemented
}
