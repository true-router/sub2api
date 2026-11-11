"""用隔离 Compose 项目验证内网映射准备和容器重建。"""

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import uuid


def run(*args, **kwargs):
    return subprocess.check_output(args, text=True, **kwargs).strip()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--receipt", type=Path, required=True)
    args = parser.parse_args()
    script = Path(__file__).resolve().parents[1] / "primaryserver" / "prepare-internal-route.sh"
    project = "internal-route-e2e-" + uuid.uuid4().hex[:12]
    env = dict(os.environ)
    for key in list(env):
        if key.startswith("COMPOSE_"):
            del env[key]
    checks = []
    receipt = {"project": project, "checks": checks, "production_changed": False}
    try:
        with tempfile.TemporaryDirectory(prefix=project) as directory:
            root = Path(directory)
            (root / "data").mkdir()
            (root / "data" / "marker").write_text("保留数据\n")
            (root / ".env").write_text("E2E_SECRET=fixture-only\n")
            compose = root / "docker-compose.yml"
            original = "name: " + project + "\n" + """services:
  app:
    image: alpine:latest
    command: [sleep, '300']
    stop_grace_period: 1s
    environment:
      E2E_SECRET: ${E2E_SECRET:?required}
    volumes:
      - ./data:/data
    extra_hosts:
      - existing.internal:192.0.2.10
    depends_on:
      - dependency
  dependency:
    image: alpine:latest
    command: [sleep, '300']
    stop_grace_period: 1s
"""
            compose.write_text(original)
            def dc(*commands):
                return run("docker", "compose", "-p", project, *commands, cwd=root, env=env)
            try:
                before = json.loads(dc("config", "--format", "json"))
                dc("up", "-d", "dependency")
                dependency = dc("ps", "-q", "dependency")
                run("sh", str(script), str(root), env=env)
                after = json.loads(dc("config", "--format", "json"))
                def host_map(value):
                    return value if isinstance(value, dict) else dict(item.replace("=", ":", 1).split(":", 1) for item in value)
                hosts = host_map(after["services"]["app"].pop("extra_hosts"))
                prior_hosts = host_map(before["services"]["app"].pop("extra_hosts"))
                assert hosts == dict(prior_hosts, **{"vm2api.truerouter.cc": "host-gateway"})
                assert before == after, "合并改变了其他 Compose 配置"
                assert (root / ".env").read_text() == "E2E_SECRET=fixture-only\n"
                assert (root / "data" / "marker").read_text() == "保留数据\n"
                checks.append("仅增加目标映射，保留其他配置、环境文件和数据")
                backups = list(root.glob(".docker-compose.before-vm2api.*"))
                assert len(backups) == 1 and backups[0].read_text() == original
                assert backups[0].stat().st_mode & 0o777 == 0o600
                checks.append("保存权限为 0600 的原始配置备份")
                first = compose.read_bytes()
                run("sh", str(script), str(root), env=env)
                assert compose.read_bytes() == first
                assert list(root.glob(".docker-compose.before-vm2api.*")) == backups
                assert "${E2E_SECRET:?required}" in compose.read_text()
                checks.append("重复准备保持幂等并保留变量插值")
                for attempt in range(2):
                    dc("up", "-d", "--no-deps", "--force-recreate", "app")
                    container = dc("ps", "-q", "app")
                    actual = run("docker", "exec", container, "cat", "/etc/hosts")
                    address = run("docker", "exec", container, "getent", "hosts", "vm2api.truerouter.cc").split()[0]
                    assert "vm2api.truerouter.cc" in actual
                    assert address not in {"", "host-gateway", "127.0.0.1"}
                    assert dc("ps", "-q", "dependency") == dependency
                receipt["resolved_address"] = address
                checks.append("两次实际创建 app 均解析宿主机映射且未重建依赖")
                compose.write_text("services:\n  unrelated:\n    image: alpine:latest\n")
                invalid = compose.read_bytes()
                result = subprocess.run(["sh", str(script), str(root)], env=env, capture_output=True)
                assert result.returncode != 0 and compose.read_bytes() == invalid
                checks.append("不匹配的服务配置失败时不修改原文件")
                receipt["prepared_compose_sha256"] = hashlib.sha256(first).hexdigest()
                receipt["status"] = "passed"
            finally:
                compose.write_text(original)
                dc("down", "--remove-orphans", "--volumes")
                receipt["isolated_project_cleaned"] = True
    except BaseException as exc:
        receipt["status"] = "failed"
        receipt["error_type"] = type(exc).__name__
        raise
    finally:
        args.receipt.parent.mkdir(parents=True, exist_ok=True)
        args.receipt.write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n")


if __name__ == "__main__":
    main()
