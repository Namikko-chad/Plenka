import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../../data/server/server.dart';
import '../../data/storage/history_store.dart';
import '../../data/storage/settings_store.dart';

enum ConnectionStatus { searching, connected, notFound }

class ServerConnectionState extends Equatable {
  final ConnectionStatus status;
  final ServerInfo? info;

  /// Manually set address: being connected to (searching) or it didn't respond (notFound).
  final String? address;

  const ServerConnectionState(this.status, [this.info, this.address]);

  bool get isConnected => status == ConnectionStatus.connected;

  @override
  List<Object?> get props => [
    status,
    address,
    info?.baseUrl,
    info?.version,
    info?.tmdb,
    info?.torrents,
    info?.indexersTotal,
    info?.indexersEnabled,
  ];
}

/// Finds plenka-server on the network and connects all APIs to it.
class ConnectionCubit extends Cubit<ServerConnectionState> {
  final ServerDiscovery _discovery;
  final ServerClient _client;
  final SettingsStore _settings;
  final HistoryStore _history;

  ConnectionCubit(this._discovery, this._client, this._settings, this._history)
    : super(const ServerConnectionState(ConnectionStatus.searching));

  /// Startup and "Retry": a manually set server is retried without auto-discovery
  /// (it could find another server and overwrite the choice); otherwise — auto-discovery.
  Future<void> connect() async {
    final manual = _settings.serverManual ? _settings.serverUrl : null;
    if (manual == null || manual.isEmpty) return _discover();
    emit(ServerConnectionState(ConnectionStatus.searching, null, manual));
    final info = await _discovery.probeWithRetry(manual);
    if (info == null) {
      emit(ServerConnectionState(ConnectionStatus.notFound, null, manual));
      return;
    }
    await _apply(info);
  }

  /// "Find the server automatically": forgets the manual choice and searches the network.
  Future<void> discoverAutomatically() async {
    await _settings.setServerManual(false);
    await _discover();
  }

  Future<void> _discover() async {
    emit(const ServerConnectionState(ConnectionStatus.searching));
    final info = await _discovery.discover(saved: _settings.serverUrl);
    if (info == null) {
      emit(const ServerConnectionState(ConnectionStatus.notFound));
      return;
    }
    await _apply(info);
  }

  /// Manually entered address: remembered and tried first on later launches.
  /// Returns false if no server responded there (the current connection stays unchanged).
  Future<bool> connectTo(String address) async {
    final info = await _discovery.probe(ServerDiscovery.normalize(address), patient: true);
    if (info == null) return false;
    await _apply(info, manual: true);
    return true;
  }

  /// Reload the server state (for the settings screen).
  Future<void> refresh() async {
    final current = state.info;
    if (current == null) return connect();
    final info = await _discovery.probe(current.baseUrl, patient: true);
    if (info != null) await _apply(info);
  }

  Future<void> _apply(ServerInfo info, {bool? manual}) async {
    final changed = _client.baseUrl != info.baseUrl;
    _client.attach(info);
    await _settings.setServerUrl(info.baseUrl);
    if (manual != null) await _settings.setServerManual(manual);
    emit(ServerConnectionState(ConnectionStatus.connected, info));
    if (changed) _history.sync();
  }
}
