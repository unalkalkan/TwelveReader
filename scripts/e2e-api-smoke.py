#!/usr/bin/env python3
"""Container-friendly TwelveReader API smoke test.

This intentionally uses only Python's standard library so it can run on hosts
or CI images without extra dependencies. It validates the deterministic stub
provider stack started by docker-compose.e2e.yaml.
"""

from __future__ import annotations

import argparse
import json
import mimetypes
import sys
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import Any


@dataclass
class Response:
    status: int
    headers: dict[str, str]
    body: bytes

    def json(self) -> Any:
        return json.loads(self.body.decode("utf-8"))

    def text(self) -> str:
        return self.body.decode("utf-8", errors="replace")


class SmokeFailure(RuntimeError):
    pass


def request(
    method: str,
    url: str,
    *,
    body: bytes | None = None,
    headers: dict[str, str] | None = None,
    timeout: float = 10,
    accept_statuses: set[int] | None = None,
) -> Response:
    req = urllib.request.Request(url, data=body, headers=headers or {}, method=method)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            response = Response(resp.status, dict(resp.headers.items()), resp.read())
    except urllib.error.HTTPError as exc:
        response = Response(exc.code, dict(exc.headers.items()), exc.read())
    except Exception as exc:  # noqa: BLE001 - want concise smoke diagnostics
        raise SmokeFailure(f"{method} {url} failed: {type(exc).__name__}: {exc}") from exc

    if accept_statuses is None:
        accept_statuses = {200}
    if response.status not in accept_statuses:
        preview = response.text()[:500]
        raise SmokeFailure(
            f"{method} {url} returned HTTP {response.status}; expected {sorted(accept_statuses)}; body={preview!r}"
        )
    return response


def endpoint(base_url: str, path: str) -> str:
    return urllib.parse.urljoin(base_url.rstrip("/") + "/", path.lstrip("/"))


def wait_for_health(base_url: str, timeout_seconds: float) -> None:
    deadline = time.time() + timeout_seconds
    last_error = "not attempted"
    while time.time() < deadline:
        try:
            resp = request("GET", endpoint(base_url, "/health/live"), timeout=3)
            data = resp.json()
            if data.get("status") == "healthy":
                print("ok health/live")
                return
            last_error = f"unexpected health payload: {data!r}"
        except Exception as exc:  # noqa: BLE001
            last_error = str(exc)
        time.sleep(1)
    raise SmokeFailure(f"backend did not become healthy within {timeout_seconds}s: {last_error}")


def multipart_form(fields: dict[str, str], file_field: str, file_path: Path, content_type: str | None = None) -> tuple[bytes, str]:
    boundary = f"----TwelveReaderSmoke{int(time.time() * 1000)}"
    chunks: list[bytes] = []

    for name, value in fields.items():
        chunks.extend(
            [
                f"--{boundary}\r\n".encode(),
                f'Content-Disposition: form-data; name="{name}"\r\n\r\n'.encode(),
                value.encode(),
                b"\r\n",
            ]
        )

    guessed_type = content_type or mimetypes.guess_type(file_path.name)[0] or "application/octet-stream"
    chunks.extend(
        [
            f"--{boundary}\r\n".encode(),
            f'Content-Disposition: form-data; name="{file_field}"; filename="{file_path.name}"\r\n'.encode(),
            f"Content-Type: {guessed_type}\r\n\r\n".encode(),
            file_path.read_bytes(),
            b"\r\n",
            f"--{boundary}--\r\n".encode(),
        ]
    )
    return b"".join(chunks), f"multipart/form-data; boundary={boundary}"


def expect_keys(label: str, data: dict[str, Any], keys: list[str]) -> None:
    missing = [key for key in keys if key not in data]
    if missing:
        raise SmokeFailure(f"{label} missing keys {missing}; payload={data!r}")


def poll_status(base_url: str, book_id: str, timeout_seconds: float) -> dict[str, Any]:
    terminal = {"voice_mapping", "ready", "synthesizing", "synthesized", "synthesis_error", "error"}
    deadline = time.time() + timeout_seconds
    last_status: dict[str, Any] | None = None
    while time.time() < deadline:
        resp = request("GET", endpoint(base_url, f"/api/v1/books/{book_id}/status"))
        data = resp.json()
        last_status = data
        status = data.get("status")
        print(f"ok status status={status} progress={data.get('progress')}")
        if status in terminal:
            return data
        time.sleep(1)
    raise SmokeFailure(f"book did not reach expected status within {timeout_seconds}s; last={last_status!r}")


def post_voice_mapping_if_needed(base_url: str, book_id: str, status: dict[str, Any]) -> None:
    if status.get("status") != "voice_mapping":
        return

    personas_resp = request("GET", endpoint(base_url, f"/api/v1/books/{book_id}/personas"))
    personas = personas_resp.json()
    discovered = personas.get("discovered") or ["narrator"]
    if not isinstance(discovered, list) or not discovered:
        discovered = ["narrator"]
    print(f"ok personas discovered={discovered}")

    payload = {
        "persons": [
            {"id": str(person), "provider_voice": "stub-voice-1"}
            for person in discovered
        ]
    }
    request(
        "POST",
        endpoint(base_url, f"/api/v1/books/{book_id}/voice-map?initial=true"),
        body=json.dumps(payload).encode("utf-8"),
        headers={"Content-Type": "application/json"},
    )
    print(f"ok voice-map mapped={len(payload['persons'])}")


