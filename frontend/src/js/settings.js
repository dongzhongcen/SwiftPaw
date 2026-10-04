// 设置页：外观（配色、风格、减少模糊效果）、背景图片和关于

import { RemoveBackgroundImage, SelectBackgroundImage } from '../../wailsjs/go/main/App';
import { getConfig, saveConfig } from './config.js';
import { registerView } from './views.js';
import { applyStyle, applyTheme, currentStyle, currentTheme, styles, themes } from './theme.js';
import { applyBackground, applyReduceBlur, backgroundUrl, reduceBlurEnabled } from './effects.js';
import { escapeHtml, guard, toast } from './util.js';
import { icons } from './icons.js';
import { isAndroid } from './platform.js';

// sections 让其他模块往设置页里加内容（返回 HTML 和绑定事件的函数）
const extraSections = [];

export function addSettingsSection(section) {
  extraSections.push(section);
}

export function initSettings() {
  registerView('settings', () => ({
    title: '设置',
    subtitle: '外观、背景图片和关于',
    actions: [],
    render: renderSettings,
  }));
}

// 背景设置的默认值和配置里存的合在一起
function background() {
  return { image: '', blur: 0, dim: 40, ...(getConfig().background || {}) };
}

function saveBackground(patch) {
  const next = { ...background(), ...patch };
  saveConfig({ background: next });
  applyBackground(next);
  return next;
}

// 风格的小预览：一块底色，上面一条文字色、一个强调色的圆点。“极简”是左黑右白
function swatch(style) {
  if (!style.swatch) return '<span class="style-swatch plain" aria-hidden="true"><i></i><b></b></span>';
  const [bg, fg, accent] = style.swatch;
  return `<span class="style-swatch" aria-hidden="true" style="--sw-bg:${bg};--sw-fg:${fg};--sw-accent:${accent}"><i></i><b></b></span>`;
}

function renderSettings(body) {
  body.innerHTML = `
    <div class="settings">
      <section class="settings-section">
        <h2>外观</h2>
        <div class="settings-row">
          <div>
            <div class="settings-label">配色</div>
            <div class="settings-hint">跟随系统时，会随 ${isAndroid ? '手机' : 'Windows'} 的深色/浅色模式自动切换；选了风格时由风格决定</div>
          </div>
          <div class="segmented" id="theme-picker" role="radiogroup" aria-label="配色">
            ${themes
              .map(
                (t) => `<button type="button" role="radio" data-theme-value="${t.value}">${escapeHtml(t.label)}</button>`,
              )
              .join('')}
          </div>
        </div>
        <div class="settings-row stacked">
          <div>
            <div class="settings-label">风格</div>
            <div class="settings-hint">换一套字体、颜色和小动画，马上生效</div>
          </div>
          <div class="style-picker" id="style-picker" role="radiogroup" aria-label="风格">
            ${styles
              .map(
                (s) => `<button type="button" class="style-card" role="radio" data-style-value="${s.value}">
                  ${swatch(s)}
                  <span class="style-name">${escapeHtml(s.label)}</span>
                  <span class="style-hint">${escapeHtml(s.hint)}</span>
                </button>`,
              )
              .join('')}
          </div>
        </div>
        <div class="settings-row">
          <div>
            <div class="settings-label">减少模糊效果</div>
            <div class="settings-hint">关掉磨砂玻璃这类模糊效果，在老电脑和手机上更流畅${isAndroid ? '（手机上默认打开）' : ''}</div>
          </div>
          <label class="switch" title="减少模糊效果">
            <input type="checkbox" id="reduce-blur" aria-label="减少模糊效果" />
            <span class="switch-track"></span>
          </label>
        </div>
      </section>

      <section class="settings-section">
        <h2>背景图片</h2>
        <div class="settings-row">
          <div class="bg-current">
            <span class="bg-thumb" id="bg-thumb">${icons.file}</span>
            <div>
              <div class="settings-label">自定义背景</div>
              <div class="settings-hint">支持 JPG、PNG、WebP、GIF，最大 20 MB。图片会复制一份保存在软件里，原图删了也不影响</div>
            </div>
          </div>
          <div class="settings-buttons">
            <button class="btn" type="button" id="bg-pick">${icons.folder}<span>选择图片</span></button>
            <button class="btn" type="button" id="bg-remove">${icons.trash}<span>移除</span></button>
          </div>
        </div>
        <div class="settings-row">
          <div>
            <div class="settings-label">模糊</div>
            <div class="settings-hint">0 是不模糊</div>
          </div>
          <div class="slider-field">
            <input type="range" class="range" id="bg-blur" min="0" max="30" step="1" aria-label="背景模糊" />
            <span class="slider-value" id="bg-blur-value"></span>
          </div>
        </div>
        <div class="settings-row">
          <div>
            <div class="settings-label">遮罩</div>
            <div class="settings-hint">用主题的底色盖住图片，越大图片越淡、文字越清楚</div>
          </div>
          <div class="slider-field">
            <input type="range" class="range" id="bg-dim" min="0" max="80" step="1" aria-label="背景遮罩" />
            <span class="slider-value" id="bg-dim-value"></span>
          </div>
        </div>
      </section>
      ${extraSections.map((s) => s.html()).join('')}
    </div>`;

  bindAppearance(body);
  bindBackground(body);
  extraSections.forEach((s) => s.bind?.(body));
}

