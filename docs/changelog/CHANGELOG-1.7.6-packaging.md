# 1.7.6 cumulative patch 3: installable packages

The release set now includes the existing ZIP distribution, the compiled
per-user Windows installer from installer-win/, and the amd64 Debian package.
All carry the same patch 3 application binaries and previous three fixes.
Fedora packaging is unchanged and has not been rebuilt.

Windows retains its original installer pages, per-user location, batch first-run
engine/model setup, LAN setup and uninstall flow. Normal launch and update
shortcuts use the new Go launch command. The separate setup/model shortcut
remains available. No engine or model is bundled in this Windows installer.

Debian retains its guided launcher, browser setup, desktop integration,
user-owned data, and pinned b10456 Vulkan engine with CPU fallback backends.
The Debian build now follows library symlinks for its payload size check and
invokes its shell guards through bash, so non-executable source file modes do
not prevent a valid package from building. These changes affect building only.
Debian revision is 2; the application build remains
1.7.6-go-314bdc9-dirty-patch3.

Validation: NSIS 3.09 compilation and extraction succeeded; the extracted
Windows executable matches the patched binary by SHA-256. Debian archive
extraction, payload and permissions checks passed. The packaged guided launcher
passed local and LAN first-run/repeat-launch tests, and the bundled engine
reported build 10456. Windows interactive installation and live model inference
are not validated by these checks. No git commits or pushes were made.
