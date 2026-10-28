# Why we chose the name bram

AI research note, October 20, 2026.

Codex chose `Bram` as a short name that fit the existing tool family and evoked a sturdy gate or threshold between AI agents. The original Codex chat records this rationale explicitly. Its working title was `bridge`; Bram supplied the product identity, while bridging remained the description of its function.

## The specific word Bram

In the September 13, 2026 Codex chat [Create macOS AI bridge daemon](codex://threads/01a09adb-1a26-75f0-aeba-c7ce512cd3f0), the owner asked Codex to create the daemon and choose a name instead of `bridge`, with this direction:

> Something that is in the same line of thinking as Vigil, Clyde, Loom, Nora, etc.

Codex then announced its choice:

> I’m going to follow that pattern and name the new daemon **Bram**, short and tool-like, with the sense of a sturdy gate/threshold between AI agents.

The recorded reasons were its brevity, its fit alongside the existing tools, and the image of a gate or threshold through which AI agents communicate. Codex proposed the name while scaffolding the project. The owner subsequently used Bram in requests about the tool.

The gate or threshold image is the association Codex stated when choosing the name. The chat does not establish a dictionary meaning, acronym expansion or namesake, so those should not be added as historical explanations.

The source is the opening turn, `01a09adb-9f8d-7fb3-83b3-35d4254fb7fa`. The naming message is `msg_0953763afb169003016aa69eaa039487d284a95a464dccc3a4`. The local conversation archive is `/Users/cs/.codex/sessions/2026/09/13/rollout-2026-09-13T07-00-45-01a09adb-1a26-75f0-aeba-c7ce512cd3f0.jsonl.zst`.

## Original naming record

The first bram commit, `7d39f7fed17918c13c711a54b720b3c39eadd179`, dated September 13, 2026 and titled “Initial Bram daemon scaffold”, contains this passage in `README.md`, lines 7 and 8:

> The name replaces the working title `bridge`. Bram is the product/repository
> name; bridging is only the job it performs.

The preceding paragraph describes the intended purpose: let Codex talk to Grok, or another command line AI tool, through one small, auditable local service. Together these passages support the following reading: **bridge described the job, and Bram supplied the product identity.** This is a paraphrase of the recorded distinction, not evidence of a separate etymology.

The name was already present in the first commit. The local bram history therefore contains no earlier committed naming discussion or implementation under the working title `bridge`.

## Why the explanation disappeared

Commit `5d81962a89fdad974bb3485f73c3cb2265b3ee20`, dated October 6, 2026 and titled “Release Bram 0.2.0 as a Starlit Digital 0BSD project”, rewrote the README for the public release and removed the naming paragraph. Its absence from the current README does not indicate a new naming decision.

Later that day, commit `a366f9f72383483c0f297ba35ccebe150932dcc1`, titled “Lowercase product names throughout documentation”, changed the displayed spelling from `Bram` to `bram`. The website's `docs/developer-product-copy.md`, under “Lowercase product names”, records the shared lowercase convention for loom, nora, bram and clyde.

## Sources and search scope

The specific naming rationale comes from the Codex chat quoted above, retrieved through the app's chat history. The repository history corroborates the replacement of `bridge`, the removal of the README explanation and the later lowercase spelling:

```sh
git show 7d39f7fed17918c13c711a54b720b3c39eadd179:README.md
git show 5d81962a89fdad974bb3485f73c3cb2265b3ee20 -- README.md
git show a366f9f72383483c0f297ba35ccebe150932dcc1 -- README.md
```

The search covered current text, including ignored and hidden text outside dependency and build directories, across `/private/var/www` and the additional local checkouts listed below. Commit messages and changes mentioning bram in Markdown, text and reStructuredText files were searched across all locally available refs in 26 discovered checkouts, including two test fixtures and the Voci profiles worktree.

The repositories covered were local95.ca; PayCal's paycal, paycal-private, paycal-ledgers, paycaltech, vigil-private and vigil-public; and the studio's nora, Voci, bram, sltd.ca, VociAudioWorks, GrandmaPizza.ca, Voci-profiles-login, s0phia, clyde, VociPulse, VociTagWorks, loom and voci.cc. Additional checkouts were `/Users/cs/goshi`, `/Users/cs/nms`, `/Users/cs/omrei` and `/Users/cs/Documents/paycal-voice`. Studio archives and `/private/ai-notes` were also searched for naming references.

The follow up chat search located and read the original September 13 conversation, including the owner's naming request and Codex's explanation before the project was committed. This resolves the earlier note's uncertainty about the specific word Bram. Remote branches absent locally and other machines were not searched.
