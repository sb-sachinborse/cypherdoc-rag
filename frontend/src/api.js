export class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.status = status
  }
}

const HEADERS = { 'X-Requested-With': 'cypherdoc' }

async function request(path, { method = 'GET', body, form } = {}) {
  const headers = { ...HEADERS }
  let payload
  if (form) payload = form
  else if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
  }
  let res
  try {
    res = await fetch('/api' + path, { method, headers, body: payload, credentials: 'same-origin' })
  } catch {
    throw new ApiError('Cannot reach the server.', 0)
  }
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new ApiError(data.error || 'Request failed.', res.status)
  return data
}

// chatStream reads the server-sent events from POST /api/chat.
async function chatStream(body, handlers, signal) {
  const res = await fetch('/api/chat', {
    method: 'POST',
    headers: { ...HEADERS, 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
    credentials: 'same-origin',
    signal,
  })
  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    throw new ApiError(data.error || 'Request failed.', res.status)
  }
  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buf = ''
  for (;;) {
    const { value, done } = await reader.read()
    if (done) break
    buf += decoder.decode(value, { stream: true })
    let i
    while ((i = buf.indexOf('\n\n')) >= 0) {
      const raw = buf.slice(0, i)
      buf = buf.slice(i + 2)
      let event = 'message'
      let data = ''
      for (const line of raw.split('\n')) {
        if (line.startsWith('event: ')) event = line.slice(7)
        else if (line.startsWith('data: ')) data += line.slice(6)
      }
      let parsed = null
      try { parsed = JSON.parse(data) } catch { continue }
      handlers[event]?.(parsed)
    }
  }
}

export const api = {
  status: () => request('/status'),
  me: () => request('/auth/me'),
  login: (email, password) => request('/auth/login', { method: 'POST', body: { email, password } }),
  register: (name, email, password) => request('/auth/register', { method: 'POST', body: { name, email, password } }),
  logout: () => request('/auth/logout', { method: 'POST' }),
  listDocs: () => request('/documents'),
  upload: (file, collection) => {
    const form = new FormData()
    form.append('collection', collection)
    form.append('file', file)
    return request('/documents', { method: 'POST', form })
  },
  deleteDoc: (id) => request('/documents/' + id, { method: 'DELETE' }),
  chatStream,
}
