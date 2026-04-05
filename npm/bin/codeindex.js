#!/usr/bin/env node
'use strict';

// Platform/arch maps used by getPlatform() and getDownloadURL() below.
// These are also imported by codeindex.test.js for unit testing.

const { spawnSync, execFileSync } = require('child_process');
const { existsSync, mkdirSync, chmodSync, unlinkSync } = require('fs');
const os = require('os');
const path = require('path');

const VERSION = '0.1.2';
const REPO = '01x-in/codeindex';

function getPlatform(platform, arch) {
  // Allow callers to inject platform/arch for testability.
  const p = platform != null ? platform : process.platform;
  const a = arch != null ? arch : process.arch;

  const platMap = { darwin: 'darwin', linux: 'linux', win32: 'windows' };
  const archMap = { x64: 'amd64', arm64: 'arm64' };

  const plat = platMap[p];
  const ar = archMap[a];

  if (!plat) throw new Error(`Unsupported platform: ${p}`);
  if (!ar) throw new Error(`Unsupported architecture: ${a}`);

  return { plat, ar };
}

function getBinaryName() {
  return process.platform === 'win32' ? 'codeindex.exe' : 'codeindex';
}

function getDownloadURL(version, plat, ar) {
  const ext = plat === 'windows' ? 'zip' : 'tar.gz';
  const name = `codeindex_${version}_${plat}_${ar}.${ext}`;
  return `https://github.com/${REPO}/releases/download/v${version}/${name}`;
}

function downloadAndExtract(url, destDir, binaryName) {
  const tmpFile = path.join(os.tmpdir(), `codeindex-download-${Date.now()}`);
  const isZip = url.endsWith('.zip');

  // Try curl first, then wget. Both are invoked with execFileSync (no shell
  // injection risk — url and tmpFile are constructed internally, not from
  // user input).
  try {
    execFileSync('curl', ['-fsSL', url, '-o', tmpFile], { stdio: 'inherit' });
  } catch (_e) {
    execFileSync('wget', ['-q', url, '-O', tmpFile], { stdio: 'inherit' });
  }

  mkdirSync(destDir, { recursive: true });

  if (isZip) {
    // On Windows, `unzip` is not guaranteed to be present. Extract the single
    // binary entry from the ZIP using PowerShell's Expand-Archive, which ships
    // with every Windows 10+ / Server 2016+ installation.
    if (process.platform === 'win32') {
      const destBin = path.join(destDir, binaryName);
      execFileSync('powershell', [
        '-NoProfile', '-NonInteractive', '-Command',
        `Add-Type -AssemblyName System.IO.Compression.FileSystem; ` +
        `$zip = [System.IO.Compression.ZipFile]::OpenRead('${tmpFile}'); ` +
        `$entry = $zip.Entries | Where-Object { $_.Name -eq '${binaryName}' } | Select-Object -First 1; ` +
        `[System.IO.Compression.ZipFileExtensions]::ExtractToFile($entry, '${destBin}', $true); ` +
        `$zip.Dispose()`,
      ], { stdio: 'inherit' });
    } else {
      execFileSync('unzip', ['-o', tmpFile, binaryName, '-d', destDir], { stdio: 'inherit' });
    }
  } else {
    execFileSync('tar', ['-xzf', tmpFile, '-C', destDir, binaryName], { stdio: 'inherit' });
  }

  unlinkSync(tmpFile);
}

// Only run the download+spawn logic when this file is the entry point,
// not when it is require()-d by the test suite.
if (require.main === module) {
  const binDir = path.join(__dirname, '..', '.bin');
  const binaryName = getBinaryName();
  const binaryPath = path.join(binDir, binaryName);

  if (!existsSync(binaryPath)) {
    const { plat, ar } = getPlatform();
    const url = getDownloadURL(VERSION, plat, ar);

    process.stderr.write(`Downloading codeindex v${VERSION} for ${plat}/${ar}...\n`);
    downloadAndExtract(url, binDir, binaryName);

    if (process.platform !== 'win32') {
      chmodSync(binaryPath, 0o755);
    }
  }

  const result = spawnSync(binaryPath, process.argv.slice(2), { stdio: 'inherit' });
  process.exit(result.status != null ? result.status : 1);
}

module.exports = { getPlatform, getDownloadURL };