function bindAppearance(body) {
  const update = () => {
    const style = currentStyle();
    body.querySelectorAll('[data-theme-value]').forEach((b) => {
      b.setAttribute('aria-checked', String(!style && b.dataset.themeValue === currentTheme()));
    });
    body.querySelector('#theme-picker').classList.toggle('inactive', !!style);
    body.querySelectorAll('[data-style-value]').forEach((b) => {
      b.setAttribute('aria-checked', String(b.dataset.styleValue === style));
    });
  };

  // 点配色时回到“极简”风格，免得点了没反应
  body.querySelectorAll('[data-theme-value]').forEach((button) => {
    button.addEventListener('click', () => {
      applyTheme(button.dataset.themeValue);
      applyStyle('');
      saveConfig({ theme: button.dataset.themeValue, style: '' });
      update();
    });
  });
  body.querySelectorAll('[data-style-value]').forEach((button) => {
    button.addEventListener('click', () => {
      applyStyle(button.dataset.styleValue);
      saveConfig({ style: button.dataset.styleValue });
      update();
    });
  });

  const reduce = body.querySelector('#reduce-blur');
  reduce.checked = reduceBlurEnabled(getConfig().reduceBlur);
  reduce.addEventListener('change', () => {
    const value = reduce.checked ? 'on' : 'off';
    applyReduceBlur(value);
    saveConfig({ reduceBlur: value });
  });
  update();
}

function bindBackground(body) {
  const blur = body.querySelector('#bg-blur');
  const dim = body.querySelector('#bg-dim');
  const remove = body.querySelector('#bg-remove');
  const thumb = body.querySelector('#bg-thumb');

  const update = () => {
    const bg = background();
    const has = !!bg.image;
    blur.value = String(bg.blur);
    dim.value = String(bg.dim);
    blur.disabled = dim.disabled = remove.disabled = !has;
    setFill(blur, (bg.blur / 30) * 100);
    setFill(dim, (bg.dim / 80) * 100);
    body.querySelector('#bg-blur-value').textContent = `${bg.blur} px`;
    body.querySelector('#bg-dim-value').textContent = `${bg.dim}%`;
    thumb.classList.toggle('has-image', has);
    thumb.style.backgroundImage = has ? `url("${backgroundUrl(bg.image)}")` : '';
  };

  body.querySelector('#bg-pick').addEventListener(
    'click',
    guard(async () => {
      const image = await SelectBackgroundImage();
      if (!image) return; // 取消了
      saveBackground({ image });
      update();
      toast('已设置背景图片');
    }),
  );
  remove.addEventListener(
    'click',
    guard(async () => {
      saveBackground({ image: '' });
      update();
      await RemoveBackgroundImage();
      toast('已移除背景图片');
    }),
  );
  blur.addEventListener('input', () => {
    saveBackground({ blur: Number(blur.value) });
    update();
  });
  dim.addEventListener('input', () => {
    saveBackground({ dim: Number(dim.value) });
    update();
  });
  update();
}

function setFill(range, percent) {
  range.style.setProperty('--fill', `${Math.min(100, Math.max(0, percent))}%`);
}
