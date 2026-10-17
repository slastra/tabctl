---
layout: page
title: Compatibility
---

## What this covers

Other tools are built on tabctl: launchers, tab strips, shell plugins,
Ansible roles and Nix packages (see Related Projects in the README). They
parse its output, call its D-Bus interface and download its release assets.
This page lists what they can rely on, and what may change without notice.

The two halves ship separately. The binaries come from the AUR or GitHub
releases, and the extension comes from the browser stores. Users update them
at different times, so either half may be newer than the other.

The promises below are enforced by `make test-contract`, which runs the
real binaries against a scripted extension and compares everything a
consumer can observe with the transcripts in `testdata/contract/`. CI runs
it on every push.

## Stable

### Tab IDs

- The format is `<prefix>.<window_id>.<tab_id>`. The prefix is the
  lowercased instance name: `firefox`, `zen`, `braveorigin`, and so on. A
  second profile of the same browser appends a digit (`firefox2`), and later
  profiles count up from there.
- `window_id` and `tab_id` are the browser's own integers.
- IDs are ephemeral. They are valid only until the tab closes or the browser
  restarts. Consumers should get them from `list` and use them promptly.
- Profile suffixes follow connection order, so the same profile can be
  `chrome` today and `chrome2` after a restart. `--browser chrome` matches
  every Chrome profile, and `--browser chrome2` matches only that one.
  Matching ignores case.

### CLI

Every command, flag and argument that exists today keeps working, with the
same stdout and the same exit status:

| Command | stdout | Exit |
|---|---|---|
| `list` (default `--format tsv`) | `id<TAB>title<TAB>url` per tab | 0 |
| `list --format json` | Indented JSON array of tab objects | 0 |
| `list --format simple` | One title per line (display only, cannot be mapped back to IDs) | 0 |
| `list --delimiter X` | TSV with `X` between fields | 0 |
| `list`, no tabs | `No tabs found` (`[]` with `--format json`) | 0 |
| `list`, some browsers fail | The tabs of the browsers that answered | 0 |
| `list`, every selected browser fails or none match `--browser` | Nothing | 1 |
| `activate [--focused] ID` | `Activated tab ID` | 0, or 1 on failure |
| `navigate ID URL` | `Navigated tab ID to URL` | 0, or 1 on failure |
| `close ID...`, or IDs on stdin | `Closed N tab(s)`, where N counts the tabs actually sent to a browser | 0, or 1 if any ID failed (the count is still printed) |
| `close`, empty stdin | `No tabs to close` | 0 |
| `open URL...`, or URLs on stdin | One new tab ID per line, in input order | 0, or 1 on failure (IDs already opened are still printed) |
| `open`, empty stdin | `No URLs to open` | 0 |
| `open` with more than one instance selected | Nothing | 1 |
| `status` | Human-readable, see below | 1 on a protocol mismatch, otherwise 0 |

The stdin forms read one item per line, trimming whitespace and skipping
blank lines.

JSON tab objects have these fields, and none will be removed, renamed or
change type:

| Field | Type | Notes |
|---|---|---|
| `id` | string | The tab ID above |
| `title` | string | May be empty |
| `url` | string | |
| `windowId` | number | |
| `index` | number | Position in its window |
| `active` | boolean | |
| `pinned` | boolean | |
| `favIconUrl` | string | Empty when there is none. Usually `http(s)`, sometimes a `data:` URI |

**Order.** Browsers appear sorted by instance name in byte order (`brave`,
`chrome`, `chrome10`, `chrome2`, `firefox`). Within a browser, tabs are
sorted by window, then by position in the window.

### D-Bus

The service names, object paths, interface, method signatures and the
`TabsUpdated` signal are listed in
[ARCHITECTURE](./ARCHITECTURE.html#methods-and-signals), and recorded in
`testdata/contract/dbus.golden`. A published signature never changes,
including the field list of the structs it returns. New data arrives
through a new method, as `ListTabsWithIcons` did. Failed calls return
`org.freedesktop.DBus.Error.Failed`.

### Native messaging

The mediator and the extension speak JSON-RPC, versioned by
`ProtocolVersion` (currently 2). Within one protocol version, either side
may add a method only if the other side tolerates its absence. The mediator
ignores notifications it does not know, and the extension answers unknown
requests with an error. A change that cannot meet that rule bumps
`ProtocolVersion`, and both halves must then be updated together. A test
fails the build if the Go and JavaScript protocol versions disagree.

The native host is named `tabctl_mediator` and its manifest file is
`tabctl_mediator.json`. The extension IDs are `tabctl@slastra.github.io`
(Firefox) and `baomblllgemcgbignhpbipgiofmjdhpn` (Chrome Web Store).

### Release assets

Each release publishes these names, which installers download directly:

- `tabctl-linux-amd64` and `tabctl-mediator-linux-amd64`, plus the `arm64`
  pair from 2.3.1
- `tabctl-linux-<arch>-<tag>.tar.gz` (for example
  `tabctl-linux-amd64-v2.3.1.tar.gz`) from 2.3.1, holding both binaries at
  the archive root. 1.1.x published the amd64 one, and 1.2.0 through 2.3.0
  did not
- `tabctl-chrome-extension.zip`, `tabctl-firefox-extension.zip`
- `aur-sha256.txt`

## Not stable

These can change in any release:

- Anything on stderr, including warnings and the wording of error messages.
  Use the exit status to detect failure.
- The stdout of `status`, `install`, `uninstall`, `--help` and `--version`.
  They are for people. `status`'s exit status is stable.
- How the JSON is formatted (indentation, which characters are escaped), as
  long as it decodes to the same values. Parse it with a JSON parser rather
  than matching on its text.
- What happens with input outside the table above, such as an unknown
  `--format` value (it currently prints TSV).
- Go packages. The module path does not resolve, and `internal/` and `pkg/`
  are implementation details.
- Log file names, locations and contents.
- Timeouts and timing.

## Known quirks that are part of the contract

These look like bugs, but consumers may depend on them, so they stay until a
major version:

- `No tabs found` is printed on stdout, not stderr, and exits 0. A TSV
  parser should treat that line as no tabs.
- Fields are not quoted or escaped. With `--delimiter ,`, a title
  containing a comma produces an ambiguous row. Browsers normally collapse
  whitespace in page titles, and URLs cannot contain a raw tab, so the
  default tab delimiter is very unlikely to collide.
- A browser that fails while others answer is left out of `list` with exit
  status 0. The only trace is a warning on stderr.
- `--format simple` prints an empty line for a tab with no title.
- `tabctl` with no command prints help to stdout and exits 0.

## Changing the contract

Additions are allowed in a minor or patch release: a new command, flag or
JSON field, a new D-Bus method or signal, or a new release asset. Consumers
must ignore JSON fields they do not know.

Everything else is a breaking change, including renaming, removing,
reordering, changing a type or changing stdout wording. A breaking change
needs:

1. A major version (3.0.0).
2. A release note naming what changed and what to do instead.
3. A heads-up to the known downstream projects before the release.
4. Where possible, one release in which the old and new behaviors coexist.

When `make test-contract` fails, it prints the difference from the golden
transcripts. If the diff only adds lines, the change is an addition:
regenerate with `UPDATE_GOLDEN=1 make test-contract` and commit the new
transcripts with the change. If it changes or removes a line, the change is
breaking, and the steps above apply before the transcripts are regenerated.
