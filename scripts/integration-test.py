#!/usr/bin/env python3
# End-to-end integration test: drives the REAL tabctl-mediator and tabctl CLI
# over a private D-Bus session, acting as a scripted v2 extension. Run under a
# session bus:
#
#   make build
#   dbus-run-session -- python3 scripts/integration-test.py
#
# Exits non-zero if any check fails.
import json, os, shutil, struct, subprocess, sys, threading, time

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MED = os.path.join(ROOT, "build", "tabctl-mediator")
CLI = os.path.join(ROOT, "build", "tabctl")
# The /.mozilla/ arg makes the mediator detect Firefox.
MED_ARGS = [MED, "/home/x/.mozilla/native-messaging-hosts/tabctl_mediator.json"]

failures = []

def check(name, cond, detail=""):
    print(("PASS" if cond else "FAIL"), name, ("" if cond else detail))
    if not cond:
        failures.append(name)

def frame_writer(pipe):
    def wf(obj):
        data = json.dumps(obj).encode()
        pipe.write(struct.pack("<I", len(data)))
        pipe.write(data)
        pipe.flush()
    return wf

def spawn_extension(hello_protocol, handlers):
    """Start a mediator wired to a scripted extension. Returns (proc, state)."""
    med = subprocess.Popen(MED_ARGS, stdin=subprocess.PIPE, stdout=subprocess.PIPE)
    wf = frame_writer(med.stdin)
    state = {"closed": [], "activated": []}

    def loop():
        while True:
            hdr = med.stdout.read(4)
            if len(hdr) < 4:
                return
            (n,) = struct.unpack("<I", hdr)
            msg = json.loads(med.stdout.read(n))
            method = msg.get("method")
            if method is None:
                continue  # response to our hello
            handlers(wf, state, msg)

    threading.Thread(target=loop, daemon=True).start()
    wf({"jsonrpc": "2.0", "id": 0, "method": "hello",
        "params": {"extensionVersion": "2.0.0", "protocolVersion": hello_protocol}})
    time.sleep(0.6)  # let the mediator claim its D-Bus name
    return med, state

def run(*args):
    return subprocess.run([CLI, *args], capture_output=True, text=True)

FAVICON = "https://one.example/favicon.ico"

TABS = [
    {"windowId": 1, "tabId": 10, "title": "Test One", "url": "https://one.example",
     "index": 0, "active": True, "pinned": False, "favIconUrl": FAVICON},
    # No favIconUrl key at all: an extension predating the field must not
    # break decoding, it just yields an empty string.
    {"windowId": 1, "tabId": 11, "title": "Test Two", "url": "https://two.example",
     "index": 1, "active": False, "pinned": False},
]

HAVE_GDBUS = shutil.which("gdbus") is not None

def gdbus(bus_name, method):
    """Call a method on a live mediator the way a downstream consumer would."""
    if not HAVE_GDBUS:
        return None
    return subprocess.run(
        ["gdbus", "call", "--session", "--dest", f"dev.slastra.TabCtl.{bus_name}",
         "--object-path", f"/dev/slastra/TabCtl/Browser/{bus_name}",
         "--method", f"dev.slastra.TabCtl.Browser.{method}"],
        capture_output=True, text=True)

def handlers_for(tabs):
    """Scripted extension responses backed by a specific tab set."""
    def handlers(wf, state, msg):
        mid, method = msg.get("id"), msg["method"]
        params = msg.get("params") or {}
        if method == "list_tabs":
            wf({"jsonrpc": "2.0", "id": mid, "result": tabs})
        elif method == "activate_tab":
            state["activated"].append((params.get("tab_id"), params.get("focused")))
            wf({"jsonrpc": "2.0", "id": mid, "result": None})
        elif method == "close_tabs":
            state["closed"].extend(params.get("tab_ids", []))
            wf({"jsonrpc": "2.0", "id": mid, "result": None})
        elif method == "open_urls":
            wf({"jsonrpc": "2.0", "id": mid, "result": [
                {"windowId": 1, "tabId": 99, "title": "", "url": params["urls"][0],
                 "index": 5, "active": True, "pinned": False}]})
    return handlers

compatible_handlers = handlers_for(TABS)

