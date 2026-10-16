// A small evaluator for the part of Make's IML the app uses, so the tests
// can run its communication sections: build the request a module or RPC
// sends from sample parameters, and its output from a mocked answer.
//
// It covers {{ … }} templates with paths (parameters.email, body.items),
// string, number and boolean literals, + - * / = != == < > <= >= && || !,
// and the functions below. A string that is exactly one {{ … }} gives the
// expression's own value (an object, an array, a boolean); otherwise the
// pieces are joined as text. Object keys whose value is undefined are left
// out, as Make leaves them out of a request.

export const FUNCTIONS = {
  if: (c, a, b) => (truthy(c) ? a : b),
  ifempty: (a, b) => (isEmpty(a) ? b : a),
  omit: (obj, ...keys) => {
    if (!obj || typeof obj !== 'object') return obj
    const out = { ...obj }
    for (const k of keys) delete out[k]
    return out
  },
  toCollection: (arr, key, value) => {
    const out = {}
    for (const x of Array.isArray(arr) ? arr : []) {
      if (x && x[key] !== undefined && x[key] !== '') out[x[key]] = x[value]
    }
    return out
  },
  map: (arr, key, filterKey, filterValues) => {
    let items = Array.isArray(arr) ? arr : []
    if (filterKey !== undefined) {
      const allowed = String(filterValues ?? '')
        .split(',')
        .map((s) => s.trim())
      items = items.filter((x) => x && allowed.includes(String(x[filterKey])))
    }
    return items.map((x) => (x ? x[key] : undefined))
  },
  first: (arr) => (Array.isArray(arr) ? arr[0] : undefined),
  length: (x) => (Array.isArray(x) || typeof x === 'string' ? x.length : x && typeof x === 'object' ? Object.keys(x).length : 0),
  encodeURL: (s) => encodeURIComponent(String(s ?? '')),
  join: (arr, sep = ', ') => (Array.isArray(arr) ? arr.join(sep) : ''),
  lower: (s) => String(s ?? '').toLowerCase(),
  trim: (s) => String(s ?? '').trim(),
}

function truthy(v) {
  return !(v === undefined || v === null || v === false || v === '' || v === 0 || (typeof v === 'number' && Number.isNaN(v)))
}

function isEmpty(v) {
  return v === undefined || v === null || v === '' || (Array.isArray(v) && v.length === 0)
}

// --- tokens ---------------------------------------------------------------

const OPS = ['&&', '||', '!=', '==', '<=', '>=', '=', '<', '>', '+', '-', '*', '/', '!', '(', ')', ',', '.']

function tokenize(src) {
  const out = []
  let i = 0
  while (i < src.length) {
    const c = src[i]
    if (/\s/.test(c)) {
      i++
      continue
    }
    if (c === "'" || c === '"') {
      let j = i + 1
      let s = ''
      while (j < src.length && src[j] !== c) {
        if (src[j] === '\\' && j + 1 < src.length) {
          s += src[j + 1]
          j += 2
          continue
        }
        s += src[j++]
      }
      if (j >= src.length) throw new SyntaxError(`unterminated string in ${src}`)
      out.push({ t: 'str', v: s })
      i = j + 1
      continue
    }
    if (/[0-9]/.test(c)) {
      const m = /^[0-9]+(\.[0-9]+)?/.exec(src.slice(i))
      out.push({ t: 'num', v: Number(m[0]) })
      i += m[0].length
      continue
    }
    if (/[A-Za-z_]/.test(c)) {
      const m = /^[A-Za-z_][A-Za-z0-9_]*/.exec(src.slice(i))
      out.push({ t: 'id', v: m[0] })
      i += m[0].length
      continue
    }
    const op = OPS.find((o) => src.startsWith(o, i))
    if (!op) throw new SyntaxError(`unexpected ${JSON.stringify(c)} in ${src}`)
    out.push({ t: 'op', v: op })
    i += op.length
  }
  return out
}

// --- parser (precedence climbing) -------------------------------------------

const BINARY = [['||'], ['&&'], ['=', '==', '!='], ['<', '>', '<=', '>='], ['+', '-'], ['*', '/']]

/** Parses one IML expression (the inside of {{ }}) into a tree. */
export function parse(src) {
  const tokens = tokenize(src)
  let pos = 0
  const peek = () => tokens[pos]
  const eat = (v) => {
    const t = tokens[pos]
    if (!t || (v !== undefined && t.v !== v)) throw new SyntaxError(`expected ${v ?? 'a value'} in ${src}`)
    pos++
    return t
  }
  const level = (n) => {
    if (n === BINARY.length) return unary()
    let left = level(n + 1)
    while (peek() && peek().t === 'op' && BINARY[n].includes(peek().v)) {
      const op = eat().v
      left = { k: 'bin', op, left, right: level(n + 1) }
    }
    return left
  }
  const unary = () => {
    const t = peek()
    if (t && t.t === 'op' && (t.v === '!' || t.v === '-')) {
      eat()
      return { k: 'un', op: t.v, arg: unary() }
    }
    return primary()
  }
  const primary = () => {
    const t = eat()
    if (t.t === 'str' || t.t === 'num') return { k: 'lit', v: t.v }
    if (t.t === 'op' && t.v === '(') {
      const e = level(0)
      eat(')')
      return e
    }
    if (t.t !== 'id') throw new SyntaxError(`unexpected ${t.v} in ${src}`)
    if (t.v === 'true' || t.v === 'false') return { k: 'lit', v: t.v === 'true' }
    if (t.v === 'null') return { k: 'lit', v: null }
    if (peek() && peek().v === '(') {
      eat('(')
      const args = []
      while (peek() && peek().v !== ')') {
        args.push(level(0))
        if (peek() && peek().v === ',') eat(',')
        else break
      }
      eat(')')
      return { k: 'call', name: t.v, args }
    }
    const path = [t.v]
    while (peek() && peek().v === '.') {
      eat('.')
      path.push(eat().v)
    }
    return { k: 'path', path }
  }
  const tree = level(0)
  if (pos !== tokens.length) throw new SyntaxError(`unexpected ${tokens[pos].v} in ${src}`)
  return tree
}

