// JS léger écrit à la main (aucun framework, aucune ressource externe).
// - confirmation avant les suppressions
// - année courante dans le pied de page si présent
document.addEventListener("DOMContentLoaded", function () {
  document.querySelectorAll("form[data-confirm]").forEach(function (form) {
    form.addEventListener("submit", function (e) {
      if (!window.confirm(form.getAttribute("data-confirm") || "Confirmer ?")) {
        e.preventDefault();
      }
    });
  });
  var yearEl = document.getElementById("annee");
  if (yearEl) {
    yearEl.textContent = String(new Date().getFullYear());
  }
});
