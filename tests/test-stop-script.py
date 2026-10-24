"""stop-gobbonet.bat stops only THIS install's processes (issue #63, PR #59).

THE BUG
-------
stop-gobbonet.bat ended llama-server.exe by image name -- `taskkill /IM` --
so stopping GobboNet, or running its uninstaller, also killed any llama.cpp the
user was running on their own. The legacy fileserver.ps1 was matched by name
anywhere on the command line, the same over-reach with a rarer name. PR #59
(orcmaul) added stopping the launch.bat windows, matched by "launch.bat" on any
command line, which would have closed other programs' launch.bat windows too.

THE FIX
-------
One routine, :stop_matching, stops the processes of a name that belong to the
install folder (GN_DIR): by where the executable lives (llama-server), or by
this folder's file on the command line (launch.bat, fileserver.ps1). The
process running the script and every process above it are skipped. GN_DIR is
the script's own folder unless one is passed; the installer passes $INSTDIR,
because it runs a copy of the script from its temp folder.

WHAT THIS RUNS
--------------
A. Static checks on the script and the installer.
B. The routine's real PowerShell, lifted out of the .bat, run in pwsh against
   a fake process table (Get-CimInstance / Stop-Process / Get-Process stubbed),
   when pwsh is available: set GOBBONET_TEST_PWSH or put pwsh on PATH.
   Otherwise B is skipped and says so. Windows PowerShell 5.1 is what runs it
   for real; the script sticks to syntax both accept.

Run: python3 tests/test-stop-script.py
"""
import json, os, pathlib, re, shutil, subprocess, sys, tempfile

ROOT = pathlib.Path(__file__).resolve().parents[1]
RAW = (ROOT / "stop-gobbonet.bat").read_bytes().decode("ascii")
BAT = RAW.replace("\r\n", "\n")
NSI = (ROOT / "installer-win" / "gobbonet.nsi").read_text()
passed = failed = 0


def ok(cond, label, extra=""):
    global passed, failed
    if cond:
        passed += 1
        print("  \u2713 " + label)
    else:
        failed += 1
        print("  \u2717 " + label + ("\n      " + str(extra) if extra else ""))


def code(text):
    return "\n".join(l for l in text.split("\n") if not l.strip().startswith("::") and not l.strip().lower().startswith("rem "))


C = code(BAT)
print("\n=== A. what the script stops, and how it picks ===")
ok(not re.search(r'(?i)(stop_image\s+"llama-server|/IM\s+"?llama-server|imagename eq llama-server)', C),
   "llama-server.exe is never stopped by image name")
calls = re.findall(r'call :stop_matching "([^"]+)" (\w+) "([^"]*)" "([^"]+)"', C)
ok(("llama-server.exe", "path", "", "llama-server") in calls,
   "llama-server.exe is stopped only when its executable is inside this install", calls)
ok(("cmd.exe", "cmdline", "launch.bat", "the launch.bat window") in calls,
   "launch.bat windows are matched by this install's launch.bat (PR #59, scoped)", calls)
ok(("powershell.exe", "cmdline", "fileserver.ps1", "the legacy PowerShell file server") in calls,
   "the legacy fileserver.ps1 is matched by this install's copy", calls)
ok("*fileserver.ps1*" not in C and "*launch.bat*" not in C, "no bare-name wildcard matches remain")
order = [C.find('call :stop_image "gobbonet.exe"'), C.find('"cmd.exe" cmdline'), C.find('"llama-server.exe" path')]
ok(-1 not in order and order == sorted(order),
   "order: gobbonet.exe, then the launch.bat windows (a legacy monitor would restart the engine), then llama-server")

ok('set "GN_DIR=%~dp0"' in C, "the install folder defaults to the script's own folder")
ok(re.search(r'else if exist "%~1\\" \(\s*set "GN_DIR=%~f1"', C) is not None, "and can be passed as an argument")
for where in ("$PLUGINSDIR", "$INSTDIR"):
    call = "nsExec::ExecToLog '\"$SYSDIR\\cmd.exe\" /c \"\"" + where + "\\stop-gobbonet.bat\" /quiet \"$INSTDIR\"\"'"
    ok(call in NSI, f"the installer passes $INSTDIR when it runs the copy in {where}", call)

