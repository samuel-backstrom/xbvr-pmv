const COMMON_WORDS = new Set([
  '180', '180x180', '2880x1440', '3d', '3dh', '3dv', '30fps', '30m', '360',
  '3840x1920', '4k', '5k', '5400x2700', '60fps', '6k', '7k', '7680x3840',
  '8k', 'fb360', 'fisheye190', 'funscript', 'cmscript', 'h264', 'h265', 'hevc',
  'hq', 'hsp', 'lq', 'lr', 'mkv', 'mkx200', 'mkx220', 'mono', 'mp4', 'oculus',
  'oculus5k', 'oculusrift', 'original', 'rf52', 'smartphone', 'srt', 'ssa',
  'tb', 'uhq', 'vrca220', 'vp9',
])

function titleCase(word) {
  if (!word) return word
  return word.charAt(0).toUpperCase() + word.slice(1).toLowerCase()
}

function wordsFromFilename(filename) {
  const baseName = String(filename || '')
    .replace(/\.[^.]+$/, '')
    .replace(/\.|_|\+|-/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()

  if (!baseName) return []

  return baseName
    .split(' ')
    .filter((word) => {
      const lower = word.toLowerCase()
      if (COMMON_WORDS.has(lower)) return false
      if (/^[0-9]{3,4}$/.test(word)) return false
      if (/^s[0-9]{2}e[0-9]{2}$/i.test(word)) return false
      return !/^[0-9]+p$/i.test(word)
    })
    .map(titleCase)
}

export function deriveMatchQueryFromFilename(filename) {
  return wordsFromFilename(filename)
    .join(' ')
    .replace(/\bS\b/g, "'s")
}

export function deriveSceneTitleFromFilename(filename) {
  return wordsFromFilename(filename)
    .join(' ')
    .replace(/\bS\b/g, "'s")
}
