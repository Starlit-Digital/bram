# Security

Bram 0.2.x is early-development software for trusted local workflows.
Report security issues privately to security@sltd.ca; include the version,
reproduction steps and impact. Do not publish secrets or live prompts in an issue.
No response-time or patch-support commitment is currently offered.

## Trust boundary

The API has no authentication. Keep its listener on 127.0.0.1 and use trusted
peer commands. A client that can reach it can invoke configured peers with its
own prompts. Do not bind it to a public or shared network. Loopback does not
isolate other local users or processes. TLS, browser-origin checks, request/output
bounds and concurrency controls are not provided in this release.

Peer commands run with the daemon user's privileges and inherited environment.
They may contact external providers and incur costs. Bram itself does not supply
credentials or guarantee a provider's privacy behavior. Command timeouts are
implemented, but do not constitute a sandbox or full descendant-process isolation.

Mailbox requests/responses contain plaintext prompts and answers and remain on
disk. Restrict access to the mailbox and only use a trusted responder. The API
buffers output in memory. Keep requests and responses modest until resource bounds
are implemented. Use only synthetic inputs in public bug reports and tests.
