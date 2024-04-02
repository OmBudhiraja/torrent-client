# Torrent Client

A simple BitTorrent Client written in Go.

## Resources

- [Build your own BitTorrent](https://app.codecrafters.io/courses/bittorrent/overview)

- [BitTorrent Protocol Specification](https://www.bittorrent.org/beps/bep_0003.html)

- [BitTorrent Economic Paper](http://bittorrent.org/bittorrentecon.pdf)

### Run locally

```bash
go build -o torrent_client ./cmd/mybittorrent/main.go

./torrent_client download -o ./sample.txt sample.torrent
```