# Security

bram 0.4.x is early-development software for trusted local workflows.
Report security issues privately to security@sltd.ca; include the version,
reproduction steps and impact. Do not publish secrets or live prompts in an issue.
No response-time or patch-support commitment is currently offered.

## Trust boundary

The API has no authentication. Keep its listener on 127.0.0.1 and use trusted
peer commands. A client that can reach it can invoke configured peers with its
own prompts. Do not bind it to a public or shared network. Loopback does not
isolate other local users or processes. TLS and browser-origin checks are not provided. The daemon admits four concurrent
peer executions; excess requests receive HTTP 429. Structured JSON/GCF wire
data and configuration are capped at 64 MiB; only generic snapshots are accepted.

Peer commands run with the daemon user's privileges and inherited environment.
They may contact external providers and incur costs. bram itself does not supply
credentials or guarantee a provider's privacy behavior. Command timeouts are
implemented, but do not constitute a sandbox or full descendant-process isolation.

Mailbox requests/responses contain plaintext prompts and answers and remain on
disk. Restrict access to the mailbox and only use a trusted responder. Peer stdout is capped at 2 MiB and stderr at 64 KiB; overflow cancels the process.
A one-second pipe-drain limit prevents inherited pipes from blocking completion.
These controls do not sandbox peers or bound their own memory/descendant processes. Use only synthetic inputs in public bug reports and tests.
