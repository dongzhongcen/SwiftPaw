// 设置页里的“关于”：版本号、项目主页、开源许可和免责声明

import { AppVersion } from '../../wailsjs/go/main/App';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import { DISCLAIMER } from './disclaimer.js';
import { addSettingsSection } from './settings.js';
import { escapeHtml, guard } from './util.js';
import { isAndroid } from './platform.js';
import { nativeCall } from './native/android.js';
import { textDialog } from './dialog.js';

const REPO_URL = 'https://github.com/dongzhongcen/SwiftPaw';

export function initAbout() {
  addSettingsSection({
    html: () => `
      <section class="settings-section">
        <h2>关于</h2>
        <div class="settings-row">
          <div>
            <div class="settings-label">极拍 SwiftPaw</div>
            <div class="settings-hint">简洁的${isAndroid ? '' : ' Windows '}本地音乐播放器，可以用插件在线搜歌</div>
          </div>
          <div class="settings-value" id="about-version">版本 …</div>
        </div>
        <div class="settings-row">
          <div>
            <div class="settings-label">项目主页</div>
            <div class="settings-hint">源代码、问题反馈和更新日志都在这里</div>
          </div>
          <button class="btn" type="button" id="about-repo">打开 GitHub</button>
        </div>
        <div class="settings-row">
          <div>
            <div class="settings-label">开源许可</div>
            <div class="settings-hint">${
              isAndroid
                ? 'MIT 许可证；用到的第三方开源软件、字体和它们的许可证点右边查看'
                : 'MIT 许可证；用到的第三方开源软件和字体见安装文件夹里的 THIRD_PARTY_NOTICES.md'
            }</div>
          </div>
          ${isAndroid ? '<button class="btn" type="button" id="about-notices">查看</button>' : ''}
        </div>
        <div class="settings-row">
          <div>
            <div class="settings-label">免责声明</div>
            <div class="settings-hint" id="about-disclaimer">${escapeHtml(DISCLAIMER)}</div>
          </div>
        </div>
      </section>`,
    bind: (body) => {
      AppVersion()
        .then((version) => {
          body.querySelector('#about-version').textContent = `版本 ${version}`;
        })
        .catch(() => {});
      body.querySelector('#about-repo').addEventListener('click', () => openRepo());
      body.querySelector('#about-notices')?.addEventListener('click', guard(showNotices));
    },
  });
}

// 在系统默认浏览器里打开项目主页（Wails 的 WebView 里直接用 window.open 打不开外部浏览器）
function openRepo() {
  try {
    BrowserOpenURL(REPO_URL);
  } catch {
    window.open(REPO_URL, '_blank');
  }
}

// Android 版：安装包里带着许可证文件（LICENSE 和 THIRD_PARTY_NOTICES.md），在这里显示
async function showNotices() {
  const { text } = await nativeCall('readNotices');
  await textDialog({ title: '开源许可', text });
}
