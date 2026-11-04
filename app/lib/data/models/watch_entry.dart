import 'media.dart';

/// Playback position of one torrent file. Key is hash + fileId (as on the server).
class WatchEntry {
  final MediaItem? media;
  final String hash;
  final int fileId;
  final String fileName;
  final String torrentTitle;
  final int positionMs;
  final int durationMs;
  final DateTime updatedAt;

  const WatchEntry({
    this.media,
    required this.hash,
    required this.fileId,
    required this.fileName,
    required this.torrentTitle,
    required this.positionMs,
    required this.durationMs,
    required this.updatedAt,
  });

  static String keyOf(String hash, int fileId) => '$hash:$fileId';

  String get key => keyOf(hash, fileId);
  double get progress => durationMs > 0 ? (positionMs / durationMs).clamp(0, 1) : 0;

  /// Finished: less than 5% left (credits).
  bool get finished => durationMs > 0 && progress > 0.95;

  WatchEntry withUpdatedAt(DateTime t) => WatchEntry(
    media: media,
    hash: hash,
    fileId: fileId,
    fileName: fileName,
    torrentTitle: torrentTitle,
    positionMs: positionMs,
    durationMs: durationMs,
    updatedAt: t,
  );

  /// PUT /api/history body: the server sets updatedAt.
  Map<String, dynamic> toServerJson() => {
    'hash': hash,
    'fileId': fileId,
    'fileName': fileName,
    'torrentTitle': torrentTitle,
    'media': media?.toJson(),
    'positionMs': positionMs,
    'durationMs': durationMs,
  };

  /// Local cache: the same plus updatedAt (RFC 3339, UTC).
  Map<String, dynamic> toJson() => {...toServerJson(), 'updatedAt': updatedAt.toUtc().toIso8601String()};

  /// Parses the server response and the local cache. updatedAt is an RFC 3339 string
  /// or milliseconds (cache from versions before server-side history).
  factory WatchEntry.fromJson(Map<String, dynamic> j) {
    final t = j['updatedAt'];
    return WatchEntry(
      media: MediaItem.tryFromJson(j['media']),
      hash: j['hash'] as String,
      fileId: (j['fileId'] as num).toInt(),
      fileName: j['fileName'] as String? ?? '',
      torrentTitle: j['torrentTitle'] as String? ?? '',
      positionMs: (j['positionMs'] as num?)?.toInt() ?? 0,
      durationMs: (j['durationMs'] as num?)?.toInt() ?? 0,
      updatedAt: switch (t) {
        final int ms => DateTime.fromMillisecondsSinceEpoch(ms, isUtc: true),
        final String s => DateTime.tryParse(s)?.toUtc() ?? DateTime.fromMillisecondsSinceEpoch(0, isUtc: true),
        _ => DateTime.fromMillisecondsSinceEpoch(0, isUtc: true),
      },
    );
  }

  static WatchEntry? tryFromJson(Object? j) {
    if (j is! Map) return null;
    try {
      return WatchEntry.fromJson(j.cast<String, dynamic>());
    } catch (_) {
      return null;
    }
  }
}
