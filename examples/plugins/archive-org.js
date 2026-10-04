/**
 * SwiftPaw 示例插件：Internet Archive（archive.org）公有领域音频
 *
 * 只搜索 Internet Archive 上标记为公有领域（Public Domain / CC0）的音频，
 * 用的是 archive.org 公开的 advancedsearch 和 metadata 接口，不需要登录。
 *
 * 这个文件也是插件格式的说明：
 *   - 插件是 CommonJS 模块，用 module.exports 导出一个对象
 *   - search / getMediaSource / getLyric 都可以是 async 函数
 *   - 可以 require("axios") 发网络请求
 *
 * 安装方法：SwiftPaw → 插件 → 从文件安装，选择这个文件。
 */

const axios = require("axios");

const PAGE_SIZE = 20;

// 只要公有领域的音频（CC0 和公有领域标记的网址里都带 publicdomain）
const LICENSE_FILTER = "licenseurl:*publicdomain*";

// 把 creator 字段（可能是字符串，也可能是数组）变成一个字符串
function joinField(value) {
  if (Array.isArray(value)) return value.join(", ");
  return value ? String(value) : "";
}

// 搜索结果里的一条 → 插件的 musicItem
function toMusicItem(doc) {
  return {
    id: doc.identifier,
    platform: "Internet Archive",
    title: joinField(doc.title) || doc.identifier,
    artist: joinField(doc.creator) || "未知",
    album: "", // Internet Archive 的条目没有“专辑”这个概念
    artwork: "https://archive.org/services/img/" + encodeURIComponent(doc.identifier),
    duration: 0, // 搜索接口拿不到时长
  };
}

// 不同音质优先选哪种文件格式
const FORMAT_PRIORITY = {
  low: ["64Kbps MP3", "VBR MP3", "Ogg Vorbis"],
  standard: ["VBR MP3", "128Kbps MP3", "64Kbps MP3", "Ogg Vorbis"],
  high: ["VBR MP3", "256Kbps MP3", "Ogg Vorbis", "Flac"],
  super: ["Flac", "VBR MP3", "Ogg Vorbis"],
};

// 解析 "3:09" 或 "189.46" 这样的时长
function parseLength(length) {
  if (!length) return 0;
  if (String(length).includes(":")) {
    return String(length).split(":").reduce((total, part) => total * 60 + Number(part), 0);
  }
  return Number(length) || 0;
}

module.exports = {
  platform: "Internet Archive",
  version: "1.0.0",
  author: "dongzhongcen",
  srcUrl: "https://raw.githubusercontent.com/dongzhongcen/SwiftPaw/main/examples/plugins/archive-org.js",
  supportedSearchType: ["music"],

  // 搜索：返回 { isEnd, data: [musicItem] }，page 从 1 开始
  async search(query, page, type) {
    const keyword = String(query).replace(/[()":]/g, " ").trim();
    const q = `(title:(${keyword}) OR creator:(${keyword})) AND mediatype:audio AND ${LICENSE_FILTER}`;
    const res = await axios.get("https://archive.org/advancedsearch.php", {
      params: {
        q,
        "fl[]": ["identifier", "title", "creator"],
        "sort[]": "downloads desc",
        rows: PAGE_SIZE,
        page,
        output: "json",
      },
      paramsSerializer(params) {
        // fl[] 要写成 fl[]=identifier&fl[]=title 的形式
        const parts = [];
        for (const [key, value] of Object.entries(params)) {
          for (const v of [].concat(value)) {
            parts.push(encodeURIComponent(key) + "=" + encodeURIComponent(v));
          }
        }
        return parts.join("&");
      },
      timeout: 10000,
    });
    if (res.data.error) {
      throw new Error("Internet Archive 搜索失败：" + res.data.error);
    }
    const response = res.data.response || { docs: [], numFound: 0 };
    return {
      isEnd: page * PAGE_SIZE >= response.numFound,
      data: response.docs.map(toMusicItem),
    };
  },

  // 播放地址：查条目的文件列表，挑一个合适格式的音频文件（一个条目有多首时取第一首）
  async getMediaSource(musicItem, quality) {
    const res = await axios.get("https://archive.org/metadata/" + encodeURIComponent(musicItem.id), {
      timeout: 10000,
    });
    const files = res.data.files || [];
    for (const format of FORMAT_PRIORITY[quality] || FORMAT_PRIORITY.standard) {
      const file = files.find((f) => f.format === format);
      if (file) {
        return {
          url: `https://archive.org/download/${encodeURIComponent(musicItem.id)}/${encodeURIComponent(file.name)}`,
          duration: parseLength(file.length),
        };
      }
    }
    return null; // 没有这种音质，SwiftPaw 会自动换一个音质再试
  },

  // Internet Archive 的音频一般没有歌词，返回 null 表示没有
  async getLyric(musicItem) {
    return null;
  },
};
