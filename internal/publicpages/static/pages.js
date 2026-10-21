// Landing and sales pages, the shop, checkout and downloads (design
// expansion phase 7). Every page works without this file: it adds the
// reader's own time to event details, loads a video's player only when
// it's clicked, goes on to Stripe from "Taking you to Stripe…" and checks
// again for a payment Stripe hasn't confirmed yet.
(function () {
  'use strict'

  // Event details: "13:00–13:30 your time · New York".
  function hhmm(d, zone) {
    return new Intl.DateTimeFormat(document.documentElement.lang || undefined, { hour: '2-digit', minute: '2-digit', hour12: false, timeZone: zone }).format(d)
  }
  document.querySelectorAll('.lt[data-start]').forEach(function (el) {
    try {
      var zone = Intl.DateTimeFormat().resolvedOptions().timeZone
      if (!zone || zone === 'UTC' || zone === 'Etc/UTC') return
      var start = new Date(el.getAttribute('data-start'))
      var end = new Date(el.getAttribute('data-end'))
      var time = hhmm(start, zone)
      if (end.getTime() > start.getTime()) time += '–' + hhmm(end, zone)
      var city = zone.split('/').pop().replace(/_/g, ' ')
      var tpl = el.getAttribute('data-tpl') || '{time} · {zone}'
      el.textContent = tpl.replace('{time}', time).replace('{zone}', city)
      el.hidden = false
    } catch (e) {
      // An old browser: the event's own time zone is enough.
    }
  })

  // Videos: the player loads nothing until the reader asks for it.
  document.querySelectorAll('.pg-video a[data-embed]').forEach(function (a) {
    a.addEventListener('click', function (ev) {
      ev.preventDefault()
      var frame = document.createElement('iframe')
      frame.src = a.getAttribute('data-embed')
      frame.title = a.textContent.trim()
      frame.allow = 'autoplay; fullscreen; picture-in-picture'
      frame.setAttribute('allowfullscreen', '')
      a.parentNode.replaceChild(frame, a)
    })
  })

  // "Taking you to Stripe…": on at once.
  var go = document.querySelector('form[data-autosubmit]')
  if (go) {
    window.setTimeout(function () {
      go.submit()
    }, 50)
  }

  // A payment Stripe hasn't confirmed yet: check again.
  var again = document.querySelector('[data-refresh]')
  if (again) {
    var seconds = parseInt(again.getAttribute('data-refresh'), 10) || 5
    window.setTimeout(function () {
      window.location.replace(again.getAttribute('href'))
    }, seconds * 1000)
  }
})()
