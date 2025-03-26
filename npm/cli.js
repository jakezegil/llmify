#!/usr/bin/env node

const os = require('os');
const fs = require('fs');
const path = require('path');
const { spawn } = require('child_process');

// Mapping from Node's process.platform/arch to binary name parts
const platformMapping = {
  'darwin': 'darwin',
  'linux': 'linux',
  'win32': 'windows',
};

const archMapping = {
  'x64': 'amd64',
  'arm64': 'arm64',
};

const platform = os.platform();
const arch = os.arch();

const goPlatform = platformMapping[platform];
const goArch = archMapping[arch];

if (!goPlatform || !goArch) {
  console.error(`Error: Unsupported platform/architecture: ${platform}/${arch}`);
  process.exit(1);
}

let binaryName = `llmify-${goPlatform}-${goArch}`;
if (goPlatform === 'windows') {
  binaryName += '.exe';
}

// __dirname is the directory where this script (cli.js) resides
const binaryPath = path.join(__dirname, 'bin', binaryName);

// Check if the binary exists
if (!fs.existsSync(binaryPath)) {
  console.error(`Error: Could not find the llmify binary for your platform (${platform}/${arch}) at ${binaryPath}`);
  console.error('Please report this issue on GitHub.');
  process.exit(1);
}

// Get arguments passed to the npm script, excluding 'node' and the script path
const args = process.argv.slice(2);

// Run the binary
const child = spawn(binaryPath, args, { stdio: 'inherit' }); // 'inherit' pipes stdin, stdout, stderr

child.on('error', (err) => {
  console.error(`Error executing binary: ${err}`);
  process.exit(1);
});

child.on('exit', (code, signal) => {
  if (signal) {
    // Process terminated by signal
    process.kill(process.pid, signal);
  } else {
    // Process exited normally
    process.exit(code);
  }
}); 