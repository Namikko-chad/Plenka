import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:plenka/data/api/history_api.dart';
import 'package:plenka/data/models/media.dart';
import 'package:plenka/data/server/server.dart';
import 'package:plenka/data/storage/history_store.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'helpers/fake_adapter.dart';

const _media = MediaItem(id: 603, type: MediaType.movie, title: 'Матрица', originalTitle: 'The Matrix');

WatchEntry _e(String hash, int fileId, int minute, {int pos = 1000, MediaItem? media}) => WatchEntry(
  media: media,
  hash: hash,
  fileId: fileId,
  fileName: 'f$fileId.mkv',
  torrentTitle: 't',
  positionMs: pos,
  durationMs: 100000,
  updatedAt: DateTime.utc(2026, 11, 3, 6, minute),
);

/// In-memory history server.
class FakeRemote implements HistoryRemote {
  bool online = true;
  bool fail = false;
  final store = <String, WatchEntry>{};
  final puts = <WatchEntry>[];
  final deletes = <String>[];
  DateTime clock = DateTime.utc(2026, 11, 3, 7);

  @override
  bool get available => online;

  @override
  Future<List<WatchEntry>> list({int limit = 300}) async {
    if (fail) throw Exception('down');
    return store.values.toList()..sort((a, b) => b.updatedAt.compareTo(a.updatedAt));
  }

  @override
  Future<WatchEntry?> put(WatchEntry e) async {
    if (fail) throw Exception('down');
    clock = clock.add(const Duration(seconds: 1));
    final saved = e.withUpdatedAt(clock);
    puts.add(e);
    store[e.key] = saved;
    return saved;
  }

  @override
  Future<void> delete(String hash, int fileId) async {
    if (fail) throw Exception('down');
    deletes.add('$hash:$fileId');
    store.remove(WatchEntry.keyOf(hash, fileId));
  }
}

Future<void> _settle() => Future<void>.delayed(const Duration(milliseconds: 20));

