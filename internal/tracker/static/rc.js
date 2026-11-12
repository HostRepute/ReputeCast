/*
 * ReputeCast site tracker (M62). A site adds:
 *
 *   <script>window.reputecast=window.reputecast||function(){(reputecast.q=reputecast.q||[]).push(arguments)};</script>
 *   <script async src="https://<ReputeCast>/track/v1/rc.js"
 *     data-workspace="<workspace>" data-site="<site id>"></script>
 *
 * Commands: reputecast('consent', true|false), ('page'), ('product', {id}),
 * ('cart', {id, quantity}), ('track', name, props), ('identify', {email,
 * signature}), ('link', token), ('notifyWhenAvailable', {id}).
 *
 * Unless the tag says data-wait-consent="false", nothing is stored or sent
 * before reputecast('consent', true) (the server also drops a waiting
 * site's events without consent). consent(false) deletes the visitor id.
 * A browser sending Global Privacy Control is never tracked. The visitor
 * id is a random first-party cookie, rc_vid, mirrored in localStorage.
 */
(function () {
  'use strict';
  var w = window, d = document, n = navigator;
  if (w.__rcTracker) return;
  var tag = d.currentScript || d.querySelector('script[data-site][src*="/track/v1/rc.js"]');
  if (!tag) return;
  w.__rcTracker = 1;
  var attr = function (k) { return tag.getAttribute(k); };
  var ws = attr('data-workspace'), site = attr('data-site');
  if (!ws || !site) return;
  var waits = attr('data-wait-consent') !== 'false';
  var autoPage = attr('data-auto-page') !== 'false';
  var gpc = n.globalPrivacyControl === true;
  var endpoint = new URL(tag.src, location.href).origin + '/p/' + encodeURIComponent(ws) + '/track/' + encodeURIComponent(site) + '/e';
  var KEY = 'rc_vid', CONSENT = 'rc_consent', MAX = 20, BYTES = 7600;
  var store = (function () { try { return w.localStorage; } catch (e) { return null; } })();
  var get = function (k) { try { return store && store.getItem(k); } catch (e) { return null; } };
  var set = function (k, v) { try { if (store) v == null ? store.removeItem(k) : store.setItem(k, v); } catch (e) { /* storage off */ } };

  // consent: true once the visitor allowed it (remembered), false once they
  // refused (remembered too), null while a waiting site waits.
  var consent = get(CONSENT) === '1' ? true : get(CONSENT) === '0' ? false : (waits ? null : true);
  var queue = [], timer = 0, vid = null, rc = null, link = null, who = null, last = '';

  try { rc = new URLSearchParams(location.search).get('rc'); } catch (e) { rc = null; }

  function cookie() {
    var m = d.cookie.match(/(?:^|;\s*)rc_vid=([A-Za-z0-9_-]{22})(?:;|$)/);
    return m ? m[1] : null;
  }
  function writeId(v) {
    var secure = location.protocol === 'https:' ? '; Secure' : '';
    d.cookie = KEY + '=' + (v || '') + '; path=/; max-age=' + (v ? 34560000 : 0) + '; SameSite=Lax' + secure;
    set(KEY, v);
  }
  function id() {
    if (vid) return vid;
    var v = cookie() || get(KEY);
    if (!v || !/^[A-Za-z0-9_-]{22}$/.test(v)) {
      var b = new Uint8Array(16);
      (w.crypto || w.msCrypto).getRandomValues(b);
      var s = '';
      for (var i = 0; i < b.length; i++) s += String.fromCharCode(b[i]);
      v = btoa(s).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
    }
    writeId(v);
    return (vid = v);
  }
  function post(body) {
    var json = JSON.stringify(body);
    if (w.fetch) {
      w.fetch(endpoint, { method: 'POST', body: json, keepalive: true, credentials: 'omit', mode: 'cors', headers: { 'Content-Type': 'text/plain' } })
        .catch(function () { /* the next page tries again */ });
    } else if (n.sendBeacon) {
      n.sendBeacon(endpoint, new Blob([json], { type: 'text/plain' }));
    }
  }
  function flush() {
    clearTimeout(timer);
    timer = 0;
    if (gpc || consent !== true) return;
    while (queue.length || rc || link || who) {
      var body = { vid: id(), events: [] };
      if (waits) body.c = 1;
      if (rc) { body.rc = rc; rc = null; }
      if (link) { body.link = link; link = null; }
      if (who) { body.identify = who; who = null; }
      var size = JSON.stringify(body).length;
      while (queue.length && body.events.length < MAX) {
        var e = JSON.stringify(queue[0]).length + 1;
        if (body.events.length && size + e > BYTES) break;
        size += e;
        body.events.push(queue.shift());
      }
      post(body);
    }
  }
  function later() {
    if (consent === false || gpc) return;
    if (consent === true && !timer) timer = setTimeout(flush, 1000);
  }
  function push(ev) {
    if (gpc || consent === false) return;
    ev.at = Date.now();
    ev.url = location.href;
    // Before consent, events wait in memory only (and go when it comes).
    if (queue.length >= 100) queue.shift();
    queue.push(ev);
    if (queue.length >= MAX) flush(); else later();
  }
  function page() {
    if (location.href === last) return;
    var ev = { t: 'page' };
    if (!last && d.referrer) ev.ref = d.referrer;
    last = location.href;
    push(ev);
  }
  function pid(p) {
    var v = p && typeof p === 'object' ? p.id : p;
    return typeof v === 'number' || typeof v === 'string' ? v : null;
  }

  var commands = {
    consent: function (yes) {
      if (gpc) return;
      if (yes === true) {
        consent = true;
        set(CONSENT, '1');
        flush();
      } else if (yes === false) {
        consent = false;
        queue.length = 0;
        rc = link = who = null;
        vid = null;
        writeId(null);
        set(CONSENT, '0');
      }
    },
    page: page,
    product: function (p) { var v = pid(p); if (v != null) push({ t: 'product', id: v }); },
    cart: function (p) {
      var v = pid(p);
      if (v != null) push({ t: 'cart', id: v, qty: (p && +p.quantity) || 1 });
    },
    notifyWhenAvailable: function (p) { var v = pid(p); if (v != null) push({ t: 'notify', id: v }); },
    track: function (name, props) {
      if (typeof name !== 'string' || !/^[a-z0-9_]{1,40}$/.test(name)) return;
      var ev = { t: 'track', name: name };
      if (props && typeof props === 'object') {
        var j = JSON.stringify(props);
        if (j.length > 2048) return;
        ev.props = props;
      }
      push(ev);
    },
    identify: function (p) {
      if (!p || typeof p.email !== 'string' || typeof p.signature !== 'string' || gpc || consent === false) return;
      who = { email: p.email, signature: p.signature };
      later();
    },
    link: function (token) {
      if (typeof token !== 'string' || !token || gpc || consent === false) return;
      link = token;
      later();
    }
  };

  function run(args) {
    var f = commands[args[0]];
    if (f) f.apply(null, Array.prototype.slice.call(args, 1));
  }
  var pending = (w.reputecast && w.reputecast.q) || [];
  w.reputecast = function () { run(arguments); };
  for (var i = 0; i < pending.length; i++) run(pending[i]);

  if (gpc) return;
  if (rc) later();
  if (autoPage) {
    page();
    var pushState = history.pushState;
    history.pushState = function () {
      var r = pushState.apply(this, arguments);
      page();
      return r;
    };
    w.addEventListener('popstate', page);
  }
  d.addEventListener('visibilitychange', function () { if (d.visibilityState === 'hidden') flush(); });
  w.addEventListener('pagehide', flush);
})();
