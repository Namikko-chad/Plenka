// Hand-drawn 24×24 stroke icons. `fill` icons are drawn solid.
const circle = (cx: number, cy: number, r: number) =>
  `M${cx - r} ${cy}a${r} ${r} 0 1 0 ${2 * r} 0a${r} ${r} 0 1 0 ${-2 * r} 0`

export const ICONS = {
  home: { d: ['M3 10.5 12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6H9v6H4a1 1 0 0 1-1-1z'] },
  search: { d: [circle(11, 11, 7), 'm20 20-4.2-4.2'] },
  download: { d: ['M12 3v12m0 0-4.5-4.5M12 15l4.5-4.5', 'M4 17v3h16v-3'] },
  radar: {
    d: [circle(12, 12, 2), 'M7.8 16.2a6 6 0 0 1 0-8.4M16.2 7.8a6 6 0 0 1 0 8.4', 'M5 19a10 10 0 0 1 0-14M19 5a10 10 0 0 1 0 14'],
  },
  settings: {
    d: ['M4 6h9M17 6h3M4 12h3M11 12h9M4 18h11M19 18h1', circle(15, 6, 2), circle(9, 12, 2), circle(17, 18, 2)],
  },
  back: { d: ['M15 18l-6-6 6-6'] },
  chevronDown: { d: ['M6 9l6 6 6-6'] },
  close: { d: ['M18 6 6 18M6 6l12 12'] },
  play: { d: ['M7 4.5v15l13-7.5z'], fill: true },
  pause: { d: ['M6.5 4h3.5v16H6.5zM14 4h3.5v16H14z'], fill: true },
  replay: { d: ['M3 12a9 9 0 1 0 3-6.7L3 8', 'M3 3v5h5'] },
  forward: { d: ['M21 12a9 9 0 1 1-3-6.7L21 8', 'M21 3v5h-5'] },
  next: { d: ['M5 4.5l10 7.5-10 7.5z', 'M16.5 5h2.5v14h-2.5z'], fill: true },
  fullscreen: { d: ['M4 9V4h5M20 9V4h-5M4 15v5h5M20 15v5h-5'] },
  fullscreenExit: { d: ['M9 4v5H4M15 4v5h5M9 20v-5H4M15 20v-5h5'] },
  external: { d: ['M14 4h6v6M20 4l-9 9', 'M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5'] },
  copy: { d: ['M10 8h9a1 1 0 0 1 1 1v10a1 1 0 0 1-1 1h-9a1 1 0 0 1-1-1V9a1 1 0 0 1 1-1z', 'M16 8V5a1 1 0 0 0-1-1H5a1 1 0 0 0-1 1v10a1 1 0 0 0 1 1h4'] },
  plus: { d: ['M12 5v14M5 12h14'] },
  upload: { d: ['M12 15V3m0 0L7.5 7.5M12 3l4.5 4.5', 'M4 17v3h16v-3'] },
  trash: { d: ['M4 7h16M9 7V4h6v3', 'M6 7l1 13h10l1-13'] },
  eject: { d: ['M5 14h14l-7-9z', 'M5 19h14'] },
  refresh: { d: ['M20 11a8 8 0 1 0-2.3 5.7', 'M20 4v7h-7'] },
  folder: { d: ['M3 6a1 1 0 0 1 1-1h5l2 2h9a1 1 0 0 1 1 1v10a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1z'] },
  info: { d: [circle(12, 12, 9), 'M12 11v6M12 7.5v.01'] },
  check: { d: ['M5 12l5 5L20 7'] },
  alert: { d: ['M12 9v4M12 17v.01', 'M10.3 3.9 2.4 18a2 2 0 0 0 1.7 3h15.8a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z'] },
  zap: { d: ['M13 2 4 14h7l-1 8 9-12h-7z'] },
  film: { d: ['M4 4h16v16H4z', 'M8 4v16M16 4v16M4 8h4M4 12h4M4 16h4M16 8h4M16 12h4M16 16h4'] },
  tv: { d: ['M3 6h18v11H3z', 'M8 21h8M12 17v4'] },
  file: { d: ['M14 3H6a1 1 0 0 0-1 1v16a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V8z', 'M14 3v5h5'] },
  checkCircle: { d: [circle(12, 12, 9), 'M8 12l3 3 5-6'] },
  playCircle: { d: [circle(12, 12, 9), 'M10 8.5v7l6-3.5z'] },
  key: { d: [circle(8, 15, 4), 'M11 12l9-9M16 7l3 3'] },
  subtitles: { d: ['M3 5h18v14H3z', 'M7 12h4M13 12h4M7 15.5h7M16 15.5h1'] },
  audio: { d: ['M9 18V6l11-2v12', circle(6, 18, 3), circle(17, 16, 3)] },
  youtube: { d: ['M3 7.5a3 3 0 0 1 3-3h12a3 3 0 0 1 3 3v9a3 3 0 0 1-3 3H6a3 3 0 0 1-3-3z', 'M10 9v6l5-3z'] },
  preload: { d: [circle(12, 12, 9), 'M12 7v7m0 0-3-3m3 3 3-3', 'M8.5 17h7'] },
} as const

export type IconName = keyof typeof ICONS
