import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../api/history_api.dart';
import '../models/media.dart';
import '../models/watch_entry.dart';

export '../models/watch_entry.dart';

/// Result of merging the local cache with the server.
typedef HistoryMerge = ({Map<String, WatchEntry> entries, Set<String> pending});

/// History merge: the newer updatedAt wins.
/// - a server entry replaces the local one unless the local one is pending upload and newer;
/// - a local entry missing on the server stays only if it is pending upload
///   (otherwise it was deleted on the server — e.g. in the web UI);
/// - entries whose deletion hasn't reached the server yet don't come back.
HistoryMerge mergeHistory({
  required Map<String, WatchEntry> local,
  required Iterable<WatchEntry> server,
  required Set<String> pending,
  Set<String> deleted = const {},
  int limit = HistoryStore.limit,
}) {
  final out = <String, WatchEntry>{};
  final stillPending = <String>{};
  for (final s in server) {
    if (deleted.contains(s.key)) continue;
    final l = local[s.key];
    if (l != null && pending.contains(s.key) && l.updatedAt.isAfter(s.updatedAt)) {
      out[s.key] = l;
      stillPending.add(s.key);
    } else {
      out[s.key] = s;
    }
  }
  for (final l in local.values) {
    if (out.containsKey(l.key) || !pending.contains(l.key) || deleted.contains(l.key)) continue;
    out[l.key] = l;
    stillPending.add(l.key);
  }
  final kept = (out.values.toList()..sort((a, b) => b.updatedAt.compareTo(a.updatedAt))).take(limit);
  final entries = {for (final e in kept) e.key: e};
  return (entries: entries, pending: stillPending.where(entries.containsKey).toSet());
}

/// Watch history: shared on the server, with a local cache in SharedPreferences —
/// the home screen shows it immediately, and writes aren't lost while the server is down.
/// Network writes don't block the UI: unsent keys accumulate in pending and go out
/// on the next save or [sync]. Listeners (the home screen) are notified on changes.
class HistoryStore extends ChangeNotifier {
  static const limit = 300;

  static const _kEntries = 'watch_history';
  static const _kPending = 'watch_history_pending';
  static const _kDeleted = 'watch_history_deleted';

  /// Flag for migrating local history (from versions before server-side history) to the server.
  static const _kMigrated = 'watch_history_server_migrated';

  final SharedPreferences _prefs;
  final HistoryRemote? _remote;

  late Map<String, WatchEntry> _entries;
  late Set<String> _pending;
  late final Set<String> _deleted;

  final _inFlight = <String>{};
  final _again = <String>{};
  Future<void>? _syncing;
  bool _flushing = false;

  HistoryStore(this._prefs, [this._remote]) {
    _entries = _loadEntries();
    _pending = {...?_prefs.getStringList(_kPending)};
    _deleted = {...?_prefs.getStringList(_kDeleted)};
    if (_prefs.getBool(_kMigrated) != true) {
      // everything that existed only on the device is uploaded to the server once
      _pending.addAll(_entries.keys);
      _prefs.setStringList(_kPending, _pending.toList());
      _prefs.setBool(_kMigrated, true);
    }
  }

  Map<String, WatchEntry> _loadEntries() {
    final raw = _prefs.getString(_kEntries);
    if (raw == null) return {};
    try {
      final list = jsonDecode(raw) as List;
      return {for (final e in list.map(WatchEntry.tryFromJson).whereType<WatchEntry>()) e.key: e};
    } catch (_) {
      return {};
    }
  }

  Future<void> _persist() async {
    final sorted = _entries.values.toList()..sort((a, b) => b.updatedAt.compareTo(a.updatedAt));
    await _prefs.setString(_kEntries, jsonEncode(sorted.take(limit).map((e) => e.toJson()).toList()));
    await _prefs.setStringList(_kPending, _pending.toList());
    await _prefs.setStringList(_kDeleted, _deleted.toList());
  }

  bool get _online => _remote?.available ?? false;

  /// Keys not yet sent to the server (for tests and diagnostics).
  Set<String> get pendingKeys => Set.unmodifiable(_pending);
  List<WatchEntry> get entries => _entries.values.toList();

