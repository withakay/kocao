#!/usr/bin/env node

import fs from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { marked } from 'marked'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const repoRoot = path.resolve(__dirname, '..', '..')
const docsRoot = path.join(repoRoot, 'docs')
const outputRoot = path.join(repoRoot, 'web', 'dist', 'docs')
const checkOnly = process.argv.includes('--check')

async function walkMarkdown(dir) {
  const entries = await fs.readdir(dir, { withFileTypes: true })
  const files = []
  for (const entry of entries) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      files.push(...await walkMarkdown(full))
      continue
    }
    if (entry.isFile() && entry.name.toLowerCase().endsWith('.md')) {
      files.push(full)
    }
  }
  return files
}

function getTitle(markdown, fallback) {
  const m = markdown.match(/^#\s+(.+)$/m)
  return m?.[1]?.trim() || fallback
}

function shellHTML({ title, content, navItems }) {
  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>${title} · kocao docs</title>
    <style>
      :root {
        color-scheme: dark;
        --bg: #0b0b0b;
        --panel: #121212;
        --border: #272727;
        --fg: #e7e7e7;
        --muted: #a0a0a0;
        --accent: #78c5af;
        --code: #1b1b1b;
      }
      * { box-sizing: border-box; }
      body {
        margin: 0;
        font-family: Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif;
        background: var(--bg);
        color: var(--fg);
      }
      .layout {
        display: grid;
        grid-template-columns: 280px 1fr;
        min-height: 100vh;
      }
      nav {
        border-right: 1px solid var(--border);
        padding: 18px 14px;
        background: #0f0f0f;
      }
      nav h1 {
        margin: 0 0 12px;
        font-size: 12px;
        letter-spacing: .08em;
        text-transform: uppercase;
        color: var(--muted);
      }
      nav a {
        display: block;
        color: var(--fg);
        text-decoration: none;
        padding: 6px 8px;
        border-radius: 6px;
        margin-bottom: 2px;
        font-size: 13px;
      }
      nav a:hover { background: #1a1a1a; }
      main {
        max-width: 980px;
        width: 100%;
        padding: 26px 28px 44px;
      }
      article {
        background: var(--panel);
        border: 1px solid var(--border);
        border-radius: 12px;
        padding: 22px;
      }
      article h1, article h2, article h3 { margin-top: 1.25em; }
      article h1:first-child { margin-top: 0; }
      article a { color: var(--accent); }
      article pre, article code {
        font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
      }
      article code {
        background: var(--code);
        border: 1px solid var(--border);
        border-radius: 6px;
        padding: 1px 5px;
        font-size: 12px;
      }
      article pre {
        background: var(--code);
        border: 1px solid var(--border);
        border-radius: 8px;
        padding: 12px;
        overflow-x: auto;
      }
      .links {
        margin-top: 14px;
        display: flex;
        gap: 10px;
        flex-wrap: wrap;
      }
      .links a {
        text-decoration: none;
        color: var(--fg);
        border: 1px solid var(--border);
        background: #151515;
        padding: 6px 9px;
        border-radius: 999px;
        font-size: 12px;
      }
      @media (max-width: 980px) {
        .layout { grid-template-columns: 1fr; }
        nav { border-right: 0; border-bottom: 1px solid var(--border); }
      }
    </style>
  </head>
  <body>
    <div class="layout">
      <nav>
        <h1>kocao docs</h1>
        ${navItems}
        <div class="links">
          <a href="/" target="_blank" rel="noreferrer">Open UI</a>
          <a href="/api/v1/scalar" target="_blank" rel="noreferrer">API Reference</a>
          <a href="/api/v1/openapi.json" target="_blank" rel="noreferrer">OpenAPI JSON</a>
        </div>
      </nav>
      <main>
        <article>${content}</article>
      </main>
    </div>
  </body>
</html>`
}

async function main() {
  const markdownFiles = (await walkMarkdown(docsRoot)).sort((a, b) => a.localeCompare(b))
  if (markdownFiles.length === 0) {
    throw new Error(`no markdown docs found under ${docsRoot}`)
  }

  const docs = []
  for (const absFile of markdownFiles) {
    const rel = path.relative(docsRoot, absFile)
    const markdown = await fs.readFile(absFile, 'utf8')
    const title = getTitle(markdown, rel)
    const html = marked.parse(markdown)
    docs.push({
      absFile,
      rel,
      title,
      html,
      outRel: rel.replace(/\.md$/i, '.html'),
    })
  }

  if (checkOnly) {
    console.log(`docs check OK (${docs.length} markdown files)`)
    return
  }

  await fs.rm(outputRoot, { recursive: true, force: true })
  await fs.mkdir(outputRoot, { recursive: true })

  const navItems = docs
    .map((d) => `<a href="/${path.posix.join('docs', d.outRel).replace(/\\/g, '/')}" title="${d.rel}">${d.title}</a>`)
    .join('\n')

  for (const doc of docs) {
    const out = path.join(outputRoot, doc.outRel)
    await fs.mkdir(path.dirname(out), { recursive: true })
    await fs.writeFile(out, shellHTML({ title: doc.title, content: doc.html, navItems }), 'utf8')
  }

  const indexHTML = shellHTML({
    title: 'Documentation',
    navItems,
    content: `
      <h1>Documentation</h1>
      <p>Published from repository markdown files at image build time.</p>
      <ul>
        ${docs.map((d) => `<li><a href="/${path.posix.join('docs', d.outRel)}">${d.title}</a> <code>${d.rel}</code></li>`).join('\n')}
      </ul>
    `,
  })
  await fs.writeFile(path.join(outputRoot, 'index.html'), indexHTML, 'utf8')

  console.log(`rendered ${docs.length} docs to ${outputRoot}`)
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
