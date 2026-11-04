import 'package:dio/dio.dart';

import '../models/watch_entry.dart';
import '../server/server.dart';

/// Where the shared watch history is stored. An interface so HistoryStore can be tested offline.
abstract interface class HistoryRemote {
  /// The server is connected — network calls are allowed.
  bool get available;

  /// Newest first.
  Future<List<WatchEntry>> list({int limit = 300});

  /// Saves an entry; returns it with the server's updatedAt (null — response not parsed).
  Future<WatchEntry?> put(WatchEntry e);

  Future<void> delete(String hash, int fileId);
}

/// Watch history on the server (/api/history), shared with the web UI.
class HistoryApi implements HistoryRemote {
  final ServerClient _server;
  HistoryApi(this._server);

  @override
  bool get available => _server.isAttached;

  @override
  Future<List<WatchEntry>> list({int limit = 300}) async {
    final r = await _server.dio.get<List<dynamic>>('/api/history', queryParameters: {'limit': limit});
    return [for (final j in r.data ?? const []) ?WatchEntry.tryFromJson(j)];
  }

  @override
  Future<WatchEntry?> put(WatchEntry e) async {
    final r = await _server.dio.put<dynamic>('/api/history', data: e.toServerJson());
    return WatchEntry.tryFromJson(r.data);
  }

  @override
  Future<void> delete(String hash, int fileId) async {
    try {
      await _server.dio.delete<void>('/api/history/${Uri.encodeComponent(hash)}/$fileId');
    } on DioException catch (e) {
      if (e.response?.statusCode != 404) rethrow; // already deleted — goal achieved
    }
  }
}
