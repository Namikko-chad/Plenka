import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';

import '../../core/config/app_config.dart';

/// Server response to GET /api/info.
class ServerInfo {
  static const serviceName = 'plenka-server';

  final String baseUrl;
  final String version;

  /// A TMDB key is set.
  final bool tmdb;

  /// The torrent engine is running.
  final bool torrents;
  final int indexersTotal;
  final int indexersEnabled;

  const ServerInfo({
    required this.baseUrl,
    this.version = '',
    required this.tmdb,
    required this.torrents,
    this.indexersTotal = 0,
    this.indexersEnabled = 0,
  });

  /// null — the responder is not plenka-server.
  static ServerInfo? fromJson(String baseUrl, Object? json) {
    if (json is! Map || json['service'] != serviceName) return null;
    final idx = json['indexers'];
    int count(String k) => idx is Map ? (idx[k] as num?)?.toInt() ?? 0 : 0;
    return ServerInfo(
      baseUrl: baseUrl,
      version: json['version'] as String? ?? '',
      tmdb: json['tmdb'] == true,
      torrents: json['torrents'] == true,
      indexersTotal: count('total'),
      indexersEnabled: count('enabled'),
    );
  }
}

/// The discovered server's address and a Dio configured for it. All APIs go through this client.
class ServerClient {
  final Dio dio;

  ServerClient({Dio? dio})
    : dio =
          dio ??
          Dio(BaseOptions(connectTimeout: const Duration(seconds: 8), receiveTimeout: const Duration(seconds: 30)));

  ServerInfo? _info;

  ServerInfo? get info => _info;
  String get baseUrl => _info?.baseUrl ?? '';
  bool get isAttached => _info != null;

  void attach(ServerInfo info) {
    _info = info;
    dio.options.baseUrl = info.baseUrl;
  }

  /// Absolute URL for the player, images and the browser.
  String url(String path) => '$baseUrl$path';

  String? image(String? tmdbPath, {String size = 'w342'}) => tmdbPath == null ? null : url('/img/t/p/$size$tmdbPath');
}

/// Finds the server on the LAN with zero configuration: sends a UDP broadcast (the server replies
/// from its own address) and in parallel probes the router address (x.y.z.1) of every device
/// network, the container address and the default MikroTik address.
class ServerDiscovery {
  /// The request plenka-server answers (UDP, port = HTTP port).
  static const udpRequest = 'plenka-server:discover';

  final Dio _probe;
  final bool _udp;

  ServerDiscovery({Dio? dio, bool udp = true})
    : _probe =
          dio ??
          Dio(
            BaseOptions(
              connectTimeout: const Duration(milliseconds: 1500),
              receiveTimeout: const Duration(milliseconds: 2500),
            ),
          ),
      _udp = udp;

  /// "192.168.88.1" → "http://192.168.88.1:8095". An address with a scheme and no port
  /// ("https://kino.example.com") stays on the scheme's default port — that is how a server
  /// behind a reverse proxy works.
  static String normalize(String input) {
    final s = input.trim();
    if (s.isEmpty) return s;
    final hasScheme = s.contains('://');
    final uri = Uri.parse(hasScheme ? s : 'http://$s');
    if (uri.hasPort) return '${uri.scheme}://${uri.host}:${uri.port}';
    return hasScheme ? '${uri.scheme}://${uri.host}' : '${uri.scheme}://${uri.host}:${AppConfig.serverPort}';
  }

  /// Server address from a UDP reply: sender IP + HTTP port from JSON. null — not plenka-server.
  static String? parseUdpReply(String host, List<int> data) {
    try {
      final json = jsonDecode(utf8.decode(data));
      if (json is! Map || json['service'] != ServerInfo.serviceName) return null;
      final port = (json['port'] as num?)?.toInt() ?? AppConfig.serverPort;
      if (port <= 0 || port > 65535) return null;
      return 'http://$host:$port';
    } catch (_) {
      return null;
    }
  }

  /// Timeouts for a known address: a TV's network right after power-on and a remote server
  /// respond slower than local auto-discovery addresses.
  static final _patient = Options(
    connectTimeout: const Duration(seconds: 6),
    receiveTimeout: const Duration(seconds: 8),
  );

