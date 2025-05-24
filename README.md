# Torrent Client

A simple BitTorrent Client written in Go.

## Resources

- [Build your own BitTorrent](https://app.codecrafters.io/courses/bittorrent/overview)

- [BitTorrent Protocol Specification](https://www.bittorrent.org/beps/bep_0003.html)

- [Magnet Uri Extension](https://www.bittorrent.org/beps/bep_0009.html)

- [UDP Tracker Protocol](https://www.bittorrent.org/beps/bep_0015.html)

### Run locally

```bash
go build -o torrent_client ./cmd/mybittorrent/main.go

./torrent_client ./sample_torrents/sample.torrent ./sample.txt
```
