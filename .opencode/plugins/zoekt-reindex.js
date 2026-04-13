/**
 * Zoekt Reindex Plugin
 *
 * Automatically reindexes the repo with zoekt when files change or the session
 * goes idle. Debounced to run at most once per 30 seconds. Non-blocking — the
 * reindex always runs in the background.
 *
 * Triggers:
 *   - session.idle — agent finished processing, good time to refresh the index
 *   - file.watcher.updated — files changed on disk (git checkout, editor saves)
 *
 * Environment:
 *   ZOEKT_REINDEX_DISABLED=1       — disable the plugin entirely
 *   ZOEKT_REINDEX_DEBOUNCE_MS      — override debounce window (default 30000)
 *   ZOEKT_INDEX_DIR                — override index directory (forwarded to zoekt-index.sh)
 */

import path from 'path';
import fs from 'fs';
import { execFile } from 'child_process';

const DEFAULT_DEBOUNCE_MS = 30000;

export const ZoektReindexPlugin = async ({ client, directory, worktree }) => {
  const disabled = process.env.ZOEKT_REINDEX_DISABLED === '1';
  if (disabled) {
    return {};
  }

  const debounceMs = (() => {
    const raw = Number.parseInt(process.env.ZOEKT_REINDEX_DEBOUNCE_MS || '', 10);
    return Number.isFinite(raw) && raw > 0 ? raw : DEFAULT_DEBOUNCE_MS;
  })();

  // Resolve the index script path.
  // Look in the worktree first (may differ from directory in worktree setups),
  // then fall back to directory.
  const resolveScript = () => {
    const candidates = [
      worktree && path.join(worktree, '.agents/skills/zoekt-search/scripts/zoekt-index.sh'),
      path.join(directory, '.agents/skills/zoekt-search/scripts/zoekt-index.sh'),
    ].filter(Boolean);

    for (const candidate of candidates) {
      try {
        fs.accessSync(candidate, fs.constants.R_OK);
        return candidate;
      } catch {
        // try next
      }
    }
    return null;
  };

  const scriptPath = resolveScript();
  if (!scriptPath) {
    // Script not present — nothing to do. Don't break OpenCode startup.
    return {};
  }

  const log = (level, message) => {
    if (!client?.app?.log) return;
    try {
      client.app.log({
        body: { service: 'zoekt-reindex', level, message },
      });
    } catch {
      // best-effort
    }
  };

  let lastRunAt = 0;
  let pending = null; // timer handle
  let running = false;

  const runReindex = () => {
    if (running) return;

    const now = Date.now();
    const elapsed = now - lastRunAt;

    if (elapsed < debounceMs) {
      // Schedule for later if not already scheduled
      if (!pending) {
        const delay = debounceMs - elapsed;
        pending = setTimeout(() => {
          pending = null;
          runReindex();
        }, delay);
      }
      return;
    }

    running = true;
    lastRunAt = Date.now();
    log('info', 'Reindexing...');

    const env = { ...process.env };
    if (process.env.ZOEKT_INDEX_DIR) {
      env.ZOEKT_INDEX_DIR = process.env.ZOEKT_INDEX_DIR;
    }

    const child = execFile('bash', [scriptPath, worktree || directory], {
      cwd: directory,
      env,
      timeout: 120_000,
    }, (error) => {
      running = false;
      if (error) {
        log('warn', `Reindex failed: ${error.message}`);
      } else {
        log('info', 'Reindex complete');
      }
    });

    // Detach stdio so the child never blocks the agent
    if (child.stdout) child.stdout.unref?.();
    if (child.stderr) child.stderr.unref?.();
    child.unref?.();
  };

  const scheduleReindex = () => {
    const now = Date.now();
    const elapsed = now - lastRunAt;

    if (elapsed >= debounceMs && !running) {
      runReindex();
    } else if (!pending) {
      const delay = Math.max(0, debounceMs - elapsed);
      pending = setTimeout(() => {
        pending = null;
        runReindex();
      }, delay);
    }
    // else: already scheduled, do nothing
  };

  return {
    event: async ({ event }) => {
      if (!event || typeof event.type !== 'string') return;

      if (event.type === 'session.idle' || event.type === 'file.watcher.updated') {
        scheduleReindex();
      }
    },
  };
};
