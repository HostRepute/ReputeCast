/* ReputeCast push prompt · https://reputecast.com
 *
 * Asks readers for permission to send notifications: ReputeCast's own
 * prompt shows first, and the browser's appears only after "Allow", so
 * "Not now" never uses up the browser's one chance to ask. On iPhone and
 * iPad, where only a site added to the Home Screen can get notifications,
 * it shows how to add the site, or stays hidden.
 *
 *   <script async src="https://<ReputeCast>/push/v1/prompt.js"
 *     data-workspace="<workspace>" data-site="<site>"></script>
 */
(function () {
  'use strict';
  if (window.__rcPush) return;
  window.__rcPush = true;

  var script = document.currentScript;
  var src = null;
  try {
    src = script && script.src ? new URL(script.src, location.href) : null;
  } catch (e) {
    src = null;
  }
  function attr(name, param) {
    return (script && script.getAttribute(name)) || (src && src.searchParams.get(param)) || '';
  }
  var workspace = attr('data-workspace', 'w');
  var siteHost = attr('data-site', 's') || location.host;
  var scriptBase = src ? src.origin : location.origin;
  if (!workspace) return;

  var supported = 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window;
  var ua = navigator.userAgent || '';
  var ios = /iPhone|iPad|iPod/.test(ua) || (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1);
  var standalone = navigator.standalone === true || (window.matchMedia && window.matchMedia('(display-mode: standalone)').matches);
  var cfg = null;
  var noop = function () {};

  var store = {
    get: function (k) {
      try {
        return window.localStorage.getItem(k);
      } catch (e) {
        return null;
      }
    },
    set: function (k, v) {
      try {
        window.localStorage.setItem(k, v);
      } catch (e) {
        /* private mode */
      }
    }
  };

  function base() {
    return cfg && cfg.base ? cfg.base : scriptBase;
  }
  function endpoint(path) {
    return base() + '/p/' + encodeURIComponent(workspace) + '/push/' + path;
  }
  function sameOrigin() {
    return base() === location.origin;
  }
  function post(path, body) {
    return fetch(endpoint(path), {
      method: 'POST',
      mode: 'cors',
      credentials: sameOrigin() ? 'include' : 'omit',
      keepalive: true,
      headers: { 'Content-Type': 'text/plain' },
      body: JSON.stringify(body)
    });
  }
  function promptEvent(p, ev) {
    if (!p) return;
    post('events', { kind: 'prompt', prompt: p.id, event: ev }).catch(noop);
  }
  function timeZone() {
    try {
      return Intl.DateTimeFormat().resolvedOptions().timeZone || '';
    } catch (e) {
      return '';
    }
  }
  // A personal link from an email or a text carries the reader: ?rc= or ?r=.
  function readerToken() {
    var t = '';
    try {
      var q = new URLSearchParams(location.search);
      t = q.get('rc') || q.get('r') || '';
      if (t) window.sessionStorage.setItem('rc-push:reader', t);
      else t = window.sessionStorage.getItem('rc-push:reader') || '';
    } catch (e) {
      /* storage blocked */
    }
    return t;
  }
  function keyBytes(b64) {
    var pad = '='.repeat((4 - (b64.length % 4)) % 4);
    var raw = window.atob((b64 + pad).replace(/-/g, '+').replace(/_/g, '/'));
    var out = new Uint8Array(raw.length);
    for (var i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i);
    return out;
  }
  function sameKey(sub) {
    try {
      var k = sub.options && sub.options.applicationServerKey;
      if (!k) return true;
      var a = new Uint8Array(k);
      var b = keyBytes(cfg.key);
      if (a.length !== b.length) return false;
      for (var i = 0; i < a.length; i++) if (a[i] !== b[i]) return false;
      return true;
    } catch (e) {
      return true;
    }
  }

  function registration() {
    return navigator.serviceWorker.register(cfg.worker, { scope: cfg.scope }).then(function (reg) {
      return navigator.serviceWorker.ready.then(function () {
        return reg;
      });
    });
  }
  function currentSubscription() {
    if (!supported) return Promise.resolve(null);
    return navigator.serviceWorker.getRegistration(cfg ? cfg.scope : '/').then(function (reg) {
      return reg ? reg.pushManager.getSubscription() : null;
    });
  }
  // subscribe asks the browser (after the reader chose Allow), then saves
  // the subscription with its consent record.
  function subscribe(from, p) {
    return registration()
      .then(function (reg) {
        return reg.pushManager.getSubscription().then(function (sub) {
          if (sub && !sameKey(sub)) {
            return sub.unsubscribe().then(function () {
              return null;
            });
          }
          return sub;
        }).then(function (sub) {
          return sub || reg.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: keyBytes(cfg.key) });
        });
      })
      .then(function (sub) {
        var j = sub.toJSON();
        return post('subscriptions', {
          endpoint: j.endpoint,
          keys: j.keys,
          timeZone: timeZone(),
          page: location.href,
          prompt: p ? p.id : '',
          version: p ? p.version : 0,
          token: readerToken(),
          from: from || ''
        }).then(function (r) {
          return r.ok;
        });
      });
  }
  function unsubscribe(reason) {
    return currentSubscription().then(function (sub) {
      if (!sub) return false;
      var ep = sub.endpoint;
      return sub.unsubscribe().catch(noop).then(function () {
        return post('unsubscribe', { endpoint: ep, reason: reason || 'turned_off' }).then(function () {
          return true;
        });
      });
    });
  }

  // ---- Where and when ----
  function pathAllowed(c) {
    var path = location.pathname;
    var hit = (c.paths || []).some(function (x) {
      var p = x.replace(/\/+$/, '');
      return path === x || path === p || path.indexOf(p + '/') === 0;
    });
    return c.pages === 'only' ? hit : !hit;
  }
  function snoozed(p) {
    var t = Number(store.get('rc-push:notnow:' + p.id)) || 0;
    return t > 0 && Date.now() - t < (p.config.askAgainDays || 14) * 86400000;
  }
  function snooze(p) {
    store.set('rc-push:notnow:' + p.id, String(Date.now()));
  }
  function postsRead() {
    var key = 'rc-push:read:' + siteHost;
    var list = [];
    try {
      list = JSON.parse(store.get(key) || '[]');
    } catch (e) {
      list = [];
    }
    if (/\/posts?\//.test(location.pathname) && list.indexOf(location.pathname) < 0) {
      list.push(location.pathname);
      if (list.length > 50) list = list.slice(-50);
      store.set(key, JSON.stringify(list));
    }
    return list.length;
  }

  // ---- The prompt ----
  var host = null;
  var root = null;
  function mount() {
    if (host) return root;
    host = document.createElement('div');
    host.setAttribute('data-rc-push-prompt', '');
    if (cfg.accent) host.style.setProperty('--rc-accent', cfg.accent);
    root = host.attachShadow ? host.attachShadow({ mode: 'open' }) : host;
    var css = document.createElement('link');
    css.rel = 'stylesheet';
    css.href = scriptBase + '/push/v1/prompt.css';
    root.appendChild(css);
    document.body.appendChild(host);
    return root;
  }
  function el(tag, cls, text) {
    var e = document.createElement(tag);
    if (cls) e.className = cls;
    if (text) e.textContent = text;
    return e;
  }
  function icon() {
    if (cfg.icon) {
      var img = el('img', 'ic');
      img.src = cfg.icon;
      img.alt = '';
      img.width = 32;
      img.height = 32;
      return img;
    }
    return el('span', 'ic mark', (cfg.name || '?').charAt(0).toUpperCase());
  }
  function close(card) {
    if (card && card.parentNode) card.parentNode.removeChild(card);
    document.removeEventListener('keydown', onKey);
  }
  var openCard = null;
  var openPrompt = null;
  function onKey(e) {
    if (e.key === 'Escape' && openCard) {
      promptEvent(openPrompt, 'closed');
      snooze(openPrompt);
      close(openCard);
      openCard = null;
    }
  }

  function showSoft(p) {
    var r = mount();
    var c = p.config;
    var wrap = el('div', 'rc ' + (c.style || 'bar') + (c.corner === 'bottom_left' ? ' left' : ''));
    var card = el('div', 'card');
    card.setAttribute('role', 'dialog');
    card.setAttribute('aria-modal', c.style === 'dialog' ? 'true' : 'false');
    card.setAttribute('aria-labelledby', 'rc-t');
    card.appendChild(icon());
    var tx = el('div', 'tx');
    var t = el('b', '', c.title);
    t.id = 'rc-t';
    tx.appendChild(t);
    if (c.message) tx.appendChild(el('span', '', c.message));
    card.appendChild(tx);
    var bt = el('div', 'bt');
    var no = el('button', 'no', c.notNow || 'Not now');
    no.type = 'button';
    var yes = el('button', 'yes', c.allow || 'Allow');
    yes.type = 'button';
    bt.appendChild(no);
    bt.appendChild(yes);
    card.appendChild(bt);
    wrap.appendChild(card);
    r.appendChild(wrap);
    openCard = wrap;
    openPrompt = p;
    document.addEventListener('keydown', onKey);
    promptEvent(p, 'shown');
    no.addEventListener('click', function () {
      promptEvent(p, 'not_now');
      snooze(p);
      close(wrap);
    });
    yes.addEventListener('click', function () {
      promptEvent(p, 'allow');
      close(wrap);
      // The browser's own prompt, now that the reader said yes.
      Notification.requestPermission().then(function (perm) {
        if (perm === 'granted') subscribe('', p).catch(noop);
        else if (perm === 'denied') promptEvent(p, 'blocked');
      });
    });
    (c.style === 'dialog' ? yes : no).focus({ preventScroll: true });
  }

  function showIOS(p) {
    var r = mount();
    var t = cfg.text || {};
    var wrap = el('div', 'rc ios');
    var card = el('div', 'card');
    card.setAttribute('role', 'dialog');
    card.setAttribute('aria-labelledby', 'rc-ios-t');
    var head = el('div', 'head');
    head.appendChild(icon());
    var h = el('b', '', t.iosTitle || 'To get notifications, add this site to your Home Screen: tap Share, then Add to Home Screen.');
    h.id = 'rc-ios-t';
    head.appendChild(h);
    card.appendChild(head);
    var steps = el('ol', 'steps');
    [t.iosShare || "Tap Share in Safari's toolbar", t.iosAdd || 'Tap Add to Home Screen', t.iosOpen || 'Open the site from the Home Screen and allow notifications'].forEach(
      function (s, i) {
        var li = el('li', 's' + (i + 1), s);
        steps.appendChild(li);
      }
    );
    card.appendChild(steps);
    var no = el('button', 'no wide', p.config.notNow || 'Not now');
    no.type = 'button';
    card.appendChild(no);
    wrap.appendChild(card);
    r.appendChild(wrap);
    openCard = wrap;
    openPrompt = p;
    document.addEventListener('keydown', onKey);
    promptEvent(p, 'ios_steps');
    no.addEventListener('click', function () {
      promptEvent(p, 'not_now');
      snooze(p);
      close(wrap);
    });
  }

  function showBell(p) {
    var r = mount();
    var c = p.config;
    var b = el('button', 'bell' + (c.corner === 'bottom_left' ? ' left' : ''));
    b.type = 'button';
    b.setAttribute('aria-label', c.title);
    // Built node by node: sites that enforce Trusted Types refuse innerHTML.
    var ns = 'http://www.w3.org/2000/svg';
    var svg = document.createElementNS(ns, 'svg');
    var attrs = { width: '22', height: '22', viewBox: '0 0 16 16', fill: 'none', stroke: 'currentColor', 'stroke-width': '1.6',
      'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'aria-hidden': 'true' };
    Object.keys(attrs).forEach(function (k) {
      svg.setAttribute(k, attrs[k]);
    });
    var path = document.createElementNS(ns, 'path');
    path.setAttribute('d', 'M4 11V7a4 4 0 0 1 8 0v4l1.5 1.5h-11zM6.5 14a1.5 1.5 0 0 0 3 0');
    svg.appendChild(path);
    b.appendChild(svg);
    r.appendChild(b);
    b.addEventListener('click', function () {
      if (openCard) return;
      promptEvent(p, 'bell');
      showSoft(p);
    });
  }

  function arm(p) {
    var c = p.config;
    if (c.style === 'bell') {
      showBell(p);
      return;
    }
    var shown = false;
    function go() {
      if (shown) return;
      shown = true;
      if (ios && !standalone) showIOS(p);
      else showSoft(p);
    }
    var w = c.when || {};
    if (w.button) {
      Array.prototype.forEach.call(document.querySelectorAll('[data-rc-push]'), function (b) {
        b.addEventListener('click', function (e) {
          e.preventDefault();
          go();
        });
      });
    }
    if (w.posts && w.posts.on && postsRead() >= w.posts.n) {
      go();
      return;
    }
    if (w.seconds && w.seconds.on) window.setTimeout(go, Math.max(0, w.seconds.n) * 1000);
    if (w.scroll && w.scroll.on) {
      var onScroll = function () {
        var d = document.documentElement;
        var max = d.scrollHeight - d.clientHeight;
        if (max <= 0 || ((window.scrollY || d.scrollTop) / max) * 100 >= w.scroll.n) {
          window.removeEventListener('scroll', onScroll);
          go();
        }
      };
      window.addEventListener('scroll', onScroll, { passive: true });
    }
  }

  // ---- The account page and the preference center ----
  // listAccount shows the member's other browsers, each with "Turn off".
  function listAccount(box, mine) {
    var list = box.querySelector('[data-rc-push-list]');
    if (!list || !sameOrigin()) return;
    fetch(endpoint('account'), { credentials: 'include' })
      .then(function (r) {
        return r.ok ? r.json() : null;
      })
      .then(function (d) {
        if (!d || !d.subscriptions) return;
        while (list.firstChild) list.removeChild(list.firstChild);
        d.subscriptions.forEach(function (s) {
          if (mine && s.endpoint === mine) return;
          var li = el('li', 'st-k');
          var since = new Date(s.since);
          var date = since.toLocaleDateString(document.documentElement.lang || undefined, { day: 'numeric', month: 'short' });
          // The browser's name in the page's language, else in English.
          var name = (s.kind && box.getAttribute('data-browser-' + s.kind)) || s.browser;
          li.appendChild(document.createTextNode(name + ' · ' + (box.getAttribute('data-since') || 'on since {date}').replace('{date}', date) + ' · '));
          var off = el('button', 'st-linkbtn', box.getAttribute('data-turnoff') || 'Turn off');
          off.type = 'button';
          off.addEventListener('click', function () {
            off.disabled = true;
            post('account/off', { id: s.id }).then(function () {
              li.parentNode && li.parentNode.removeChild(li);
            });
          });
          li.appendChild(off);
          list.appendChild(li);
        });
      })
      .catch(noop);
  }

  function wireAccount(box) {
    box.hidden = false;
    var toggle = box.querySelector('[data-rc-push-toggle]');
    var state = box.querySelector('[data-rc-push-state]');
    var blocked = box.querySelector('[data-rc-push-blocked]');
    var words = { on: box.getAttribute('data-on') || 'On in this browser', off: box.getAttribute('data-off') || 'Off in this browser',
      blocked: box.getAttribute('data-blocked') || 'Blocked in this browser', unsupported: box.getAttribute('data-unsupported') || "This browser can't get notifications" };
    function show(s) {
      if (state) {
        state.textContent = words[s] || '';
        state.className = state.className.replace(/\bis-\w+/g, '') + ' is-' + s;
      }
      if (blocked) blocked.hidden = s !== 'blocked';
      if (toggle) {
        toggle.checked = s === 'on';
        toggle.disabled = s === 'blocked' || s === 'unsupported';
      }
    }
    if (!supported || (ios && !standalone)) {
      show('unsupported');
      listAccount(box, '');
      return;
    }
    if (Notification.permission === 'denied') {
      show('blocked');
      listAccount(box, '');
      return;
    }
    currentSubscription().then(function (sub) {
      show(sub && Notification.permission === 'granted' ? 'on' : 'off');
      listAccount(box, sub ? sub.endpoint : '');
    });
    if (toggle) {
      toggle.addEventListener('change', function () {
        toggle.disabled = true;
        var next = toggle.checked
          ? Notification.requestPermission().then(function (perm) {
              if (perm !== 'granted') return perm === 'denied' ? 'blocked' : 'off';
              return subscribe('account', null).then(function () {
                return 'on';
              });
            })
          : unsubscribe('turned_off').then(function () {
              return 'off';
            });
        next.then(show, function () {
          show('off');
        });
      });
    }
  }

  function wireTurnOff(btn) {
    var note = document.querySelector('[data-rc-push-turnoff-note]');
    if (!supported) {
      btn.hidden = true;
      return;
    }
    currentSubscription().then(function (sub) {
      if (!sub) {
        btn.disabled = true;
        if (note) note.textContent = btn.getAttribute('data-none') || 'Notifications are off in this browser.';
      }
    });
    btn.addEventListener('click', function (e) {
      e.preventDefault();
      btn.disabled = true;
      unsubscribe('turned_off').then(function () {
        if (note) note.textContent = btn.getAttribute('data-done') || 'Notifications are off in this browser.';
      });
    });
  }

  // ---- Start ----
  function start() {
    fetch(scriptBase + '/p/' + encodeURIComponent(workspace) + '/push/config?site=' + encodeURIComponent(siteHost), {
      mode: 'cors',
      credentials: 'omit'
    })
      .then(function (r) {
        return r.ok ? r.json() : null;
      })
      .then(function (c) {
        if (!c || !c.key) return;
        cfg = c;
        Array.prototype.forEach.call(document.querySelectorAll('[data-rc-push-account]'), wireAccount);
        Array.prototype.forEach.call(document.querySelectorAll('[data-rc-push-turnoff]'), wireTurnOff);
        if (!supported && !ios) return;
        if (ios && standalone && !store.get('rc-push:ios-added')) {
          store.set('rc-push:ios-added', '1');
          if (c.prompts[0]) promptEvent(c.prompts[0], 'ios_added');
        }
        var p = (c.prompts || []).filter(function (x) {
          return pathAllowed(x.config);
        })[0];
        if (!p) return;
        if (ios && !standalone) {
          if (p.config.ios === 'hidden') return;
        } else if (!supported) {
          return;
        }
        // Never shown in a browser that blocked notifications.
        if (supported && Notification.permission === 'denied') return;
        if (supported && Notification.permission === 'granted') {
          // Already allowed: keep the subscription saved, quietly.
          currentSubscription().then(function (sub) {
            if (!sub) subscribe('', p).catch(noop);
          });
          return;
        }
        if (snoozed(p)) return;
        arm(p);
      })
      .catch(noop);
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', start);
  else start();
})();
