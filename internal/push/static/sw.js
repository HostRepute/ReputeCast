/* ReputeCast push service worker · version 1 · https://reputecast.com
 *
 * Shows the notifications a ReputeCast workspace sends, reports when the
 * browser shows or closes one, and opens its link when it's clicked. Host it
 * at your site's root as rc-push-sw.js, or add one line to the service
 * worker your site already has:
 *
 *   importScripts("https://<your ReputeCast address>/push/v1/sw.js");
 */
(function () {
  'use strict';
  var VERSION = 1;
  self.RC_PUSH_VERSION = VERSION;

  // report tells ReputeCast a notification was shown or closed. The answer
  // is never read, so the request needs no CORS.
  function report(data, event) {
    if (!data || !data.events || !data.id || !data.t) return Promise.resolve();
    try {
      return fetch(data.events, {
        method: 'POST',
        mode: 'no-cors',
        credentials: 'omit',
        keepalive: true,
        headers: { 'Content-Type': 'text/plain' },
        body: JSON.stringify({ kind: 'notification', id: data.id, t: data.t, event: event })
      }).catch(function () {});
    } catch (e) {
      return Promise.resolve();
    }
  }

  self.addEventListener('push', function (event) {
    var p = null;
    try {
      p = event.data ? event.data.json() : null;
    } catch (e) {
      p = null;
    }
    if (!p || !p.title || !p.id) return;
    var actions = (p.actions || []).slice(0, 2);
    var data = { url: p.url, id: p.id, t: p.t, events: p.events, actions: actions };
    var options = {
      body: p.body || '',
      data: data,
      actions: actions.map(function (a) {
        return { action: a.action, title: a.title };
      })
    };
    if (p.icon) options.icon = p.icon;
    if (p.image) options.image = p.image;
    if (p.tag) {
      options.tag = p.tag;
      options.renotify = !!p.renotify;
    }
    event.waitUntil(
      self.registration.showNotification(p.title, options).then(function () {
        return report(data, 'shown');
      })
    );
  });

  self.addEventListener('notificationclick', function (event) {
    var n = event.notification;
    var data = n.data || {};
    n.close();
    var url = data.url;
    if (event.action) {
      (data.actions || []).forEach(function (a) {
        if (a.action === event.action && a.url) url = a.url;
      });
    }
    if (!url) return;
    // The link goes through ReputeCast, which counts the click.
    event.waitUntil(self.clients.openWindow(url));
  });

  self.addEventListener('notificationclose', function (event) {
    event.waitUntil(report(event.notification.data || {}, 'closed'));
  });

  // As a site's own file (rc-push-sw.js), the worker takes over at once;
  // imported into another worker, it leaves that worker's lifecycle alone.
  if (self.RC_PUSH_OWN) {
    self.addEventListener('install', function () {
      self.skipWaiting();
    });
    self.addEventListener('activate', function (event) {
      event.waitUntil(self.clients.claim());
    });
  }
})();
