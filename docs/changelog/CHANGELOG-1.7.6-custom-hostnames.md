# Patch: custom hostnames for DuckDNS and Tailscale

## Report

After upgrading, an IP address worked but the same server reached through a
DuckDNS hostname returned `unrecognised Host header`. Rollback restored access.

## Cause

The Go server validates Host before login or routing. IP literals, localhost
and `.local` names are allowed automatically; other names require
`allowed_hosts`. The check exists in the Go server since commit dda9698
(v1.7.3). The report does not identify the prior version or server path, so the
exact upgrade transition is not yet established. This is GobboNet's rejection,
not evidence of a DuckDNS failure. DNS resolving both addresses to the same
machine does not give them the same HTTP Host header.

## Change

CONFIG now exposes a Network access panel. Owners can connect through a working
IP, sign in and save exact hostnames or pasted HTTP(S) addresses. Names are
normalized and deduplicated, saved to the existing config, and applied live.
Failed persistence leaves the running allowlist unchanged. Removing the active
hostname requires connecting by IP first, preventing accidental lockout.

Unknown-host errors explain recovery. Config-file and CLI workflows remain
available, and configured names with a trailing dot now compare consistently.

The endpoint remains behind Host and authentication checks, and browser writes
must carry a matching Origin even if the owner disabled authentication. It does
not resolve DNS to infer trust, trust forwarded headers, allow wildcards, or
change network listeners. A blocked hostname cannot approve itself. All other
hostnames remain blocked and password authentication continues to apply.

## Validation

Go tests cover the reported Host mismatch, live updates and reload, exact-name
boundaries, authentication, cross-origin rejection, invalid input, failed
writes, removal and concurrent reads/writes. The Node suite exercises the
actual UI handlers, including load/save/clear and error states. Actual VPN
routing, TLS termination and Windows launcher behavior are separate from these
Host-header regressions and require their respective environments.
