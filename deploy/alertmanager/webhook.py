#!/usr/bin/env python3
"""Local Alertmanager webhook receiver used only by the R0-08 smoke test.

It stores the raw Alertmanager payload as one JSON line per request and exposes
it over GET /alerts so the test can assert delivery. It never forwards anything
to an external channel.
"""

import http.server
import json
import os

DATA = "/data/alerts.jsonl"


class Handler(http.server.BaseHTTPRequestHandler):
    def _send(self, code, body=b"", content_type="text/plain"):
        self.send_response(code)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_POST(self):
        if self.path != "/alert":
            self._send(404, b"not found")
            return
        length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(length)
        try:
            json.loads(body)
        except ValueError:
            self._send(400, b"invalid json")
            return
        os.makedirs("/data", exist_ok=True)
        with open(DATA, "ab") as fh:
            fh.write(body + b"\n")
        self._send(200, b"ok")

    def do_GET(self):
        if self.path == "/health":
            self._send(200, b"ok")
            return
        if self.path == "/alerts":
            data = b""
            if os.path.exists(DATA):
                with open(DATA, "rb") as fh:
                    data = fh.read()
            self._send(200, data, "application/x-ndjson")
            return
        self._send(404, b"not found")

    def log_message(self, *_args):
        pass


if __name__ == "__main__":
    http.server.HTTPServer(("0.0.0.0", 8080), Handler).serve_forever()
