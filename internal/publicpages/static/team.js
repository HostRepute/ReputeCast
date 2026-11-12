// The team pages (M65): the seats stepper, and the team page's total as
// the seats change (the server writes the same total; this keeps it in
// step while typing). Nothing here is needed for the forms to work.
(function () {
  'use strict';
  document.querySelectorAll('.tm-step').forEach(function (step) {
    var input = step.querySelector('input[type=number]');
    if (!input) return;
    step.querySelectorAll('[data-step]').forEach(function (b) {
      b.addEventListener('click', function () {
        var min = parseInt(input.min, 10) || 1;
        var max = parseInt(input.max, 10) || 1000;
        var n = parseInt(input.value, 10);
        if (isNaN(n)) n = min;
        n = Math.min(max, Math.max(min, n + parseInt(b.getAttribute('data-step'), 10)));
        input.value = String(n);
        input.dispatchEvent(new Event('input', { bubbles: true }));
      });
    });
  });
  var box = document.querySelector('[data-team-total]');
  var seats = document.getElementById('tm-seats');
  if (!box || !seats || !box.querySelector('[data-amount]')) return;
  var d = box.dataset;
  var unit = parseInt(d.unit, 10);
  var min = parseInt(d.min, 10);
  var max = parseInt(d.max, 10);
  var rules = new Intl.PluralRules(d.lang || 'en');
  var thousands = function (n) { return String(n).replace(/\B(?=(\d{3})+(?!\d))/g, ','); };
  var money = function (c) {
    if (d.whole === 'true') return d.sym + thousands(c);
    var f = c % 100;
    return d.sym + thousands(Math.floor(c / 100)) + (f ? '.' + String(f).padStart(2, '0') : '');
  };
  seats.addEventListener('input', function () {
    var n = parseInt(seats.value, 10);
    if (isNaN(n) || n < min || n > max) return;
    var form = rules.select(n) === 'one' ? d.one : d.other;
    box.querySelector('[data-times]').textContent = form.replace('#', n.toLocaleString(d.lang || 'en'));
    box.querySelector('[data-amount]').textContent = money(unit * n);
  });
})();
