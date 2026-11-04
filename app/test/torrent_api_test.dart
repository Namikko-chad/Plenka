import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:plenka/data/api/torrent_api.dart';
import 'package:plenka/data/models/media.dart';
import 'package:plenka/data/models/torrent.dart';
import 'package:plenka/data/server/server.dart';

import 'helpers/fake_adapter.dart';

void main() {
  const media = MediaItem(id: 603, type: MediaType.movie, title: 'Матрица', originalTitle: 'The Matrix', year: 1999);

  test('stream url: /api/stream/{hash}/{id}/{encoded file name}', () {
    const f = TorrentFile(id: 3, path: 'Season 1/S01E03 [1080p] #1.mkv', length: 1);
    expect(
      TorrentApi.buildStreamUrl('http://srv:8095', 'abc', f),
      'http://srv:8095/api/stream/abc/3/S01E03%20%5B1080p%5D%20%231.mkv',
    );
  });

  test('add body carries link as-is and media as JSON string in data', () {
    final b = TorrentApi.addBody(link: '/api/indexers/rutor/download?token=t', title: 'Матрица', media: media);
    expect(b.keys, unorderedEquals(['link', 'title', 'data']));
    expect(b['link'], '/api/indexers/rutor/download?token=t');
    final data = jsonDecode(b['data'] as String) as Map<String, dynamic>;
    expect(data['id'], 603);
    expect(data['type'], 'movie');
  });

  test('Torrent.fromJson reads the camelCase shape', () {
    final t = Torrent.fromJson({
      'hash': 'h1',
      'title': 'Матрица',
      'poster': '',
      'data': '{}',
      'name': 'The.Matrix',
      'stat': 3,
      'statString': 'Torrent working',
      'addedAt': '2026-11-03T06:00:00Z',
      'torrentSize': 1000,
      'downloadSpeed': 1.5,
      'totalPeers': 9,
      'activePeers': 4,
      'files': [
        {'id': 1, 'path': 'The.Matrix/movie.mkv', 'length': 100},
      ],
    });
    expect(t.name, 'The.Matrix');
    expect(t.statString, 'Torrent working');
    expect(t.addedAt, '2026-11-03T06:00:00Z');
    expect(t.torrentSize, 1000);
    expect(t.downloadSpeed, 1.5);
    expect(t.activePeers, 4);
    expect(t.totalPeers, 9);
    expect(t.files.single.id, 1);
    expect(t.files.single.isVideo, isTrue);
  });

  test('add posts to /api/torrents and waitForFiles polls GET /api/torrents/{hash}', () async {
    var gets = 0;
    final dio = fakeDio((o) {
      if (o.method == 'POST') {
        return (200, {'hash': 'h1', 'title': 'Матрица', 'stat': 1, 'statString': 'Torrent getting info', 'files': []});
      }
      gets++;
      if (gets == 1) return (404, {'error': 'not found'});
      return (
        200,
        {
          'hash': 'h1',
          'stat': 3,
          'files': [
            {'id': 1, 'path': 'm/movie.mkv', 'length': 100},
          ],
        },
      );
    });
    final c = ServerClient(dio: dio)..attach(const ServerInfo(baseUrl: 'http://srv:8095', tmdb: true, torrents: true));
    final api = TorrentApi(c);
    final st = await api.add(link: 'magnet:?xt=urn:btih:h1', title: 'Матрица', media: media);
    expect(st.hash, 'h1');
    final reqs = adapterOf(dio).requests;
    expect(reqs.first.uri.path, '/api/torrents');
    expect((reqs.first.data as Map).containsKey('action'), isFalse);
    final full = await api.waitForFiles('h1', timeout: const Duration(seconds: 5));
    expect(reqs.skip(1).every((r) => r.method == 'GET' && r.uri.path == '/api/torrents/h1'), isTrue);
    expect(full.files.single.name, 'movie.mkv');
    expect(api.streamUrl('h1', full.files.single), 'http://srv:8095/api/stream/h1/1/movie.mkv');
  });

  test('preload posts {fileId} to /api/torrents/{hash}/preload', () async {
    final dio = fakeDio((o) => (202, null));
    final c = ServerClient(dio: dio)..attach(const ServerInfo(baseUrl: 'http://srv:8095', tmdb: true, torrents: true));
    await TorrentApi(c).preload('h1', 4);
    final r = adapterOf(dio).requests.single;
    expect(r.method, 'POST');
    expect(r.path, '/api/torrents/h1/preload');
    expect(r.data, {'fileId': 4});
  });

  test('downloadAll posts to /download and parses fullDownload; cancelDownload deletes it', () async {
    final dio = fakeDio(
      (o) => o.method == 'POST'
          ? (200, {'hash': 'h1', 'fullDownload': true, 'torrentSize': 100, 'loadedSize': 40, 'files': []})
          : (204, null),
    );
    final c = ServerClient(dio: dio)..attach(const ServerInfo(baseUrl: 'http://srv:8095', tmdb: true, torrents: true));
    final api = TorrentApi(c);
    final st = await api.downloadAll('h1');
    expect(st.fullDownload, isTrue);
    expect(st.loadedSize, 40);
    await api.cancelDownload('h1');
    final reqs = adapterOf(dio).requests;
    expect(
      [for (final r in reqs) '${r.method} ${r.path}'],
      ['POST /api/torrents/h1/download', 'DELETE /api/torrents/h1/download'],
    );
  });
}