m = re.search(r'^powershell -NoProfile -ExecutionPolicy Bypass -Command "([^"]+)" >nul 2>&1$', BAT[BAT.find(":stop_matching\n"):], re.M)
PS = m.group(1) if m else ""
ok(PS != "", ":stop_matching runs one PowerShell command")
ok(PS and not any(c in PS for c in "!%^"), "that command carries no ! % or ^, which cmd eats with delayed expansion on")
ok("$env:GN_DIR" in PS and "$env:GN_NAME" in PS and "$env:GN_FILE" in PS,
   "folder, name and file arrive through the environment, not through cmd quoting")
for bad in ("??", "?.", "&&", "||"):
    ok(bad not in PS, f"no PowerShell 7-only operator {bad!r} (Windows PowerShell 5.1 runs this)")

end = BAT[BAT.find("rem Pause only when something needs reading"):BAT.find("if defined FAILED exit /b 1")]
ok(end.count("pause") == 1 and end.find("pause") < end.find(") else if"), "it pauses only when something would not stop (PR #59)")
ok("timeout /t 10 >nul" in end and "/nobreak" not in BAT, "otherwise it closes itself, and a keypress still closes it at once")
depth, inside = 0, []
for i, l in enumerate(BAT.split("\n"), 1):
    s = l.strip()
    if s.startswith("::"):
        continue
    if depth and re.match(r"(?i)(echo|rem)\b", s) and ("(" in s or ")" in s):
        inside.append(i)
    if not re.match(r"(?i)(echo|rem)\b", s):
        depth = max(0, depth + s.count("(") - s.count(")"))
ok(not inside, "no echo or rem inside a ( ) block carries a parenthesis", inside)
ok(RAW.count("\n") == RAW.count("\r\n"), "stop-gobbonet.bat is still CRLF throughout")


# ---------------------------------------------------------------------------
print("\n=== B. the PowerShell routine against a fake process table ===")
PWSH = os.environ.get("GOBBONET_TEST_PWSH") or shutil.which("pwsh")
if not PS or not PWSH:
    print("  - SKIPPED: no pwsh (set GOBBONET_TEST_PWSH to a PowerShell 7 binary)")