void main() {
  group('WatchEntry json', () {
    test('reads legacy local cache with millisecond updatedAt', () {
      final e = WatchEntry.fromJson({
        'media': _media.toJson(),
        'hash': 'h',
        'fileId': 2,
        'fileName': 'a.mkv',
        'torrentTitle': 't',
        'positionMs': 5,
        'durationMs': 10,
        'updatedAt': DateTime.utc(2026, 1, 1).millisecondsSinceEpoch,
      });
      expect(e.updatedAt, DateTime.utc(2026, 1, 1));
      expect(e.media, _media);
    });

    test('reads server entry (RFC 3339, null media) and writes PUT body without updatedAt', () {
      final e = WatchEntry.fromJson({
        'hash': 'h',
        'fileId': 3,
        'fileName': 'S01E03.mkv',
        'torrentTitle': 't',
        'media': null,
        'positionMs': 120000,
        'durationMs': 7200000,
        'updatedAt': '2026-11-03T06:00:00Z',
      });
      expect(e.media, isNull);
      expect(e.updatedAt, DateTime.utc(2026, 11, 3, 6));
      expect(e.toServerJson().containsKey('updatedAt'), isFalse);
      expect(e.toServerJson()['media'], isNull);
      expect(WatchEntry.fromJson(e.toJson()).updatedAt, e.updatedAt);
    });
  });

  group('mergeHistory', () {
    test('server wins unless a pending local entry is newer', () {
      final local = {
        for (final e in [_e('a', 1, 10, pos: 1), _e('b', 1, 30, pos: 2), _e('c', 1, 5), _e('d', 1, 5)]) e.key: e,
      };
      final server = [_e('a', 1, 20, pos: 9), _e('b', 1, 25, pos: 9), _e('e', 1, 1)];
      final m = mergeHistory(local: local, server: server, pending: {'a:1', 'b:1', 'c:1'});
      expect(m.entries['a:1']!.positionMs, 9); // server is newer
      expect(m.entries['b:1']!.positionMs, 2); // local is newer and not sent
      expect(m.entries.containsKey('c:1'), isTrue); // local only, pending upload
      expect(m.entries.containsKey('d:1'), isFalse); // deleted on the server
      expect(m.entries.containsKey('e:1'), isTrue); // new from the server
      expect(m.pending, {'b:1', 'c:1'});
    });

    test('does not resurrect entries whose delete is pending and respects the limit', () {
      final m = mergeHistory(
        local: const {},
        server: [_e('a', 1, 1), _e('b', 1, 2), _e('c', 1, 3)],
        pending: const {},
        deleted: {'b:1'},
        limit: 1,
      );
      expect(m.entries.keys, ['c:1']);
    });
  });

  group('HistoryStore', () {
    test('migrates old local-only history to the server once, oldest first', () async {
      SharedPreferences.setMockInitialValues({
        'watch_history': jsonEncode([
          {..._e('new', 1, 30).toJson(), 'updatedAt': DateTime.utc(2026, 1, 2).millisecondsSinceEpoch},
          {..._e('old', 1, 10).toJson(), 'updatedAt': DateTime.utc(2026, 1, 1).millisecondsSinceEpoch},
        ]),
      });
      final prefs = await SharedPreferences.getInstance();
      final remote = FakeRemote();
      final store = HistoryStore(prefs, remote);
      expect(store.pendingKeys, {'new:1', 'old:1'});
      await store.sync();
      expect(remote.puts.map((e) => e.hash), ['old', 'new']);
      expect(store.pendingKeys, isEmpty);
      expect(store.entry('old', 1)!.updatedAt, remote.store['old:1']!.updatedAt);

      // second run: nothing left to migrate
      final again = HistoryStore(prefs, FakeRemote()..store.addAll(remote.store));
      expect(again.pendingKeys, isEmpty);
      expect(again.entries.length, 2);
    });

    test('pulls entries from the server (web app) and drops ones deleted there', () async {
      SharedPreferences.setMockInitialValues({'watch_history_server_migrated': true});
      final prefs = await SharedPreferences.getInstance();
      final remote = FakeRemote()..store['w:1'] = _e('w', 1, 40, media: _media);
      final store = HistoryStore(prefs, remote);
      var notified = 0;
      store.addListener(() => notified++);
      await store.sync();
      expect(store.continueWatching().single.hash, 'w');
      expect(notified, greaterThan(0));

      remote.store.clear();
      await store.sync();
      expect(store.entries, isEmpty);
    });

    test('save is local-first and survives the server being down', () async {
      SharedPreferences.setMockInitialValues({'watch_history_server_migrated': true});
      final prefs = await SharedPreferences.getInstance();
      final remote = FakeRemote()..fail = true;
      final store = HistoryStore(prefs, remote);
      await store.save(_e('h', 1, 1, pos: 500));
      await _settle();
      expect(store.entry('h', 1)!.positionMs, 500);
      expect(store.pendingKeys, {'h:1'});

      // offline restart: cache and queue are intact
      final restarted = HistoryStore(prefs, remote);
      expect(restarted.entry('h', 1)!.positionMs, 500);
      expect(restarted.pendingKeys, {'h:1'});

      remote.fail = false;
      await restarted.sync();
      expect(remote.store['h:1']!.positionMs, 500);
      expect(restarted.pendingKeys, isEmpty);
    });

    test('save pushes in the background; remove deletes on the server', () async {
      SharedPreferences.setMockInitialValues({'watch_history_server_migrated': true});
      final prefs = await SharedPreferences.getInstance();
      final remote = FakeRemote();
      final store = HistoryStore(prefs, remote);
      await store.save(_e('h', 2, 1));
      await _settle();
      expect(remote.store.containsKey('h:2'), isTrue);
      expect(store.pendingKeys, isEmpty);

      await store.remove(store.entry('h', 2)!);
      await _settle();
      expect(remote.deletes, ['h:2']);
      expect(store.entry('h', 2), isNull);
    });

    test('offline (no server attached) never touches the network', () async {
      SharedPreferences.setMockInitialValues({});
      final prefs = await SharedPreferences.getInstance();
      final remote = FakeRemote()..online = false;
      final store = HistoryStore(prefs, remote);
      await store.save(_e('h', 1, 1));
      await store.sync();
      expect(remote.puts, isEmpty);
      expect(store.pendingKeys, {'h:1'});
    });
  });

  group('HistoryApi', () {
    test('talks to /api/history', () async {
      final dio = fakeDio(
        (o) => switch (o.method) {
          'GET' => (
            200,
            [
              {
                'hash': 'h',
                'fileId': 1,
                'fileName': 'a.mkv',
                'torrentTitle': 't',
                'media': _media.toJson(),
                'positionMs': 1,
                'durationMs': 2,
                'updatedAt': '2026-11-03T06:00:00Z',
              },
            ],
          ),
          'PUT' => (200, {...(o.data as Map<String, dynamic>), 'updatedAt': '2026-11-03T07:00:00Z'}),
          _ => (204, null),
        },
      );
      final c = ServerClient(dio: dio)
        ..attach(const ServerInfo(baseUrl: 'http://srv:8095', tmdb: true, torrents: true));
      final api = HistoryApi(c);
      final list = await api.list();
      expect(list.single.media, _media);
      final saved = await api.put(_e('h', 1, 1));
      expect(saved!.updatedAt, DateTime.utc(2026, 11, 3, 7));
      await api.delete('h', 1);
      final reqs = adapterOf(dio).requests;
      expect(reqs[0].uri.toString(), 'http://srv:8095/api/history?limit=300');
      expect((reqs[1].data as Map).containsKey('updatedAt'), isFalse);
      expect(reqs[2].method, 'DELETE');
      expect(reqs[2].uri.path, '/api/history/h/1');
    });
  });
}
