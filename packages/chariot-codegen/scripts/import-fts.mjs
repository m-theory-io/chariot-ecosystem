#!/usr/bin/env node
import { readFileSync, realpathSync, writeFileSync, renameSync, linkSync, unlinkSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { randomUUID } from 'node:crypto';
import { importFTSModel, loadFTSModel } from '../dist/index.js';

const args = process.argv.slice(2);
const usage = 'Usage: npm run import:fts -- INPUT.modelseed.json OUTPUT.ch [--force]';
try {
  if (args.length < 2 || args.length > 3 || (args[2] !== undefined && args[2] !== '--force') ||
      args.slice(0, 2).some(a => a.startsWith('--'))) throw new Error(usage);
  const [input, output, force] = args;
  const root = realpathSync(input);
  const sourcePaths = new Set();
  const model = loadFTSModel(root, id => {
    sourcePaths.add(id);
    try { return JSON.parse(readFileSync(id, 'utf8')); }
    catch (error) { throw new Error(`Cannot read FTS model ${id}: ${error.message}`, { cause: error }); }
  }, (parent, source) => realpathSync(resolve(dirname(parent), source)));
  const destination = resolve(output);
  let existing;
  try { existing = realpathSync(destination); }
  catch (error) { if (error.code !== 'ENOENT') throw error; }
  if (sourcePaths.has(existing ?? destination)) throw new Error('Output must not overwrite an input model');
  const result = importFTSModel(model);
  const temporary = `${destination}.${randomUUID()}.tmp`;
  try {
    writeFileSync(temporary, result.source, { flag: 'wx' });
    if (force) renameSync(temporary, destination);
    else linkSync(temporary, destination);
  } finally {
    try { unlinkSync(temporary); }
    catch (error) { if (error.code !== 'ENOENT') throw error; }
  }
  console.log(`Created ${destination}: ${result.classNames.length} classes, ${result.relationshipKeys.length} relationships; excluded ${result.excludedInstanceFrames} instance frames and ${result.excludedInstanceEdges} instance edges.`);
} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
}
