import { chmod, mkdir, rename, rm, writeFile } from "node:fs/promises";
import { arch, platform } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const repo = "Kartik-2239/lightcode";
const version = process.env.LIGHTCODE_VERSION || "latest";
const packageRoot = dirname(fileURLToPath(import.meta.url));
const binDir = join(packageRoot, "bin");

const osByPlatform = {
	darwin: "darwin",
	linux: "linux",
	win32: "windows",
};

const archByProcessArch = {
	x64: "amd64",
	arm64: "arm64",
};

const os = osByPlatform[platform()];
const cpu = archByProcessArch[arch()];

if (!os || !cpu) {
	throw new Error(`Unsupported platform: ${platform()} ${arch()}`);
}

const exe = os === "windows" ? ".exe" : "";
const asset = `lightcode-${os}-${cpu}${exe}`;
const downloadBase =
	version === "latest"
		? `https://github.com/${repo}/releases/latest/download`
		: `https://github.com/${repo}/releases/download/${version}`;
const url = `${downloadBase}/${asset}`;
const tmpPath = join(binDir, `${asset}.download`);
const targetPath = join(binDir, `lightcode${exe}`);

async function download(from, to) {
	const response = await fetch(from);
	if (!response.ok) {
		throw new Error(`Failed to download ${from}: ${response.status} ${response.statusText}`);
	}

	const bytes = Buffer.from(await response.arrayBuffer())
    await writeFile(to, bytes)
}

console.log(`Installing Lightcode ${version} for ${os}/${cpu}...`);

await mkdir(binDir, { recursive: true });
await rm(tmpPath, { force: true });
await download(url, tmpPath);
await chmod(tmpPath, 0o755);
await rename(tmpPath, targetPath);

console.log(`Installed Lightcode@${version} to ${targetPath}`);
