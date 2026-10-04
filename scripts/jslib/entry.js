// 插件可以 require 的 JS 库。每个库包在一个函数里，插件真正 require 时才执行，
// 这样没用到 cheerio 的插件不用花时间初始化它。
globalThis.__swiftpawLibs = {
  'crypto-js': () => require('crypto-js'),
  qs: () => require('qs'),
  dayjs: () => require('dayjs'),
  he: () => require('he'),
  'big-integer': () => require('big-integer'),
  cheerio: () => require('cheerio'),
};
