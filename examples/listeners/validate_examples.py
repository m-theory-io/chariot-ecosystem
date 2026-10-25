"""Real-server fixture checks; no Go source edits and no external NSQD required.

Usage: python3 examples/listeners/validate_examples.py /absolute/path/to/chariot-server
Uses fixed example ports plus 18089 for Dev REST. Child termination is automated
test cleanup, not a substitute for a manual/container process-signal smoke test.
"""

import json
import os
from pathlib import Path
import shutil
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request


REPO = Path(__file__).resolve().parents[2]
FIXTURES = Path(__file__).resolve().parent


def request(url, payload=None):
    data = None if payload is None else json.dumps(payload).encode()
    req = urllib.request.Request(url, data=data, headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=1) as response:
        return response.status, json.load(response)


def environment(root, script, rest):
    env = os.environ.copy()
    env.update(
        CHARIOT_DATA_PATH=str(root), CHARIOT_TREE_PATH=str(root),
        CHARIOT_DIAGRAM_PATH=str(root / "diagrams"), CHARIOT_SANDBOX_ROOT=str(root / "sandboxes"),
        CHARIOT_LISTENER_SCRIPTS_DIR="listeners", CHARIOT_LISTENERS_FILE="listeners.json",
        CHARIOT_FUNCTION_LIB="stlib.json", CHARIOT_BOOTSTRAP=script,
        CHARIOT_SECRET_PROVIDER="file", CHARIOT_SECRET_FILE_PATH=str(root / "secrets.local.json"),
        CHARIOT_HEADLESS="true", CHARIOT_DEV_REST_ENABLED=str(rest).lower(),
        CHARIOT_PORT="18089", CHARIOT_SSL="false", CHARIOT_MCP_ENABLED="false",
        CHARIOT_MCP_EXECUTE_ENABLED="false", CHARIOT_SHUTDOWN_TIMEOUT="5",
    )
    return env


def run_case(binary, root, script, url, check, rest=False, payload=None):
    with tempfile.TemporaryFile() as log:
        proc = subprocess.Popen([binary], env=environment(root, script, rest),
                                stdout=log, stderr=subprocess.STDOUT)
        try:
            deadline = time.monotonic() + 15
            while True:
                if proc.poll() is not None or time.monotonic() > deadline:
                    log.seek(0)
                    raise AssertionError(log.read().decode())
                try:
                    response = request(url, payload)
                    break
                except (OSError, urllib.error.URLError):
                    continue
            check(response)
            if rest:
                while True:
                    try:
                        assert request("http://127.0.0.1:18089/health")[0] == 200
                        break
                    except (OSError, urllib.error.URLError):
                        if time.monotonic() > deadline:
                            raise
                check(request(url, payload))
        finally:
            proc.terminate()
            try:
                proc.wait(timeout=10)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait()
                raise
        assert proc.returncode == 0, f"cleanup exit {proc.returncode}"


def health_check(expected):
    def check(response):
        assert response[0] == 200 and response[1]["bootstrap_runs"] == expected, response
        _, echo = request("http://127.0.0.1:18081/echo", {"input": "hello"})
        assert echo["request"]["body"]["input"] == "hello", echo
        assert echo["bootstrap_runs"] == expected, echo
        req = urllib.request.Request("http://127.0.0.1:18081/echo", data=b"{",
                                     headers={"Content-Type": "application/json"})
        try:
            with urllib.request.urlopen(req, timeout=1):
                raise AssertionError("malformed JSON accepted")
        except urllib.error.HTTPError as error:
            assert error.code == 400, error.code
    return check


def main(binary):
    with tempfile.TemporaryDirectory(prefix="chariot-wp8-") as tmp:
        root = Path(tmp)
        for path in FIXTURES.iterdir():
            if path.suffix in (".ch", ".json") or path.name == "bootstrap-count.txt":
                shutil.copy(path, root / path.name)
        for expected in (1, 2):
            run_case(binary, root, "bootstrap-http-only.ch", "http://127.0.0.1:18080/health",
                     health_check(expected), rest=True)
            assert (root / "bootstrap-count.txt").read_text() == str(expected)
        print("PASS two HTTP services, strict JSON 400, Dev REST, persistent counter 1 then 2")

        def devcheck(response):
            assert response[0] == 200 and response[1]["mode"] == "devtest", response
        run_case(binary, root, "devtest.ch", "http://127.0.0.1:18082/devtest", devcheck)
        print("PASS DevTest script through real interpreter/host (not supervisor API acceptance)")

        shutil.copy(REPO / "data/go-chariot/listeners/listenDecision.ch", root / "listenDecision.ch")

        def decisioncheck(response):
            assert response[0] == 200 and response[1]["service"] == "decision-example", response
        run_case(binary, root, "listenDecision.ch", "http://127.0.0.1:8050/decision",
                 decisioncheck, payload={"input": "example"})
        print("PASS decision script POST returns defined JSON body")

        with socket.socket() as reserve:
            reserve.bind(("127.0.0.1", 0))
            unused_port = reserve.getsockname()[1]
        env = environment(root, "bootstrap.ch", False)
        env["CHARIOT_NSQ_ADDR"] = f"127.0.0.1:{unused_port}"
        result = subprocess.run([binary], env=env, stdout=subprocess.PIPE,
                                stderr=subprocess.STDOUT, timeout=20)
        output = result.stdout.decode()
        assert result.returncode != 0 and "connection refused" in output, output
        for port in (18080, 18081):
            with socket.socket() as probe:
                assert probe.connect_ex(("127.0.0.1", port)) != 0, f"rollback leaked {port}"
        print("PASS full bootstrap reaches NSQ connection; absent-daemon rollback closes HTTP ports")
        print("NOT RUN connected NSQ delivery, images/push, parent package/race tests, manual signal smoke")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        raise SystemExit(__doc__)
    main(str(Path(sys.argv[1]).resolve(strict=True)))