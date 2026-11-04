import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:plenka/data/server/server.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:plenka/data/storage/settings_store.dart';

import 'helpers/fake_adapter.dart';

void main() {
  group('ServerInfo.fromJson', () {
    test('parses plenka-server info', () {
      final info = ServerInfo.fromJson('http://a:8095', {
        'service': 'plenka-server',
        'version': '1.0.0',
        'tmdb': true,
        'torrents': false,
        'indexers': {'total': 120, 'enabled': 6},
      })!;
      expect(info.baseUrl, 'http://a:8095');
      expect(info.version, '1.0.0');
      expect(info.tmdb, isTrue);
      expect(info.torrents, isFalse);
      expect(info.indexersTotal, 120);
      expect(info.indexersEnabled, 6);
    });

    test('rejects other services and garbage', () {
      expect(ServerInfo.fromJson('x', {'service': 'other', 'tmdb': true}), isNull);
      expect(ServerInfo.fromJson('x', 'hello'), isNull);
      expect(ServerInfo.fromJson('x', null), isNull);
    });

    test('tolerates missing indexers block', () {
      final info = ServerInfo.fromJson('x', {'service': 'plenka-server'})!;
      expect(info.indexersTotal, 0);
      expect(info.tmdb, isFalse);
    });
  });

  group('ServerDiscovery', () {
    test('normalize adds scheme and default port', () {
      expect(ServerDiscovery.normalize('192.168.88.1'), 'http://192.168.88.1:8095');
      expect(ServerDiscovery.normalize(' http://10.0.0.5:9000/ '), 'http://10.0.0.5:9000');
    });

    test('normalize keeps the scheme port when a scheme is given (server behind a proxy)', () {
      expect(ServerDiscovery.normalize('https://kino.example.com/'), 'https://kino.example.com');
      expect(ServerDiscovery.normalize('kino.example.com'), 'http://kino.example.com:8095');
    });

    test('probeWithRetry retries a server that is not up yet', () async {
      var calls = 0;
      final dio = fakeDio((o) => ++calls < 3 ? (503, null) : (200, {'service': 'plenka-server'}));
      final d = ServerDiscovery(dio: dio, udp: false);
      final info = await d.probeWithRetry('http://late:8095', pause: Duration.zero);
      expect(info?.baseUrl, 'http://late:8095');
      expect(calls, 3);
      expect(
        await ServerDiscovery(dio: fakeDio((_) => (503, null))).probeWithRetry('http://x', pause: Duration.zero),
        isNull,
      );
    });

    test('probe hits /api/info and accepts only plenka-server', () async {
      final dio = fakeDio(
        (o) => o.uri.host == 'good'
            ? (200, {'service': 'plenka-server', 'version': '1.2.3', 'tmdb': true, 'torrents': true})
            : (200, {'service': 'other'}),
      );
      final d = ServerDiscovery(dio: dio);
      final ok = await d.probe('http://good:8095');
      expect(ok?.version, '1.2.3');
      expect(adapterOf(dio).requests.single.uri.toString(), 'http://good:8095/api/info');
      expect(await d.probe('http://old:8095'), isNull);
    });

    test('parseUdpReply takes the sender address and HTTP port', () {
      List<int> b(Object o) => utf8.encode(jsonEncode(o));
      expect(
        ServerDiscovery.parseUdpReply('192.168.88.12', b({'service': 'plenka-server', 'port': 9000})),
        'http://192.168.88.12:9000',
      );
      expect(ServerDiscovery.parseUdpReply('h', b({'service': 'plenka-server'})), 'http://h:8095');
      expect(ServerDiscovery.parseUdpReply('h', b({'service': 'other', 'port': 1})), isNull);
      expect(ServerDiscovery.parseUdpReply('h', b({'service': 'plenka-server', 'port': 0})), isNull);
      expect(ServerDiscovery.parseUdpReply('h', utf8.encode('garbage')), isNull);
    });

    test('broadcast sends the request and yields replying servers', () async {
      final server = await RawDatagramSocket.bind(InternetAddress.loopbackIPv4, 0);
      server.listen((e) {
        final d = server.receive();
        if (d == null || utf8.decode(d.data) != ServerDiscovery.udpRequest) return;
        server.send(
          utf8.encode(jsonEncode({'service': 'plenka-server', 'version': 't', 'port': 8095})),
          d.address,
          d.port,
        );
      });
      addTearDown(server.close);
      final urls = await ServerDiscovery()
          .broadcast(port: server.port, targets: ['127.0.0.1'], timeout: const Duration(milliseconds: 1200))
          .toList();
      expect(urls, ['http://127.0.0.1:8095']);
    });

    test('probe returns null on HTTP errors', () async {
      final d = ServerDiscovery(dio: fakeDio((_) => (500, {'error': 'boom'})));
      expect(await d.probe('http://x:8095'), isNull);
    });
  });

  group('ServerClient', () {
    test('builds image urls against the attached server', () {
      final c = ServerClient()..attach(const ServerInfo(baseUrl: 'http://s:8095', tmdb: true, torrents: true));
      expect(c.image('/abc.jpg', size: 'w500'), 'http://s:8095/img/t/p/w500/abc.jpg');
      expect(c.image(null), isNull);
      expect(c.dio.options.baseUrl, 'http://s:8095');
    });
  });

  group('SettingsStore', () {
    test('stores and clears server_url', () async {
      SharedPreferences.setMockInitialValues({'server_url': 'http://192.168.88.1:8095'});
      final prefs = await SharedPreferences.getInstance();
      final s = SettingsStore(prefs);
      expect(s.serverUrl, 'http://192.168.88.1:8095');
      await s.setServerUrl(null);
      expect(prefs.getString('server_url'), isNull);
    });
  });
}
