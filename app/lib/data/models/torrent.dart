import 'dart:convert';

import 'package:equatable/equatable.dart';

import '../../core/utils/format.dart';
import 'media.dart';

/// A torrent from the tracker search (GET /api/search).
class TorrentResult extends Equatable {
  final String title;
  final String tracker;
  final int size;
  final int seeders;
  final int peers;

  /// What is handed to the torrent engine: the magnet if present, otherwise the result link —
  /// often a relative `/api/indexers/{id}/download?token=…` that the server resolves itself.
  final String link;
  final String? infoHash;
  final DateTime? publishDate;

  const TorrentResult({
    required this.title,
    required this.tracker,
    required this.size,
    required this.seeders,
    required this.peers,
    required this.link,
    this.infoHash,
    this.publishDate,
  });

  /// null — the torrent has neither a magnet nor a link.
  static TorrentResult? fromJson(Map<String, dynamic> j) {
    final magnet = j['magnet'] as String? ?? '';
    final link = magnet.isNotEmpty ? magnet : j['link'] as String? ?? '';
    if (link.isEmpty) return null;
    final hash = (j['infoHash'] as String? ?? '').trim().toLowerCase();
    return TorrentResult(
      title: j['title'] as String? ?? '',
      tracker: j['tracker'] as String? ?? '',
      size: (j['size'] as num?)?.toInt() ?? 0,
      seeders: (j['seeders'] as num?)?.toInt() ?? 0,
      peers: (j['peers'] as num?)?.toInt() ?? 0,
      link: link,
      infoHash: hash.isEmpty ? null : hash,
      publishDate: DateTime.tryParse(j['publishDate'] as String? ?? ''),
    );
  }

  /// Key for merging identical torrents from different requests.
  String get dedupKey => infoHash ?? '$title|$size';

  Quality get quality => Quality.of(title);

  bool get isHdr => RegExp(r'\b(hdr(10)?|dolby ?vision|dv)\b', caseSensitive: false).hasMatch(title);

  @override
  List<Object?> get props => [dedupKey];
}

enum Quality {
  uhd('4K'),
  fhd('1080p'),
  hd('720p'),
  sd('SD');

  final String label;
  const Quality(this.label);

  static Quality of(String title) {
    final t = title.toLowerCase();
    if (RegExp(r'2160p|\b4k\b|\buhd\b').hasMatch(t)) return uhd;
    if (t.contains('1080')) return fhd;
    if (t.contains('720')) return hd;
    return sd;
  }
}

class TorrentFile extends Equatable {
  /// File index in the torrent (1-based), used in the stream URL.
  final int id;
  final String path;
  final int length;

  const TorrentFile({required this.id, required this.path, required this.length});

  factory TorrentFile.fromJson(Map<String, dynamic> j) =>
      TorrentFile(id: j['id'] as int, path: j['path'] as String? ?? '', length: (j['length'] as num?)?.toInt() ?? 0);

  String get name => path.split('/').last;

  static const _videoExt = {
    'mkv',
    'mp4',
    'avi',
    'm4v',
    'mov',
    'ts',
    'm2ts',
    'mts',
    'webm',
    'wmv',
    'mpg',
    'mpeg',
    'vob',
    'flv',
  };

  bool get isVideo => _videoExt.contains(name.split('.').last.toLowerCase());

  @override
  List<Object?> get props => [id, path];
}

/// A torrent on the server (`Torrent` from GET/POST /api/torrents, camelCase).
class Torrent {
  final String hash;
  final String title;
  final String poster;

  /// Opaque client JSON (the movie card), may be empty.
  final String data;

  /// Name from the metadata; empty until it arrives.
  final String name;

  /// 0 added, 1 fetching metadata, 2 preloading, 3 working, 4 closed, 5 in DB (not loaded).
  final int stat;
  final String statString;

  /// Time added, in UTC (ISO 8601).
  final String addedAt;
  final int torrentSize;
  final int loadedSize;
  final int preloadedBytes;
  final int preloadSize;

  /// The whole torrent is downloaded and kept in the server cache;
  /// progress is [loadedSize] / [torrentSize].
  final bool fullDownload;
  final double downloadSpeed;
  final double uploadSpeed;
  final int totalPeers;
  final int pendingPeers;
  final int activePeers;
  final int connectedSeeders;
  final int halfOpenPeers;
  final int bytesRead;

  /// Empty until the metadata arrives.
  final List<TorrentFile> files;

  const Torrent({
    required this.hash,
    this.title = '',
    this.poster = '',
    this.data = '',
    this.name = '',
    this.stat = 0,
    this.statString = '',
    this.addedAt = '',
    this.torrentSize = 0,
    this.loadedSize = 0,
    this.preloadedBytes = 0,
    this.preloadSize = 0,
    this.fullDownload = false,
    this.downloadSpeed = 0,
    this.uploadSpeed = 0,
    this.totalPeers = 0,
    this.pendingPeers = 0,
    this.activePeers = 0,
    this.connectedSeeders = 0,
    this.halfOpenPeers = 0,
    this.bytesRead = 0,
    this.files = const [],
  });

