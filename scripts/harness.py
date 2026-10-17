# Shared plumbing for the end-to-end tests: runs the REAL tabctl-mediator
# against a scripted extension, and the REAL tabctl CLI against it. Callers
# must already be on a private session bus (dbus-run-session); the mediators
# claim dev.slastra.TabCtl.* names on whatever bus they find.
import json, os, shutil, struct, subprocess, threading, time

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MED = os.path.join(ROOT, "build", "tabctl-mediator")
CLI = os.path.join(ROOT, "build", "tabctl")

# Firefox-family browsers pass their manifest path as argv[1], and the
# mediator detects the browser from the profile root it names.
FIREFOX_ARG = "/home/x/.mozilla/native-messaging-hosts/tabctl_mediator.json"
ZEN_ARG = "/home/x/.zen/native-messaging-hosts/tabctl_mediator.json"

HAVE_GDBUS = shutil.which("gdbus") is not None


def frame_writer(pipe):
    def wf(obj):
        data = json.dumps(obj).encode()
        pipe.write(struct.pack("<I", len(data)))
        pipe.write(data)
        pipe.flush()
    return wf


def spawn_extension(hello_protocol, handlers, manifest_arg=FIREFOX_ARG):
    """Start a mediator wired to a scripted extension. Returns (proc, state).

    handlers(wf, state, msg) answers each request the mediator sends. Replies
    the mediator sends back (to our hello) are collected in state["replies"].
    """
    med = subprocess.Popen([MED, manifest_arg], stdin=subprocess.PIPE, stdout=subprocess.PIPE)
    wf = frame_writer(med.stdin)
    state = {"closed": [], "activated": [], "navigated": [], "replies": [], "wf": wf}

    def loop():
        while True:
            hdr = med.stdout.read(4)
            if len(hdr) < 4:
                return
            (n,) = struct.unpack("<I", hdr)
            msg = json.loads(med.stdout.read(n))
            if msg.get("method") is None:
                state["replies"].append(msg)
                continue
            handlers(wf, state, msg)

    threading.Thread(target=loop, daemon=True).start()
    wf({"jsonrpc": "2.0", "id": 0, "method": "hello",
        "params": {"extensionVersion": "2.0.0", "protocolVersion": hello_protocol}})
    time.sleep(0.6)  # let the mediator claim its D-Bus name
    return med, state


def stop(*procs):
    for p in procs:
        if p is not None:
            p.terminate()
            p.wait()


def run(*args, input=None):
    return subprocess.run([CLI, *args], capture_output=True, text=True, input=input)


def gdbus(bus_name, *args):
    """Run gdbus against a mediator the way a downstream consumer would.

    args are appended after the destination and object path, so pass e.g.
    ("call", "--method", "dev.slastra.TabCtl.Browser.ListTabs"). Returns None
    when gdbus is not installed.
    """
    if not HAVE_GDBUS:
        return None
    verb, rest = args[0], args[1:]
    return subprocess.run(
        ["gdbus", verb, "--session", "--dest", f"dev.slastra.TabCtl.{bus_name}",
         "--object-path", f"/dev/slastra/TabCtl/Browser/{bus_name}", *rest],
        capture_output=True, text=True)
