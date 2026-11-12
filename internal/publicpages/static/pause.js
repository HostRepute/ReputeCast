// The cancel page's "Pause instead" (M65): the return date and the button
// follow the length chosen.
(function () {
  'use strict';
  var back = document.querySelector('[data-pause-back]');
  var button = document.querySelector('[data-pause-button]');
  if (!back || !button) return;
  document.querySelectorAll('input[name=months][data-back]').forEach(function (r) {
    r.addEventListener('change', function () {
      if (!r.checked) return;
      back.textContent = r.getAttribute('data-back');
      button.textContent = r.getAttribute('data-button');
    });
  });
})();
