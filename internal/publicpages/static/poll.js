// Poll answer pages and survey pages (design expansion, X11). The pages work
// without this script: the answer is confirmed with a button, a survey is
// sent with its form. With it, the answer picked in the email is saved once
// the page has been visible for a moment, picking another answer saves it
// right away, and text boxes count their characters.
(function () {
  'use strict';

  // Numbers as the page's language writes them ("1,234", "1.234").
  var numbers = null;
  try {
    numbers = new Intl.NumberFormat(document.documentElement.lang || 'en');
  } catch (e) {
    numbers = null;
  }
  function fmt(n) {
    return numbers ? numbers.format(n) : String(n).replace(/\B(?=(\d{3})+(?!\d))/g, ',');
  }

  // A counter's words come with the page ("{n} of 500"), in its language.
  function counters(root) {
    root.querySelectorAll('[data-count]').forEach(function (out) {
      var id = out.id;
      var box = root.querySelector('[aria-describedby~="' + id + '"]');
      if (!box) return;
      var max = Number(out.getAttribute('data-count'));
      var tpl = out.getAttribute('data-tpl');
      var update = function () {
        var n = fmt(Array.from(box.value).length);
        out.textContent = tpl ? tpl.replace('{n}', n) : n + ' of ' + fmt(max) + (max > 500 ? ' characters' : '');
      };
      box.addEventListener('input', update);
    });
  }

  // The page's own words for what the script says, in its language.
  function words(main, name, english) {
    return main.getAttribute('data-' + name) || english;
  }

  function whenVisible(fn) {
    if (document.visibilityState === 'visible') return fn();
    var once = function () {
      if (document.visibilityState !== 'visible') return;
      document.removeEventListener('visibilitychange', once);
      fn();
    };
    document.addEventListener('visibilitychange', once);
  }

  function status(main, text, tone) {
    var box = main.querySelector('[role="status"]');
    if (!box) return;
    box.className = 'pstat ' + (tone || 'neutral');
    box.textContent = '';
    var b = document.createElement('b');
    b.textContent = text;
    var div = document.createElement('div');
    div.appendChild(b);
    box.appendChild(div);
  }

  function replace(html) {
    var doc = new DOMParser().parseFromString(html, 'text/html');
    var next = doc.querySelector('main');
    var cur = document.querySelector('main');
    if (!next || !cur) return false;
    cur.replaceWith(next);
    bind();
    return true;
  }

  function send(form, main) {
    var confirm = form.querySelector('.confirm');
    var buttons = form.querySelectorAll('input, button');
    buttons.forEach(function (b) { b.disabled = true; });
    var body = new URLSearchParams(new FormData(form));
    // Disabled radios aren't sent: add the picked answer back.
    var picked = form.querySelector('input[name="answer"]:checked');
    if (picked) body.set('answer', picked.value);
    fetch(form.action, { method: 'POST', body: body, credentials: 'omit', headers: { 'X-RC-Script': '1' } })
      .then(function (r) {
        if (!r.ok && r.status !== 503 && r.status !== 404) throw new Error(String(r.status));
        return r.text();
      })
      .then(function (html) {
        if (!replace(html)) throw new Error('page');
      })
      .catch(function () {
        buttons.forEach(function (b) { b.disabled = false; });
        main.setAttribute('data-state', 'error');
        status(main, words(main, 'failed', "We couldn't save your answer. Nothing was recorded. Check your connection and try again."), 'error');
        if (confirm) {
          confirm.hidden = false;
          var btn = confirm.querySelector('button');
          if (btn) btn.textContent = words(main, 'retry', 'Try again');
        }
      });
  }

  function bindPoll(main) {
    var form = main.querySelector('form.answer');
    var state = main.getAttribute('data-state');
    if (form) {
      var confirm = form.querySelector('.confirm');
      if (confirm && state !== 'error') confirm.hidden = true;
      form.addEventListener('change', function (e) {
        if (e.target && e.target.name === 'answer') send(form, main);
      });
      form.addEventListener('submit', function (e) {
        e.preventDefault();
        send(form, main);
      });
      if (state === 'pending') {
        status(main, words(main, 'saving', 'Saving your answer…'), 'neutral');
        whenVisible(function () { setTimeout(function () { send(form, main); }, 900); });
      }
    }
    var edit = main.querySelector('a.self');
    if (edit) {
      edit.addEventListener('click', function (e) {
        e.preventDefault();
        var said = main.querySelector('.said');
        var url = edit.getAttribute('href');
        fetch(url, { credentials: 'omit' }).then(function (r) { return r.text(); }).then(function (html) {
          if (replace(html)) {
            var box = document.querySelector('#cmt');
            if (box) box.focus();
          } else if (said) {
            location.assign(url);
          }
        });
      });
    }
    var next = main.getAttribute('data-next');
    if (next && state !== 'pending') setTimeout(function () { location.assign(next); }, 1200);
  }

  function bindSurvey(main) {
    var errors = main.querySelector('#errors');
    if (errors) errors.focus();
  }

  function bind() {
    var main = document.querySelector('main');
    if (!main) return;
    counters(main);
    if (main.hasAttribute('data-poll')) bindPoll(main);
    if (main.hasAttribute('data-survey')) bindSurvey(main);
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', bind);
  else bind();
})();
