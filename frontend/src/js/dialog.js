// 自定义的对话框：输入框（prompt）、确认框（confirm）、选择框（pick）。
// Wails 的 WebView 里 window.prompt 用不了，所以用 HTML 的 <dialog> 自己做一个。
// 每个函数都返回 Promise，用法和原生的差不多：const name = await promptDialog({...})

import { escapeHtml } from './util.js';

let dialog;

function ensureDialog() {
  if (dialog) return dialog;
  dialog = document.createElement('dialog');
  dialog.id = 'dialog';
  dialog.className = 'dialog';
  document.body.appendChild(dialog);
  return dialog;
}

// open 显示对话框，render 负责填内容并在用户做出选择时调用 done(结果)
function open(render) {
  const d = ensureDialog();
  return new Promise((resolve) => {
    let finished = false;
    const done = (value) => {
      if (finished) return;
      finished = true;
      d.close();
      resolve(value);
    };
    // 按 Esc 或点关闭时，结果是 null
    d.oncancel = (event) => {
      event.preventDefault();
      done(null);
    };
    d.onclick = (event) => {
      if (event.target === d) done(null); // 点到对话框外面的遮罩
    };
    render(d, done);
    d.showModal();
    const focusable = d.querySelector('[autofocus]') || d.querySelector('input, button.primary');
    focusable?.focus();
  });
}

function frame(title, body, buttons) {
  return `
    <form method="dialog" class="dialog-body">
      <h2 class="dialog-title">${escapeHtml(title)}</h2>
      ${body}
      <div class="dialog-buttons">${buttons}</div>
    </form>`;
}

// promptDialog 让用户输入一段文字，取消时返回 null
export function promptDialog({ title, label = '', value = '', placeholder = '', okText = '确定' }) {
  return open((d, done) => {
    d.innerHTML = frame(
      title,
      `<label class="dialog-field">
        <span>${escapeHtml(label)}</span>
        <input class="text-input" name="value" maxlength="200" autocomplete="off"
          value="${escapeHtml(value)}" placeholder="${escapeHtml(placeholder)}" autofocus />
      </label>`,
      `<button type="button" class="btn" data-role="cancel">取消</button>
       <button type="submit" class="btn primary">${escapeHtml(okText)}</button>`,
    );
    const input = d.querySelector('input');
    input.select();
    d.querySelector('form').onsubmit = (event) => {
      event.preventDefault();
      const text = input.value.trim();
      if (text) done(text);
      else input.focus();
    };
    d.querySelector('[data-role=cancel]').onclick = () => done(null);
  });
}

// confirmDialog 让用户确认一个操作，返回 true / false
export async function confirmDialog({ title, message, okText = '确定', danger = false }) {
  const result = await open((d, done) => {
    d.innerHTML = frame(
      title,
      `<p class="dialog-message">${escapeHtml(message)}</p>`,
      `<button type="button" class="btn" data-role="cancel">取消</button>
       <button type="submit" class="btn primary ${danger ? 'danger' : ''}" autofocus>${escapeHtml(okText)}</button>`,
    );
    d.querySelector('form').onsubmit = (event) => {
      event.preventDefault();
      done(true);
    };
    d.querySelector('[data-role=cancel]').onclick = () => done(false);
  });
  return result === true;
}

// pickDialog 从一组选项里选一个，返回选中项的 value；取消返回 null。
// items: [{ value, label, hint }]
export function pickDialog({ title, items, emptyText = '没有可选的项目' }) {
  return open((d, done) => {
    const list = items.length
      ? items
          .map(
            (item, i) => `
          <li><button type="button" class="pick-item" data-index="${i}">
            <span class="pick-label">${escapeHtml(item.label)}</span>
            <span class="pick-hint">${escapeHtml(item.hint ?? '')}</span>
          </button></li>`,
          )
          .join('')
      : `<li class="pick-empty">${escapeHtml(emptyText)}</li>`;
    d.innerHTML = frame(
      title,
      `<ul class="pick-list">${list}</ul>`,
      `<button type="button" class="btn" data-role="cancel" autofocus>取消</button>`,
    );
    d.querySelectorAll('.pick-item').forEach((button) => {
      button.onclick = () => done(items[Number(button.dataset.index)].value);
    });
    d.querySelector('[data-role=cancel]').onclick = () => done(null);
  });
}
