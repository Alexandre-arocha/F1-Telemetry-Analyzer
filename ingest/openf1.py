"""Tiny OpenF1 client: public API, no key, rate limited, responses cached on disk."""
import hashlib
import json
import os
import threading
import time
import urllib.error
import urllib.parse
import urllib.request

BASE = "https://api.openf1.org/v1"
MIN_INTERVAL_S = 0.4  # the API documents 3 requests per second
MAX_INTERVAL_S = 3.0  # it also throttles harder under sustained load, so we slow down on 429
ATTEMPTS = 8


class OpenF1:
    def __init__(self, cache_dir: str):
        self.cache_dir = cache_dir
        os.makedirs(cache_dir, exist_ok=True)
        self._interval = MIN_INTERVAL_S
        self._next = 0.0
        self._lock = threading.Lock()  # get() runs in a thread pool; the spacing is global

    def get(self, endpoint: str, **params) -> list[dict]:
        # keys may carry an operator, e.g. {"date>": "..."}; only the values are url-encoded
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

    def _wait_turn(self) -> None:
        with self._lock:
            now = time.monotonic()
            start = max(now, self._next)
            self._next = start + self._interval
        if start > now:
            time.sleep(start - now)

    def _adjust(self, throttled: bool) -> None:
        with self._lock:
            if throttled:
                self._interval = min(self._interval * 1.5, MAX_INTERVAL_S)
            else:
                self._interval = max(self._interval * 0.98, MIN_INTERVAL_S)

    def _fetch(self, url: str) -> list[dict]:
        for attempt in range(ATTEMPTS):
            self._wait_turn()
            try:
                with urllib.request.urlopen(url, timeout=90) as resp:
                    data = json.load(resp)
                self._adjust(throttled=False)
                return data
            except urllib.error.HTTPError as e:
                if e.code == 404:  # OpenF1 answers 404 when the filter matches no rows
                    return []
                if e.code not in (429, 500, 502, 503, 504):
                    raise
                self._adjust(throttled=e.code == 429)
            except (urllib.error.URLError, TimeoutError):
                pass
            time.sleep(min(2**attempt, 20))
        raise RuntimeError(f"OpenF1 kept failing: {url}")
