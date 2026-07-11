#!/usr/bin/env python3
# End-to-end integration test: drives the REAL tabctl-mediator and tabctl CLI
# over a private D-Bus session, acting as a scripted v2 extension. Run under a
# session bus:
#
#   make build
#   dbus-run-session -- python3 scripts/integration-test.py
#
# Exits non-zero if any check fails.
import json, os, struct, subprocess, sys, threading, time

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
    state = {"closed": []}

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

TABS = [
    {"windowId": 1, "tabId": 10, "title": "Test One", "url": "https://one.example",
     "index": 0, "active": True, "pinned": False},
    {"windowId": 1, "tabId": 11, "title": "Test Two", "url": "https://two.example",
     "index": 1, "active": False, "pinned": False},
]

def compatible_handlers(wf, state, msg):
    mid, method = msg.get("id"), msg["method"]
    params = msg.get("params") or {}
    if method == "list_tabs":
        wf({"jsonrpc": "2.0", "id": mid, "result": TABS})
    elif method in ("activate_tab",):
        wf({"jsonrpc": "2.0", "id": mid, "result": None})
    elif method == "close_tabs":
        state["closed"].extend(params.get("tab_ids", []))
        wf({"jsonrpc": "2.0", "id": mid, "result": None})
    elif method == "open_urls":
        wf({"jsonrpc": "2.0", "id": mid, "result": [
            {"windowId": 1, "tabId": 99, "title": "", "url": params["urls"][0],
             "index": 5, "active": True, "pinned": False}]})

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

print("\nRESULT:", "ALL PASS" if not failures else f"FAILURES: {failures}")
sys.exit(1 if failures else 0)
