import 'package:flutter_test/flutter_test.dart';
import 'package:plenka/data/server/server.dart';
import 'package:plenka/data/storage/history_store.dart';
import 'package:plenka/data/storage/settings_store.dart';
import 'package:plenka/presentation/connect/connection_cubit.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'history_test.dart' show FakeRemote;

ServerInfo _info(String url) => ServerInfo(baseUrl: url, tmdb: true, torrents: true);

/// A network of the given servers; auto-discovery finds [found].
class _FakeDiscovery extends ServerDiscovery {
  final Set<String> up;
  final String? found;
  final probed = <String>[];
  int discoveries = 0;

  _FakeDiscovery({this.up = const {}, this.found}) : super(udp: false);

  @override
  Future<ServerInfo?> probe(String baseUrl, {bool patient = false}) async {
    probed.add(baseUrl);
    return up.contains(baseUrl) ? _info(baseUrl) : null;
  }

  @override
  Future<ServerInfo?> probeWithRetry(String baseUrl, {int attempts = 3, Duration pause = Duration.zero}) =>
      probe(baseUrl, patient: true);

  @override
  Future<ServerInfo?> discover({String? saved}) async {
    discoveries++;
    return found == null ? null : _info(found!);
  }
}

void main() {
  late SettingsStore settings;
  late HistoryStore history;

  Future<void> init(Map<String, Object> prefs) async {
    SharedPreferences.setMockInitialValues(prefs);
    final p = await SharedPreferences.getInstance();
    settings = SettingsStore(p);
    history = HistoryStore(p, FakeRemote());
  }

  ConnectionCubit cubit(ServerDiscovery d) => ConnectionCubit(d, ServerClient(), settings, history);

  test('an address entered by hand is remembered and used on the next start', () async {
    await init({});
    final d = _FakeDiscovery(up: {'http://10.0.0.9:8095'});
    expect(await cubit(d).connectTo('10.0.0.9'), isTrue);
    expect(settings.serverUrl, 'http://10.0.0.9:8095');
    expect(settings.serverManual, isTrue);

    final next = cubit(d);
    await next.connect();
    expect(next.state.isConnected, isTrue);
    expect(next.state.info?.baseUrl, 'http://10.0.0.9:8095');
    expect(d.discoveries, 0);
  });

  test('a manual server that does not answer is not replaced by auto-discovery', () async {
    await init({'server_url': 'http://10.0.0.9:8095', 'server_manual': true});
    final d = _FakeDiscovery(found: 'http://192.168.88.1:8095');
    final c = cubit(d);
    await c.connect();
    expect(c.state.status, ConnectionStatus.notFound);
    expect(c.state.address, 'http://10.0.0.9:8095');
    expect(d.discoveries, 0);
    expect(settings.serverUrl, 'http://10.0.0.9:8095');
  });

  test('"search automatically" drops the manual choice', () async {
    await init({'server_url': 'http://10.0.0.9:8095', 'server_manual': true});
    final c = cubit(_FakeDiscovery(found: 'http://192.168.88.1:8095'));
    await c.discoverAutomatically();
    expect(c.state.info?.baseUrl, 'http://192.168.88.1:8095');
    expect(settings.serverManual, isFalse);
    expect(settings.serverUrl, 'http://192.168.88.1:8095');
  });

  test('a failed manual change keeps the current server', () async {
    await init({});
    final d = _FakeDiscovery(found: 'http://192.168.88.1:8095');
    final c = cubit(d);
    await c.connect();
    expect(await c.connectTo('10.9.9.9'), isFalse);
    expect(c.state.info?.baseUrl, 'http://192.168.88.1:8095');
    expect(settings.serverManual, isFalse);
  });
}