  WatchEntry? entry(String hash, int fileId) => _entries[WatchEntry.keyOf(hash, fileId)];

  /// Saves locally right away, to the server in the background.
  Future<void> save(WatchEntry e) async {
    _entries[e.key] = e;
    _pending.add(e.key);
    _deleted.remove(e.key);
    notifyListeners();
    await _persist();
    unawaited(_push(e.key));
  }

  Future<void> remove(WatchEntry e) async {
    _entries.remove(e.key);
    _pending.remove(e.key);
    _deleted.add(e.key);
    notifyListeners();
    await _persist();
    unawaited(_delete(e.key));
  }

  /// Pulls history from the server and sends anything unsent. Safe to call often.
  Future<void> sync() => _syncing ??= _sync().whenComplete(() => _syncing = null);

  Future<void> _sync() async {
    if (!_online) return;
    for (final key in [..._deleted]) {
      await _delete(key);
    }
    final List<WatchEntry> server;
    try {
      server = await _remote!.list(limit: limit);
    } catch (_) {
      return; // server unreachable — work from the cache
    }
    final merged = mergeHistory(local: _entries, server: server, pending: _pending, deleted: _deleted);
    _entries = merged.entries;
    _pending = merged.pending;
    notifyListeners();
    await _persist();
    await _flushPending();
  }

  /// Sends the queue one entry at a time, oldest first, so the server assigns updatedAt in order.
  Future<void> _flushPending() async {
    if (_flushing || !_online) return;
    _flushing = true;
    try {
      final queue = [for (final k in _pending) ?_entries[k]]..sort((a, b) => a.updatedAt.compareTo(b.updatedAt));
      for (final e in queue) {
        if (!await _push(e.key, flush: false)) break; // network is down — stop trying
      }
    } finally {
      _flushing = false;
    }
  }

  /// true — the entry is on the server (or there is nothing left to send).
  Future<bool> _push(String key, {bool flush = true}) async {
    if (!_online) return false;
    if (_inFlight.contains(key)) {
      _again.add(key); // send the latest position after the current request
      return true;
    }
    final e = _entries[key];
    if (e == null || !_pending.contains(key)) return true;
    _inFlight.add(key);
    var ok = false;
    try {
      final saved = await _remote!.put(e);
      ok = true;
      if (identical(_entries[key], e)) {
        _pending.remove(key);
        // one clock for all clients — the server's
        if (saved != null) _entries[key] = e.withUpdatedAt(saved.updatedAt);
        await _persist();
      }
    } catch (_) {
      // stays in pending
    } finally {
      _inFlight.remove(key);
    }
    if (_again.remove(key)) return _push(key, flush: flush);
    if (ok && flush && _pending.isNotEmpty) unawaited(_flushPending());
    return ok;
  }

  Future<void> _delete(String key) async {
    if (!_online) return;
    final i = key.lastIndexOf(':');
    final fileId = int.tryParse(key.substring(i + 1));
    if (i <= 0 || fileId == null) {
      _deleted.remove(key);
      return;
    }
    try {
      await _remote!.delete(key.substring(0, i), fileId);
      _deleted.remove(key);
      await _persist();
    } catch (_) {
      // retry on the next sync
    }
  }

  /// The last watched file for a movie/TV show.
  WatchEntry? lastFor(MediaItem media) {
    WatchEntry? best;
    for (final e in _entries.values) {
      if (e.media == media && (best == null || e.updatedAt.isAfter(best.updatedAt))) best = e;
    }
    return best;
  }

  /// "Continue watching": one unfinished file per movie/torrent, most recent first.
  List<WatchEntry> continueWatching({int limit = 20}) {
    final sorted = _entries.values.toList()..sort((a, b) => b.updatedAt.compareTo(a.updatedAt));
    final seen = <Object>{};
    final out = <WatchEntry>[];
    for (final e in sorted) {
      if (!seen.add(e.media ?? e.hash)) continue;
      if (e.finished) continue;
      out.add(e);
      if (out.length >= limit) break;
    }
    return out;
  }
}
