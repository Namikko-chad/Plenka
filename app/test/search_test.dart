import 'package:flutter_test/flutter_test.dart';
import 'package:plenka/data/api/search_api.dart';
import 'package:plenka/data/models/torrent.dart';
import 'package:plenka/data/server/server.dart';

import 'helpers/fake_adapter.dart';

Map<String, dynamic> _r({
  String title = 'Матрица 1999 1080p',
  int size = 1000,
  int seeders = 1,
  String link = '/api/indexers/rutor/download?token=t1',
  String magnet = '',
  String infoHash = '',
}) => {
  'title': title,
  'tracker': 'Rutor',
  'trackerId': 'rutor',
  'size': size,
  'seeders': seeders,
  'peers': 3,
  'link': link,
  'magnet': magnet,
  'infoHash': infoHash,
  'details': 'https://rutor.info/t/1',
  'publishDate': '2026-10-01T00:00:00Z',
  'category': [2000],
  'categoryDesc': 'Movies/HD',
};

void main() {
  group('TorrentResult.fromJson', () {
    test('parses camelCase fields and keeps a relative link as-is', () {
      final t = TorrentResult.fromJson(_r())!;
      expect(t.title, 'Матрица 1999 1080p');
      expect(t.peers, 3);
      expect(t.link, '/api/indexers/rutor/download?token=t1');
      expect(t.infoHash, isNull);
      expect(t.publishDate, DateTime.utc(2026, 10, 1));
      expect(t.quality, Quality.fhd);
    });

    test('prefers a non-empty magnet over link', () {
      final t = TorrentResult.fromJson(_r(magnet: 'magnet:?xt=urn:btih:ABC', infoHash: 'ABCDEF'))!;
      expect(t.link, 'magnet:?xt=urn:btih:ABC');
      expect(t.infoHash, 'abcdef');
      expect(t.dedupKey, 'abcdef');
    });

    test('null publishDate and missing link', () {
      expect(TorrentResult.fromJson({..._r(), 'publishDate': null})!.publishDate, isNull);
      expect(TorrentResult.fromJson(_r(link: '')), isNull);
    });
  });

  group('SearchOutcome.merge', () {
    test('dedupes by infoHash, else title+size, and sorts by seeders', () {
      final a = SearchResponse.fromJson({
        'results': [_r(title: 'A', infoHash: 'h1', seeders: 5), _r(title: 'B', size: 10, seeders: 50)],
        'indexers': [
          {'id': 'rutor', 'name': 'Rutor', 'status': 'ok', 'results': 2, 'error': '', 'elapsedMs': 100},
          {'id': 'nnm', 'name': 'NNM', 'status': 'error', 'results': 0, 'error': 'timeout', 'elapsedMs': 30000},
        ],
      });
      final b = SearchResponse.fromJson({
        'results': [
          _r(title: 'A (other title)', infoHash: 'H1', seeders: 7), // same hash
          _r(title: 'B', size: 10, seeders: 50), // same title+size
          _r(title: 'B', size: 11, seeders: 9), // different size
        ],
        'indexers': [
          {'id': 'rutor', 'name': 'Rutor', 'status': 'ok', 'results': 3, 'elapsedMs': 200},
          {'id': 'nnm', 'name': 'NNM', 'status': 'ok', 'results': 0, 'elapsedMs': 900},
        ],
      });
      final m = SearchOutcome.merge([a, b]);
      expect(m.results.map((t) => '${t.title}/${t.size}'), ['B/10', 'B/11', 'A/1000']);
      expect(m.indexers.length, 2);
      expect(m.indexers.every((i) => i.ok), isTrue); // NNM responded to the second query
    });

    test('reports failed indexers', () {
      final m = SearchOutcome.merge([
        SearchResponse.fromJson({
          'results': [],
          'indexers': [
            {'id': 'x', 'name': 'X', 'status': 'error', 'error': 'login failed'},
          ],
        }),
      ]);
      expect(m.indexers.single.ok, isFalse);
      expect(m.indexers.single.name, 'X');
    });
  });

  group('SearchApi', () {
    ServerClient client(FakeResponse Function(String q) handler) {
      final c = ServerClient(dio: fakeDio((o) => handler(o.queryParameters['q'] as String)));
      c.attach(const ServerInfo(baseUrl: 'http://srv:8095', tmdb: true, torrents: true));
      return c;
    }

    test('queries /api/search in parallel and tolerates one failure', () async {
      final c = client(
        (q) => q == 'bad'
            ? (502, {'error': 'x'})
            : (
                200,
                {
                  'results': [_r(title: q, seeders: q.length)],
                  'indexers': [
                    {'id': 'rutor', 'name': 'Rutor', 'status': 'ok', 'results': 1},
                  ],
                },
              ),
      );
      final out = await SearchApi(c).search(['Matrix 1999', 'bad', ' Matrix 1999 ', '']);
      expect(out.results.single.title, 'Matrix 1999');
      final paths = adapterOf(c.dio).requests.map((r) => r.uri.path).toSet();
      expect(paths, {'/api/search'});
      expect(adapterOf(c.dio).requests.length, 2);
    });

    test('throws when every query fails', () async {
      final c = client((_) => (500, {'error': 'down'}));
      expect(SearchApi(c).search(['a', 'b']), throwsA(anything));
    });
  });
}