# --- Scenario 1: compatible extension -------------------------------------
med, state = spawn_extension(2, compatible_handlers)
try:
    r = run("list")
    check("list composes firefox.1.10", "firefox.1.10" in r.stdout, r.stdout)
    r = run("list", "--format", "json")
    data = json.loads(r.stdout) if r.returncode == 0 else []
    check("json id + windowId", any(t["id"] == "firefox.1.11" for t in data) and data[0]["windowId"] == 1, r.stdout)
    check("activate ok", run("activate", "firefox.1.10").returncode == 0)
    check("open prints firefox.1.99", "firefox.1.99" in run("open", "https://new.example").stdout)
    check("close ok", run("close", "firefox.1.11").returncode == 0)
    check("close routed numeric 11", 11 in state["closed"], str(state["closed"]))
    s = run("status")
    check("status OK", s.returncode == 0 and "OK" in s.stdout, s.stdout)
    check("unroutable close errors", run("close", "zen.1.5").returncode != 0)

    # --- favicons ---------------------------------------------------------
    r = run("list", "--format", "json")
    data = json.loads(r.stdout) if r.returncode == 0 else []
    check("json carries favIconUrl", data and data[0]["favIconUrl"] == FAVICON, r.stdout)
    check("missing favicon is an empty string, not null",
          len(data) > 1 and data[1]["favIconUrl"] == "", r.stdout)

    # tsv is positional, so adding a field must not shift its columns.
    r = run("list")
    check("tsv columns unchanged",
          all(len(line.split("\t")) == 3 for line in r.stdout.strip().split("\n")), r.stdout)

    # The frozen signature: a consumer built against a(iissibb) must still be
    # able to call ListTabs. This is the whole reason icons went on a new
    # method rather than being appended to the existing one.
    legacy = gdbus("Firefox", "ListTabs")
    if legacy is None:
        print("SKIP frozen-signature checks (gdbus not installed)")
    else:
        check("legacy ListTabs still marshals", legacy.returncode == 0, legacy.stderr)
        check("legacy ListTabs has no favicon field", FAVICON not in legacy.stdout, legacy.stdout)

        rich = gdbus("Firefox", "ListTabsWithIcons")
        check("ListTabsWithIcons marshals", rich.returncode == 0, rich.stderr)
        check("ListTabsWithIcons carries the favicon", FAVICON in rich.stdout, rich.stdout)
finally:
    med.terminate()
    med.wait()
    time.sleep(0.3)

# --- Scenario 2: incompatible extension (version guard) -------------------
med, _ = spawn_extension(1, compatible_handlers)
try:
    r = run("list")
    check("guard: list fails loudly", r.returncode != 0 and "protocol v1" in r.stderr, r.stderr)
    check("guard: status shows MISMATCH", "MISMATCH" in run("status").stdout)
finally:
    med.terminate()
    med.wait()
    time.sleep(0.3)

# --- Scenario 3: two profiles of the same browser (issue #2) ---------------
# Each browser profile spawns its own mediator, and they all detect the same
# browser. The second must claim a distinct bus name instead of dying on the
# collision, so both profiles' tabs stay reachable at once.
PROFILE2_TABS = [
    {"windowId": 7, "tabId": 70, "title": "Profile Two", "url": "https://p2.example",
     "index": 0, "active": True, "pinned": False},
]
med1, state1 = spawn_extension(2, compatible_handlers)
med2, state2 = spawn_extension(2, handlers_for(PROFILE2_TABS))
med3 = None
try:
    r = run("list")
    check("both profiles listed at once",
          "firefox.1.10" in r.stdout and "firefox2.7.70" in r.stdout, r.stdout)

    r = run("list", "--browser", "firefox")
    check("--browser firefox spans every profile",
          "firefox.1.10" in r.stdout and "firefox2.7.70" in r.stdout, r.stdout)

    r = run("list", "--browser", "firefox2")
    check("--browser firefox2 narrows to one profile",
          "firefox2.7.70" in r.stdout and "firefox.1.10" not in r.stdout, r.stdout)

    # The switch-to-tab path: activate must reach the profile that owns the
    # tab, and only that one. Numeric tab IDs are per-profile, so a misroute
    # would silently activate an unrelated tab in the other profile.
    check("activate a second-profile tab", run("activate", "firefox2.7.70").returncode == 0)
    check("profile 2 received the activate", (70, False) in state2["activated"], str(state2["activated"]))
    check("profile 1 did not see it", state1["activated"] == [], str(state1["activated"]))

    check("--focused reaches the owning profile",
          run("activate", "--focused", "firefox2.7.70").returncode == 0)
    check("focused flag carried through", (70, True) in state2["activated"], str(state2["activated"]))

    check("activate routes back to profile 1", run("activate", "firefox.1.10").returncode == 0)
    check("profile 1 received its own activate", (10, False) in state1["activated"], str(state1["activated"]))

    check("close routes to the right profile", run("close", "firefox2.7.70").returncode == 0)
    check("profile 2 received the close", 70 in state2["closed"], str(state2["closed"]))

    s = run("status")
    check("status lists both instances",
          "Firefox" in s.stdout and "Firefox2" in s.stdout, s.stdout)

    # First profile's browser quits: the survivor keeps serving on its own name.
    med1.terminate()
    med1.wait()
    time.sleep(0.5)
    r = run("list")
    check("survivor still served after the other profile quits",
          "firefox2.7.70" in r.stdout and "firefox.1.10" not in r.stdout, r.stdout)

    # The freed name goes to the next profile that connects.
    med3, _ = spawn_extension(2, compatible_handlers)
    r = run("list")
    check("freed instance name is reused", "firefox.1.10" in r.stdout, r.stdout)
finally:
    for p in (med1, med2, med3):
        if p is not None:
            p.terminate()
            p.wait()

print("\nRESULT:", "ALL PASS" if not failures else f"FAILURES: {failures}")
sys.exit(1 if failures else 0)
