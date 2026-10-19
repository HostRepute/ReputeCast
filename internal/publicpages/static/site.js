// The publication website's small enhancements (every page works without
// them): a sign-in link's page continues by itself, the code's six boxes
// move along as you type or paste, "Send again" waits its minute, toasts
// close without a reload.
(function () {
  'use strict';

  // A sign-in link's page: continue for a person in a browser. Link
  // scanners that fetch the address run no scripts, so they never use it.
  var auto = document.querySelector('form[data-auto-submit]');
  if (auto && !document.body.hasAttribute('inert')) {
    window.setTimeout(function () {
      if (typeof auto.requestSubmit === 'function') auto.requestSubmit();
      else auto.submit();
    }, 300);
  }

  // The six boxes of the code.
  var code = document.querySelector('[data-code]');
  if (code) {
    var boxes = Array.prototype.slice.call(code.querySelectorAll('input'));
    var fill = function (start, digits) {
      for (var i = 0; i < digits.length && start + i < boxes.length; i++) boxes[start + i].value = digits[i];
      var next = Math.min(start + digits.length, boxes.length - 1);
      boxes[next].focus();
      if (boxes.every(function (b) { return b.value !== ''; })) {
        var form = code.closest('form');
        if (form && typeof form.requestSubmit === 'function') form.requestSubmit();
      }
    };
    boxes.forEach(function (box, i) {
      box.addEventListener('input', function () {
        var digits = box.value.replace(/\D/g, '');
        box.value = '';
        if (digits) fill(i, digits.split(''));
      });
      box.addEventListener('keydown', function (e) {
        if (e.key === 'Backspace' && box.value === '' && i > 0) {
          boxes[i - 1].value = '';
          boxes[i - 1].focus();
          e.preventDefault();
        } else if (e.key === 'ArrowLeft' && i > 0) {
          boxes[i - 1].focus();
        } else if (e.key === 'ArrowRight' && i < boxes.length - 1) {
          boxes[i + 1].focus();
        }
      });
      box.addEventListener('paste', function (e) {
        var text = (e.clipboardData || window.clipboardData).getData('text') || '';
        var digits = text.replace(/\D/g, '').slice(0, boxes.length);
        if (digits) {
          e.preventDefault();
          fill(0, digits.split(''));
        }
      });
    });
    if (!boxes.some(function (b) { return b.value; })) boxes[0].focus();
  }

  // "Send again in 0:42".
  var resend = document.querySelector('[data-resend]');
  if (resend) {
    var left = parseInt(resend.getAttribute('data-resend'), 10) || 0;
    var label = resend.getAttribute('data-label');
    var wait = resend.getAttribute('data-wait') || '';
    var show = function () {
      if (left <= 0) {
        resend.disabled = false;
        resend.textContent = label;
        return;
      }
      var m = Math.floor(left / 60), s = left % 60;
      resend.disabled = true;
      resend.textContent = wait.replace(/\d+:\d\d/, m + ':' + (s < 10 ? '0' : '') + s);
    };
    show();
    if (left > 0) {
      var timer = window.setInterval(function () {
        left--;
        show();
        if (left <= 0) window.clearInterval(timer);
      }, 1000);
    }
  }

  // Toasts close where they are.
  document.addEventListener('click', function (e) {
    var x = e.target.closest && e.target.closest('.st-toast .x');
    if (!x) return;
    e.preventDefault();
    var toast = x.closest('.st-toast');
    if (toast) toast.remove();
  });
})();
