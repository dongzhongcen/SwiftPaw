// 插件页：查看已安装的插件，安装、卸载、启用/停用，填写插件设置。
// 插件是一个 .js 文件，负责告诉 SwiftPaw 怎么搜索歌曲、怎么拿到播放地址和歌词。

import {
  InstallPluginFromFile,
  InstallPluginFromURL,
  PluginUserVariables,
  Plugins,
  ReloadPlugins,
  SetPluginEnabled,
  SetPluginUserVariables,
  UninstallPlugin,
} from '../../wailsjs/go/main/App';
import { icons } from './icons.js';
import { confirmDialog, formDialog, promptDialog } from './dialog.js';
import { DISCLAIMER } from './disclaimer.js';
import { registerView, renderView } from './views.js';
import { escapeHtml, guard, toast } from './util.js';

export function initPlugins() {
  registerView('plugins', loadPluginsView);
}

async function loadPluginsView() {
  const plugins = (await Plugins()) || [];
  const enabled = plugins.filter((p) => p.enabled && !p.error).length;
  return {
    title: '插件',
    subtitle: plugins.length ? `${plugins.length} 个插件 · ${enabled} 个启用` : '在线搜索和播放靠插件提供',
    actions: [
      { label: '从文件安装', icon: icons.file, primary: true, run: installFromFile },
      { label: '从网址安装', icon: icons.link, run: installFromURL },
      { label: '重新加载', icon: icons.refresh, run: reload },
    ],
    render: (body) => renderPlugins(body, plugins),
  };
}

function renderPlugins(body, plugins) {
  if (!plugins.length) {
    body.innerHTML = `
      <div class="empty-state">
        <p>还没有安装插件</p>
        <p class="muted">点右上角的“从文件安装”选择一个 .js 插件，或者“从网址安装”粘贴插件地址</p>
        <p class="muted">项目里的 examples/plugins/archive-org.js 是一个可以直接用的示例插件</p>
      </div>`;
    return;
  }
  body.innerHTML = `<div class="plugin-list">${plugins.map(pluginCard).join('')}</div>`;

  body.querySelectorAll('[data-toggle]').forEach((input) => {
    input.addEventListener('change', guard(async () => {
      const plugin = plugins.find((p) => p.id === input.dataset.toggle);
      try {
        await SetPluginEnabled(plugin.id, input.checked);
        toast(`${input.checked ? '已启用' : '已停用'}：${plugin.platform || plugin.id}`);
      } finally {
        await renderView();
      }
    }));
  });
  body.querySelectorAll('[data-settings]').forEach((button) => {
    button.addEventListener('click', guard(() => openSettings(plugins.find((p) => p.id === button.dataset.settings))));
  });
  body.querySelectorAll('[data-uninstall]').forEach((button) => {
    button.addEventListener('click', guard(async () => {
      const plugin = plugins.find((p) => p.id === button.dataset.uninstall);
      const name = plugin.platform || plugin.id;
      const ok = await confirmDialog({
        title: '卸载插件',
        message: `确定要卸载「${name}」吗？插件文件会被删除，已经收藏的在线歌曲会保留，但需要重新安装插件才能播放。`,
        okText: '卸载',
        danger: true,
      });
      if (!ok) return;
      await UninstallPlugin(plugin.id);
      toast(`已卸载：${name}`);
      await renderView();
    }));
  });
}

// openSettings 打开插件设置对话框：插件声明的每个用户变量（比如 API Key）一个输入框。
// 保存后插件会重新加载，新的设置马上生效
async function openSettings(plugin) {
  const name = plugin.platform || plugin.id;
  const saved = (await PluginUserVariables(plugin.id)) || {};
  const values = await formDialog({
    title: `插件设置：${name}`,
    message: '这些内容只保存在这台电脑上，由插件自己使用。',
    fields: plugin.userVariables.map((v) => ({ name: v.key, label: v.name || v.key, hint: v.hint, value: saved[v.key] })),
  });
  if (!values) return; // 用户点了取消
  const info = await SetPluginUserVariables(plugin.id, values);
  if (info?.error) toast(`设置已保存，但插件重新加载失败：${info.error}`, true);
  else toast(`已保存「${name}」的设置`);
  await renderView();
}

// pluginCard 是一个插件的卡片：名字、版本、作者、能做什么、出错原因、启用开关、设置和卸载按钮。
// 只有声明了用户变量（userVariables）的插件才有“设置”按钮
function pluginCard(p) {
  const abilities = [p.canSearch && '搜索', p.canPlay && '播放', p.canLyric && '歌词'].filter(Boolean);
  let status = p.enabled ? '已启用' : '已停用';
  if (p.error) status = '加载失败';
  const meta = [p.author && `作者：${p.author}`, abilities.length && `支持：${abilities.join('、')}`, `文件：${p.id}.js`]
    .filter(Boolean)
    .join(' · ');
  return `
    <div class="plugin-card ${p.error ? 'broken' : ''} ${p.enabled ? '' : 'disabled'}">
      <div class="plugin-main">
        <div class="plugin-name">
          <span>${escapeHtml(p.platform || p.id)}</span>
          ${p.version ? `<span class="chip">v${escapeHtml(p.version)}</span>` : ''}
          <span class="plugin-status">${status}</span>
        </div>
        <div class="plugin-meta">${escapeHtml(meta)}</div>
        ${p.error ? `<div class="plugin-error">${escapeHtml(p.error)}</div>` : ''}
        ${p.missing?.length ? `<div class="plugin-missing">用到了暂不支持的模块：${escapeHtml(p.missing.join('、'))}，部分功能可能用不了</div>` : ''}
      </div>
      <div class="plugin-actions">
        <label class="switch" title="${p.enabled ? '停用' : '启用'}">
          <input type="checkbox" data-toggle="${escapeHtml(p.id)}" ${p.enabled ? 'checked' : ''} ${p.error ? 'disabled' : ''} />
          <span class="switch-track"></span>
        </label>
        ${p.userVariables?.length ? `<button class="btn" type="button" data-settings="${escapeHtml(p.id)}">${icons.settings}<span>设置</span></button>` : ''}
        <button class="btn danger" type="button" data-uninstall="${escapeHtml(p.id)}">${icons.trash}<span>卸载</span></button>
      </div>
    </div>`;
}

async function installFromFile() {
  const info = await InstallPluginFromFile();
  if (!info?.id) return; // 用户取消了
  toast(`已安装插件：${info.platform}`);
  await renderView();
}

async function installFromURL() {
  const url = await promptDialog({
    title: '从网址安装插件',
    label: '插件地址（.js 文件，或者插件列表 .json）',
    placeholder: 'https://',
    okText: '安装',
    maxLength: 2000,
  });
  if (!url) return;
  // 安装前再确认一次：插件来自第三方，提醒用户免责声明
  const ok = await confirmDialog({
    title: '安装插件',
    message: `将从这个网址下载并安装插件：\n${url}\n\n${DISCLAIMER}`,
    okText: '安装',
  });
  if (!ok) return;
  toast('正在下载插件…');
  const list = (await InstallPluginFromURL(url)) || [];
  toast(`已安装 ${list.length} 个插件：${list.map((p) => p.platform).join('、')}`);
  await renderView();
}

async function reload() {
  const list = (await ReloadPlugins()) || [];
  const broken = list.filter((p) => p.error).length;
  toast(broken ? `已重新加载，${broken} 个插件加载失败` : `已重新加载 ${list.length} 个插件`, broken > 0);
  await renderView();
}