function run(node, scope) {
  switch (node.k) {
    case 'lit':
      return node.v
    case 'path': {
      let v = scope
      for (const p of node.path) v = v === undefined || v === null ? undefined : v[p]
      return v
    }
    case 'call': {
      const fn = FUNCTIONS[node.name]
      if (!fn) throw new Error(`unknown IML function ${node.name}`)
      return fn(...node.args.map((a) => run(a, scope)))
    }
    case 'un': {
      const v = run(node.arg, scope)
      return node.op === '!' ? !truthy(v) : -Number(v)
    }
    case 'bin': {
      if (node.op === '&&') return truthy(run(node.left, scope)) && truthy(run(node.right, scope))
      if (node.op === '||') return truthy(run(node.left, scope)) || truthy(run(node.right, scope))
      const a = run(node.left, scope)
      const b = run(node.right, scope)
      switch (node.op) {
        case '+':
          return typeof a === 'string' || typeof b === 'string' ? `${a ?? ''}${b ?? ''}` : Number(a) + Number(b)
        case '-':
          return Number(a) - Number(b)
        case '*':
          return Number(a) * Number(b)
        case '/':
          return Number(a) / Number(b)
        case '=':
        case '==':
          return a == b // eslint-disable-line eqeqeq
        case '!=':
          return a != b // eslint-disable-line eqeqeq
        case '<':
          return a < b
        case '>':
          return a > b
        case '<=':
          return a <= b
        case '>=':
          return a >= b
      }
    }
  }
  throw new Error(`can't run ${JSON.stringify(node)}`)
}

const EXPR = /\{\{([\s\S]*?)\}\}/g

/** The {{ }} expressions of a string. */
export function expressions(s) {
  return [...String(s).matchAll(EXPR)].map((m) => m[1].trim())
}

/** Evaluates a section (any JSON value) against a scope. */
export function evaluate(template, scope) {
  if (typeof template === 'string') {
    const whole = /^\{\{([\s\S]*?)\}\}$/.exec(template)
    if (whole && expressions(template).length === 1) return run(parse(whole[1].trim()), scope)
    return template.replace(EXPR, (_, e) => {
      const v = run(parse(e.trim()), scope)
      return v === undefined || v === null ? '' : typeof v === 'object' ? JSON.stringify(v) : String(v)
    })
  }
  if (Array.isArray(template)) return template.map((x) => evaluate(x, scope))
  if (template && typeof template === 'object') {
    const out = {}
    for (const [k, v] of Object.entries(template)) {
      const val = evaluate(v, scope)
      if (val !== undefined) out[k] = val
    }
    return out
  }
  return template
}

/** The paths (parameters.email, connection.workspace) and functions a section uses. */
export function references(template) {
  const paths = new Set()
  const fns = new Set()
  const walk = (node) => {
    if (!node) return
    if (node.k === 'path') paths.add(node.path.join('.'))
    if (node.k === 'call') {
      fns.add(node.name)
      node.args.forEach(walk)
    }
    if (node.k === 'bin') {
      walk(node.left)
      walk(node.right)
    }
    if (node.k === 'un') walk(node.arg)
  }
  const visit = (v) => {
    if (typeof v === 'string') expressions(v).forEach((e) => walk(parse(e)))
    else if (Array.isArray(v)) v.forEach(visit)
    else if (v && typeof v === 'object') Object.values(v).forEach(visit)
  }
  visit(template)
  return { paths, fns }
}

/**
 * The request a communication section sends: base and section merged as
 * Make merges them (the section's url is relative to the base URL, its
 * headers add to the base's).
 */
export function request(base, api, scope) {
  const b = evaluate({ baseUrl: base.baseUrl, headers: base.headers ?? {} }, scope)
  const s = evaluate({ url: api.url, method: api.method ?? 'GET', headers: api.headers ?? {}, qs: api.qs, body: api.body }, scope)
  const url = /^https?:\/\//.test(s.url) ? s.url : `${b.baseUrl}${s.url}`
  const headers = Object.fromEntries(Object.entries({ ...b.headers, ...s.headers }).filter(([, v]) => v !== '' && v !== undefined))
  return { method: s.method, url, headers, body: s.body }
}

/** A response section's output for an answer: iterate, then output per item. */
export function output(api, scope) {
  const r = api.response ?? {}
  if (r.iterate !== undefined) {
    const items = evaluate(r.iterate, scope)
    return (Array.isArray(items) ? items : []).map((item) => evaluate(r.output ?? '{{item}}', { ...scope, item }))
  }
  return r.output !== undefined ? evaluate(r.output, scope) : scope.body
}
