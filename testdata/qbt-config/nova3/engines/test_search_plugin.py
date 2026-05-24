# test_search_plugin.py — Minimal qBittorrent search plugin for CI testing.
# Not a real indexer; returns hard-coded results.

from novaprinter import prettyPrinter

# Use known magnets already present in the test suite so the "add" flow works too.
RESULTS = [
    {
        "link": "magnet:?xt=urn:btih:3b245504cf5f11bbdbe1201cea6a6bf45aee1bc0&dn=ubuntu-24.04-desktop-amd64.iso",
        "name": "Ubuntu 24.04 Desktop AMD64",
        "size": "6000000000",
        "seeds": "500",
        "leech": "120",
        "engine_url": "http://tests.local",
        "desc_link": "http://tests.local/desc1",
    },
    {
        "link": "magnet:?xt=urn:btih:7d5210a711291d7181d6e074ce5ebd56f3fedd60&dn=debian-12.10.0-amd64-netinst.iso",
        "name": "Debian 12.10 amd64 netinst",
        "size": "660000000",
        "seeds": "300",
        "leech": "50",
        "engine_url": "http://tests.local",
        "desc_link": "http://tests.local/desc2",
    },
    {
        # Non-magnet result — exercises the "non-magnet" error path.
        "link": "http://tests.local/torrent/not-a-magnet",
        "name": "Non Magnet Result",
        "size": "1000000",
        "seeds": "10",
        "leech": "5",
        "engine_url": "http://tests.local",
        "desc_link": "http://tests.local/desc3",
    },
]


class test_search_plugin:
    """Minimal test plugin name: "CI Test Search"."""

    name = "CI Test Search"
    url = "http://tests.local"
    supported_categories = {"all": "all"}

    def search(self, what: str, cat: str = "all") -> None:
        """Return hard-coded results regardless of the query."""
        for r in RESULTS:
            prettyPrinter(r)

    def download_torrent(self, download_url: str) -> None:
        """Mock downloader."""
        pass
