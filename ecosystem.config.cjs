// PM2 local dev: API on 8000 plus the admin and cashier dev servers. Secrets come from .env, never from this file.
const fs = require("fs");
const path = require("path");

const envFile = path.join(__dirname, ".env");
const env = fs.existsSync(envFile)
  ? Object.fromEntries(
      fs
        .readFileSync(envFile, "utf8")
        .split(/\r?\n/)
        .filter((line) => line.includes("=") && !line.trimStart().startsWith("#"))
        .map((line) => {
          const i = line.indexOf("=");
          return [line.slice(0, i).trim(), line.slice(i + 1).trim()];
        }),
    )
  : {};

module.exports = {
  apps: [
    {
      name: "pos-api-8000",
      cwd: "./apps/api",
      script: "go",
      interpreter: "none",
      args: "run ./cmd/api",
      env: { ...env, PORT: "8000" },
    },
    {
      name: "pos-admin-5173",
      cwd: "./apps/admin",
      script: "node_modules/vite/bin/vite.js",
      args: "dev --port 5173 --strictPort",
      env: { GATEWAY_URL: "http://localhost:8000" },
    },
    {
      name: "pos-cashier-5174",
      cwd: "./apps/cashier",
      script: "node_modules/vite/bin/vite.js",
      args: "dev --port 5174 --strictPort",
      env: { GATEWAY_URL: "http://localhost:8000" },
    },
  ],
};
