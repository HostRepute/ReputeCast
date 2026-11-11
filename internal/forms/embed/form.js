/* ReputeCast signup form. Renders the form below into every element with
   data-reputecast-form="<id>", or opens it as a popup. No third-party code,
   no cookies, no iframe: the form takes the site's fonts. Built with DOM
   APIs only (no HTML strings), so nothing in the configuration can inject
   markup into the host page.

   The form shows in one of its Live languages (cfg.langs): the page's
   (the nearest lang attribute, or data-reputecast-lang on the element),
   the browser's, or always one, as the form says (cfg.mode); a language
   it isn't Live in shows the fallback.

   A form may show its fields a step at a time (t.steps): Next checks the
   step, Back keeps the answers, and nothing is sent before the last step.
   It may show only on some pages, to visitors from some sources, on some
   devices, to new or returning visitors (cfg.target, checked here), and
   only in a schedule or some countries (the session's answer, cfg.wait).
   An A/B test's variants (cfg.ab) change its copy, steps and popup timing
   for visitors who see it in its own language (cfg.source); the session
   picks the variant from this browser's random id (localStorage rcf:vid). */
(function (cfg) {
  'use strict'
  if (!cfg || window['__rcf_' + cfg.id]) return
  window['__rcf_' + cfg.id] = true

  function el(tag, cls, text) {
    var e = document.createElement(tag)
    if (cls) e.className = cls
    if (text !== undefined && text !== null) e.textContent = text
    return e
  }
  function uid(s) {
    return 'rcf-' + cfg.id + '-' + s + '-' + Math.random().toString(36).slice(2, 7)
  }
  function stylesheet() {
    if (document.querySelector('link[data-rcf-css]')) return
    var l = document.createElement('link')
    l.rel = 'stylesheet'
    l.href = cfg.css
    l.setAttribute('data-rcf-css', '')
    document.head.appendChild(l)
  }
  function lsGet(k) {
    try {
      return localStorage.getItem(k)
    } catch (e) {
      return null
    }
  }
  function lsSet(k, v) {
    try {
      localStorage.setItem(k, v)
    } catch (e) {}
  }
  function fill(s, n, total) {
    return s.split('{n}').join(n).split('{total}').join(total)
  }

  // The browser remembers a signup through any of the workspace's forms:
  // forms that hide from subscribers show nothing to it.
  var signedKey = 'rcf:ws:' + cfg.workspace + ':done'
  function signedUp() {
    return !!lsGet(signedKey)
  }
  function markSignedUp() {
    lsSet(signedKey, '1')
  }
  function hidden() {
    return cfg.hideSubscribed && signedUp()
  }

  // Returning visitors: this browser saw one of the workspace's forms
  // before today.
  var seenKey = 'rcf:ws:' + cfg.workspace + ':seen'
  var firstSeen = Number(lsGet(seenKey) || 0)
  if (!firstSeen) lsSet(seenKey, String(Date.now()))
  function returning() {
    var today = new Date()
    today.setHours(0, 0, 0, 0)
    return firstSeen > 0 && firstSeen < today.getTime()
  }

  // This browser's random id, for A/B tests: 128 random bits.
  function visitorId() {
    var v = lsGet('rcf:vid')
    if (v && /^[A-Za-z0-9_-]{16,64}$/.test(v)) return v
    var b = new Uint8Array(16)
    window.crypto.getRandomValues(b)
    v = btoa(String.fromCharCode.apply(null, b)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
    lsSet('rcf:vid', v)
    return v
  }

  // Where the visit started (the tab's first page): its UTM tags and the
  // kind of site that sent it.
  var SEARCH = /(^|\.)(google\.[a-z.]+|bing\.com|duckduckgo\.com|yahoo\.[a-z.]+|baidu\.com|yandex\.[a-z.]+|ecosia\.org|search\.brave\.com|startpage\.com|qwant\.com|naver\.com)$/
  var SOCIAL = /(^|\.)(facebook\.com|fb\.com|t\.co|twitter\.com|x\.com|linkedin\.com|lnkd\.in|instagram\.com|reddit\.com|youtube\.com|tiktok\.com|pinterest\.[a-z.]+|bsky\.app|threads\.net|mastodon\.social|news\.ycombinator\.com|t\.me|whatsapp\.com)$/
  function landing() {
    try {
      var kept = JSON.parse(sessionStorage.getItem('rcf:src') || 'null')
      if (kept) return kept
    } catch (e) {}
    var q = new URLSearchParams(window.location.search)
    var ref = ''
    try {
      ref = document.referrer ? new URL(document.referrer).hostname.toLowerCase() : ''
    } catch (e) {}
    var cls = !ref ? 'direct' : ref === window.location.hostname.toLowerCase() ? '' : SEARCH.test(ref) ? 'search' : SOCIAL.test(ref) ? 'social' : 'other_site'
    var src = { ref: cls, s: (q.get('utm_source') || '').toLowerCase(), m: (q.get('utm_medium') || '').toLowerCase(), c: (q.get('utm_campaign') || '').toLowerCase() }
    try {
      sessionStorage.setItem('rcf:src', JSON.stringify(src))
    } catch (e) {}
    return src
  }
  function device() {
    var mq = function (q) {
      return window.matchMedia && window.matchMedia(q).matches
    }
    if (mq('(max-width: 767px)')) return 'mobile'
    if (mq('(max-width: 1099px)') || mq('(pointer: coarse)')) return 'tablet'
    return 'desktop'
  }
  function trimSlash(p) {
    return p.length > 1 ? p.replace(/\/+$/, '') : p
  }
  function pageHit(r) {
    var path = window.location.pathname
    var full = path + window.location.search
    if (r.op === 'equals') return trimSlash(path) === trimSlash(r.value) || full === r.value
    if (r.op === 'starts_with') return full.indexOf(r.value) === 0
    return full.indexOf(r.value) >= 0
  }
  // targeted reports whether this page and visitor match the form's rules.
  function targeted() {
    var t = cfg.target
    if (!t) return true
    if (t.pages && t.pages.mode) {
      var hit = (t.pages.rules || []).some(pageHit)
      if (t.pages.mode === 'only' ? !hit : hit) return false
    }
    var s = t.sources || {}
    var l = landing()
    var any = function (list, v) {
      return !list || !list.length || list.some(function (x) { return String(x).toLowerCase() === v })
    }
    if (!any(s.utmSource, l.s) || !any(s.utmMedium, l.m) || !any(s.utmCampaign, l.c)) return false
    if (s.referrer && s.referrer !== l.ref) return false
    if (t.devices && t.devices.length && t.devices.indexOf(device()) < 0) return false
    if (t.visitors === 'new' && returning()) return false
    if (t.visitors === 'returning' && !returning()) return false
    return true
  }

  // pick returns the copy to show where spot is.
  function pick(spot) {
    var wants = []
    if (cfg.mode === 'fixed') wants.push(cfg.fixed)
    else if (cfg.mode === 'browser') wants = (navigator.languages && navigator.languages.length ? navigator.languages : [navigator.language || '']).slice()
    else {
      var own = spot && spot.getAttribute ? spot.getAttribute('data-reputecast-lang') : ''
      var near = spot && spot.closest ? spot.closest('[lang]') : null
      wants.push(own || (near ? near.getAttribute('lang') : '') || document.documentElement.lang || '')
    }
    for (var i = 0; i < wants.length; i++) {
      var base = String(wants[i] || '').toLowerCase().split(/[-_]/)[0]
      if (base && cfg.langs[base]) return cfg.langs[base]
    }
    return cfg.langs[cfg.fallback]
  }

  // The session gives the form its timestamp token, says whether it shows
  // (its schedule and countries) and, in an A/B test, which variant. A peek
  // (a popup before it opens) counts no view and has no token.
  var token = ''
  var variantKey = 'rcf:' + cfg.id + ':v'
  function session(inTest, peek, done) {
    var u = cfg.session + '?vid=' + encodeURIComponent(visitorId()) + (inTest ? '&ab=1' : '') + (peek ? '&peek=1' : '')
    fetch(u, { method: 'GET', mode: 'cors', credentials: 'omit', headers: { Accept: 'application/json' } })
      .then(function (r) {
        if (r.status === 429) return { show: false }
        return r.ok ? r.json() : { show: !cfg.wait }
      })
      .then(function (j) {
        if (j.t) token = j.t
        var v = j.variant || ''
        if (v) lsSet(variantKey, v)
        done({ show: j.show !== false, variant: v })
      })
      .catch(function () {
        done({ show: !cfg.wait, variant: inTest ? lsGet(variantKey) || '' : '' })
      })
  }

  // The consent sentence, with its words for the privacy policy linked when
  // there is a policy. The search ignores case and returns an index into
  // the text itself.
  function consentText(t, parent) {
    var text = t.consent
    var m = null
    if (t.privacy && t.privacyText) {
      m = text.match(new RegExp(t.privacyText.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'i'))
    }
    if (!m) {
      parent.appendChild(document.createTextNode(text))
      return
    }
    parent.appendChild(document.createTextNode(text.slice(0, m.index)))
    var a = el('a', '', m[0])
    a.href = t.privacy
    a.target = '_blank'
    a.rel = 'noopener'
    parent.appendChild(a)
    parent.appendChild(document.createTextNode(text.slice(m.index + m[0].length)))
  }

  // linkedText writes text into parent with each link's words linked (the
  // first match, in any case): the SMS consent text's terms and privacy.
  function linkedText(parent, text, links) {
    var spans = []
    links.forEach(function (l) {
      if (!l.href || !l.words) return
      var m = text.match(new RegExp(l.words.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'i'))
      if (m) spans.push({ at: m.index, end: m.index + m[0].length, words: m[0], href: l.href })
    })
    spans.sort(function (a, b) { return a.at - b.at })
    var pos = 0
    spans.forEach(function (sp) {
      if (sp.at < pos) return
      parent.appendChild(document.createTextNode(text.slice(pos, sp.at)))
      var a = el('a', '', sp.words)
      a.href = sp.href
      a.target = '_blank'
      a.rel = 'noopener'
      parent.appendChild(a)
      pos = sp.end
    })
    parent.appendChild(document.createTextNode(text.slice(pos)))
  }

  // build renders the form in copy t; variant is the A/B test's ("": none).
  function build(t, variant) {
    var shownIn = variant && t.lang === cfg.source ? variant : ''
    var v = shownIn && cfg.ab ? cfg.ab[shownIn] : null
    var heading = v ? v.heading : t.heading
    var description = v ? v.description : t.description
    var buttonText = v ? v.button : t.button
    var steps = v ? v.steps : t.steps || []
    var messages = t.messages
    if (v && v.done) {
      messages = {
        confirm: { title: t.messages.confirm.title, text: v.done },
        subscribed: { title: t.messages.subscribed.title, text: v.done },
        error: t.messages.error,
      }
    }

    var form = el('form', 'rcf')
    form.noValidate = true
    form.method = 'post'
    form.action = cfg.hosted
    form.lang = t.lang
    form.dir = t.dir
    form.style.setProperty('--rcf-accent', cfg.accent)
    form.style.setProperty('--rcf-hover', cfg.hover)
    form.style.setProperty('--rcf-on', cfg.onAccent)
    form.style.setProperty('--rcf-link', cfg.link)

    var brand = el('div', 'rcf-brand')
    var mark = el('span', 'rcf-mark')
    mark.setAttribute('aria-hidden', 'true')
    if (cfg.logo) {
      var img = el('img')
      img.src = cfg.logo
      img.alt = ''
      mark.appendChild(img)
    } else {
      mark.textContent = cfg.initial
    }
    brand.appendChild(mark)
    brand.appendChild(el('span', 'rcf-heading', heading || cfg.publisher))
    form.appendChild(brand)
    if (description) form.appendChild(el('p', 'rcf-desc', description))

    var alert = el('div', 'rcf-alert')
    alert.setAttribute('role', 'alert')
    alert.hidden = true
    var errors = {}
    var wraps = {}
    var fieldOf = {}

    t.fields.forEach(function (f) {
      fieldOf[f.key] = f
      var inputId = uid(f.key)
      var err = el('span', 'rcf-err')
      err.id = inputId + '-err'
      err.hidden = true
      if (f.type === 'choice') {
        // A choice question: each answer adds its tag to the contact.
        var group = el('fieldset', 'rcf-lists rcf-choice')
        var lg = el('legend', '', f.label)
        if (!f.required) {
          lg.appendChild(document.createTextNode(' '))
          lg.appendChild(el('span', 'rcf-opt', t.text.optional))
        }
        group.appendChild(lg)
        var boxes = []
        f.options.forEach(function (o, i) {
          var lab = el('label', 'rcf-check')
          var box = el('input')
          box.type = f.multiple ? 'checkbox' : 'radio'
          box.name = f.name
          box.value = String(i)
          boxes.push(box)
          lab.appendChild(box)
          lab.appendChild(document.createTextNode(o))
          group.appendChild(lab)
        })
        group.appendChild(err)
        errors[f.key] = { input: boxes[0], err: err, group: boxes }
        wraps[f.key] = group
        return
      }
      var wrap = el('div', 'rcf-field')
      var label = el('label', 'rcf-label')
      label.htmlFor = inputId
      var title = el('span', '', f.label)
      if (!f.required && f.key !== 'email') {
        title.appendChild(document.createTextNode(' '))
        title.appendChild(el('span', 'rcf-opt', t.text.optional))
      }
      label.appendChild(title)
      var input
      if (f.type === 'select') {
        input = el('select')
        input.appendChild(el('option', '', f.placeholder || t.text.choose)).value = ''
        f.options.forEach(function (o) {
          var opt = el('option', '', o)
          opt.value = o
          input.appendChild(opt)
        })
      } else {
        input = el('input')
        input.type = f.key === 'email' ? 'email' : f.type === 'tel' ? 'tel' : f.type === 'number' ? 'number' : f.type === 'date' ? 'date' : 'text'
        if (f.placeholder) input.placeholder = f.placeholder
        if (f.key === 'email') input.autocomplete = 'email'
        if (f.key === 'first_name') input.autocomplete = 'given-name'
        if (f.key === 'last_name') input.autocomplete = 'family-name'
        if (f.key === 'company') input.autocomplete = 'organization'
        if (f.key === 'phone') input.autocomplete = 'tel'
      }
      input.id = inputId
      input.name = f.name
      if (f.required) input.required = true
      if (f.key === 'phone' && t.phone && t.phone.countries.length) {
        // The country picker reads numbers typed without +.
        var row = el('span', 'rcf-phone')
        var cc = el('select', 'rcf-cc')
        cc.name = 'phone_country'
        cc.setAttribute('aria-label', t.phone.label)
        t.phone.countries.forEach(function (c) {
          var o = el('option', '', c.name + ' ' + c.dial)
          o.value = c.code
          if (c.code === t.phone.default) o.selected = true
          cc.appendChild(o)
        })
        row.appendChild(cc)
        row.appendChild(input)
        label.appendChild(row)
      } else {
        label.appendChild(input)
      }
      wrap.appendChild(label)
      wrap.appendChild(err)
      errors[f.key] = { input: input, err: err }
      wraps[f.key] = wrap
    })

    // The last step (or the form) ends with the lists, the consent boxes
    // and the button.
    var tail = []
    if (cfg.lists.length > 1 && cfg.listChoice) {
      var fs = el('fieldset', 'rcf-lists')
      fs.appendChild(el('legend', '', t.listsLabel))
      cfg.lists.forEach(function (l) {
        var lab = el('label', 'rcf-check')
        var cb = el('input')
        cb.type = 'checkbox'
        cb.name = 'list'
        cb.value = l.id
        cb.checked = !!l.checked
        lab.appendChild(cb)
        lab.appendChild(document.createTextNode(t.lists[l.id] || ''))
        fs.appendChild(lab)
      })
      var lerr = el('span', 'rcf-err')
      lerr.hidden = true
      fs.appendChild(lerr)
      errors.lists = { err: lerr }
      tail.push(fs)
    }

    if (t.consent) {
      var cl = el('label', 'rcf-consent')
      var ccb = el('input')
      ccb.type = 'checkbox'
      ccb.name = 'consent'
      ccb.value = '1'
      ccb.required = true
      cl.appendChild(ccb)
      var ctext = el('span')
      consentText(t, ctext)
      cl.appendChild(ctext)
      tail.push(cl)
      var cerr = el('span', 'rcf-err')
      cerr.hidden = true
      tail.push(cerr)
      errors.consent = { input: ccb, err: cerr }
    }

    // The SMS consent box: its own text, unchecked; email consent never
    // signs anyone up for texts.
    if (t.smsConsent) {
      var sl = el('label', 'rcf-consent')
      var scb = el('input')
      scb.type = 'checkbox'
      scb.name = 'sms_consent'
      scb.value = '1'
      sl.appendChild(scb)
      var stext = el('span')
      linkedText(stext, t.smsConsent, [{ words: t.termsText, href: t.terms }, { words: t.privacyText, href: t.privacy }])
      sl.appendChild(stext)
      tail.push(sl)
      errors.sms = { input: scb, err: el('span', 'rcf-err') }
    }

    if (cfg.honeypot) {
      var hp = el('div', 'rcf-hp')
      hp.setAttribute('aria-hidden', 'true')
      var hpl = el('label', '', t.text.honeypot)
      var hpi = el('input')
      hpi.type = 'text'
      hpi.name = 'website'
      hpi.tabIndex = -1
      hpi.autocomplete = 'off'
      hpl.appendChild(hpi)
      hp.appendChild(hpl)
      tail.push(hp)
    }

    var button = el('button', 'rcf-submit', buttonText)
    button.type = 'submit'

    // Steps: one shows at a time; a polite live region says which.
    var pages = []
    var current = 0
    var counted = {}
    var live = el('div', 'rcf-vh')
    live.setAttribute('aria-live', 'polite')
    if (steps.length > 1) {
      steps.forEach(function (st, i) {
        var box = el('div', 'rcf-step')
        box.hidden = i > 0
        // The dots, then "Step 2 of 3" (FormsStates' several steps).
        var top = el('div', 'rcf-stepbar')
        var dots = el('span', 'rcf-dots')
        dots.setAttribute('aria-hidden', 'true')
        for (var d = 0; d < steps.length; d++) dots.appendChild(el('span', d === i ? 'on' : d < i ? 'done' : ''))
        top.appendChild(dots)
        var of = el('span', 'rcf-stepof', fill(t.text.stepOf, i + 1, steps.length))
        of.tabIndex = -1
        top.appendChild(of)
        box.appendChild(top)
        var focus = of
        if (st.heading) {
          var h = el('strong', 'rcf-steph', st.heading)
          h.tabIndex = -1
          box.appendChild(h)
          focus = h
        }
        if (st.description) box.appendChild(el('p', 'rcf-desc', st.description))
        st.fields.forEach(function (k) {
          if (wraps[k]) box.appendChild(wraps[k])
        })
        var last = i === steps.length - 1
        if (last) tail.forEach(function (n) { box.appendChild(n) })
        var nav = el('div', 'rcf-nav')
        if (i > 0) {
          var back = el('button', 'rcf-btn rcf-btn2', t.text.back)
          back.type = 'button'
          back.addEventListener('click', function () { go(i - 1) })
          nav.appendChild(back)
        }
        if (last) {
          if (st.button) button.textContent = st.button
          nav.appendChild(button)
        } else {
          var next = el('button', 'rcf-submit', st.button || t.text.next)
          next.type = 'button'
          next.addEventListener('click', function () { forward() })
          nav.appendChild(next)
        }
        box.appendChild(nav)
        form.appendChild(box)
        pages.push({ box: box, focus: focus, fields: st.fields })
      })
    } else {
      t.fields.forEach(function (f) { form.appendChild(wraps[f.key]) })
      tail.forEach(function (n) { form.appendChild(n) })
      form.appendChild(button)
    }
    if (t.footnote) form.appendChild(el('span', 'rcf-foot', t.footnote))
    form.appendChild(alert)
    form.appendChild(live)

    function go(n) {
      if (n === current || !pages[n]) return
      pages[current].box.hidden = true
      pages[n].box.hidden = false
      current = n
      live.textContent = fill(t.text.stepOf, n + 1, pages.length)
      pages[n].focus.focus()
      // A view of a later step, once per page view (the funnel).
      if (n > 0 && !counted[n] && navigator.sendBeacon) {
        counted[n] = true
        navigator.sendBeacon(cfg.step, new URLSearchParams({ n: String(n + 1), v: shownIn, t: token }))
      }
    }
    function stepOf(key) {
      for (var i = 0; i < pages.length; i++) if (pages[i].fields.indexOf(key) >= 0) return i
      return pages.length - 1
    }
    // checkFields is the browser's own check of some fields: the email's
    // shape and the required ones.
    function checkFields(keys) {
      var bad = {}
      keys.forEach(function (k) {
        var f = fieldOf[k]
        var e = errors[k]
        if (!f || !e) return
        if (k === 'email') {
          var typed = e.input.value.trim()
          if (!typed || typed.indexOf('@') < 1 || typed.lastIndexOf('.') < typed.indexOf('@')) bad.email = typed ? t.text.emailInvalid : t.text.emailMissing
          return
        }
        if (!f.required) return
        var filled = e.group ? e.group.some(function (b) { return b.checked }) : !!e.input.value.trim()
        if (!filled) bad[k] = t.text.required
      })
      return bad
    }
    function forward() {
      clear()
      var bad = checkFields(pages[current].fields)
      if (Object.keys(bad).length) {
        showErrors(bad)
        return
      }
      go(current + 1)
    }

    function clear() {
      alert.hidden = true
      Object.keys(errors).forEach(function (k) {
        var e = errors[k]
        e.err.hidden = true
        e.err.textContent = ''
        if (e.input) {
          ;(e.group || [e.input]).forEach(function (i) {
            i.removeAttribute('aria-invalid')
            i.removeAttribute('aria-describedby')
          })
        }
      })
    }
    function showErrors(map, message) {
      var first = null
      var keys = Object.keys(map || {})
      // A server error on an earlier step opens that step first.
      if (pages.length && keys.length) {
        var to = pages.length - 1
        keys.forEach(function (k) { to = Math.min(to, stepOf(k)) })
        if (to !== current) go(to)
      }
      keys.forEach(function (k) {
        var e = errors[k]
        if (!e) return
        e.err.textContent = map[k]
        e.err.hidden = false
        if (e.input) {
          ;(e.group || [e.input]).forEach(function (i) {
            i.setAttribute('aria-invalid', 'true')
            if (e.err.id) i.setAttribute('aria-describedby', e.err.id)
          })
          if (!first) first = e.input
        }
      })
      if (!first && message) {
        alert.textContent = message
        alert.hidden = false
      }
      if (first) first.focus()
    }
    function done(status, texted, inbox) {
      var msg = messages[status] || messages.confirm
      var box = el('div', 'rcf-done')
      box.setAttribute('role', 'status')
      box.appendChild(el('strong', 'rcf-heading', msg.title))
      box.appendChild(el('p', 'rcf-desc', msg.text))
      if (texted) box.appendChild(el('p', 'rcf-desc', texted))
      // "Open Gmail": the server builds the address from a fixed https one.
      if (inbox && /^https:\/\//.test(inbox.url)) {
        var open = el('a', 'rcf-btn', t.text.inboxOpen.split('{provider}').join(inbox.name))
        open.href = inbox.url
        open.target = '_blank'
        open.rel = 'noopener noreferrer'
        open.appendChild(el('span', 'rcf-vh', ' ' + t.text.newTab))
        box.appendChild(open)
      }
      while (form.firstChild) form.removeChild(form.firstChild)
      form.appendChild(box)
      lsSet('rcf:' + cfg.id + ':done', '1')
      markSignedUp()
    }

    // "Did you mean linh@gmail.com?": the server suggests, the visitor
    // takes the suggestion or keeps the address as typed.
    var email = errors.email
    var choice = null
    function dropChoice() {
      if (choice) email.err.parentNode.removeChild(choice)
      choice = null
    }
    function offer(suggested, typed) {
      choice = el('div', 'rcf-typo')
      var use = el('button', 'rcf-btn', t.text.typoUse.split('{email}').join(suggested))
      var keep = el('button', 'rcf-btn rcf-btn2', t.text.typoKeep.split('{email}').join(typed))
      use.type = keep.type = 'button'
      use.addEventListener('click', function () { send('use') })
      keep.addEventListener('click', function () { send('keep') })
      choice.appendChild(use)
      choice.appendChild(keep)
      email.err.parentNode.insertBefore(choice, email.err.nextSibling)
      use.focus()
    }
    email.input.addEventListener('input', function () {
      if (!choice) return
      dropChoice()
      email.err.hidden = true
      email.input.removeAttribute('aria-invalid')
      email.input.removeAttribute('aria-describedby')
    })

    function send(typo) {
      if (button.disabled) return
      var typed = email.input.value.trim()
      var body = new URLSearchParams(new FormData(form))
      body.set('_t', token)
      body.set('_js', '1')
      body.set('_lang', t.lang)
      if (shownIn) body.set('_v', shownIn)
      if (typo) body.set('typo', typo)
      // Where the signup happened, for growth sources: this page (its UTM
      // parameters) and the page before it.
      body.set('_page', String(window.location.href).slice(0, 2000))
      body.set('_ref', String(document.referrer || '').slice(0, 2000))
      button.disabled = true
      button.setAttribute('aria-busy', 'true')
      fetch(cfg.submit, { method: 'POST', mode: 'cors', credentials: 'omit', headers: { Accept: 'application/json' }, body: body })
        .then(function (r) {
          return r.json().then(function (j) { return j }, function () { return { status: 'error' } })
        })
        .then(function (j) {
          button.disabled = false
          button.removeAttribute('aria-busy')
          if (j.redirect) {
            markSignedUp()
            window.location.href = j.redirect
            return
          }
          if (j.status === 'confirm' || j.status === 'subscribed') return done(j.status, j.texted, j.inbox)
          if (j.status === 'challenge' && j.challengeUrl) {
            window.location.href = j.challengeUrl
            return
          }
          dropChoice()
          clear()
          showErrors(j.errors, j.message || messages.error.text)
          if (j.suggestion) offer(j.suggestion, typed)
        })
        .catch(function () {
          button.disabled = false
          button.removeAttribute('aria-busy')
          showErrors(null, messages.error.text)
        })
    }

    form.addEventListener('submit', function (ev) {
      ev.preventDefault()
      // Enter on an earlier step moves on; only the last step sends.
      if (pages.length && current < pages.length - 1) {
        forward()
        return
      }
      dropChoice()
      clear()
      var bad = checkFields(Object.keys(fieldOf))
      if (bad.email) {
        showErrors({ email: bad.email })
        return
      }
      if (Object.keys(bad).length) {
        showErrors(bad)
        return
      }
      if (errors.consent && !errors.consent.input.checked) {
        showErrors({ consent: t.text.consent })
        return
      }
      if (errors.sms && errors.sms.input.checked && errors.phone && !errors.phone.input.value.trim()) {
        showErrors({ phone: t.text.phoneNeeded })
        return
      }
      send('')
    })
    return form
  }

  function mountInline() {
    var spots = document.querySelectorAll('[data-reputecast-form="' + cfg.id + '"]')
    if (!spots.length) return false
    if (hidden() || !targeted()) return true // no view counted: no /session request
    stylesheet()
    var inTest = !!cfg.ab && pick(spots[0]).lang === cfg.source
    function render(variant) {
      for (var i = 0; i < spots.length; i++) {
        while (spots[i].firstChild) spots[i].removeChild(spots[i].firstChild)
        spots[i].appendChild(build(pick(spots[i]), variant))
      }
    }
    if (cfg.wait) {
      session(inTest, false, function (a) {
        if (a.show) render(a.variant)
      })
    } else {
      render('')
      session(false, false, function () {})
    }
    return true
  }

  function mountPopup() {
    if (hidden() || !targeted()) return
    var key = 'rcf:' + cfg.id
    if (lsGet(key + ':done')) return
    var closed = Number(lsGet(key + ':closed') || 0)
    if (closed && Date.now() - closed < cfg.popup.repeatDays * 86400000) return
    var t = pick(document.documentElement)
    var inTest = !!cfg.ab && t.lang === cfg.source
    // arm sets the popup's triggers: its variant's, in an A/B test.
    function arm(variant) {
      var pop = (variant && cfg.ab && cfg.ab[variant] && cfg.ab[variant].popup) || cfg.popup
      var shown = false
      function onScroll() {
        var h = document.documentElement
        var seen = (h.scrollTop + window.innerHeight) / Math.max(h.scrollHeight, 1)
        if (seen * 100 >= pop.scroll) show()
      }
      // Exit intent: the pointer leaves through the top of the window.
      function onOut(e) {
        if (!e.relatedTarget && e.clientY <= 0) show()
      }
      function show() {
        if (shown) return
        shown = true
        window.removeEventListener('scroll', onScroll)
        document.removeEventListener('mouseout', onOut)
        // Another form on the page may have signed the visitor up since.
        if (hidden()) return
        stylesheet()
        session(inTest, false, function (a) {
          if (a.show) open(a.variant || variant)
        })
      }
      if (pop.delay > 0) setTimeout(show, pop.delay * 1000)
      if (pop.scroll > 0) window.addEventListener('scroll', onScroll, { passive: true })
      // Only with a mouse: touch screens have no exit intent. Armed after 2 s
      // so a pointer passing the top edge while the page loads doesn't count.
      if (pop.exit && window.matchMedia && window.matchMedia('(pointer: fine)').matches) {
        setTimeout(function () {
          if (!shown) document.addEventListener('mouseout', onOut, { passive: true })
        }, 2000)
      }
    }
    function open(variant) {
      var v = variant && t.lang === cfg.source && cfg.ab ? cfg.ab[variant] : null
      var dlg = el('dialog', 'rcf-dialog')
      dlg.lang = t.lang
      dlg.dir = t.dir
      dlg.style.setProperty('--rcf-accent', cfg.accent)
      dlg.setAttribute('aria-label', (v ? v.heading : t.heading) || cfg.publisher)
      var close = el('button', 'rcf-close', '×')
      close.type = 'button'
      close.setAttribute('aria-label', t.text.close)
      close.addEventListener('click', function () { dlg.close() })
      dlg.appendChild(close)
      dlg.appendChild(build(t, variant))
      dlg.addEventListener('close', function () {
        lsSet(key + ':closed', String(Date.now()))
        dlg.parentNode && dlg.parentNode.removeChild(dlg)
      })
      document.body.appendChild(dlg)
      if (dlg.showModal) dlg.showModal()
      else dlg.setAttribute('open', '')
    }
    // With a schedule, countries or a test, ask first (a peek counts no view).
    if (cfg.wait) {
      session(inTest, true, function (a) {
        if (a.show) arm(a.variant)
      })
    } else {
      arm('')
    }
  }

  function start() {
    if (!mountInline() && cfg.kind === 'popup') mountPopup()
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', start)
  else start()
})(__RC_FORM__)
