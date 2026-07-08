const fs = require('fs');
const path = require('path');
const os = require('os');

const BIN_DIR = path.join(__dirname, 'bin');
const BIN_NAME = os.platform() === 'win32' ? 'markdown2confluence.exe' : 'markdown2confluence';

function getPlatformDir() {
  const platform = os.platform();
  const arch = os.arch();

  let platformStr, archStr;

  switch (platform) {
    case 'linux':
      platformStr = 'linux';
      break;
    case 'darwin':
      platformStr = 'darwin';
      break;
    case 'win32':
      platformStr = 'windows';
      break;
    default:
      throw new Error(`Unsupported platform: ${platform}`);
  }

  switch (arch) {
    case 'x64':
      archStr = 'amd64';
      break;
    case 'arm64':
      archStr = 'arm64';
      break;
    case 'ia32':
      archStr = 'amd64';
      break;
    default:
      throw new Error(`Unsupported architecture: ${arch}`);
  }

  return `${platformStr}-${archStr}`;
}

function install() {
  const platformDir = getPlatformDir();
  const binPath = path.join(BIN_DIR, platformDir, BIN_NAME);

  if (!fs.existsSync(binPath)) {
    console.error(`Error: Binary not found for ${platformDir}.`);
    console.error('Please run: npm run build');
    process.exit(1);
  }

  // Make executable on Unix
  if (os.platform() !== 'win32') {
    fs.chmodSync(binPath, 0o755);
  }

  console.log(`markdown2confluence ready for ${platformDir}!`);
}

install();
