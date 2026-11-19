"""Run against fresh local databases and the real HTTP servers, with an isolated binary terminal source."""

import argparse, json, os, pathlib, re, secrets, socket, subprocess, tempfile, time
import urllib.request, urllib.error
from contextlib import ExitStack

parser = argparse.ArgumentParser()
for arg in ("vm-repo", "binary", "output"):
    parser.add_argument("--" + arg, required=True, type=pathlib.Path)
parser.add_argument(
    "--pg-bin",
    type=pathlib.Path,
    default=pathlib.Path("/opt/homebrew/opt/postgresql@17/bin"),
)
parser.add_argument("--browser", action="store_true")
args = parser.parse_args()
root = args.vm_repo.resolve()
out = args.output.resolve()
out.mkdir(parents=True, exist_ok=True)
here = pathlib.Path(__file__).resolve().parent
results = []


def check(label, condition):
    if not condition:
        raise AssertionError(label)
    results.append(label)
    print("PASS", label, flush=True)


def port():
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def http(base, method, path, key="", body=None, status=200):
    h = {"Content-Type": "application/json", "Idempotency-Key": secrets.token_hex(12)}
    if key:
        h["Authorization"] = "Bearer " + key
    req = urllib.request.Request(
        base + path,
        method=method,
        headers=h,
        data=None if body is None else json.dumps(body).encode(),
    )
    try:
        res = urllib.request.urlopen(req, timeout=30)
    except urllib.error.HTTPError as exc:
        res = exc
    with res:
        data = json.loads(res.read() or "{}")
        if res.status != status:
            raise RuntimeError(
                f"{method} {path} expected {status}, got {res.status}: "
                + str(data.get("message", data.get("error", "")))[:180]
            )
        return data.get("data", data)


def run(*cmd, **kwargs):
    return subprocess.check_output(
        cmd, text=True, stderr=subprocess.STDOUT, **kwargs
    ).strip()


def wait(base, path, proc):
    for _ in range(900):
        if proc.poll() is not None:
            raise RuntimeError("isolated server exited: " + path)
        try:
            urllib.request.urlopen(base + path, timeout=0.4).close()
            return
        except OSError:
            time.sleep(0.1)
    raise RuntimeError("startup timed out " + path)