  factory Torrent.fromJson(Map<String, dynamic> j) {
    int i(String k) => (j[k] as num?)?.toInt() ?? 0;
    String s(String k) => j[k] as String? ?? '';
    return Torrent(
      hash: s('hash'),
      title: s('title'),
      poster: s('poster'),
      data: s('data'),
      name: s('name'),
      stat: i('stat'),
      statString: s('statString'),
      addedAt: s('addedAt'),
      torrentSize: i('torrentSize'),
      loadedSize: i('loadedSize'),
      preloadedBytes: i('preloadedBytes'),
      preloadSize: i('preloadSize'),
      fullDownload: j['fullDownload'] == true,
      downloadSpeed: (j['downloadSpeed'] as num?)?.toDouble() ?? 0,
      uploadSpeed: (j['uploadSpeed'] as num?)?.toDouble() ?? 0,
      totalPeers: i('totalPeers'),
      pendingPeers: i('pendingPeers'),
      activePeers: i('activePeers'),
      connectedSeeders: i('connectedSeeders'),
      halfOpenPeers: i('halfOpenPeers'),
      bytesRead: i('bytesRead'),
      files: [for (final f in (j['files'] as List? ?? [])) TorrentFile.fromJson(f as Map<String, dynamic>)],
    );
  }

  /// What to show as the name: the title, else the metadata name, else the hash.
  String get displayTitle => title.isNotEmpty ? title : (name.isNotEmpty ? name : hash);

  /// Loaded in the engine: fetching metadata, preloading or working.
  bool get isActive => stat >= 1 && stat <= 3;

  /// Full download progress 0…1; null — no full download.
  double? get fullProgress =>
      fullDownload && torrentSize > 0 ? (loadedSize / torrentSize).clamp(0, 1).toDouble() : null;

  /// Preload progress 0…1 while it runs (stat 2); otherwise null.
  double? get preloadProgress =>
      stat == 2 && preloadSize > 0 ? (preloadedBytes / preloadSize).clamp(0, 1).toDouble() : null;

  /// The movie card the web UI or the app stored in [data]; null — none or unreadable.
  MediaItem? get media {
    if (data.isEmpty) return null;
    try {
      return MediaItem.tryFromJson(jsonDecode(data));
    } on FormatException {
      return null;
    }
  }

  /// Video files in natural path order.
  List<TorrentFile> get videos =>
      files.where((f) => f.isVideo).toList()..sort((a, b) => naturalCompare(a.path, b.path));

  Torrent copyWith({
    int? stat,
    String? statString,
    bool? fullDownload,
    int? activePeers,
    double? downloadSpeed,
    double? uploadSpeed,
  }) => Torrent(
    hash: hash,
    title: title,
    poster: poster,
    data: data,
    name: name,
    stat: stat ?? this.stat,
    statString: statString ?? this.statString,
    addedAt: addedAt,
    torrentSize: torrentSize,
    loadedSize: loadedSize,
    preloadedBytes: preloadedBytes,
    preloadSize: preloadSize,
    fullDownload: fullDownload ?? this.fullDownload,
    downloadSpeed: downloadSpeed ?? this.downloadSpeed,
    uploadSpeed: uploadSpeed ?? this.uploadSpeed,
    totalPeers: totalPeers,
    pendingPeers: pendingPeers,
    activePeers: activePeers ?? this.activePeers,
    connectedSeeders: connectedSeeders,
    halfOpenPeers: halfOpenPeers,
    bytesRead: bytesRead,
    files: files,
  );
}

/// One cache piece (GET /api/torrents/{hash}/cache): with data, or queued with nothing yet.
class CachePiece {
  final int id;
  final int length;
  final int size;
  final bool completed;
  final int priority;

  const CachePiece({required this.id, this.length = 0, this.size = 0, this.completed = false, this.priority = 0});

  factory CachePiece.fromJson(Map<String, dynamic> j) => CachePiece(
    id: (j['id'] as num?)?.toInt() ?? 0,
    length: (j['length'] as num?)?.toInt() ?? 0,
    size: (j['size'] as num?)?.toInt() ?? 0,
    completed: j['completed'] == true,
    priority: (j['priority'] as num?)?.toInt() ?? 0,
  );

  /// How much of the piece has arrived, 0…1.
  double get fill => completed ? 1 : (length > 0 ? (size / length).clamp(0, 1).toDouble() : 0);
}

/// Torrent cache state (GET /api/torrents/{hash}/cache).
class CacheState {
  final int capacity;
  final int filled;
  final int piecesLength;
  final int piecesCount;
  final List<CachePiece> pieces;

  /// Piece ranges the active readers read: (start, end), inclusive.
  final List<(int, int)> readers;

  const CacheState({
    this.capacity = 0,
    this.filled = 0,
    this.piecesLength = 0,
    this.piecesCount = 0,
    this.pieces = const [],
    this.readers = const [],
  });

  factory CacheState.fromJson(Map<String, dynamic> j) {
    int i(String k) => (j[k] as num?)?.toInt() ?? 0;
    final pieces = j['pieces'];
    return CacheState(
      capacity: i('capacity'),
      filled: i('filled'),
      piecesLength: i('piecesLength'),
      piecesCount: i('piecesCount'),
      pieces: [
        if (pieces is Map)
          for (final p in pieces.values) CachePiece.fromJson((p as Map).cast<String, dynamic>()),
      ],
      readers: [
        for (final r in (j['readers'] as List? ?? const []))
          (((r as Map)['start'] as num?)?.toInt() ?? 0, (r['end'] as num?)?.toInt() ?? 0),
      ],
    );
  }
}
