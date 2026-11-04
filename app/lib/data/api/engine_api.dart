import '../models/server_settings.dart';
import '../server/server.dart';

/// Torrent engine settings (/api/engine/settings).
class EngineApi {
  final ServerClient _server;
  EngineApi(this._server);

  Future<EngineSettings> get() async {
    final r = await _server.dio.get<Map<String, dynamic>>('/api/engine/settings');
    return EngineSettings.fromJson(r.data);
  }

  /// The whole object; the engine restarts with the new settings.
  Future<EngineSettings> save(EngineSettings s) async {
    final r = await _server.dio.put<Map<String, dynamic>>('/api/engine/settings', data: s.json);
    return EngineSettings.fromJson(r.data);
  }

  /// Resets to defaults and returns them.
  Future<EngineSettings> reset() async {
    final r = await _server.dio.delete<Map<String, dynamic>>('/api/engine/settings');
    return EngineSettings.fromJson(r.data);
  }
}
