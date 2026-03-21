/* shared.js — Common utilities for all Next templates */

function toast(msg) {
  var t = document.getElementById('toast');
  if (!t) return;
  t.textContent = msg;
  t.classList.add('show');
  setTimeout(function() { t.classList.remove('show'); }, 3000);
}

function escapeHtml(s) {
  if (!s) return '';
  var d = document.createElement('div');
  d.textContent = s;
  return d.innerHTML;
}
var esc = escapeHtml;

async function apiFetch(url, opts) {
  var r = await fetch(url, opts);
  if (r.status === 401) { location.href = '/login'; return null; }
  if (r.status === 403) { toast('Sem permissao'); return null; }
  return r;
}

async function getError(r) {
  try { var d = await r.json(); return d.error || 'Erro desconhecido'; }
  catch(e) { return 'Erro desconhecido'; }
}

async function doLogout() {
  await fetch('/api/logout', {method:'POST'});
  location.href = '/login';
}

function formatTimeRelative(ts) {
  var d = new Date(ts * 1000);
  var now = new Date();
  var diff = now - d;
  if (diff < 86400000 && d.getDate() === now.getDate()) {
    return d.toLocaleTimeString('pt-BR', {hour:'2-digit', minute:'2-digit'});
  }
  if (diff < 172800000) return 'Ontem ' + d.toLocaleTimeString('pt-BR', {hour:'2-digit', minute:'2-digit'});
  return d.toLocaleDateString('pt-BR', {day:'2-digit', month:'2-digit'}) + ' ' + d.toLocaleTimeString('pt-BR', {hour:'2-digit', minute:'2-digit'});
}

function formatTimeAbsolute(ts) {
  return new Date(ts * 1000).toLocaleString('pt-BR', {
    day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit'
  });
}

function initAuth(cb) {
  fetch('/api/auth/status').then(function(r) { return r.json(); }).then(function(d) {
    if (d.enabled && d.user) {
      var el = document.getElementById('auth-user');
      if (el) { el.textContent = d.user.username; el.classList.remove('hidden'); }
      var lo = document.getElementById('logout-link');
      if (lo) lo.classList.remove('hidden');
      if (cb) cb(d.user);
    }
  }).catch(function() {});
}
