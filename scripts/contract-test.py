#!/usr/bin/env python3
# Contract test: records everything a downstream consumer can observe from the
# REAL tabctl and tabctl-mediator binaries, and compares it with the golden
# transcripts in testdata/contract/. COMPATIBILITY.md says what is promised;
# these files are the executable form of that promise.
#
#   make test-contract                   # compare
#   UPDATE_GOLDEN=1 make test-contract   # rewrite the goldens
#
# A golden diff that only adds lines is an addition to the contract. A diff
# that changes or removes a line breaks a consumer somewhere, and needs a
# major version and a note in COMPATIBILITY.md, not a regenerated file.
#
# Needs a private session bus (the Makefile target wraps it in
# dbus-run-session) and gdbus, since the D-Bus transcript calls the mediator
# the way out-of-tree consumers do.
import difflib, json, os, re, shlex, subprocess, sys, time

import harness
from harness import FIREFOX_ARG, ZEN_ARG, run, spawn_extension, stop

GOLDEN_DIR = os.path.join(harness.ROOT, "testdata", "contract")
UPDATE = os.environ.get("UPDATE_GOLDEN") == "1"

if not harness.HAVE_GDBUS:
    sys.exit("contract test needs gdbus (glib2 / libglib2.0-bin)")
if "/run/user/" in os.environ.get("DBUS_SESSION_BUS_ADDRESS", ""):
    # That is the desktop session bus. The fake mediators would collide with
    # the user's real browsers there, and the CLI would list their real tabs.
    sys.exit("run under dbus-run-session (make test-contract), not on the desktop bus")

VERSION = run("--version").stdout.split()[-1]

# --- Fixtures ---------------------------------------------------------------
# The extension sorts tabs by window, then index, before replying; the fake
# extensions return them already in that order.
FIREFOX_TABS = [
    {"windowId": 1, "tabId": 10, "title": "Inbox (3), Mail", "url": "https://mail.example/inbox",
     "index": 0, "active": True, "pinned": True, "favIconUrl": "https://mail.example/favicon.ico"},
    {"windowId": 1, "tabId": 11, "title": "日本語 ✓ café 🦊", "url": "https://unicode.example/%E8%B7%AF?q=1&x=2",
     "index": 1, "active": False, "pinned": False, "favIconUrl": "data:image/png;base64,iVBORw0KGgo="},
    # No favIconUrl key, as from an extension older than 2.2.0, and no title.
    {"windowId": 1, "tabId": 12, "title": "", "url": "about:blank",
     "index": 2, "active": False, "pinned": False},
    {"windowId": 2, "tabId": 20, "title": "Second window", "url": "https://two.example/",
     "index": 0, "active": True, "pinned": False, "favIconUrl": ""},
]
FIREFOX2_TABS = [
    {"windowId": 7, "tabId": 70, "title": "Profile Two", "url": "https://p2.example/",
     "index": 0, "active": True, "pinned": False, "favIconUrl": ""},
]
ZEN_TABS = [
    {"windowId": 3, "tabId": 30, "title": "Zen", "url": "https://zen.example/",
     "index": 0, "active": True, "pinned": False, "favIconUrl": "https://zen.example/icon.png"},
]


def masked(msg):
    """A wire message with the parts that are not contract masked out."""
    msg = json.loads(json.dumps(msg))
    if "id" in msg:
        msg["id"] = "<id>"
    result = msg.get("result")
    if isinstance(result, dict) and "mediatorVersion" in result:
        result["mediatorVersion"] = "<version>"
    return json.dumps(msg, sort_keys=True, ensure_ascii=False)


def extension(tabs, open_window):
    """A scripted extension. state["mode"] switches list_tabs between
    answering, answering with no tabs, and failing; every request is logged."""
    def handlers(wf, state, msg):
        state["log"].append(masked(msg))
        mid, method = msg.get("id"), msg["method"]
        params = msg.get("params") or {}

        def ok(result):
            wf({"jsonrpc": "2.0", "id": mid, "result": result})

        def err(message):
            wf({"jsonrpc": "2.0", "id": mid, "error": {"code": -32000, "message": message}})

        if method == "list_tabs":
            mode = state.get("mode", "ok")
            if mode == "fail":
                err("Failed to list tabs: scripted failure")
            else:
                ok([] if mode == "empty" else tabs)
        elif method in ("activate_tab", "navigate_tab"):
            if params.get("tab_id") == 999:
                err("No tab with id: 999.")
            else:
                ok(None)
        elif method == "close_tabs":
            ok(None)
        elif method == "open_urls":
            state["next_tab"] = state.get("next_tab", 100) + 1
            ok([{"windowId": open_window, "tabId": state["next_tab"], "title": "",
                 "url": params["urls"][0], "index": 9, "active": True, "pinned": False}])
    return handlers


