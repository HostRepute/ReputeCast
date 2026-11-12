// The sponsor request form (M65): characters left under the texts, the
// chosen image's name, and the slot's issues when the slot changes.
(function () {
  'use strict';
  var form = document.querySelector('[data-sponsor-form]');
  if (!form) return;
  var lang = document.documentElement.lang || 'en';
  var rules = new Intl.PluralRules(lang);
  form.querySelectorAll('textarea[data-left]').forEach(function (ta) {
    var out = document.getElementById(ta.getAttribute('data-left') || '');
    if (!out) return;
    var max = parseInt(out.getAttribute('data-max'), 10);
    var update = function () {
      var n = Math.max(0, max - Array.from(ta.value).length);
      var pattern = rules.select(n) === 'one' ? out.getAttribute('data-one') : out.getAttribute('data-other');
      out.textContent = (pattern || '#').replace('#', n.toLocaleString(lang));
    };
    ta.addEventListener('input', update);
  });
  form.querySelectorAll('.sp-pick input[type=file]').forEach(function (input) {
    input.addEventListener('change', function () {
      var box = input.closest('.sp-file');
      var name = box && box.querySelector('[data-file-name]');
      if (name && input.files && input.files[0]) name.textContent = input.files[0].name;
      var keep = box && box.querySelector('input[type=hidden][name^="keep_"]');
      if (keep) keep.remove();
    });
  });
  var slot = form.querySelector('[data-sponsor-slot]');
  var issue = form.querySelector('select[name=issue]');
  if (!slot || !issue) return;
  slot.addEventListener('change', function () {
    var opt = slot.options[slot.selectedIndex];
    var list = [];
    try { list = JSON.parse(opt.getAttribute('data-issues') || '[]'); } catch (e) { list = []; }
    while (issue.options.length > 1) issue.remove(1);
    list.forEach(function (o) { issue.add(new Option(o.label, o.value)); });
    issue.selectedIndex = 0;
  });
})();
