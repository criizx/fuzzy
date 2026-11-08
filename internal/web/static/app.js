(() => {
  
  // Keep experiment settings visible until the user explicitly collapses them.
  let activePlot = 'cuts';
  const selectPlot = (name, focus = false) => {
    activePlot = name;
    for (const button of document.querySelectorAll('[data-plot]')) {
      const selected = button.dataset.plot === name;
      button.setAttribute('aria-selected', String(selected));
      button.tabIndex = selected ? 0 : -1;
      document.getElementById(button.getAttribute('aria-controls')).hidden = !selected;
      if (selected && focus) button.focus();
    }
  };
  document.body.addEventListener('click', event => {
    const tab = event.target.closest('[data-plot]');
    if (tab) selectPlot(tab.dataset.plot);
  });
  document.body.addEventListener('keydown', event => {
    const tab = event.target.closest('[data-plot]');
    if (!tab || !['ArrowLeft','ArrowRight','Home','End'].includes(event.key)) return;
    event.preventDefault();
    const name = event.key === 'Home' ? 'cuts' : event.key === 'End' ? 'plane' : activePlot === 'cuts' ? 'plane' : 'cuts';
    selectPlot(name, true);
  });
  document.body.addEventListener('htmx:afterSwap', () => selectPlot(activePlot));

  const root = document.documentElement;
  const toggle = document.querySelector('#theme-toggle');
  const syncTheme = () => toggle?.setAttribute('aria-pressed', String(root.dataset.theme === 'dark'));
  syncTheme();
  toggle?.addEventListener('click', () => {
    root.dataset.theme = root.dataset.theme === 'dark' ? 'light' : 'dark';
    try { localStorage.setItem('ln-theme', root.dataset.theme); } catch (_) {}
    syncTheme();
  });
  for (const input of document.querySelectorAll('input[name^=sigma]')) {
    const validateSigma = () => input.setCustomValidity(input.value !== '' && Number(input.value) <= 0 ? 'σ должно быть больше нуля' : '');
    input.addEventListener('input', validateSigma);
    validateSigma();
  }
  const range = document.querySelector('#alpha-range');
  const number = document.querySelector('#alpha-number');
  range?.addEventListener('input', () => { number.value = range.value; });
  number?.addEventListener('input', () => {
    if (number.value === '' || !number.validity.valid) return;
    range.value = number.value;
  });
  const algorithm = document.querySelector('#algorithm');
  const syncAlgorithm = () => {
    const hierarchy = algorithm?.value === 'agglo';
    const k = document.querySelector('#k-field');
    const threshold = document.querySelector('#threshold-field');
    if (k) k.hidden = hierarchy;
    if (threshold) threshold.hidden = !hierarchy;
  };
  algorithm?.addEventListener('change', syncAlgorithm);
  syncAlgorithm();
  const mode = document.querySelector('#generation-mode');
  const syncMode = () => {
    for (const id of ['groups-field', 'noise-field']) {
      const field = document.getElementById(id);
      if (!field) continue;
      const hidden = mode.value !== 'clouds';
      field.hidden = hidden;
      for (const input of field.querySelectorAll('input')) input.disabled = hidden;
    }
  };
  mode?.addEventListener('change', syncMode);
  if (mode) syncMode();
  // A connection that never completes must eventually return control to the user.
  if (window.htmx) window.htmx.config.timeout = 30000;
  const resultTarget = event => event.detail.target || document.querySelector('#view, #opsview');
  const showRequestError = (event, text) => {
    const target = resultTarget(event);
    if (!target) return;
    target.setAttribute('aria-busy', 'false');
    let message = target.querySelector('[data-request-error]');
    if (!message) {
      message = document.createElement('p');
      message.className = 'error';
      message.dataset.requestError = '';
      message.setAttribute('role', 'alert');
      target.prepend(message);
    }
    message.textContent = text;
  };
  document.body.addEventListener('htmx:beforeRequest', event => {
    const target = resultTarget(event);
    target?.querySelector('[data-request-error]')?.remove();
    target?.setAttribute('aria-busy', 'true');
  });
  document.body.addEventListener('htmx:afterRequest', event => {
    resultTarget(event)?.setAttribute('aria-busy', 'false');
  });
  document.body.addEventListener('htmx:responseError', event => {
    showRequestError(event, 'Сервер не смог выполнить расчёт. Предыдущий результат не обновлён. Повторите действие.');
  });
  document.body.addEventListener('htmx:sendError', event => {
    showRequestError(event, 'Нет связи с сервером. Предыдущий результат не обновлён. Проверьте соединение и повторите действие.');
  });
  document.body.addEventListener('htmx:timeout', event => {
    showRequestError(event, 'Сервер не ответил вовремя. Предыдущий результат не обновлён. Повторите действие.');
  });
})();
