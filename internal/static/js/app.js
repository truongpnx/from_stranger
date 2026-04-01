/* ═══════════════════════════════════════════
   From Stranger – Main JS
   Handles: mobile tabs, sub-tabs, card lifecycle,
   keyboard & swipe reactions, flip animations
   ═══════════════════════════════════════════ */

(function () {
  'use strict';

  // ─── State ───
  let currentSentence = null;
  let isAnimating = false;
  let touchStartX = 0;
  let touchStartY = 0;

  const REACTION_ICONS = {
    heart: '❤️',
    hate: '💔',
    ignore: '🙈',
  };

  // ─── DOM helpers ───
  const $ = (sel, ctx) => (ctx || document).querySelector(sel);
  const $$ = (sel, ctx) => [...(ctx || document).querySelectorAll(sel)];

  // ─── Mobile bottom nav ───
  function initMobileNav() {
    $$('.nav-btn').forEach((btn) => {
      btn.addEventListener('click', () => {
        const tab = btn.dataset.tab;
        $$('.nav-btn').forEach((b) => b.classList.remove('active'));
        btn.classList.add('active');
        $$('.panel').forEach((p) => {
          p.classList.toggle('active', p.dataset.tabContent === tab);
        });
      });
    });
  }

  // ─── Sub-tabs (publish panel) ───
  function initSubTabs() {
    $$('.sub-tab').forEach((btn) => {
      btn.addEventListener('click', () => {
        const target = btn.dataset.subtab;
        $$('.sub-tab').forEach((b) => b.classList.remove('active'));
        btn.classList.add('active');
        $$('.sub-content').forEach((c) => {
          c.classList.toggle('active', c.dataset.subtabContent === target);
        });
      });
    });
  }

  // ─── Card lifecycle ───
  function fetchSentence() {
    if (isAnimating) return;
    showLoading();
    fetch('/random', { credentials: 'same-origin' })
      .then((r) => r.json())
      .then((data) => {
        currentSentence = data;
        flipToSentence(data.text);
      })
      .catch(() => {
        currentSentence = null;
        flipToSentence('Could not load. Tap to retry.');
      });
  }

  function showLoading() {
    const content = $('#card-content');
    if (content) {
      content.innerHTML = '<div class="loading-dots"><span></span><span></span><span></span></div>';
    }
  }

  function flipToSentence(text) {
    const card = $('#sentence-card');
    const content = $('#card-content');
    if (!card || !content) return;

    isAnimating = true;
    card.classList.remove('flip-in');
    card.classList.add('flip-out');

    setTimeout(() => {
      content.innerHTML = '<p class="sentence-text">' + escapeHtml(text) + '</p>';
      card.classList.remove('flip-out');
      card.classList.add('flip-in');
      setTimeout(() => {
        isAnimating = false;
      }, 300);
    }, 300);
  }

  function showReactionThenNext(type) {
    if (isAnimating || !currentSentence) return;
    isAnimating = true;

    const card = $('#sentence-card');
    const content = $('#card-content');
    if (!card || !content) return;

    // Send reaction
    const sentenceId = currentSentence.id;
    sendReaction(sentenceId, type);

    // Flip to show reaction icon
    card.classList.remove('flip-in');
    card.classList.add('flip-out');

    setTimeout(() => {
      const icon = REACTION_ICONS[type] || '⭕';
      content.innerHTML = '<div class="reaction-display">' + icon + '</div>';
      card.classList.remove('flip-out');
      card.classList.add('flip-in');

      // After showing icon 1s, load next
      setTimeout(() => {
        isAnimating = false;
        fetchSentence();
      }, 1000);
    }, 300);
  }

  function sendReaction(sentenceId, type) {
    if (!sentenceId || sentenceId === 'fallback') return;
    const body = new URLSearchParams({ sentence_id: sentenceId, reaction_type: type });
    fetch('/react', { method: 'POST', body: body, credentials: 'same-origin' }).catch(() => {});
  }

  // ─── Keyboard ───
  function initKeyboard() {
    document.addEventListener('keydown', (e) => {
      if (e.target.tagName === 'TEXTAREA' || e.target.tagName === 'INPUT') return;
      switch (e.key) {
        case 'ArrowRight':
          e.preventDefault();
          showReactionThenNext('heart');
          break;
        case 'ArrowLeft':
          e.preventDefault();
          showReactionThenNext('hate');
          break;
        case 'ArrowUp':
          e.preventDefault();
          showReactionThenNext('ignore');
          break;
      }
    });
  }

  // ─── Touch / Swipe ───
  function initSwipe() {
    const scene = $('.card-scene');
    if (!scene) return;

    scene.addEventListener('touchstart', (e) => {
      touchStartX = e.changedTouches[0].clientX;
      touchStartY = e.changedTouches[0].clientY;
    }, { passive: true });

    scene.addEventListener('touchend', (e) => {
      const dx = e.changedTouches[0].clientX - touchStartX;
      const dy = e.changedTouches[0].clientY - touchStartY;
      const absDx = Math.abs(dx);
      const absDy = Math.abs(dy);
      const threshold = 50;

      if (absDx < threshold && absDy < threshold) return;

      if (absDy > absDx && dy < 0) {
        showReactionThenNext('ignore');
      } else if (absDx > absDy) {
        showReactionThenNext(dx > 0 ? 'heart' : 'hate');
      }
    }, { passive: true });
  }

  // ─── Utils ───
  function escapeHtml(text) {
    const d = document.createElement('div');
    d.textContent = text;
    return d.innerHTML;
  }

  // ─── Publish form ───
  function initPublishForm() {
    const form = $('form[hx-post="/publish"]');
    const textarea = form && $('textarea', form);
    const btn = form && $('.btn-publish', form);
    const result = $('#publish-result');
    if (!form || !textarea || !btn) return;

    textarea.addEventListener('input', () => {
      btn.disabled = textarea.value.trim().length === 0;
    });

    document.addEventListener('htmx:afterSwap', (e) => {
      if (e.detail.target !== result) return;
      const isSuccess = !!$('.status.ok', result);
      if (isSuccess) {
        textarea.value = '';
        btn.disabled = true;
      }
      setTimeout(() => { result.innerHTML = ''; }, 5000);
    });
  }

  // ─── Init ───
  function init() {
    initMobileNav();
    initSubTabs();
    initKeyboard();
    initSwipe();
    initPublishForm();
    fetchSentence();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