def poll_final_status(base_url: str, book_id: str, timeout_seconds: float) -> dict[str, Any]:
    terminal = {"synthesized", "synthesis_error", "error"}
    deadline = time.time() + timeout_seconds
    last_status: dict[str, Any] | None = None
    while time.time() < deadline:
        resp = request("GET", endpoint(base_url, f"/api/v1/books/{book_id}/status"))
        data = resp.json()
        last_status = data
        status = data.get("status")
        print(
            "ok final-poll status=%s synthesized=%s/%s"
            % (status, data.get("synthesized_segments"), data.get("total_segments"))
        )
        if status in terminal:
            return data
        time.sleep(1)
    raise SmokeFailure(f"book did not finish within {timeout_seconds}s; last={last_status!r}")


def smoke(base_url: str, startup_timeout: float, processing_timeout: float) -> None:
    wait_for_health(base_url, startup_timeout)

    ready = request("GET", endpoint(base_url, "/health/ready"), accept_statuses={200, 503})
    print(f"ok health/ready http={ready.status}")

    info = request("GET", endpoint(base_url, "/api/v1/info")).json()
    expect_keys("info", info, ["version", "storage_adapter"])
    print(f"ok info version={info['version']} storage={info['storage_adapter']}")

    providers = request("GET", endpoint(base_url, "/api/v1/providers")).json()
    for key in ("llm", "tts", "ocr"):
        if key not in providers or not providers[key]:
            raise SmokeFailure(f"providers.{key} is empty; payload={providers!r}")
    print(f"ok providers {providers}")

    voices = request("GET", endpoint(base_url, "/api/v1/voices")).json()
    voice_list = voices.get("voices") or []
    if not voice_list:
        raise SmokeFailure(f"voices response has no voices; payload={voices!r}")
    print(f"ok voices count={len(voice_list)}")

    with tempfile.TemporaryDirectory(prefix="twelvereader-smoke-") as tmp:
        fixture = Path(tmp) / "smoke-book.txt"
        # Six paragraphs trigger the initial voice mapping gate configured by the pipeline.
        fixture.write_text(
            "Chapter 1\n\n"
            "This is the first paragraph for the smoke test.\n\n"
            "This is the second paragraph for the smoke test.\n\n"
            "This is the third paragraph for the smoke test.\n\n"
            "This is the fourth paragraph for the smoke test.\n\n"
            "This is the fifth paragraph for the smoke test.\n\n"
            "This is the sixth paragraph for the smoke test.\n",
            encoding="utf-8",
        )
        body, content_type = multipart_form(
            {"title": "Smoke Test Book", "author": "Hermes", "language": "en"},
            "file",
            fixture,
            "text/plain",
        )
        upload = request(
            "POST",
            endpoint(base_url, "/api/v1/books"),
            body=body,
            headers={"Content-Type": content_type},
            accept_statuses={201},
            timeout=20,
        ).json()
    expect_keys("upload", upload, ["id", "status", "orig_format"])
    book_id = upload["id"]
    print(f"ok upload book_id={book_id} status={upload.get('status')}")

    status = poll_status(base_url, book_id, processing_timeout)
    post_voice_mapping_if_needed(base_url, book_id, status)
    if status.get("status") == "voice_mapping":
        status = poll_final_status(base_url, book_id, processing_timeout)
    if status.get("status") == "error":
        raise SmokeFailure(f"book pipeline ended in error: {status!r}")

    book = request("GET", endpoint(base_url, f"/api/v1/books/{book_id}")).json()
    print(f"ok book status={book.get('status')} total_segments={book.get('total_segments')}")

    personas = request("GET", endpoint(base_url, f"/api/v1/books/{book_id}/personas")).json()
    print(f"ok personas final={personas}")

    segments = request("GET", endpoint(base_url, f"/api/v1/books/{book_id}/segments")).json()
    if not isinstance(segments, list) or not segments:
        raise SmokeFailure(f"segments response is empty; payload={segments!r}")
    print(f"ok segments count={len(segments)}")

    stream = request("GET", endpoint(base_url, f"/api/v1/books/{book_id}/stream"))
    stream_lines = [line for line in stream.text().splitlines() if line.strip()]
    if not stream_lines:
        raise SmokeFailure("stream endpoint returned no NDJSON lines")
    first_stream = json.loads(stream_lines[0])
    audio_url = first_stream.get("audio_url")
    print(f"ok stream lines={len(stream_lines)} first_audio_url={audio_url}")

    if audio_url:
        audio = request("GET", endpoint(base_url, audio_url), accept_statuses={200, 404})
        if status.get("status") == "synthesized" and audio.status != 200:
            raise SmokeFailure(f"expected synthesized audio to be available, got HTTP {audio.status}")
        print(f"ok audio http={audio.status} content_type={audio.headers.get('Content-Type')}")

    download = request(
        "GET",
        endpoint(base_url, f"/api/v1/books/{book_id}/download"),
        accept_statuses={200, 500},
        timeout=20,
    )
    if status.get("status") == "synthesized" and download.status != 200:
        raise SmokeFailure(f"expected synthesized download to be available, got HTTP {download.status}: {download.text()[:300]!r}")
    print(f"ok download http={download.status} bytes={len(download.body)}")


def main() -> int:
    parser = argparse.ArgumentParser(description="Run TwelveReader API smoke tests against a running backend.")
    parser.add_argument("--base-url", default="http://localhost:8080")
    parser.add_argument("--startup-timeout", type=float, default=90)
    parser.add_argument("--processing-timeout", type=float, default=60)
    args = parser.parse_args()

    try:
        smoke(args.base_url, args.startup_timeout, args.processing_timeout)
    except SmokeFailure as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        return 1
    except Exception as exc:  # noqa: BLE001
        print(f"FAIL: unexpected {type(exc).__name__}: {exc}", file=sys.stderr)
        return 1

    print("PASS TwelveReader API smoke")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