class Transcript:
    def __init__(self):
        self.parts = []

    def section(self, title):
        self.parts.append(f"## {title}\n\n")

    def raw(self, text):
        self.parts.append(text if text.endswith("\n") else text + "\n")

    def cli(self, *args, stdin=None, show_stdout=True):
        r = run(*args, input=stdin)
        cmd = shlex.join(["tabctl", *args])
        if stdin is not None:
            shown = stdin.replace("\\", "\\\\").replace("\n", "\\n")
            cmd = f"printf {shlex.quote(shown)} | {cmd}"
        out = r.stdout if show_stdout else "(stdout is not part of the contract)\n"
        if out and not out.endswith("\n"):
            out += "\n[no trailing newline]\n"
        self.parts.append(f"$ {cmd}\n{out}[exit {r.returncode}]\n\n")
        return r

    def text(self):
        return "".join(self.parts)


def gdbus_lines(bus, *args):
    r = harness.gdbus(bus, *args)
    out = (r.stdout + r.stderr).replace(VERSION, "<version>")
    # Error replies end with the browser's message, which is not contract;
    # the error name is.
    out = re.sub(r"(GDBus\.Error:[\w.]+): .*", r"\1: <message>", out)
    return out.strip() + "\n"


cli, wire, bus = Transcript(), Transcript(), Transcript()
procs = []
try:
    ff, ff_state = spawn_extension(2, extension(FIREFOX_TABS, 1), FIREFOX_ARG)
    ff2, ff2_state = spawn_extension(2, extension(FIREFOX2_TABS, 7), FIREFOX_ARG)
    zen, zen_state = spawn_extension(2, extension(ZEN_TABS, 3), ZEN_ARG)
    procs = [ff, ff2, zen]
    for st in (ff_state, ff2_state, zen_state):
        st.setdefault("log", [])

    # --- CLI ----------------------------------------------------------------
    cli.raw("# tabctl CLI contract: stdout and exit status.\n"
            "# Three mediators: Firefox, a second Firefox profile (Firefox2), and Zen.\n"
            "# stderr is not part of the contract and is not recorded.\n\n")

    cli.section("Listing")
    cli.cli("list")
    cli.cli("list", "--format", "tsv")
    cli.cli("list", "--format", "json")
    cli.cli("list", "--format", "simple")
    cli.cli("list", "--delimiter", ",")
    cli.cli("list", "--browser", "firefox")
    cli.cli("list", "--browser", "firefox2")
    cli.cli("list", "--browser", "ZEN")
    cli.cli("list", "--browser", "nosuch")

    cli.section("A browser with no tabs")
    zen_state["mode"] = "empty"
    cli.cli("list", "--browser", "zen")
    cli.cli("list", "--browser", "zen", "--format", "json")
    cli.cli("list", "--browser", "zen", "--format", "simple")

    cli.section("A browser that fails while others answer")
    zen_state["mode"] = "fail"
    cli.cli("list")
    cli.cli("list", "--browser", "zen")
    zen_state["mode"] = "ok"

    cli.section("Activate and navigate")
    cli.cli("activate", "firefox.1.10")
    cli.cli("activate", "--focused", "firefox2.7.70")
    cli.cli("activate", "firefox.1.999")
    cli.cli("activate", "nosuch.1.2")
    cli.cli("activate", "firefox.x.y")
    cli.cli("activate")
    cli.cli("navigate", "zen.3.30", "https://navigated.example/")
    cli.cli("navigate", "firefox.1.999", "https://navigated.example/")
    cli.cli("navigate", "zen.3.30", "")

    cli.section("Close")
    cli.cli("close", "firefox.1.11", "firefox.2.20")
    cli.cli("close", stdin="firefox.1.12\nzen.3.30\n")
    cli.cli("close", "firefox.1.10", "nosuch.1.1")
    cli.cli("close", stdin="")

    cli.section("Open")
    cli.cli("open", "--browser", "zen", "https://a.example/", "https://b.example/")
    cli.cli("open", "--browser", "firefox2", stdin="https://c.example/\n\n  https://d.example/  \n")
    cli.cli("open", "https://e.example/")
    cli.cli("open", "--browser", "firefox", "https://e.example/")
    cli.cli("open", stdin="")

    cli.section("Status (exit status only)")
    cli.cli("status", show_stdout=False)

    # --- D-Bus --------------------------------------------------------------
    bus.raw("# D-Bus contract, as a consumer sees it through gdbus.\n"
            "# <version> is the mediator's build version.\n\n")
    bus.section("Introspection (dev.slastra.TabCtl.Firefox)")
    bus.raw(gdbus_lines("Firefox", "introspect", "--xml"))
    bus.raw("")
    for method, args in (
        ("ListTabs", []),
        ("ListTabsWithIcons", []),
        ("GetInfo", []),
        ("ActivateTab", ["10", "false"]),
        ("ActivateTab", ["999", "false"]),
        ("NavigateTab", ["10", "https://navigated.example/"]),
        ("CloseTabs", ["[10, 11]"]),
        ("OpenTab", ["https://f.example/"]),
    ):
        bus.section(f"{method}({', '.join(args)})")
        bus.raw(gdbus_lines("Firefox", "call", "--method",
                            f"dev.slastra.TabCtl.Browser.{method}", *args))
        bus.raw("")

    bus.section("TabsUpdated signal after a tabs_changed notification")
    mon = subprocess.Popen(
        ["stdbuf", "-oL", "gdbus", "monitor", "--session", "--dest", "dev.slastra.TabCtl.Firefox",
         "--object-path", "/dev/slastra/TabCtl/Browser/Firefox"],
        stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True)
    time.sleep(0.5)
    ff_state["wf"]({"jsonrpc": "2.0", "method": "tabs_changed"})
    time.sleep(0.5)
    mon.terminate()
    signals = [l for l in mon.communicate()[0].splitlines() if "TabsUpdated" in l]
    bus.raw("\n".join(signals) + "\n")

    # --- Native messaging ---------------------------------------------------
    # The extension and the mediator ship separately (stores vs AUR), so the
    # contract is the set of message shapes the extension must understand,
    # not how many calls a given CLI command makes. Each distinct message is
    # recorded once, grouped by mediator in the order each first sent it.
    wire.raw("# Native-messaging contract: every distinct message the mediator sends\n"
             "# the extension during the CLI and D-Bus sections, once each.\n"
             "# Message ids and the mediator version are masked; keys are sorted.\n\n")
    states = (ff_state, ff2_state, zen_state)
    wire.section("Reply to the extension's hello")
    wire.raw("\n".join(dict.fromkeys(masked(r) for st in states for r in st["replies"])) + "\n")
    wire.raw("")
    wire.section("Requests")
    wire.raw("\n".join(dict.fromkeys(line for st in states for line in st["log"])) + "\n")
finally:
    stop(*procs)

# --- Compare ----------------------------------------------------------------
failed = False
for name, transcript in (("cli.golden", cli), ("dbus.golden", bus), ("native-messaging.golden", wire)):
    path = os.path.join(GOLDEN_DIR, name)
    got = transcript.text()
    if UPDATE:
        os.makedirs(GOLDEN_DIR, exist_ok=True)
        with open(path, "w") as f:
            f.write(got)
        print(f"WROTE {os.path.relpath(path, harness.ROOT)}")
        continue
    try:
        with open(path) as f:
            want = f.read()
    except FileNotFoundError:
        want = ""
    if got == want:
        print(f"PASS {name}")
    else:
        failed = True
        print(f"FAIL {name}: output differs from the golden transcript")
        sys.stdout.writelines(difflib.unified_diff(
            want.splitlines(True), got.splitlines(True),
            f"testdata/contract/{name}", "actual"))

if failed:
    print("\nIf this change is intended, it is a contract change: read COMPATIBILITY.md,\n"
          "then regenerate with UPDATE_GOLDEN=1 make test-contract and review the diff.")
sys.exit(1 if failed else 0)
