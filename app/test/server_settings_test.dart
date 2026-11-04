import 'package:flutter_test/flutter_test.dart';
import 'package:plenka/core/utils/format.dart';
import 'package:plenka/data/api/config_api.dart';
import 'package:plenka/data/api/engine_api.dart';
import 'package:plenka/data/api/indexers_api.dart';
import 'package:plenka/data/models/server_settings.dart';
import 'package:plenka/data/server/server.dart';

import 'helpers/fake_adapter.dart';

const _s = IndexerSetting.new;

void main() {
  group('indexer form values', () {
    test('checkboxes accept bool or string and are written back in the same shape', () {
      final c = _s(name: 'x', type: 'checkbox', defaultValue: false);
      expect(c.formValue('true'), isTrue);
      expect(c.formValue(null), isFalse);
      expect(c.wireValue(true, 'false'), 'true');
      expect(c.wireValue(true, false), true);
      expect(c.wireValue(false, null), false);
    });

    test('multi-selects accept a list or a comma string', () {
      final m = _s(name: 'x', type: 'multi-select', defaultValue: 'a,b');
      expect(m.formValue(null), ['a', 'b']);
      expect(m.formValue(['c']), ['c']);
      expect(m.wireValue(['a', 'c'], null), 'a,c');
      expect(m.wireValue(['a'], ['b']), ['a']);
    });

    test('passwords start empty so that an empty value keeps the stored one', () {
      final p = _s(name: 'password', type: 'password', defaultValue: 'zzz');
      expect(p.formValue(null), '');
      expect(p.formValue(''), '');
    });

    test('text and select fall back to the default', () {
      expect(_s(name: 'x', type: 'select', defaultValue: 0).formValue(null), '0');
      expect(_s(name: 'x', type: 'text').formValue('me'), 'me');
      expect(_s(name: 'x', type: 'info_cookie').isInfo, isTrue);
    });
  });

  test('IndexerDetail reads the site link, set passwords and form fields', () {
    final d = IndexerDetail.fromJson({
      'id': 'rutracker',
      'name': 'RuTracker.org',
      'type': 'private',
      'links': ['https://rutracker.org/', 'https://rutracker.net/'],
      'enabled': false,
      'configured': false,
      'settings': [
        {'name': 'sitelink', 'type': 'text', 'label': 'Site'},
        {'name': 'username', 'type': 'text', 'label': 'Username'},
        {'name': 'password', 'type': 'password', 'label': 'Password'},
        {
          'name': 'sort',
          'type': 'select',
          'default': '0',
          'options': [
            {'value': '0', 'label': 'created'},
          ],
        },
      ],
      'config': {'username': 'me', 'password': '', 'password__set': '1'},
    });
    expect(d.siteLink, 'https://rutracker.org/');
    expect(d.passwordSet('password'), isTrue);
    expect(d.fields.map((s) => s.name), ['username', 'password', 'sort']);
    expect(d.fields.last.options.single.label, 'created');
    expect(d.indexer.configured, isFalse);
  });

  test('EngineSettings keeps fields the app does not know', () {
    final s = EngineSettings.fromJson({'cacheSize': 1, 'futureField': 'x'}).copyWith('cacheSize', 2);
    expect(s.json, {'cacheSize': 2, 'futureField': 'x'});
  });

  group('APIs', () {
    ServerClient client(FakeResponse Function(dynamic o) h) =>
        ServerClient(dio: fakeDio(h))..attach(const ServerInfo(baseUrl: 'http://srv:8095', tmdb: true, torrents: true));

    test('indexer save sends only the given fields', () async {
      final c = client((_) => (200, {'id': 'rutor', 'enabled': true}));
      final d = await IndexersApi(c).save('rutor', enabled: true);
      final req = adapterOf(c.dio).requests.single;
      expect(req.method, 'PUT');
      expect(req.path, '/api/indexers/rutor');
      expect(req.data, {'enabled': true});
      expect(d.indexer.enabled, isTrue);
    });

    test('indexer test and definitions update are POSTs', () async {
      final c = client(
        (o) => o.path.endsWith('/test')
            ? (200, {'ok': true, 'results': 50, 'elapsedMs': 900})
            : (200, {'count': 586, 'updatedAt': '2026-11-03T06:00:00Z'}),
      );
      final api = IndexersApi(c);
      final r = await api.test('rutor');
      expect((r.ok, r.results, r.elapsedMs), (true, 50, 900));
      final defs = await api.updateDefinitions();
      expect(defs.count, 586);
      expect(adapterOf(c.dio).requests.map((r) => '${r.method} ${r.path}'), [
        'POST /api/indexers/rutor/test',
        'POST /api/definitions/update',
      ]);
    });

    test('config put and engine reset', () async {
      final c = client(
        (o) => o.path == '/api/config'
            ? (200, {'tmdbApiKeySet': false, 'searchTimeoutSec': 45})
            : (200, {'cacheSize': 67108864}),
      );
      final cfg = await ConfigApi(c).put({'tmdbApiKey': '-'});
      expect((cfg.tmdbApiKeySet, cfg.searchTimeoutSec), (false, 45));
      final e = await EngineApi(c).reset();
      expect(e.intOf('cacheSize'), 67108864);
      final reqs = adapterOf(c.dio).requests;
      expect(reqs.first.data, {'tmdbApiKey': '-'});
      expect('${reqs.last.method} ${reqs.last.path}', 'DELETE /api/engine/settings');
    });
  });

  test('htmlToText keeps line breaks and drops tags', () {
    expect(
      htmlToText('<p>Use your <b>login</b> &amp; password.</p><ul><li>one</li><li>two</li></ul>Line<br>break'),
      'Use your login & password.\n• one\n• two\nLine\nbreak',
    );
  });

  test('formatDateTime uses YYYY-MM-DD HH:MM:SS', () {
    expect(formatDateTime(DateTime(2026, 6, 15, 14, 30, 5)), '2026-06-15 14:30:05');
  });
}
