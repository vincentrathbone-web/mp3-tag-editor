type P = { className?: string }
const S = ({ children, className }: P & { children: React.ReactNode }) => (
  <svg className={'i ' + (className ?? '')} viewBox="0 0 24 24" aria-hidden="true">{children}</svg>
)

export const IconMusic = (p: P) => <S {...p}><path d="M9 18V5l12-2v13" /><circle cx="6" cy="18" r="3" /><circle cx="18" cy="16" r="3" /></S>
export const IconPlus = (p: P) => <S {...p}><path d="M12 5v14M5 12h14" /></S>
export const IconSave = (p: P) => <S {...p}><path d="M5 4h11l3 3v13H5z" /><path d="M8 4v5h8V4" /><path d="M8 20v-6h8v6" /></S>
export const IconUndo = (p: P) => <S {...p}><path d="M9 14 4 9l5-5" /><path d="M4 9h10a6 6 0 0 1 0 12h-3" /></S>
export const IconColumns = (p: P) => <S {...p}><rect x="3" y="4" width="18" height="16" rx="2" /><path d="M9 4v16M15 4v16" /></S>
export const IconRemove = (p: P) => <S {...p}><path d="M4 7h16M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3" /></S>
export const IconFolder = (p: P) => <S {...p}><path d="M3 6a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" /></S>
export const IconFile = (p: P) => <S {...p}><path d="M14 3H6v18h12V7z" /><path d="M14 3v4h4" /></S>
export const IconImage = (p: P) => <S {...p}><rect x="3" y="3" width="18" height="18" rx="2" /><circle cx="9" cy="9" r="2" /><path d="m21 15-5-5L5 21" /></S>
export const IconClose = (p: P) => <S {...p}><path d="M6 6l12 12M18 6 6 18" /></S>
export const IconSearch = (p: P) => <S {...p}><circle cx="11" cy="11" r="7" /><path d="m20 20-3.5-3.5" /></S>
export const IconRows = (p: P) => <S {...p}><path d="M4 6h16M4 12h16M4 18h10" /></S>
export const IconRename = (p: P) => <S {...p}><path d="M4 7h9M4 12h6M4 17h5" /><path d="m14 19 1-4 5-5 3 3-5 5z" /></S>
export const IconWarn = (p: P) => <S {...p}><path d="M12 3 2 20h20z" /><path d="M12 10v4M12 17h.01" /></S>
