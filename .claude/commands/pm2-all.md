Start all PM2 services and open monitor.
```bash
cd "C:/Users/solui/moon-eye/stery-suite/POS-Stery" && pm2 start ecosystem.config.cjs && start wt.exe -d "C:/Users/solui/moon-eye/stery-suite/POS-Stery" pwsh -NoExit -c "pm2 monit"
```
