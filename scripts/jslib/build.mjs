// 用 esbuild 把 entry.js 和它依赖的 npm 包打成一个文件：internal/jsrt/libs/libs.js，
// 同时生成这些 JS 库的许可证说明 internal/jsrt/libs/NOTICES.md，
// 再由 scripts/notices 把它和 Go 依赖、字体的许可证合并成根目录的 THIRD_PARTY_NOTICES.md。
// 用法：cd scripts/jslib && npm ci && npm run build
import { build } from 'esbuild';
import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve(import.meta.dirname, '../..');
const outfile = path.join(root, 'internal/jsrt/libs/libs.js');

const result = await build({
  entryPoints: [path.join(import.meta.dirname, 'entry.js')],
  bundle: true,
  format: 'iife',
  platform: 'browser', // 用各个包的浏览器版本，不依赖 Node 内置模块
  target: 'es2017', // goja 对新语法支持不完整，让 esbuild 转成老一点的写法
  minify: true,
  legalComments: 'none', // 许可证统一写在 NOTICES.md / THIRD_PARTY_NOTICES.md 里
  metafile: true,
  outfile,
  logLevel: 'info',
});

// 从打包用到的文件里找出所有 npm 包，读取它们的许可证
const packages = new Map();
for (const input of Object.keys(result.metafile.inputs)) {
  if (input.startsWith("(disabled)")) continue; // 被浏览器版本禁用的 Node 模块，没有打包进来
  const match = input.match(/node_modules\/((?:@[^/]+\/)?[^/]+)\//g);
  if (!match) continue;
  const dir = input.slice(0, input.lastIndexOf(match[match.length - 1]) + match[match.length - 1].length);
  const pkgDir = path.resolve(import.meta.dirname, dir);
  if (packages.has(pkgDir)) continue;
  const pkg = JSON.parse(fs.readFileSync(path.join(pkgDir, 'package.json'), 'utf8'));
  const licenseFile = fs.readdirSync(pkgDir).find((f) => /^(license|licence|copying)/i.test(f));
  packages.set(pkgDir, {
    name: pkg.name,
    version: pkg.version,
    license: typeof pkg.license === 'string' ? pkg.license : pkg.license?.type || '未知',
    repo: typeof pkg.repository === 'string' ? pkg.repository : pkg.repository?.url || pkg.homepage || '',
    text: licenseFile ? fs.readFileSync(path.join(pkgDir, licenseFile), 'utf8').trim() : '',
  });
}

const allowed = /^(MIT|ISC|BSD-2-Clause|BSD-3-Clause|Unlicense|0BSD|Apache-2.0)$/;
const list = [...packages.values()].sort((a, b) => a.name.localeCompare(b.name));
const bad = list.filter((p) => !allowed.test(p.license));
if (bad.length) {
  console.error('有不在白名单里的许可证：', bad.map((p) => `${p.name} (${p.license})`));
  process.exit(1);
}

// 这里只写 JS 库这一部分（从二级标题开始），scripts/notices 会把它原样放进 THIRD_PARTY_NOTICES.md
let md = `## 插件运行时内置的 JS 库

用 esbuild 打包进 \`internal/jsrt/libs/libs.js\`（由 \`scripts/jslib/build.mjs\` 自动生成这一部分）。

| 包 | 版本 | 许可证 |
|----|------|--------|
${list.map((p) => `| ${p.name} | ${p.version} | ${p.license} |`).join('\n')}

`;
for (const p of list) {
  md += `### ${p.name}@${p.version}（${p.license}）\n\n${p.repo ? p.repo.replace(/^git\+/, '') + '\n\n' : ''}`;
  md += p.text ? '```\n' + p.text + '\n```\n\n' : '（包里没有单独的许可证文件）\n\n';
}
fs.writeFileSync(path.join(root, 'internal/jsrt/libs/NOTICES.md'), md);
console.log(`打包完成：${list.length} 个包（记得再运行 go run ./scripts/notices），${(fs.statSync(outfile).size / 1024).toFixed(0)} KB`);
