import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:plenka/data/api/torrent_api.dart';
import 'package:plenka/data/models/media.dart';
import 'package:plenka/data/models/torrent.dart';
import 'package:plenka/data/models/watch_entry.dart';
import 'package:plenka/data/server/server.dart';
import 'package:plenka/presentation/torrents/torrent_details_screen.dart';
import 'package:plenka/presentation/torrents/torrents_cubit.dart';

import 'helpers/fake_adapter.dart';

Map<String, dynamic> _torrent(String hash, {int stat = 3, List<Map<String, dynamic>> files = const []}) => {
  'hash': hash,
  'title': 'T $hash',
  'stat': stat,
  'statString': 'Torrent working',
  'activePeers': 4,
  'files': files,
};

WatchEntry _watched(String hash, int fileId, DateTime at) => WatchEntry(
  hash: hash,
  fileId: fileId,
  fileName: 'f$fileId',
  torrentTitle: 't',
  positionMs: 1,
  durationMs: 10,
  updatedAt: at,
);

void main() {
  (TorrentApi, FakeAdapter) api(FakeResponse Function(dynamic o) h) {
    final dio = fakeDio(h);
    final c = ServerClient(dio: dio)..attach(const ServerInfo(baseUrl: 'http://srv:8095', tmdb: true, torrents: true));
    return (TorrentApi(c), adapterOf(dio));
  }

  test('list, drop, remove and cache hit the engine endpoints', () async {
    final (a, adapter) = api((o) {
      if (o.path == '/api/torrents') return (200, [_torrent('h1'), _torrent('h2', stat: 5)]);
      if (o.path.endsWith('/cache')) {
        return (
          200,
          {
            'hash': 'h1',
            'capacity': 100,
            'filled': 10,
            'piecesLength': 4,
            'piecesCount': 8,
            'pieces': {
              '2': {'id': 2, 'length': 4, 'size': 2, 'completed': false, 'priority': 1},
            },
            'readers': [
              {'start': 1, 'end': 3, 'reader': 2},
            ],
          },
        );
      }
      return (204, null);
    });
    final list = await a.list();
    expect(list.map((t) => t.hash), ['h1', 'h2']);
    expect(list.first.isActive, isTrue);
    expect(list.last.isActive, isFalse);

    await a.drop('h1');
    await a.remove('h2');
    final cache = await a.cache('h1');
    expect(cache.piecesCount, 8);
    expect(cache.pieces.single.fill, 0.5);
    expect(cache.readers.single, (1, 3));

    final reqs = adapter.requests.map((o) => '${o.method} ${o.path}').toList();
    expect(reqs, [
      'GET /api/torrents',
      'POST /api/torrents/h1/drop',
      'DELETE /api/torrents/h2',
      'GET /api/torrents/h1/cache',
    ]);
  });

  test('a null list is empty', () async {
    final (a, _) = api((o) => (200, null));
    expect(await a.list(), isEmpty);
  });

  test('cubit: drop marks the torrent unloaded, remove takes it out', () async {
    final (a, _) = api((o) => o.path == '/api/torrents' ? (200, [_torrent('h1'), _torrent('h2')]) : (204, null));
    final cubit = TorrentsCubit(a);
    await cubit.refresh();
    expect(cubit.state.loaded, isTrue);

    await cubit.drop('h1');
    final dropped = cubit.state.list.first;
    expect(dropped.stat, 5);
    expect(dropped.activePeers, 0);

    await cubit.remove('h2');
    expect(cubit.state.list.map((t) => t.hash), ['h1']);
    await cubit.close();
  });

  test('cubit keeps the list on a failed refresh and reports the error', () async {
    var fail = false;
    final (a, _) = api((o) => fail ? (500, {'error': 'boom'}) : (200, [_torrent('h1')]));
    final cubit = TorrentsCubit(a);
    await cubit.refresh();
    fail = true;
    await cubit.refresh();
    expect(cubit.state.list, hasLength(1));
    expect(cubit.state.error, isNotNull);
    await cubit.close();
  });

  test('Torrent.media reads the stored card; junk is null', () {
    const m = MediaItem(id: 603, type: MediaType.movie, title: 'Матрица', originalTitle: 'The Matrix');
    expect(Torrent(hash: 'h', data: jsonEncode(m.toJson())).media, m);
    expect(const Torrent(hash: 'h', data: '{oops').media, isNull);
    expect(const Torrent(hash: 'h').media, isNull);
    expect(const Torrent(hash: 'h', name: 'n').displayTitle, 'n');
  });

  test('preload target: the last watched file of the torrent, else the first video', () {
    const files = [
      TorrentFile(id: 1, path: 'S01E10.mkv', length: 1),
      TorrentFile(id: 2, path: 'S01E02.mkv', length: 1),
      TorrentFile(id: 3, path: 'readme.txt', length: 1),
    ];
    const t = Torrent(hash: 'h', files: files);
    expect(preloadTarget(t, const [])!.id, 2);
    expect(
      preloadTarget(t, [
        _watched('h', 1, DateTime(2026, 1, 2)),
        _watched('h', 2, DateTime(2026, 1, 1)),
        _watched('other', 3, DateTime(2026, 1, 3)),
      ])!.id,
      1,
    );
    expect(preloadTarget(const Torrent(hash: 'h'), const []), isNull);
  });

  test('piece map buckets pieces into cells', () {
    const c = CacheState(
      piecesCount: 4,
      pieces: [CachePiece(id: 0, length: 4, size: 4, completed: true), CachePiece(id: 3, length: 4, priority: 1)],
      readers: [(1, 2)],
    );
    final m = PieceMapLayout.of(c, 700);
    expect(m.fill, [1, 0, 0, 0]);
    expect(m.queued, [false, false, false, true]);
    expect(m.reading, [false, true, true, false]);
    expect(m.rows, 1);
  });
}
