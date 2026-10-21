/* ReputeCast signup form. Renders the form below into every element with
   data-reputecast-form="<id>", or opens it as a popup. No third-party code,
   no cookies, no iframe: the form takes the site's fonts. Built with DOM
   APIs only (no HTML strings), so nothing in the configuration can inject
   markup into the host page.

   The form shows in one of its Live languages (cfg.langs): the page's
   (the nearest lang attribute, or data-reputecast-lang on the element),
   the browser's, or always one, as the form says (cfg.mode); a language
   it isn't Live in shows the fallback. */
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

  var token = ''
  var sessionLoaded = false
  function session() {
    if (sessionLoaded) return
    sessionLoaded = true
    fetch(cfg.session, { method: 'GET', mode: 'cors', credentials: 'omit', headers: { Accept: 'application/json' } })
      .then(function (r) { return r.ok ? r.json() : {} })
      .then(function (j) { token = (j && j.t) || '' })
      .catch(function () {})
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

  function build(t) {
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
    brand.appendChild(el('span', 'rcf-heading', t.heading || cfg.publisher))
    form.appendChild(brand)
    if (t.description) form.appendChild(el('p', 'rcf-desc', t.description))

    var alert = el('div', 'rcf-alert')
    alert.setAttribute('role', 'alert')
    alert.hidden = true
    var errors = {}

    t.fields.forEach(function (f) {
      var wrap = el('div', 'rcf-field')
      var inputId = uid(f.key)
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
      var err = el('span', 'rcf-err')
      err.id = inputId + '-err'
      err.hidden = true
      wrap.appendChild(err)
      errors[f.key] = { input: input, err: err }
      form.appendChild(wrap)
    })

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
      form.appendChild(fs)
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
      form.appendChild(cl)
      var cerr = el('span', 'rcf-err')
      cerr.hidden = true
      form.appendChild(cerr)
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
      form.appendChild(sl)
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
      form.appendChild(hp)
    }

    var button = el('button', 'rcf-submit', t.button)
    button.type = 'submit'
    form.appendChild(button)
    if (t.footnote) form.appendChild(el('span', 'rcf-foot', t.footnote))
    form.appendChild(alert)

    function clear() {
      alert.hidden = true
      Object.keys(errors).forEach(function (k) {
        var e = errors[k]
        e.err.hidden = true
        e.err.textContent = ''
        if (e.input) {
          e.input.removeAttribute('aria-invalid')
          e.input.removeAttribute('aria-describedby')
        }
      })
    }
    function showErrors(map, message) {
      var first = null
      Object.keys(map || {}).forEach(function (k) {
        var e = errors[k]
        if (!e) return
        e.err.textContent = map[k]
        e.err.hidden = false
        if (e.input) {
          e.input.setAttribute('aria-invalid', 'true')
          if (e.err.id) e.input.setAttribute('aria-describedby', e.err.id)
          if (!first) first = e.input
        }
      })
      if (!first && message) {
        alert.textContent = message
        alert.hidden = false
      }
      if (first) first.focus()
    }
    function done(status, texted) {
      var msg = t.messages[status] || t.messages.confirm
      var box = el('div', 'rcf-done')
      box.setAttribute('role', 'status')
      box.appendChild(el('strong', 'rcf-heading', msg.title))
      box.appendChild(el('p', 'rcf-desc', msg.text))
      if (texted) box.appendChild(el('p', 'rcf-desc', texted))
      while (form.firstChild) form.removeChild(form.firstChild)
      form.appendChild(box)
      try {
        localStorage.setItem('rcf:' + cfg.id + ':done', '1')
      } catch (e) {}
    }

    form.addEventListener('submit', function (ev) {
      ev.preventDefault()
      clear()
      var email = errors.email.input.value.trim()
      if (!email || email.indexOf('@') < 1 || email.lastIndexOf('.') < email.indexOf('@')) {
        showErrors({ email: email ? t.text.emailInvalid : t.text.emailMissing })
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
      var body = new URLSearchParams(new FormData(form))
      body.set('_t', token)
      body.set('_js', '1')
      body.set('_lang', t.lang)
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
            window.location.href = j.redirect
            return
          }
          if (j.status === 'confirm' || j.status === 'subscribed') return done(j.status, j.texted)
          if (j.status === 'challenge' && j.challengeUrl) {
            window.location.href = j.challengeUrl
            return
          }
          showErrors(j.errors, j.message || t.messages.error.text)
        })
        .catch(function () {
          button.disabled = false
          button.removeAttribute('aria-busy')
          showErrors(null, t.messages.error.text)
        })
    })
    return form
  }

  function mountInline() {
    var spots = document.querySelectorAll('[data-reputecast-form="' + cfg.id + '"]')
    if (!spots.length) return false
    stylesheet()
    session()
    for (var i = 0; i < spots.length; i++) {
      while (spots[i].firstChild) spots[i].removeChild(spots[i].firstChild)
      spots[i].appendChild(build(pick(spots[i])))
    }
    return true
  }

  function mountPopup() {
    var key = 'rcf:' + cfg.id
    try {
      if (localStorage.getItem(key + ':done')) return
      var closed = Number(localStorage.getItem(key + ':closed') || 0)
      if (closed && Date.now() - closed < cfg.popup.repeatDays * 86400000) return
    } catch (e) {}
    var shown = false
    function show() {
      if (shown) return
      shown = true
      stylesheet()
      session()
      var t = pick(document.documentElement)
      var dlg = el('dialog', 'rcf-dialog')
      dlg.lang = t.lang
      dlg.dir = t.dir
      dlg.style.setProperty('--rcf-accent', cfg.accent)
      dlg.setAttribute('aria-label', t.heading || cfg.publisher)
      var close = el('button', 'rcf-close', '×')
      close.type = 'button'
      close.setAttribute('aria-label', t.text.close)
      close.addEventListener('click', function () { dlg.close() })
      dlg.appendChild(close)
      dlg.appendChild(build(t))
      dlg.addEventListener('close', function () {
        try {
          localStorage.setItem(key + ':closed', String(Date.now()))
        } catch (e) {}
        dlg.parentNode && dlg.parentNode.removeChild(dlg)
      })
      document.body.appendChild(dlg)
      if (dlg.showModal) dlg.showModal()
      else dlg.setAttribute('open', '')
    }
    if (cfg.popup.delay > 0) setTimeout(show, cfg.popup.delay * 1000)
    if (cfg.popup.scroll > 0) {
      window.addEventListener('scroll', function onScroll() {
        var h = document.documentElement
        var seen = (h.scrollTop + window.innerHeight) / Math.max(h.scrollHeight, 1)
        if (seen * 100 >= cfg.popup.scroll) {
          window.removeEventListener('scroll', onScroll)
          show()
        }
      }, { passive: true })
    }
  }

  function start() {
    if (!mountInline() && cfg.kind === 'popup') mountPopup()
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', start)
  else start()
})(__RC_FORM__)
