// Writes .libvips/vips.pc, so that pkg-config finds the libvips that npm
// installs from sharp's prebuilt packages instead of a system install.
// Commands that compile govips need PKG_CONFIG_LIBDIR=.libvips: the dev and
// test:go scripts in package.json set it.
//
// @img/sharp-libvips-dev has the headers. @img/sharp-libvips-<os>-<arch> has a
// single library with libvips and all its dependencies built in, and npm
// installs only the one for the current machine.

import {
	existsSync,
	mkdirSync,
	readdirSync,
	readFileSync,
	rmSync,
	symlinkSync,
	writeFileSync,
} from 'node:fs';
import path from 'node:path';

const root = path.resolve(import.meta.dirname, '..');
const modules = path.join(root, 'node_modules', '@img');
const out = path.join(root, '.libvips');

const dev = path.join(modules, 'sharp-libvips-dev');
const platform = `sharp-libvips-${process.platform}-${process.arch}`;
const lib = path.join(modules, platform, 'lib');
for (const dir of [dev, lib]) {
	if (!existsSync(dir)) {
		console.error(`${dir} is missing: run npm install, on a platform listed in package.json`);
		process.exit(1);
	}
}

const library = readdirSync(lib).find((name) => /^libvips-cpp\.(.+\.dylib|so\..+)$/.test(name));
if (!library) {
	console.error(`No libvips library in ${lib}`);
	process.exit(1);
}
const { vips: version } = JSON.parse(readFileSync(path.join(dev, 'versions.json'), 'utf8'));

// The linker's -lvips-cpp looks for an unversioned file name
rmSync(out, { recursive: true, force: true });
mkdirSync(path.join(out, 'lib'), { recursive: true });
const extension = process.platform === 'darwin' ? 'dylib' : 'so';
symlinkSync(path.join(lib, library), path.join(out, 'lib', `libvips-cpp.${extension}`));

const include = path.join(dev, 'include');
writeFileSync(
	path.join(out, 'vips.pc'),
	[
		'Name: vips',
		`Description: libvips ${version} from ${platform}`,
		`Version: ${version}`,
		`Cflags: -I${include} -I${path.join(include, 'glib-2.0')} -I${path.join(lib, 'glib-2.0', 'include')}`,
		// The rpath lets binaries built here find the library in node_modules
		`Libs: -L${path.join(out, 'lib')} -lvips-cpp -Wl,-rpath,${lib}`,
		'',
	].join('\n'),
);
console.log(`Wrote ${path.relative(root, out)}/vips.pc for libvips ${version} (${platform})`);
