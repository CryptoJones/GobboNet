# 1.7.7 application hotfix integration

Recovered the cumulative three-patch hotfix source from the published
GobboNet-1.7.6-hotfix.zip and merged it onto 1.7.7 main fdf3a8c.
The hotfix ZIP identifies base 314bdc9 and contains changes absent from Git.

Preserved 1.7.7 thinking controls, per-character context budgets, user limits,
LAN address picker, model filters, web refresh and ZIP launcher behavior.
Both Phone & Other Devices and Network access panels now coexist in CONFIG.

Restored exact custom hostname management, server legacy-history import,
atomic browser migration/restores, recovery UI and sync failure protection,
and the Windows direct launch command and ZIP shortcut repair scripts.

Application packaging now includes the shortcut repair scripts and update
instructions. Installer integration remains a separate phase. (It followed in
1.7.8: see `CHANGELOG-1.7.8-hotfix-installers.md`.) The historical
1.7.6 changelogs describe the recovered patches, not validation performed now.

Validation: all 40 JavaScript suites passed. Go tests for 18 packages passed,
including cmd/gobbonet, config, state, server, and webui. Seven supervisor tests
fail at os.Executable because this runtime has no /proc/self/exe. The existing
version test compares 1.7.7 against the nearest v-prefixed tag (v1.7.6) and
fails; the source/tag mismatch is documented in the audit, and no refs were
changed or checks weakened. Native Windows execution remains untested.

The same supervisor and version-check failures were reproduced on untouched
1.7.7 main fdf3a8c. Packaging script, ZIP entry and version string checks
also passed (14 Python tests).
