// Tiny, safe Markdown subset for model answers: paragraphs, headings, lists,
// tables, **bold**, *italic*, `code`. Output is plain data (no HTML strings),
// so Vue renders it as text and nothing can inject markup.

const SEP = /^\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)*\|?\s*$/
const LIST = /^(\s*)([*+-]|\d+[.)])\s+(.*)$/
const HEADING = /^(#{1,4})\s+(.*)$/

function splitRow(line) {
  let s = line.trim()
  if (s.startsWith('|')) s = s.slice(1)
  if (s.endsWith('|')) s = s.slice(0, -1)
  return s.split('|').map((c) => c.trim())
}

export function inline(text) {
  const parts = []
  const re = /(\*\*[^*]+\*\*|`[^`]+`|\*[^*\s][^*]*\*)/g
  let last = 0
  let m
  while ((m = re.exec(text))) {
    if (m.index > last) parts.push({ t: 'text', v: text.slice(last, m.index) })
    const tok = m[0]
    if (tok.startsWith('**')) parts.push({ t: 'b', v: tok.slice(2, -2) })
    else if (tok.startsWith('`')) parts.push({ t: 'code', v: tok.slice(1, -1) })
    else parts.push({ t: 'i', v: tok.slice(1, -1) })
    last = m.index + tok.length
  }
  if (last < text.length) parts.push({ t: 'text', v: text.slice(last) })
  return parts
}

function startsBlock(lines, i) {
  const line = lines[i]
  if (!line.trim()) return true
  if (HEADING.test(line) || LIST.test(line)) return true
  return line.includes('|') && i + 1 < lines.length && SEP.test(lines[i + 1])
}

export function parseMarkdown(src) {
  const lines = String(src).replace(/\r/g, '').split('\n')
  const out = []
  let i = 0
  while (i < lines.length) {
    const line = lines[i]
    if (!line.trim()) { i++; continue }

    if (line.includes('|') && i + 1 < lines.length && SEP.test(lines[i + 1])) {
      const head = splitRow(line)
      i += 2
      const rows = []
      while (i < lines.length && lines[i].trim() && lines[i].includes('|')) {
        const cells = splitRow(lines[i])
        while (cells.length < head.length) cells.push('')
        rows.push(cells.slice(0, head.length))
        i++
      }
      out.push({ type: 'table', head, rows })
      continue
    }

    const h = HEADING.exec(line)
    if (h) { out.push({ type: 'h', level: h[1].length, text: h[2] }); i++; continue }

    if (LIST.test(line)) {
      const items = []
      while (i < lines.length && LIST.test(lines[i])) {
        const m = LIST.exec(lines[i])
        const ordered = /\d/.test(m[2])
        items.push({ text: m[3], depth: Math.min(3, Math.floor(m[1].replace(/\t/g, '  ').length / 2)), ordered, n: ordered ? parseInt(m[2], 10) : 0 })
        i++
      }
      out.push({ type: 'list', items })
      continue
    }

    const para = [line]
    i++
    while (i < lines.length && !startsBlock(lines, i)) { para.push(lines[i]); i++ }
    out.push({ type: 'p', text: para.join('\n') })
  }
  return out
}
