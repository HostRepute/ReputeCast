/* Challenge page: report Turnstile widget errors (a site key Cloudflare
   refuses, a domain it doesn't allow) so the publisher sees them in
   ReputeCast. Returning false lets the widget show its own message. */
window.rcTurnstileError = function (code) {
  var form = document.getElementById('rc-challenge')
  var url = form && form.getAttribute('data-error-url')
  if (url) {
    var body = new URLSearchParams()
    body.set('code', String(code))
    try {
      if (navigator.sendBeacon) navigator.sendBeacon(url, body)
      else fetch(url, { method: 'POST', body: body, credentials: 'omit' })
    } catch (e) {
      /* nothing to do: the page still works */
    }
  }
  return false
}
