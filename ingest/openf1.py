"""Tiny OpenF1 client: public API, no key, 3 requests/s, responses cached on disk."""
import hashlib
import json
import os
import time
import urllib.error
import urllib.parse
import urllib.request

BASE = "https://api.openf1.org/v1"
MIN_INTERVAL_S = 0.4  # the API allows 3 requests per second


class OpenF1:
    def __init__(self, cache_dir: str):
        self.cache_dir = cache_dir
        os.makedirs(cache_dir, exist_ok=True)
        self._last = 0.0

    def get(self, endpoint: str, **params) -> list[dict]:
        # keys may carry an operator, e.g. {"date>=": "..."}; only the values are url-encoded
        query = "&".join(f"{k}={urllib.parse.quote(str(v))}" for k, v in params.items())
        url = f"{BASE}/{endpoint}?{query}"
        path = os.path.join(self.cache_dir, hashlib.sha1(url.encode()).hexdigest() + ".json")
        if os.path.exists(path):
            with open(path, encoding="utf-8") as f:
                return json.load(f)
        data = self._fetch(url)
        with open(path, "w", encoding="utf-8") as f:
            json.dump(data, f)
        return data

    def _fetch(self, url: str) -> list[dict]:
        for attempt in range(6):
            wait = MIN_INTERVAL_S - (time.monotonic() - self._last)
            if wait > 0:
                time.sleep(wait)
            self._last = time.monotonic()
            try:
                with urllib.request.urlopen(url, timeout=60) as resp:
                    return json.load(resp)
            except urllib.error.HTTPError as e:
                if e.code == 404:  # OpenF1 answers 404 when the filter matches no rows
                    return []
                if e.code not in (429, 500, 502, 503, 504):
                    raise
            except (urllib.error.URLError, TimeoutError):
                pass
            time.sleep(2**attempt)
        raise RuntimeError(f"OpenF1 kept failing: {url}")
