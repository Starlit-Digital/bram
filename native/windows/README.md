# Bram for Windows

This companion implements the existing Bram for Mac protocol in `~/BramMac/main.swift`.
The current Windows target is WIN11 at 10.0.0.244, user win11\cs, profile C:\Users\cs.
CSPC at 10.0.0.200 is a different machine and is out of scope for installation and tests.
Chris redirected the work to WIN11 on October 20, 2026. The CSPC build checks were
read only, with no Bram, firewall or startup installation there.

Build from the repository on a .NET 10 SDK host:

```sh
dotnet publish native/windows/BramWindows.csproj -c Release -r win-x64 --self-contained true -o .build/windows
```

The package includes .NET 10. No SDK, runtime or Bonjour service is installed on
Windows. Windows DNS-SD APIs publish and browse the Bonjour service. A bounded legacy
query over multicast and the local IPv4 broadcast address also refreshes discovery
every 15 seconds, receiving unicast replies on an ephemeral socket; BouncyCastle.Cryptography provides
the same ChaCha20 Poly1305 combined format used by Swift CryptoKit. Exact NuGet
versions are recorded in the project and lock file. Their license notices ship
with the package. Bram's own source remains under the repository's 0BSD license.

Run BramWindows.exe with no arguments, or `tray`, for the tray app. Use `help`
for local commands. CLI commands communicate with the tray process using a named
pipe restricted to the same Windows user. Peer secrets never appear in status.
The machine identity is saved once. Data defaults to
C:\Users\cs\Downloads\grok-muse\bram, with outbox, delivered, failed, machine.json,
peers.json, network.json and discovered.json. Activity is logged to the parent
folder's bram.log. BRAM_HUB selects a different parent folder for isolated tests.

peers.json is a map keyed by machine ID, with id, name, secret and paired fields.
network.json contains only the local machine ID and TCP listening port. Times
on disk use ISO 8601 UTC with Z. The data directory ACL allows the owner and SYSTEM.
Only TestNoop delivery is currently supported on Windows. It never types into apps.

The tray includes pairing code display, nearby device pairing, peer status,
recent activity, log opening and the messages folder. Codes contain 128 random
bits, expire in two minutes, and are consumed on successful pairing. Windows
initiated pairing checks the returned machine ID against the discovered device.

Installer: install.ps1 stages immutable releases below %LOCALAPPDATA%\Bram,
retains prior releases, records provenance, and registers the Bram Windows task
for the current user's interactive sign in. The process runs at Limited privilege.
TCP 47831 and UDP 5353 inbound rules permit only the Private network profile and
LocalSubnet, scoped to the installed app. The installer does not change network
categories. WIN11's network CC was initially Public; Chris approved changing it
to Private for this test. Password and UAC prompts are handled by Chris.

Receipts mean queue acceptance, not final app delivery. Sender records gain a
relayed timestamp. Receiver records gain delivered after TestNoop processing.
Offline peers stay queued. Duplicate IDs are ignored while records are retained.
Replay history depends on retaining those records. A lost pairing response can
leave trust on one side; back up and remove only that new peer entry on both
sides before retrying. There is no automatic repair or key replacement.

Mac changes for this work are limited to generic nearby device wording.
The old ca.starlitdigital.bram and bram-mailbox services are not modified.

WIN11 already had Smart App Control enabled and its Windows Firewall disabled.
Chris approved enabling only the Private firewall and adding exact hash scoped
supplemental policies for Bram test builds during this task. approve-build.ps1
prepares such a policy; -Apply is explicit. install.ps1 -ApproveBuildHash invokes
it for the staged BramWindows.dll only. It adds no signer or writable path trust,
and does not disable Smart App Control. Records and policy removal IDs are kept
under %LOCALAPPDATA%\Bram\policies. Other firewall profiles retain their existing state.

## Qualification, October 20, 2026

The final native DNS-SD build passed on the Mac mini and WIN11. Each machine
found the other under _bram._tcp. Pairing succeeded with the Mac as initiator;
the Windows initiator was separately verified against an isolated Mac identity.
Only the production Mac and win11 trust entries remain after verification.
Queued TestNoop messages reached delivered and were logged at both ends in both
directions, before and after restarting both apps. Wrong codes were rejected.
Both receivers rejected modified authentication tags. The Windows tray menu and
its paired peer list were inspected in the connected WIN11 desktop. Open Log
was verified to display bram.log in Notepad.

Run `python3 native/windows/test-interop.py` on the Mac to repeat qualification.
It needs the installed win11ctl helper and both apps running, creates only
TestNoop messages, and leaves an evidence JSON in a private temporary folder.
The latest successful result is also copied to .build/windows-qualification/latest.json
and C:\Users\cs\BramWindows\qualification.json. Repeat runs retain message records
and trust backups. They do not invoke the old Go daemon or mailbox services.

Verified assembly SHA256:
BD52F48FC14CFAB1D0A3B91171B9469E1E552E205D31B3FFB1EEDBCAE808139C
Installed release:
C:\Users\cs\AppData\Local\Bram\releases\20261020234545\BramWindows.exe
Source copy: C:\Users\cs\BramWindows\source
Active exact hash policy: {3D5DC785-7D14-4DF7-BE1A-CFBFCEB0C965}
Prior test build exceptions were removed, retaining their policy artifacts and
release files as history. Smart App Control remains enabled. Only the Private
Windows Firewall profile was enabled; other profiles retain their prior state.

A Mac Bonjour cache refresh was needed while diagnosing discovery. The final
run rediscovered peers after both app restarts without that refresh. A later
Windows peer menu check still showed the Mac online. The log timestamps exposed
about two and a half minutes of clock skew between the machines; time formats
are UTC with Z, but this can affect comparisons across their logs. System clocks
were not changed. The initial full WIN11 reboot exposed discovery recovery
that needed a Mac app restart. Packet captures showed WIN11 multicast queries
leaving WiFi but not arriving on the Mac. A direct local broadcast Bonjour query
reached the Mac and received a unicast reply. The Windows recovery query now uses
that route alongside multicast, without a hardcoded Mac address or cache. It
accepts only same subnet responses on port 5353 with matching query IDs and uses
Windows' DNS parser with packet and record bounds. Discovery still grants no
trust; pairing and authenticated encryption remain required. No firewall or
Smart App Control protection was disabled for this fix.

The final WIN11 reboot passed. Bram started at cs sign in, found the Mac, retained
pairing and delivered TestNoop messages both ways. Bram on the Mac retained PID
67495 across the test and its pairing file did not change. Evidence is
.build/windows-qualification/reboot.json and
C:\Users\cs\BramWindows\reboot-qualification.json. The fallback needs an IPv4
LAN where local broadcast reaches responders; IPv6 only recovery and adapter
changes during an already running session are not qualified.
Delivery receipts and retry policy beyond queue acceptance remain future work.
