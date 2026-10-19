# 1.7.6 patch 3: Windows launch and idle stand-down access

Both Windows installer sources now replace the main Start Menu and desktop
shortcuts with `gobbonet.exe launch`, and add an explicit **Launch GobboNet**
Start Menu entry. Installer finish-page launch uses the same entry point.
Working directories are explicitly set to the installation folder.

The new launch command serves an existing configuration directly and opens the
browser. It does not run the batch questionnaire or rewrite engine settings.
When no configuration exists, Windows runs the existing first-run batch setup
from beside the executable, with the legacy-server override removed for that
process. An explicitly selected missing configuration, or a corrupt existing
configuration, reports an error instead of silently replacing settings.

The separate setup/model shortcut keeps the batch workflow available. Advanced
users may still invoke the batch launcher explicitly with its legacy override.
Idle stand-down remains available only when Go manages the local engine; remote
servers are not forcibly taken over.

For an existing ZIP installation: stop GobboNet, extract the complete update
into the existing installation folder, then double-click **Repair Windows
shortcuts.bat**. This recreates only the current user's GobboNet links. It needs
no administrator rights. Its PowerShell execution-policy bypass is scoped to
that process; it does not change machine or user policy. Managed policy may
still prevent execution. Old taskbar pins may need to be unpinned and repinned.

The initial patch archive included rebuilt Go binaries and installer source.
The packaged release adds a compiled setup EXE; see the packaging changelog.
Native Windows installation and shortcut execution still need a Windows smoke test.
