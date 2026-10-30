const api = async (path, method = 'GET', body) => {
  const r = await fetch('/api' + path, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: body && JSON.stringify(body),
  })
  if (!r.ok) throw new Error((await r.json().catch(() => ({}))).error || r.statusText)
  return r.status === 204 ? null : r.json()
}

Vue.createApp({
  data: () => ({
    tasks: [],
    filter: '',
    form: null,
    orig: null,
    err: '',
    drag: null,
    priorities: ['critical', 'high', 'medium', 'low'],
    cols: [
      { key: 'todo', title: 'Todo' },
      { key: 'in_progress', title: 'In Progress' },
      { key: 'done', title: 'Done' },
    ],
  }),
  mounted() {
    this.load()
    setInterval(() => !this.form && this.load(), 3000)
  },
  methods: {
    async run(fn) {
      try { this.err = ''; await fn() } catch (e) { this.err = e.message }
    },
    load() { return this.run(async () => { this.tasks = await api('/tasks') }) },
    by(status) {
      const f = this.filter.trim().toLowerCase().replace(/^#/, '')
      return this.tasks.filter(t => t.status === status &&
        (!f || String(t.id) === f || (t.title + t.description + t.plan).toLowerCase().includes(f)))
    },
    openNew(status) {
      this.form = { id: 0, title: '', description: '', plan: '', priority: 'medium', status }
    },
    edit(t) { this.orig = t; this.form = { ...t } },
    save() {
      const id = this.form.id
      // Only send edited fields, so a plan or status an agent wrote meanwhile is kept.
      const body = Object.fromEntries(['title', 'description', 'plan', 'priority', 'status']
        .filter(k => !id || this.form[k] !== this.orig[k]).map(k => [k, this.form[k]]))
      if (id && !Object.keys(body).length) { this.form = null; return }
      return this.run(async () => {
        await (id ? api('/tasks/' + id, 'PATCH', body) : api('/tasks', 'POST', body))
        this.form = null
        await this.load()
      })
    },
    remove() {
      if (!confirm('Delete task #' + this.form.id + '?')) return
      return this.run(async () => {
        await api('/tasks/' + this.form.id, 'DELETE')
        this.form = null
        await this.load()
      })
    },
    drop(status) {
      const id = this.drag
      this.drag = null
      if (id) this.run(async () => { await api('/tasks/' + id, 'PATCH', { status }); await this.load() })
    },
  },
}).mount('#app')
