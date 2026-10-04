// 和歌曲、歌单有关的操作：收藏、下一首播放、加入歌单、新建/重命名/删除歌单。
// 列表行上的按钮、播放栏的红心、侧边栏的“+”都调用这里的函数。

import {
  AddToPlaylist,
  CreatePlaylist,
  DeletePlaylist,
  FavoriteKeys,
  Playlists,
  QueueAddNext,
  RenamePlaylist,
  ToggleFavorite,
} from '../../wailsjs/go/main/App';
import { applyQueue } from './player.js';
import { confirmDialog, pickDialog, promptDialog } from './dialog.js';
import { state, emit } from './state.js';
import { displayName, songKey, toast } from './util.js';

// FAVORITES_ID 是内置歌单“我喜欢”的 id，和 Go 里的 store.FavoritesID 一致
export const FAVORITES_ID = 1;

export async function refreshPlaylists() {
  state.playlists = (await Playlists()) || [];
  emit('playlists-changed');
}

export async function refreshFavorites() {
  state.favorites = new Set((await FavoriteKeys()) || []);
  emit('favorites-changed');
}

export function isFavorite(song) {
  return !!song && state.favorites.has(songKey(song));
}

export async function toggleFavorite(song) {
  const liked = await ToggleFavorite(song);
  if (liked) state.favorites.add(songKey(song));
  else state.favorites.delete(songKey(song));
  emit('favorites-changed');
  toast(liked ? `已收藏：${displayName(song)}` : `已取消收藏：${displayName(song)}`);
  await refreshPlaylists(); // 更新“我喜欢”的歌曲数
}

export async function addNext(song) {
  await applyQueue(await QueueAddNext(song), false);
  toast(`已设为下一首播放：${displayName(song)}`);
}

// addToPlaylist 弹出歌单列表让用户选，也可以当场新建一个
export async function addToPlaylist(song) {
  const NEW = 'new';
  const items = state.playlists.map((p) => ({ value: p.id, label: p.name, hint: `${p.count} 首` }));
  items.push({ value: NEW, label: '＋ 新建歌单', hint: '' });

  let id = await pickDialog({ title: `加入歌单：${displayName(song)}`, items });
  if (id === null) return;
  if (id === NEW) {
    const created = await createPlaylist();
    if (!created) return;
    id = created.id;
  }

  const added = await AddToPlaylist(id, song);
  const name = state.playlists.find((p) => p.id === id)?.name || '';
  toast(added > 0 ? `已加入歌单：${name}` : `「${name}」里已经有这首歌了`);
  if (id === FAVORITES_ID) await refreshFavorites();
  await refreshPlaylists();
}

// createPlaylist 让用户输入名字并新建歌单，取消时返回 null
export async function createPlaylist() {
  const name = await promptDialog({ title: '新建歌单', label: '歌单名称', placeholder: '比如：通勤路上', okText: '创建' });
  if (!name) return null;
  const playlist = await CreatePlaylist(name);
  await refreshPlaylists();
  toast(`已创建歌单：${playlist.name}`);
  return playlist;
}

export async function renamePlaylist(playlist) {
  const name = await promptDialog({ title: '重命名歌单', label: '新的名称', value: playlist.name, okText: '保存' });
  if (!name || name === playlist.name) return;
  await RenamePlaylist(playlist.id, name);
  await refreshPlaylists();
  toast('已重命名');
}

// deletePlaylist 确认后删除歌单，返回是否真的删了
export async function deletePlaylist(playlist) {
  const ok = await confirmDialog({
    title: '删除歌单',
    message: `确定要删除歌单「${playlist.name}」吗？里面的 ${playlist.count} 首歌会从歌单里移除（不会删除音乐文件）。`,
    okText: '删除',
    danger: true,
  });
  if (!ok) return false;
  await DeletePlaylist(playlist.id);
  await refreshPlaylists();
  toast(`已删除歌单：${playlist.name}`);
  return true;
}