else:
    PRELUDE = r"""
$global:procs = @($env:GN_TEST_PROCS | ConvertFrom-Json)
$global:dead = @{}
function Get-CimInstance { param([string]$ClassName, [string]$Filter, $ErrorAction)
  if ($Filter -match '^ProcessId=(\d+)$') {
    $id = [int]$Matches[1]
    if ($id -eq $PID) { return [pscustomobject]@{ ProcessId = $PID; ParentProcessId = [int]$env:GN_TEST_SELF_PARENT } }
    return $global:procs | Where-Object { $_.ProcessId -eq $id } }
  if ($Filter -match "^Name='(.+)'$") { $n = $Matches[1]; return $global:procs | Where-Object { $_.Name -eq $n } }
}
function Stop-Process { param([int]$Id, [switch]$Force, $ErrorAction)
  Add-Content -Path $env:GN_TEST_LOG -Value $Id
  $p = $global:procs | Where-Object { $_.ProcessId -eq $Id }
  if (-not $p.Survives) { $global:dead[$Id] = 1 } }
function Get-Process { param([int]$Id, $ErrorAction) if (-not $global:dead.ContainsKey($Id)) { [pscustomobject]@{ Id = $Id } } }
function Start-Sleep { param($Milliseconds) }
"""
    GN = r"C:\Users\ana\AppData\Local\GobboNet" + "\\"

    def run(name, mode, file, procs, self_parent=5000, gn_dir=GN):
        with tempfile.TemporaryDirectory() as t:
            log = pathlib.Path(t) / "killed.txt"
            env = dict(os.environ, GN_DIR=gn_dir, GN_NAME=name, GN_MODE=mode, GN_FILE=file,
                       GN_TEST_PROCS=json.dumps(procs), GN_TEST_SELF_PARENT=str(self_parent), GN_TEST_LOG=str(log))
            if not file:
                env.pop("GN_FILE")
            r = subprocess.run([PWSH, "-NoProfile", "-NonInteractive", "-Command", PRELUDE + PS],
                               env=env, capture_output=True, text=True, timeout=60)
            killed = sorted(int(x) for x in log.read_text().split()) if log.exists() else []
            return r.returncode, killed, r.stderr.strip()

    def proc(pid, name, exe=None, cmdline=None, parent=1, survives=False):
        return {"ProcessId": pid, "Name": name, "ExecutablePath": exe, "CommandLine": cmdline,
                "ParentProcessId": parent, "Survives": survives}

    servers = [
        proc(101, "llama-server.exe", GN + r"llama-cpp\llama-server.exe"),
        proc(102, "llama-server.exe", r"C:\Users\ana\AppData\Local\GOBBONET\llama-cpp\llama-b10456\llama-server.exe"),
        proc(201, "llama-server.exe", r"C:\src\llama.cpp\build\bin\Release\llama-server.exe"),
        proc(202, "llama-server.exe", r"C:\Users\ana\AppData\Local\GobboNet2\llama-cpp\llama-server.exe"),
        proc(203, "llama-server.exe", None),  # another user's, or elevated: no path visible
    ]
    rc, killed, err = run("llama-server.exe", "path", "", servers)
    ok(rc == 10 and killed == [101, 102],
       "llama-server: this install's (any case, any subfolder) stopped; the user's own build, "
       "GobboNet2's and one with no visible path left running", (rc, killed, err))
    rc, killed, err = run("llama-server.exe", "path", "", servers[2:])
    ok(rc == 0 and killed == [], "nothing of this install's running: exit 0, nothing stopped", (rc, killed, err))
    stuck = [proc(101, "llama-server.exe", GN + r"llama-cpp\llama-server.exe", survives=True)]
    rc, killed, err = run("llama-server.exe", "path", "", stuck)
    ok(rc == 11 and killed == [101], "one that will not die: exit 11, so the script reports it and pauses", (rc, killed, err))

    windows = [
        proc(301, "cmd.exe", cmdline=r'C:\WINDOWS\system32\cmd.exe /c ""' + GN + r'launch.bat" "'),
        proc(302, "cmd.exe", cmdline=r'cmd.exe /k "' + GN + r'launch.bat"', parent=1),
        proc(303, "cmd.exe", cmdline=r'C:\WINDOWS\system32\cmd.exe /c "D:\Games\Valheim\launch.bat"'),
        proc(304, "cmd.exe", cmdline=r'"C:\WINDOWS\system32\cmd.exe"'),
        # The stop script's own cmd (5000), started from a launch.bat window (4000).
        proc(5000, "cmd.exe", cmdline=r'C:\WINDOWS\system32\cmd.exe /c "' + GN + r'stop-gobbonet.bat"', parent=4000),
        proc(4000, "cmd.exe", cmdline=r'cmd.exe /k "' + GN + r'launch.bat"', parent=1),
    ]
    rc, killed, err = run("cmd.exe", "cmdline", "launch.bat", windows)
    ok(rc == 10 and killed == [301, 302],
       "launch.bat windows: this install's stopped; another program's launch.bat and a plain prompt left", (rc, killed, err))
    ok(4000 not in killed and 5000 not in killed,
       "the window running the stop script, and the launch.bat window it was started from, are never ended")

    ps = [
        proc(401, "powershell.exe", cmdline=r'powershell -NoProfile -ExecutionPolicy Bypass -File "' + GN + r'fileserver.ps1"'),
        proc(402, "powershell.exe", cmdline=r'powershell -File "C:\tools\homelab\fileserver.ps1"'),
    ]
    rc, killed, err = run("powershell.exe", "cmdline", "fileserver.ps1", ps)
    ok(rc == 10 and killed == [401], "legacy fileserver.ps1: this install's stopped, another folder's left", (rc, killed, err))

    odd = r"C:\Users\ana [old]\GobboNet's (copy)" + "\\"
    rc, killed, err = run("llama-server.exe", "path", "", [proc(501, "llama-server.exe", odd + r"llama-cpp\llama-server.exe")], gn_dir=odd)
    ok(rc == 10 and killed == [501], "a folder with brackets, an apostrophe and spaces still matches", (rc, killed, err))

print(f"\n{passed} passed, {failed} failed")
sys.exit(1 if failed else 0)
