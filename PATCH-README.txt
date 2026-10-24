GobboNet 1.7.8 - 1.7.6 hotfix integration

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
The top-level ./gobbonet remains the setup/launch entry point (since 1.7.7).
This update includes freshly rebuilt Windows x64 and Linux x64 binaries with
the merged interface embedded. No model is bundled. A full archive also
carries the Linux engine; an update archive that leaves it out says so in
NO-ENGINE-IN-THIS-ZIP.txt.

SOURCE AND INSTALLERS
Full application source and regression tests are included. The installer
sources now carry the hotfix as well: installer-win/gobbonet.nsi makes the
same GobboNet, Launch GobboNet and Setup or add a model shortcuts as the
repair script, and removes them on uninstall; installer-linux/build-deb.sh has
the hotfix's library-size fix. They were checked as source only (no NSIS
compiler was available): run one Windows install and uninstall before
publishing an installer. See docs/changelog/CHANGELOG-1.7.8-hotfix-installers.md.
No tags were changed and nothing was pushed.

VALIDATION (application integration, f2a0d4a)
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
