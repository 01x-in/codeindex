'use strict';
const assert = require('assert');
const { getPlatform, getDownloadURL } = require('./codeindex');

// Platform detection
assert.deepStrictEqual(getPlatform('darwin', 'arm64'), { plat: 'darwin', ar: 'arm64' });
assert.deepStrictEqual(getPlatform('darwin', 'x64'), { plat: 'darwin', ar: 'amd64' });
assert.deepStrictEqual(getPlatform('linux', 'x64'), { plat: 'linux', ar: 'amd64' });
assert.deepStrictEqual(getPlatform('linux', 'arm64'), { plat: 'linux', ar: 'arm64' });
assert.deepStrictEqual(getPlatform('win32', 'x64'), { plat: 'windows', ar: 'amd64' });
assert.deepStrictEqual(getPlatform('win32', 'arm64'), { plat: 'windows', ar: 'arm64' });

assert.throws(() => getPlatform('freebsd', 'x64'), /Unsupported platform/);
assert.throws(() => getPlatform('darwin', 'ia32'), /Unsupported architecture/);

// Download URL construction
const darwinURL = getDownloadURL('0.1.2', 'darwin', 'arm64');
assert.match(darwinURL, /darwin_arm64\.tar\.gz$/);
assert.match(darwinURL, /v0\.1\.0/);
assert.match(darwinURL, /github\.com\/01x-in\/codeindex/);

const linuxURL = getDownloadURL('0.1.2', 'linux', 'amd64');
assert.match(linuxURL, /linux_amd64\.tar\.gz$/);

const winURL = getDownloadURL('0.1.2', 'windows', 'amd64');
assert.match(winURL, /windows_amd64\.zip$/);

console.log('All npm package tests pass.');
