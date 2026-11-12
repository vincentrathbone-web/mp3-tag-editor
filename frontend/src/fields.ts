// Every ID3v2.3 field the editor can show as a column and edit.
// Must match Fields in fields.go.
export type FieldKind = '' | 'num' | 'wide'

export interface Field {
  key: string
  label: string
  frame: string
  kind: FieldKind
}

const f = (key: string, label: string, frame: string, kind: FieldKind = ''): Field => ({ key, label, frame, kind })

export const FIELDS: Field[] = [
  f('title', 'Title', 'TIT2'),
  f('artist', 'Artist', 'TPE1'),
  f('album', 'Album', 'TALB'),
  f('albumArtist', 'Album artist', 'TPE2'),
  f('track', 'Track', 'TRCK', 'num'),
  f('disc', 'Disc', 'TPOS', 'num'),
  f('year', 'Year', 'TYER', 'num'),
  f('genre', 'Genre', 'TCON'),
  f('comment', 'Comment', 'COMM', 'wide'),
  f('composer', 'Composer', 'TCOM'),
  f('bpm', 'BPM', 'TBPM', 'num'),
  f('key', 'Initial key', 'TKEY', 'num'),
  f('grouping', 'Grouping', 'TIT1'),
  f('subtitle', 'Subtitle', 'TIT3'),
  f('lyricist', 'Lyricist', 'TEXT'),
  f('conductor', 'Conductor', 'TPE3'),
  f('remixer', 'Remixed by', 'TPE4'),
  f('originalArtist', 'Original artist', 'TOPE'),
  f('originalAlbum', 'Original album', 'TOAL'),
  f('originalYear', 'Original year', 'TORY', 'num'),
  f('publisher', 'Publisher', 'TPUB'),
  f('copyright', 'Copyright', 'TCOP', 'wide'),
  f('encodedBy', 'Encoded by', 'TENC'),
  f('language', 'Language', 'TLAN', 'num'),
  f('isrc', 'ISRC', 'TSRC'),
  f('mediaType', 'Media type', 'TMED'),
  f('lyrics', 'Lyrics', 'USLT', 'wide'),
  f('url', 'Artist URL', 'WOAR', 'wide'),
]

export const FIELD: Record<string, Field> = Object.fromEntries(FIELDS.map(x => [x.key, x]))

export const DEFAULT_COLUMNS = ['title', 'artist', 'album']

export const GENRES = [
  'Alternative', 'Ambient', 'Blues', 'Classical', 'Country', 'Dance', 'Electronic', 'Folk', 'Hip-Hop',
  'House', 'Indie', 'Indie Folk', 'Jazz', 'Metal', 'Pop', 'Punk', 'R&B', 'Rap', 'Reggae', 'Rock',
  'Singer-Songwriter', 'Soul', 'Soundtrack', 'Techno', 'Trance', 'World',
]
