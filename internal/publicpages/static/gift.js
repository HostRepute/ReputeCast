// The gift pages (design expansion phase 6, X17). They work without this
// script: a delivery date is read in the publication's time zone, and the
// gift link can be copied by hand. With it, the date is read in the
// buyer's own time zone, the message counts down its characters, and the
// gift link copies with one click.
(function () {
  'use strict';

  function timeZone() {
    var input = document.querySelector('[data-gift-tz]');
    var note = document.querySelector('[data-gift-zone]');
    if (!input) return;
    var zone = '';
    try {
      zone = Intl.DateTimeFormat().resolvedOptions().timeZone || '';
    } catch (e) {
      zone = '';
    }
    if (!zone || zone === input.value) return;
    var was = input.value;
    input.value = zone;
    if (note && was) note.textContent = note.textContent.replace(was, zone);
    if (note) note.setAttribute('data-gift-zone', zone);
  }

  function counter() {
    var box = document.querySelector('[data-gift-message]');
    var out = document.getElementById('gf-left');
    if (!box || !out) return;
    var max = Number(box.getAttribute('maxlength')) || 300;
    var one = out.getAttribute('data-one') || '';
    var other = out.getAttribute('data-other') || '';
    var update = function () {
      var left = Math.max(0, max - Array.from(box.value).length);
      out.textContent = (left === 1 ? one : other).replace('{n}', String(left));
    };
    box.addEventListener('input', update);
    update();
  }

  function copy() {
    var button = document.querySelector('[data-gift-copy]');
    var field = document.querySelector('[data-gift-link]');
    if (!button || !field) return;
    var label = button.textContent;
    button.addEventListener('click', function () {
      var done = function () {
        button.textContent = button.getAttribute('data-copied') || label;
        setTimeout(function () { button.textContent = label; }, 2000);
      };
      var url = field.getAttribute('data-url') || field.value;
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(url).then(done, function () { field.select(); });
      } else {
        field.select();
      }
    });
  }

  function ready() {
    timeZone();
    counter();
    copy();
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', ready);
  else ready();
})();
