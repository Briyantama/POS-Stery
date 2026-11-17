Manage PM2 services. Arguments: `<app> <action>`, app is all|8000|5173|5174, action is start|stop|restart|status|logs (default: all status).

Process names: 8000 = pos-api-8000, 5173 = pos-admin-5173, 5174 = pos-cashier-5174. For `all`, omit the name (stop/restart/logs/status take no name; start uses `ecosystem.config.cjs` with no `--only`).

Run from `C:/Users/solui/moon-eye/stery-suite/POS-Stery`:
- start: `pm2 start ecosystem.config.cjs --only <name>` (all: no `--only`), then open logs in a new window: `start wt.exe -d "C:/Users/solui/moon-eye/stery-suite/POS-Stery" pwsh -NoExit -c "pm2 logs <name>"` (all: `pm2 monit`)
- stop / restart: `pm2 stop|restart <name>` (all: `all`)
- status: `pm2 status`
- logs: `pm2 logs <name>` (all: `pm2 logs`)

$ARGUMENTS
