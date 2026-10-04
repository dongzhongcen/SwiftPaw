// 设置页：外观（主题）和关于

import { saveConfig } from './config.js';
import { registerView } from './views.js';
import { applyTheme, currentTheme, themes } from './theme.js';
import { escapeHtml } from './util.js';

// sections 让其他模块往设置页里加内容（返回 HTML 和绑定事件的函数）
const extraSections = [];

export function addSettingsSection(section) {
  extraSections.push(section);
}

export function initSettings() {
  registerView('settings', () => ({
    title: '设置',
    subtitle: '外观和关于',
    actions: [],
    render: renderSettings,
  }));
}

function renderSettings(body) {
  body.innerHTML = `
    <div class="settings">
      <section class="settings-section">
        <h2>外观</h2>
        <div class="settings-row">
          <div>
            <div class="settings-label">主题</div>
            <div class="settings-hint">跟随系统时，会随 Windows 的深色/浅色模式自动切换</div>
          </div>
          <div class="segmented" id="theme-picker" role="radiogroup" aria-label="主题">
            ${themes
              .map(
                (t) => `<button type="button" role="radio" data-theme-value="${t.value}"
                  aria-checked="${t.value === currentTheme()}">${escapeHtml(t.label)}</button>`,
              )
              .join('')}
          </div>
        </div>
      </section>
      ${extraSections.map((s) => s.html()).join('')}
    </div>`;

  body.querySelectorAll('[data-theme-value]').forEach((button) => {
    button.addEventListener('click', () => {
      applyTheme(button.dataset.themeValue);
      saveConfig({ theme: button.dataset.themeValue });
      body.querySelectorAll('[data-theme-value]').forEach((b) => {
        b.setAttribute('aria-checked', String(b === button));
      });
    });
  });
  extraSections.forEach((s) => s.bind?.(body));
}
