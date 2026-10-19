GobboNet 1.7.7 - hotfix integration, application files

Includes all three cumulative 1.7.6 hotfix groups while preserving 1.7.7:
- CONFIG custom hostnames (DuckDNS/Tailscale), saved and applied immediately.
- Atomic browser history migration and legacy server-history recovery.
- Direct Windows launch using existing settings, with shortcut repair.

WINDOWS UPDATE
Close GobboNet. Back up your installation and data. Extract this complete
update over the existing GobboNet application folder. Keep your configuration,
engine, models and history. Run Repair Windows shortcuts.bat, then use
Launch GobboNet from the Start menu. Re-pin old taskbar shortcuts if needed.
The repair affects only the current user's shortcuts and needs no admin rights.
Its PowerShell policy bypass is limited to that repair process.
Setup/model selection remains available through launch.bat or its shortcut.

LINUX UPDATE
Close GobboNet. Extract into the existing application folder; retain your
configuration, models, data and linux-amd64/llama-cpp engine directory.
The top-level ./gobbonet remains the 1.7.7 setup/launch entry point.
This update includes freshly rebuilt Windows x64 and Linux x64 binaries with
the merged interface embedded. No engine or model downloads are bundled.

SOURCE AND INSTALLERS
Full application source and regression tests are included. Installer sources
are still the original 1.7.7 versions in this application-phase package; do not
use them to publish installers yet. Installer integration is the next phase.
No Git commits, tag changes or pushes were made.

VALIDATION
All 40 JavaScript suites passed, including both hotfix and 1.7.7 checks.
Go application launch, config, server, state and web UI package tests passed.
Full Go run: 18 test packages passed; supervisor tests that need /proc/self/exe
could not run in this environment, and the existing version/tag consistency
check fails because upstream 1.7.7 is tagged at the old 1.7.6 commit and that
check only recognizes v-prefixed release tags. See integration notes.
Native Windows installation and shortcut execution need a Windows smoke test.

The same supervisor and version-check failures were reproduced on untouched
1.7.7 main fdf3a8c. Packaging script, ZIP entry and version string checks
also passed (14 Python tests).
