#!/usr/bin/env node
// Runs the built CLI. Inside the ReputeCast repository, before a build, it
// runs from the TypeScript sources (Node strips the types).
import { existsSync } from "node:fs";

const built = new URL("../dist/cli.js", import.meta.url);
const { main } = await import(existsSync(built) ? built.href : new URL("../src/cli.ts", import.meta.url).href);
process.exitCode = await main(process.argv.slice(2));