with tempfile.TemporaryDirectory(prefix="seat-stream-") as tmp, ExitStack() as stack:
    w = pathlib.Path(tmp)
    (w / "data").mkdir()
    pg, redis, vm, sub = [port() for _ in range(4)]
    vmurl = f"http://127.0.0.1:{vm}"
    base = f"http://127.0.0.1:{sub}"
    key = "sk-kin-test-sim-001"
    pwd = "Isolated-vm-user-123!"
    bin = args.pg_bin

    def pgcmd(*parts):
        return run(
            bin / "psql",
            "-q",
            "-h",
            "127.0.0.1",
            "-p",
            str(pg),
            "-U",
            "postgres",
            "-v",
            "ON_ERROR_STOP=1",
            "-At",
            *parts,
        )

    def proc(name, *cmd, env=None, cwd=w):
        log = stack.enter_context((w / (name + ".log")).open("w+"))
        p = subprocess.Popen(cmd, stdout=log, stderr=log, cwd=cwd, env=env)

        def stop():
            if p.poll() is None:
                p.terminate()
                p.wait(timeout=30)

        stack.callback(stop)
        return p

    def dump_failure():
        import sys

        if sys.exc_info()[0]:
            for file in w.glob("*.log"):
                print(
                    file.name,
                    "\n".join(
                        l[:240]
                        for l in file.read_text(errors="replace").splitlines()[-14:]
                        if "password" not in l.lower() and "token" not in l.lower()
                    ),
                    flush=True,
                )

    run(
        bin / "initdb",
        "-D",
        w / "db",
        "-U",
        "postgres",
        "--auth=trust",
        "--no-locale",
        "--encoding=UTF8",
    )
    run(
        bin / "pg_ctl",
        "-D",
        w / "db",
        "-o",
        f"-p {pg} -k {w} -h 127.0.0.1",
        "-l",
        w / "pg.log",
        "-w",
        "start",
    )
    stack.callback(run, bin / "pg_ctl", "-D", w / "db", "-w", "stop")
    for db in ["vm", "sub"]:
        pgcmd("-d", "postgres", "-c", "create database " + db)
    pgcmd("-d", "vm", "-f", str(root / "src/lib/db/postgresql/001_baseline.sql"))
    proc(
        "redis",
        "redis-server",
        "--port",
        str(redis),
        "--bind",
        "127.0.0.1",
        "--save",
        "",
        "--appendonly",
        "no",
    )
    gateway = proc(
        "vm",
        "node",
        here / "gateway.mjs",
        str(root),
        str(w / "vm"),
        str(vm),
        f"postgres://postgres@127.0.0.1:{pg}/vm",
    )
    env = {
        **os.environ,
        "DATA_DIR": str(w / "data"),
        "AUTO_SETUP": "true",
        "SERVER_HOST": "127.0.0.1",
        "SERVER_PORT": str(sub),
        "SERVER_MODE": "release",
        "DATABASE_HOST": "127.0.0.1",
        "DATABASE_PORT": str(pg),
        "DATABASE_USER": "postgres",
        "DATABASE_PASSWORD": pwd,
        "DATABASE_DBNAME": "sub",
        "DATABASE_SSLMODE": "disable",
        "REDIS_HOST": "127.0.0.1",
        "REDIS_PORT": str(redis),
        "ADMIN_EMAIL": "admin@fixture.invalid",
        "ADMIN_PASSWORD": pwd,
        "JWT_SECRET": secrets.token_hex(32),
        "TOTP_ENCRYPTION_KEY": secrets.token_hex(32),
        "VM2API_INTEGRATION_URL": vmurl,
        "VM2API_PANEL_TOKEN": key,
        "SECURITY_URL_ALLOWLIST_ENABLED": "false",
        "SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS": "true",
        "SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP": "true",
        "LOG_OUTPUT_TO_FILE": "false",
    }
    server = proc("sub", args.binary.resolve(), env=env)
    stack.callback(dump_failure)
    wait(base, "/health", server)
    wait(vmurl, "/health", gateway)
    admin = http(
        base,
        "POST",
        "/api/v1/auth/login",
        body={"email": "admin@fixture.invalid", "password": pwd},
    )["access_token"]

    def api(m, p, b=None, token=admin, status=200):
        return http(base, m, "/api/v1" + p, token, b, status)

    compliance = api("GET", "/admin/compliance")
    api(
        "POST",
        "/admin/compliance/accept",
        {"phrase": compliance["ack_phrase_en"], "language": "en"},
    )

    seat_id = "ba579c2d7e541beca399402d86f9e5b3"
    other_seat = "bb579c2d7e541beca399402d86f9e5b3"
    unassigned_seat = "bc579c2d7e541beca399402d86f9e5b3"

    def member(name):
        email = name + "@fixture.invalid"
        user = api("POST", "/admin/users", {"email": email, "password": pwd, "username": name,
            "balance": 0, "concurrency": 1})
        token = http(base, "POST", "/api/v1/auth/login",
            body={"email": email, "password": pwd})["access_token"]
        return user, token

    def grant_subscription(user, name):
        pgcmd("-d", "sub", "-c", f"update users set username='{name}' where id={int(user['id'])}")
        preview = api("GET", f"/admin/subscriptions/vm-user-preview?user_id={user['id']}")
        api("POST", "/admin/subscriptions/assign-vm-user", {
            "user_id": user["id"], "username": preview["username"], "email": user["email"],
            "confirmed_remote_id": "", "validity_days": 30})
        return pgcmd("-d", "sub", "-c",
            f"SELECT remote_user_id FROM vm2api_subscription_bindings WHERE user_id={int(user['id'])}")

    def assign_slot(user, number, position=1):
        assert len(user) == 36 and all(c in "0123456789abcdef-" for c in user)
        pgcmd("-d", "vm", "-c", f"INSERT INTO user_slot_grants(user_id,vm_id,position) "
            f"VALUES ('{user}','private-fixture-slot-{int(number)}',{int(position)})")

    first, member_token = member("member-one")
    second, second_token = member("member-two")
    check("a user without an active subscription sees no seats",
          api("GET", "/seat-terminals", token=member_token) == {"seats": []})
    first_remote = grant_subscription(first, "member-one")
    second_remote = grant_subscription(second, "member-two")
    assign_slot(first_remote, 1)
    assign_slot(second_remote, 2)
    seats = api("GET", "/admin/seat-terminals?timezone=Asia%2FShanghai")["seats"]
    check("administrators list all assigned slots; the unassigned slot stays hidden",
          {seat["id"] for seat in seats} == {seat_id, other_seat} and
          all(set(seat) == {"id", "state", "slot", "session_id"} and len(seat["slot"]) == 12 and
              seat["session_id"] == "original-user-session-7f52cef9-5830" for seat in seats))
    check("each regular user lists only their assigned slot",
          [seat["id"] for seat in api("GET", "/seat-terminals", token=member_token)["seats"]] == [seat_id] and
          [seat["id"] for seat in api("GET", "/seat-terminals", token=second_token)["seats"]] == [other_seat])
    api("GET", "/seat-terminals", token="", status=401)
    api("GET", "/admin/seat-terminals", token=member_token, status=403)
    check("unauthenticated callers and ordinary users cannot read the administrator route", True)
    throughput = json.loads(run("node", here / "stream.mjs", base, seat_id, other_seat, unassigned_seat, admin, member_token).splitlines()[-1])
    check("slow viewer stays bounded while a second viewer receives the complete output",
          throughput["slowBytes"] <= 256*1024 and throughput["fastBytes"] >= 384*1024)
    check("binary WebSocket uses the public route, preserves both frames and rejects unauthorized users", True)
    for _ in range(60):
        if (w / "vm" / "terminal-detached").exists():
            break
        time.sleep(.05)
    check("browser disconnect reaches the Unix-socket upstream", (w / "vm" / "terminal-detached").exists())
    assign_slot(first_remote, 2, 2)
    check("an explicitly shared assignment is visible to both assigned users",
          {seat["id"] for seat in api("GET", "/seat-terminals", token=member_token)["seats"]} == {seat_id, other_seat} and
          {seat["id"] for seat in api("GET", "/seat-terminals", token=second_token)["seats"]} == {other_seat})
    pgcmd("-d", "sub", "-c", f"UPDATE user_subscriptions SET expires_at = now() - interval '1 minute' WHERE user_id={int(second['id'])}")
    check("expired subscriptions lose terminal access",
          api("GET", "/seat-terminals", token=second_token) == {"seats": []})
    api("GET", f"/seat-terminals/{other_seat}/stream", token=second_token, status=404)
    if args.browser:
        script = (here / "browser.mjs").read_text()
        screenshot = out / "seat-terminal.png"
        list_screenshot = out / "seat-list.png"
        screenshot.unlink(missing_ok=True)
        list_screenshot.unlink(missing_ok=True)
        params = json.dumps({"base": base, "email": "member-one@fixture.invalid", "password": pwd,
                             "screenshot": str(screenshot), "listScreenshot": str(list_screenshot)})
        browser = subprocess.run(["ego-browser", "nodejs", "-e", "const fixture = " + params + ";\n" + script],
            text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=90)
        print(browser.stdout, flush=True)
        if browser.returncode:
            raise RuntimeError("browser acceptance failed")
        check("regular user browser displays only assigned slots and live terminal", screenshot.is_file() and list_screenshot.is_file())
        space = int(re.search(r"EGO space (\d+)", browser.stdout).group(1))
        run("ego-browser", "nodejs", "-e", f"const task = await taskSpace({space}); await task.finish({{keep: []}})")
    with subprocess.Popen(["node", here / "revoke.mjs", base, other_seat, member_token],
                          text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE) as view:
        assert view.stdout.readline().strip() == "ready"
        try:
            pgcmd("-d", "vm", "-c",
                  f"DELETE FROM user_slot_grants WHERE user_id='{first_remote}' AND vm_id='private-fixture-slot-2'")
            _, error = view.communicate(timeout=14)
            if view.returncode != 0:
                print(error[-1400:], flush=True)
            check("removing a slot assignment closes an already open stream", view.returncode == 0)
        finally:
            if view.poll() is None:
                view.terminate()
                view.wait(timeout=3)
    check("the revoked slot is removed from the user list, while it remains assigned for the administrator",
          {seat["id"] for seat in api("GET", "/seat-terminals", token=member_token)["seats"]} == {seat_id} and
          {seat["id"] for seat in api("GET", "/admin/seat-terminals")["seats"]} == {seat_id, other_seat})
    (out / "summary.json").write_text(json.dumps({"result": "PASS", "checks": results, "flow_control": throughput,
        "scope": "real vm2api and sub2api applications; synthetic Hermit Unix-socket stream"}, indent=2)+"\n")
