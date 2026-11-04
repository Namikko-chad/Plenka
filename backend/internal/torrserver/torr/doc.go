// Package torr is the torrent engine: the BitTorrent client, loaded torrents, the
// piece cache readers and HTTP streaming of torrent files.
//
// The engine under internal/torrserver is derived from TorrServer MatriX
// (https://github.com/YouROK/TorrServer, GPL-3.0) and modified for Plenka: its HTTP
// API, DLNA, ffprobe, viewed marks and the JSON settings store were removed; the
// native API in internal/torrents drives it.
package torr