  /// [patient] — wait longer (a saved or manually entered address).
  Future<ServerInfo?> probe(String baseUrl, {bool patient = false}) async {
    try {
      final r = await _probe.get<dynamic>('$baseUrl/api/info', options: patient ? _patient : null);
      return ServerInfo.fromJson(baseUrl, r.data);
    } catch (_) {
      return null;
    }
  }

  static Future<List<InternetAddress>> _ownAddresses() async {
    try {
      final ifaces = await NetworkInterface.list(type: InternetAddressType.IPv4);
      return [
        for (final iface in ifaces)
          for (final a in iface.addresses)
            if (!a.isLoopback && !a.isLinkLocal) a,
      ];
    } catch (_) {
      return const []; // no access to interfaces — only the default addresses remain
    }
  }

  Future<List<String>> candidates() async {
    final hosts = <String>{};
    for (final a in await _ownAddresses()) {
      final p = a.address.split('.');
      hosts.add('${p[0]}.${p[1]}.${p[2]}.1');
    }
    hosts.addAll(['192.168.88.1', '10.77.0.2', '10.0.2.2', '127.0.0.1']);
    return [for (final h in hosts) 'http://$h:${AppConfig.serverPort}'];
  }

  /// Broadcasts a UDP request (255.255.255.255 and x.y.z.255 of every network, or [targets]) and
  /// yields the addresses of responding servers until [timeout] expires.
  Stream<String> broadcast({
    int? port,
    List<String>? targets,
    Duration timeout = const Duration(milliseconds: 2500),
  }) async* {
    final dst = port ?? AppConfig.serverPort;
    final hosts =
        targets ??
        {
          '255.255.255.255',
          for (final a in await _ownAddresses()) '${a.address.substring(0, a.address.lastIndexOf('.'))}.255',
        }.toList();
    RawDatagramSocket socket;
    try {
      socket = await RawDatagramSocket.bind(InternetAddress.anyIPv4, 0);
    } catch (_) {
      return;
    }
    socket.broadcastEnabled = true;
    final out = StreamController<String>();
    final seen = <String>{};
    final sub = socket.listen((event) {
      if (event != RawSocketEvent.read) return;
      final d = socket.receive();
      if (d == null) return;
      final url = parseUdpReply(d.address.address, d.data);
      if (url != null && seen.add(url)) out.add(url);
    });
    final payload = utf8.encode(udpRequest);
    void send() {
      for (final h in hosts) {
        try {
          socket.send(payload, InternetAddress(h), dst);
        } catch (_) {
          // network without broadcast — skip the address
        }
      }
    }

    send();
    // UDP may be lost: repeat the request once.
    final resend = Timer(const Duration(milliseconds: 700), send);
    final stop = Timer(timeout, () => out.close());
    try {
      yield* out.stream;
    } finally {
      resend.cancel();
      stop.cancel();
      await sub.cancel();
      socket.close();
    }
  }

  /// A known address, with retries: after a TV powers on, the network takes a while to come up.
  Future<ServerInfo?> probeWithRetry(
    String baseUrl, {
    int attempts = 3,
    Duration pause = const Duration(seconds: 2),
  }) async {
    for (var i = 0; i < attempts; i++) {
      if (i > 0) await Future<void>.delayed(pause);
      final info = await probe(baseUrl, patient: true);
      if (info != null) return info;
    }
    return null;
  }

  /// First the build-time and saved addresses, then the UDP broadcast and all candidates
  /// in parallel; the first to respond wins.
  Future<ServerInfo?> discover({String? saved}) async {
    for (final s in [AppConfig.serverUrl, saved]) {
      if (s != null && s.isNotEmpty) {
        final info = await probe(normalize(s), patient: true);
        if (info != null) return info;
      }
    }
    final done = Completer<ServerInfo?>();
    var pending = 0;
    void track(Future<ServerInfo?> f) {
      pending++;
      f.then((info) {
        pending--;
        if (info != null && !done.isCompleted) done.complete(info);
        if (pending == 0 && !done.isCompleted) done.complete(null);
      });
    }

    StreamSubscription<String>? udp;
    if (_udp) {
      final udpDone = Completer<ServerInfo?>();
      track(udpDone.future);
      udp = broadcast().listen((url) => track(probe(url)), onDone: () => udpDone.complete(null));
    }
    for (final u in await candidates()) {
      track(probe(u));
    }
    final info = await done.future;
    await udp?.cancel();
    return info;
  }
}
