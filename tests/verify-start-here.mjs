import fs from 'node:fs';

const [guidePath, configPath] = process.argv.slice(2);
if (!guidePath || !configPath) {
  throw new Error('Usage: node verify-start-here.mjs <START-HERE.html> <default_config.json>');
}

const html = fs.readFileSync(guidePath, 'utf8');
const config = JSON.parse(fs.readFileSync(configPath, 'utf8'));

const required = [
  'id="preset"', 'id="proxyType"', 'id="proxyHost"', 'id="proxyPort"',
  'id="disableQuic"', 'id="logMode"', 'id="logSummary"', 'id="bypassList"', 'id="save"', 'id="resetAll"',
  'id="jsonOutput"', 'id="commandOutput"', 'V2RayN', 'Clash Verge Rev',
  'Clash / Mihomo', '通用 SOCKS5', '出厂配置', '恢复全部出厂值',
  'START-HERE.html', 'CodexProxyLauncher.exe', '仅错误（推荐）', '始终记录', '日志已关闭',
  'id="saveEnv"', 'id="downloadEnv"', 'id="copyEnv"', 'id="envOutput"',
  'networkingMode=mirrored', 'wsl --shutdown', 'wslinfo --networking-mode'
];
for (const marker of required) {
  if (!html.includes(marker)) throw new Error(`START-HERE missing marker: ${marker}`);
}

if (/<script\b[^>]*\bsrc\s*=/i.test(html) || /<link\b[^>]*\bhref\s*=/i.test(html)) {
  throw new Error('START-HERE must not load external scripts or stylesheets');
}

const scripts = [...html.matchAll(/<script>([\s\S]*?)<\/script>/gi)].map(match => match[1]);
if (scripts.length !== 1) throw new Error(`Expected one inline script, found ${scripts.length}`);
new Function(scripts[0]);

// Run the guide against a minimal DOM stub so the generated .env preview is
// actually verified, not merely present in the markup.
const elements = new Map();
function stub(id) {
  if (!elements.has(id)) {
    elements.set(id, {
      id, value: '', checked: false, textContent: '', className: '',
      classList: {
        items: new Set(),
        add(value) { this.items.add(value); },
        remove(value) { this.items.delete(value); },
        contains(value) { return this.items.has(value); }
      },
      addEventListener() {}, scrollIntoView() {}
    });
  }
  return elements.get(id);
}
const noop = () => {};
const documentStub = {
  getElementById: stub,
  createElement: () => ({ style: {}, click: noop, select: noop, remove: noop }),
  body: { appendChild: noop }
};
new Function('document', 'window', 'navigator', scripts[0])(documentStub, {}, {});

const factoryProxy = 'http://127.0.0.1:10808';
const envLines = stub('envOutput').textContent.split('\n');
for (const line of [
  `no_proxy="localhost,127.0.0.1,::1"`,
  `http_proxy="${factoryProxy}"`,
  `https_proxy="${factoryProxy}"`,
  `all_proxy="${factoryProxy}"`,
  `ALL_PROXY="${factoryProxy}"`
]) {
  if (!envLines.includes(line)) throw new Error(`.env preview is missing line: ${line}`);
}

const preview = JSON.parse(stub('jsonOutput').textContent);
if (preview.proxy.host !== '127.0.0.1' || preview.proxy.port !== 10808) {
  throw new Error('config preview does not use the factory proxy');
}

if (config.proxy.type !== 'http' || config.proxy.host !== '127.0.0.1' || config.proxy.port !== 10808) {
  throw new Error('default_config.json no longer matches the documented factory proxy');
}
if (config.disable_quic !== true || !Array.isArray(config.bypass_list) || config.bypass_list.length < 4) {
  throw new Error('default_config.json is missing required safe defaults');
}
if (config.log_mode !== 'errors') {
  throw new Error('default_config.json log_mode must default to errors');
}
for (const value of ['http', '127.0.0.1', '10808']) {
  if (!html.includes(value)) throw new Error(`START-HERE does not expose factory value: ${value}`);
}

console.log('START-HERE verification passed');
