// Grow's public pages work without JavaScript; this only adds "Copy" to a
// referral link (the field stays selectable without it).
(function () {
  var buttons = document.querySelectorAll('[data-copy]');
  for (var i = 0; i < buttons.length; i++) {
    (function (b) {
      var field = document.getElementById(b.getAttribute('data-copy'));
      if (!field) return;
      b.hidden = false;
      var label = b.textContent;
      b.addEventListener('click', function () {
        var done = function () {
          b.textContent = '✓ ' + label;
          setTimeout(function () { b.textContent = label; }, 1600);
        };
        if (navigator.clipboard && navigator.clipboard.writeText) {
          navigator.clipboard.writeText(field.value).then(done, function () { field.select(); });
        } else {
          field.select();
          done();
        }
      });
    })(buttons[i]);
  }
})();
