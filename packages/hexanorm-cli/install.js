const fs = require("fs");
const path = require("path");
const axios = require("axios");
const tar = require("tar");
const AdmZip = require("adm-zip");
const { execSync } = require("child_process");

const PACKAGE_VERSION = require("./package.json").version;
const REPO = "pmaojo/hexanorm-mcp";

function getPlatform() {
  const os = process.platform;
  const arch = process.arch;

  let goos, goarch;

  if (os === "darwin") goos = "Darwin";
  else if (os === "linux") goos = "Linux";
  else if (os === "win32") goos = "Windows";
  else throw new Error(`Unsupported OS: ${os}`);

  if (arch === "x64") goarch = "x86_64";
  else if (arch === "arm64") goarch = "arm64";
  else throw new Error(`Unsupported Arch: ${arch}`);

  return { goos, goarch };
}

async function install() {
  const { goos, goarch } = getPlatform();
  const version = `v${PACKAGE_VERSION}`;
  const ext = process.platform === "win32" ? "zip" : "tar.gz";
  const filename = `hexanorm_${goos}_${goarch}.${ext}`;
  const url = `https://github.com/${REPO}/releases/download/${version}/${filename}`;

  console.log(`Downloading Hexanorm ${version} for ${goos}/${goarch}...`);
  console.log(`URL: ${url}`);

  const binDir = path.join(__dirname, "bin");
  if (!fs.existsSync(binDir)) fs.mkdirSync(binDir);

  const writer = fs.createWriteStream(path.join(binDir, filename));

  try {
    const response = await axios({
      url,
      method: "GET",
      responseType: "stream",
    });

    response.data.pipe(writer);

    await new Promise((resolve, reject) => {
      writer.on("finish", resolve);
      writer.on("error", reject);
    });

    console.log("Extracting...");
    if (ext === "zip") {
      const zip = new AdmZip(path.join(binDir, filename));
      zip.extractAllTo(binDir, true);
    } else {
      await tar.x({
        file: path.join(binDir, filename),
        cwd: binDir,
      });
    }

    // Cleanup
    fs.unlinkSync(path.join(binDir, filename));

    // Rename to generic 'hexanorm' (or hexanorm.exe)
    const exeName = process.platform === "win32" ? "hexanorm.exe" : "hexanorm";
    // GoReleaser might put it in a folder or just the file.
    // Usually with tar.gz/zip it's in the root of archive or a folder.
    // Let's assume root for now based on goreleaser config.

    // If goreleaser puts it in a folder named after the archive, we might need to move it.
    // But standard goreleaser archive usually has the binary at top level or we can config it.
    // My goreleaser config didn't specify 'wrap_in_directory', so it defaults to false (I think) or true?
    // Default is usually true. Let's check.
    // Actually, let's just find the binary in binDir and move it.

    const files = fs.readdirSync(binDir);
    const binFile = files.find(
      (f) =>
        f.startsWith("hexanorm") &&
        !f.endsWith(".tar.gz") &&
        !f.endsWith(".zip")
    );

    if (binFile) {
      // If it's a directory (goreleaser default wrapping), move contents
      const binPath = path.join(binDir, binFile);
      if (fs.lstatSync(binPath).isDirectory()) {
        const innerFiles = fs.readdirSync(binPath);
        const innerBin = innerFiles.find((f) => f.startsWith("hexanorm"));
        fs.renameSync(
          path.join(binPath, innerBin),
          path.join(binDir, "hexanorm")
        );
        fs.rmdirSync(binPath);
      } else {
        // It's the file
        if (binFile !== "hexanorm") {
          fs.renameSync(
            path.join(binDir, binFile),
            path.join(binDir, "hexanorm")
          );
        }
      }
    }

    if (process.platform !== "win32") {
      fs.chmodSync(path.join(binDir, "hexanorm"), 0o755);
    }

    console.log("Hexanorm installed successfully!");
  } catch (error) {
    console.error("Installation failed:", error.message);
    process.exit(1);
  }
}

install();
