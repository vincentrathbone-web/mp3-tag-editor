package main

// Field is one editable tag field. Key is the name the UI uses, Frame the ID3v2.3 frame id.
type Field struct {
	Key   string
	Frame string
}

// Fields lists every tag field the editor can show and edit, in display order.
// It must match FIELDS in frontend/src/fields.ts.
var Fields = []Field{
	{"title", "TIT2"}, {"artist", "TPE1"}, {"album", "TALB"}, {"albumArtist", "TPE2"},
	{"track", "TRCK"}, {"disc", "TPOS"}, {"year", "TYER"}, {"genre", "TCON"},
	{"comment", "COMM"}, {"composer", "TCOM"}, {"bpm", "TBPM"}, {"key", "TKEY"},
	{"grouping", "TIT1"}, {"subtitle", "TIT3"}, {"lyricist", "TEXT"}, {"conductor", "TPE3"},
	{"remixer", "TPE4"}, {"originalArtist", "TOPE"}, {"originalAlbum", "TOAL"},
	{"originalYear", "TORY"}, {"publisher", "TPUB"}, {"copyright", "TCOP"},
	{"encodedBy", "TENC"}, {"language", "TLAN"}, {"isrc", "TSRC"}, {"mediaType", "TMED"},
	{"lyrics", "USLT"}, {"url", "WOAR"},
}

// v24Aliases maps ID3v2.4 frames onto the v2.3 field that holds the same data.
var v24Aliases = map[string]string{"TDRC": "year", "TDOR": "originalYear"}

// v24Only are frames that do not exist in ID3v2.3. They are dropped when a tag is
// rewritten as v2.3 (their useful content is carried over through v24Aliases first).
var v24Only = map[string]bool{
	"TDRC": true, "TDOR": true, "TDRL": true, "TDEN": true, "TDTG": true, "TSST": true,
	"TMOO": true, "TPRO": true, "TIPL": true, "TMCL": true, "ASPI": true, "EQU2": true,
	"RVA2": true, "SEIS": true, "SIGN": true, "SEEK": true,
}

// v22Frames maps the 3-letter ID3v2.2 frame ids onto their v2.3 equivalents.
var v22Frames = map[string]string{
	"TT1": "TIT1", "TT2": "TIT2", "TT3": "TIT3", "TP1": "TPE1", "TP2": "TPE2", "TP3": "TPE3",
	"TP4": "TPE4", "TCM": "TCOM", "TXT": "TEXT", "TLA": "TLAN", "TCO": "TCON", "TAL": "TALB",
	"TPA": "TPOS", "TRK": "TRCK", "TRC": "TSRC", "TYE": "TYER", "TBP": "TBPM", "TCR": "TCOP",
	"TPB": "TPUB", "TEN": "TENC", "TOA": "TOPE", "TOT": "TOAL", "TOR": "TORY", "TKE": "TKEY",
	"TMT": "TMED", "COM": "COMM", "ULT": "USLT", "PIC": "APIC", "WAR": "WOAR", "TXX": "TXXX",
}

// id3v1Genres is the standard ID3v1 genre list (with the common Winamp extensions).
var id3v1Genres = []string{
	"Blues", "Classic Rock", "Country", "Dance", "Disco", "Funk", "Grunge", "Hip-Hop", "Jazz", "Metal",
	"New Age", "Oldies", "Other", "Pop", "R&B", "Rap", "Reggae", "Rock", "Techno", "Industrial",
	"Alternative", "Ska", "Death Metal", "Pranks", "Soundtrack", "Euro-Techno", "Ambient", "Trip-Hop", "Vocal", "Jazz+Funk",
	"Fusion", "Trance", "Classical", "Instrumental", "Acid", "House", "Game", "Sound Clip", "Gospel", "Noise",
	"AlternRock", "Bass", "Soul", "Punk", "Space", "Meditative", "Instrumental Pop", "Instrumental Rock", "Ethnic", "Gothic",
	"Darkwave", "Techno-Industrial", "Electronic", "Pop-Folk", "Eurodance", "Dream", "Southern Rock", "Comedy", "Cult", "Gangsta",
	"Top 40", "Christian Rap", "Pop/Funk", "Jungle", "Native American", "Cabaret", "New Wave", "Psychadelic", "Rave", "Showtunes",
	"Trailer", "Lo-Fi", "Tribal", "Acid Punk", "Acid Jazz", "Polka", "Retro", "Musical", "Rock & Roll", "Hard Rock",
	"Folk", "Folk-Rock", "National Folk", "Swing", "Fast Fusion", "Bebob", "Latin", "Revival", "Celtic", "Bluegrass",
	"Avantgarde", "Gothic Rock", "Progressive Rock", "Psychedelic Rock", "Symphonic Rock", "Slow Rock", "Big Band", "Chorus", "Easy Listening", "Acoustic",
	"Humour", "Speech", "Chanson", "Opera", "Chamber Music", "Sonata", "Symphony", "Booty Bass", "Primus", "Porn Groove",
	"Satire", "Slow Jam", "Club", "Tango", "Samba", "Folklore", "Ballad", "Power Ballad", "Rhythmic Soul", "Freestyle",
	"Duet", "Punk Rock", "Drum Solo", "A capella", "Euro-House", "Dance Hall",
}
