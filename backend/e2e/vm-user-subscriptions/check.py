"""Run against fresh local databases and the real HTTP servers, with simulated inference."""

import uuid
import argparse, concurrent.futures, hashlib, json, os, pathlib, secrets, shutil, socket, subprocess, tempfile, time
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


with tempfile.TemporaryDirectory(prefix="vm-sub-") as tmp, ExitStack() as stack:
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

    def member(name):
        u = api(
            "POST",
            "/admin/users",
            {
                "email": name + "@fixture.invalid",
                "password": pwd,
                "username": name,
                "balance": 100,
                "concurrency": 5,
            },
        )
        pgcmd(
            "-d",
            "sub",
            "-c",
            f"update users set username='{name}' where id={int(u['id'])}",
        )
        return (
            u["id"],
            api(
                "POST",
                "/auth/login",
                {"email": name + "@fixture.invalid", "password": pwd},
                token="",
            )["access_token"],
        )

    uid, user = member("member-a")
    other, other_token = member("member-b")

    def preview(id):
        return api("GET", f"/admin/subscriptions/vm-user-preview?user_id={id}")

    def assign(p):
        return api(
            "POST",
            "/admin/subscriptions/assign-vm-user",
            {
                "user_id": p["user_id"],
                "username": p["username"],
                "confirmed_remote_id": (p["remote"] or {}).get("id", ""),
                "validity_days": 30,
            },
        )

    p = preview(uid)
    check(
        "preview has no side effects",
        p["remote"] is None
        and pgcmd(
            "-d", "vm", "-c", "select count(*) from users where username='member-a'"
        )
        == "0",
    )
    with concurrent.futures.ThreadPoolExecutor(2) as pool:
        initial = list(pool.map(lambda _: assign(p), range(2)))
    first = initial[0]
    check(
        "concurrent first assignment reuses subscription",
        first["id"] == initial[1]["id"],
    )
    p = preview(uid)
    rid = p["remote"]["id"]
    check("new local binding", p["bound"] and first["user_id"] == uid)
    remote = "/api/panel/users/" + rid
    http(vmurl, "PATCH", remote, key, {"enabled": False})
    pgcmd("-d", "sub", "-c", f"update users set username='renamed' where id={int(uid)}")
    p = preview(uid)
    with concurrent.futures.ThreadPoolExecutor(2) as pool:
        retries = list(pool.map(lambda _: assign(p), range(2)))
    check(
        "fixed ID, activation and concurrent retry",
        all(
            s["id"] == first["id"] and s["expires_at"] == first["expires_at"]
            for s in retries
        )
        and preview(uid)["remote"]["enabled"],
    )
    http(
        vmurl,
        "POST",
        "/api/panel/users",
        key,
        {"username": "member-b", "password": pwd},
        201,
    )
    q = preview(other)
    check(
        "existing user requires first confirmation",
        not q["bound"] and q["remote"]["id"] != rid,
    )
    wrong = {**q, "remote": p["remote"]}
    try:
        assign(wrong)
    except RuntimeError as e:
        check("stale confirmation rejected", "got 409" in str(e))
    else:
        raise AssertionError("wrong identity accepted")
    second = assign(q)
    check(
        "no external binding in vm2api",
        pgcmd("-d", "vm", "-c", "select count(*) from integration_users") == "0",
    )
    check(
        "empty slot list",
        api("GET", "/subscriptions", token=user)[0]["slot_usage"]
        == {"status": "available", "slots": []},
    )
    http(
        vmurl,
        "PUT",
        remote + "/slots",
        key,
        {"vm_ids": ["vm-sim-01", "vm-sim-02"], "expected_vm_ids": []},
    )

    def observation(account, a, b):
        doc = json.dumps(
            {
                "unified": {
                    "official": {
                        "updated_at": "2026-11-19T00:00:00Z",
                        "5h": {"utilization": a},
                        "7d": {"utilization": b},
                    }
                }
            }
        )
        pgcmd(
            "-d",
            "vm",
            "-c",
            f"insert into accounts(id,extra) values('{account}','{doc}') on conflict(id) do update set extra=excluded.extra",
        )

    observation("acct-seed", 0.12, 0.64)
    observation("acct-backup", 0.81, 0.09)

    def slots():
        return api("GET", f"/subscriptions/{first['id']}/progress", token=user)[
            "slot_usage"
        ]["slots"]

    check(
        "multi-slot values match real vm2api",
        [(s["utilization_5h"], s["utilization_7d"]) for s in slots()]
        == [(0.12, 0.64), (0.81, 0.09)],
    )
    check(
        "second user isolated",
        api("GET", "/subscriptions", token=other_token)[0]["slot_usage"]["slots"] == [],
    )
    api("GET", f"/subscriptions/{first['id']}/progress", token=other_token, status=404)
    regular = api(
        "POST",
        "/admin/groups",
        {
            "name": "ordinary",
            "platform": "anthropic",
            "subscription_type": "subscription",
            "daily_limit_usd": 21,
            "rate_multiplier": 1,
        },
    )
    regular_sub = api(
        "POST",
        "/admin/subscriptions/assign",
        {"user_id": other, "group_id": regular["id"], "validity_days": 30},
    )
    check(
        "existing group retains manual limit",
        regular_sub["group"]["daily_limit_usd"] == 21
        and "slot_usage" not in regular_sub,
    )
    observation("acct-seed", 0, None)
    check(
        "null and zero remain distinct",
        slots()[0]["utilization_5h"] == 0 and slots()[0]["utilization_7d"] is None,
    )
    observation("acct-seed", 0.12, 0.64)
    access = api(
        "POST",
        "/keys",
        {"name": "vm-fixture", "group_id": first["group_id"]},
        token=user,
    )
    key_usage = http(base, "GET", "/v1/usage", access["key"])
    check(
        "API key usage exposes both slots",
        len(key_usage["subscription"]["slot_usage"]["slots"]) == 2,
    )
    result = http(
        base,
        "POST",
        "/v1/messages",
        access["key"],
        {
            "model": "claude-haiku-4-5-20251001",
            "max_tokens": 16,
            "messages": [{"role": "user", "content": "fixture"}],
        },
    )
    check("sub2api Claude call through vm2api", result.get("type") == "message")
    remote_id = str(uuid.UUID(rid))
    for _ in range(60):
        attributed = pgcmd(
            "-d",
            "vm",
            "-c",
            f"SELECT count(*) FROM usage_logs l JOIN api_keys k ON k.id=l.api_key_id WHERE k.user_id='{remote_id}' AND l.status=200 AND l.path='/v1/messages'",
        )
        if attributed != "0":
            break
        time.sleep(0.1)
    check("vm2api recorded the bound user's key", attributed != "0")
    if args.browser:
        ui_id, _ = member("ui-member")
        http(
            vmurl,
            "POST",
            "/api/panel/users",
            key,
            {"username": "ui-member", "password": pwd},
            201,
        )
        subprocess.run(["node", here / "browser.mjs", base, str(out)], check=True)
        check("browser confirmed local binding", preview(ui_id)["bound"])
    gateway.terminate()
    gateway.wait(timeout=30)
    check(
        "upstream unavailable",
        api("GET", "/subscriptions", token=user)[0]["slot_usage"]["status"]
        == "unavailable",
    )
    (out / "checks.json").write_text(
        json.dumps(
            {
                "passed": results,
                "vm_commit": run("git", "-C", root, "rev-parse", "HEAD"),
                "sub_commit": run("git", "-C", here, "rev-parse", "HEAD"),
                "binary_sha256": hashlib.file_digest(
                    args.binary.open("rb"), "sha256"
                ).hexdigest(),
            },
            indent=2,
        )
        + "\n"
    )
