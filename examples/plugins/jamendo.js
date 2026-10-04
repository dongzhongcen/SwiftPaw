/**
 * SwiftPaw 示例插件：Jamendo（jamendo.com）
 *
 * Jamendo 上的音乐由音乐人以知识共享（Creative Commons）许可证发布。
 * 这个插件用 Jamendo 官方的 API v3.0（https://developer.jamendo.com/v3.0），
 * 需要你自己申请一个 Client ID：
 *   1. 到 https://devportal.jamendo.com 注册并登录
 *   2. 创建一个应用（app），复制它的 Client ID
 *   3. SwiftPaw → 插件 → 找到 Jamendo → 「设置」，粘贴到「Client ID」里保存
 * 使用时请遵守 Jamendo 的 API 使用条款，以及每首歌各自的知识共享许可证。
 *
 * 支持：搜索歌曲、播放（不同音质）、歌词（Jamendo 有歌词的歌曲才有，没有时间轴）。
 *
 * 用到的插件格式（详见 internal/plugin/types.go）：
 *   - userVariables 声明需要用户填写的设置，插件页会出现「设置」按钮
 *   - 运行时用 env.getUserVariables().client_id 读到用户填的值
 *
 * 安装方法：SwiftPaw → 插件 → 从文件安装，选择这个文件；或者从网址安装 srcUrl 里的地址。
 */

const axios = require("axios");

const API_BASE = "https://api.jamendo.com/v3.0";
const PAGE_SIZE = 20;

// 每种音质对应的 Jamendo 音频格式：mp31 约 96kbps，mp32 是 VBR 高品质 mp3，flac 是无损
const STREAM_FORMAT = {
  low: "mp31",
  standard: "mp32",
  high: "mp32",
  super: "flac",
};

// Jamendo 按 id 查歌曲时偶尔会返回空结果，再试一次通常就有了
const MAX_TRIES = 3;

// 读取用户在「设置」里填的 Client ID，没填时给出中文提示
function getClientId() {
  const vars = (typeof env !== "undefined" && env.getUserVariables && env.getUserVariables()) || {};
  const clientId = String(vars.client_id || "").trim();
  if (!clientId) {
    throw new Error(
      "还没有填写 Jamendo 的 Client ID：请到 devportal.jamendo.com 创建应用并复制 Client ID，然后在插件页 Jamendo 的「设置」里填写"
    );
  }
  return clientId;
}

// 调用 Jamendo API。出错时 Jamendo 也返回 HTTP 200，错误写在 headers.status / headers.error_message 里
async function callApi(path, params) {
  const clientId = getClientId();
  const res = await axios.get(API_BASE + path, {
    params: Object.assign({ client_id: clientId, format: "json" }, params),
    timeout: 10000,
  });
  const data = res.data || {};
  const headers = data.headers || {};
  if (headers.status !== "success") {
    const message = headers.error_message || "未知错误";
    if (/client\s*id/i.test(message)) {
      throw new Error("Client ID 无效，请检查插件「设置」里填写的 Client ID 是否正确（" + message + "）");
    }
    throw new Error("Jamendo 接口出错：" + message);
  }
  return Array.isArray(data.results) ? data.results : [];
}

// 按 id 查一首歌的详细信息；查不到（歌曲已下架）时返回 null
async function getTrack(id, extraParams) {
  for (let i = 0; i < MAX_TRIES; i++) {
    const results = await callApi("/tracks/", Object.assign({ id: String(id) }, extraParams));
    if (results.length > 0) return results[0];
  }
  return null;
}

// Jamendo 的 track → 插件的 musicItem。额外的字段会原样保存，播放时传回 getMediaSource
function toMusicItem(track) {
  return {
    id: String(track.id),
    platform: "Jamendo",
    title: track.name || String(track.id),
    artist: track.artist_name || "未知",
    album: track.album_name || "", // 单曲（single）没有专辑
    artwork: track.image || track.album_image || "",
    duration: Number(track.duration) || 0,
    audio: track.audio || "", // 搜索时请求的是 mp32 格式的播放地址
    audiodownload_allowed: track.audiodownload_allowed === true,
    license: track.license_ccurl || "",
    shareurl: track.shareurl || "",
  };
}

// Jamendo 的歌词是纯文本，可能带 <br> 和 \r\n
function cleanLyrics(text) {
  return String(text || "")
    .replace(/<br\s*\/?>/gi, "\n")
    .replace(/\r\n?/g, "\n")
    .trim();
}

module.exports = {
  platform: "Jamendo",
  version: "1.0.0",
  author: "dongzhongcen",
  srcUrl: "https://raw.githubusercontent.com/dongzhongcen/SwiftPaw/main/examples/plugins/jamendo.js",
  supportedSearchType: ["music"],
  userVariables: [
    {
      key: "client_id",
      name: "Client ID",
      hint: "在 devportal.jamendo.com 注册并创建应用后获得",
    },
  ],

  // 搜索歌曲：返回 { isEnd, data: [musicItem] }，page 从 1 开始
  async search(query, page, type) {
    if (type && type !== "music") {
      return { isEnd: true, data: [] }; // 目前只支持搜索歌曲
    }
    const results = await callApi("/tracks/", {
      search: String(query).trim(),
      limit: PAGE_SIZE,
      offset: (page - 1) * PAGE_SIZE,
      type: "single albumtrack", // 默认只返回专辑里的歌，加上单曲
      audioformat: "mp32",
      imagesize: 300,
    });
    return {
      isEnd: results.length < PAGE_SIZE,
      data: results.map(toMusicItem),
    };
  },

  // 播放地址：都用 audio 字段（在线播放的地址），不用 audiodownload（那是下载地址）。
  // 无损（super）只在音乐人允许下载（audiodownload_allowed）的歌曲上提供；
  // 不允许时返回 null，SwiftPaw 会自动换一个音质。
  async getMediaSource(musicItem, quality) {
    const format = STREAM_FORMAT[quality] || STREAM_FORMAT.standard;
    if (format === "flac" && musicItem.audiodownload_allowed === false) return null;
    if (format === "mp32" && musicItem.audio) {
      return { url: musicItem.audio }; // 搜索结果里已经有了，不用再请求一次
    }
    const track = await getTrack(musicItem.id, { audioformat: format });
    if (!track || !track.audio) return null;
    if (format === "flac" && track.audiodownload_allowed !== true) return null;
    return { url: track.audio };
  },

  // 歌词：Jamendo 只有部分歌曲有歌词，而且没有时间轴；没有时返回 null
  async getLyric(musicItem) {
    const track = await getTrack(musicItem.id, { include: "lyrics" });
    if (!track) return null;
    const lyrics = cleanLyrics(track.lyrics || (track.musicinfo && track.musicinfo.lyrics));
    return lyrics ? { rawLrc: lyrics } : null;
  },
};
