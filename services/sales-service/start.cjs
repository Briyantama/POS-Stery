const { spawn } = require('child_process');

const proc = spawn('go', ['run', './cmd/server'], {
  cwd: __dirname,
  stdio: 'inherit',
  windowsHide: true,
  env: { ...process.env },
});

proc.on('close', (code) => process.exit(code ?? 0));
