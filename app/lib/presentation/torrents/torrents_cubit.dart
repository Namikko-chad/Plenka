import 'dart:async';

import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../../data/api/torrent_api.dart';
import '../../data/models/torrent.dart';
import '../../data/models/watch_entry.dart';

class TorrentsState extends Equatable {
  final bool loaded;
  final List<Torrent> list;
  final Object? error;

  /// Bumped on every refresh: the torrent objects have no equality of their own.
  final int revision;

  const TorrentsState({this.loaded = false, this.list = const [], this.error, this.revision = 0});

  @override
  List<Object?> get props => [loaded, error, revision];
}

/// The engine's saved torrents (Torrents tab). Polls the list while [start]ed.
class TorrentsCubit extends Cubit<TorrentsState> {
  final TorrentApi _api;
  Timer? _timer;
  int _rev = 0;

  TorrentsCubit(this._api) : super(const TorrentsState());

  void start({Duration every = const Duration(seconds: 3)}) {
    refresh();
    _timer?.cancel();
    _timer = Timer.periodic(every, (_) => refresh());
  }

  Future<void> refresh() async {
    try {
      final list = await _api.list();
      if (!isClosed) emit(TorrentsState(loaded: true, list: list, revision: ++_rev));
    } catch (e) {
      if (!isClosed) emit(TorrentsState(loaded: state.loaded, list: state.list, error: e, revision: ++_rev));
    }
  }

  void _set(List<Torrent> list) => emit(TorrentsState(loaded: true, list: list, error: state.error, revision: ++_rev));

  void _upsert(Torrent t) {
    final list = [...state.list];
    final i = list.indexWhere((x) => x.hash == t.hash);
    if (i >= 0) {
      list[i] = t;
    } else {
      list.insert(0, t);
    }
    _set(list);
  }

  Future<Torrent> add({required String link, required String title}) async {
    final t = await _api.add(link: link, title: title);
    _upsert(t);
    return t;
  }

  Future<void> drop(String hash) async {
    await _api.drop(hash);
    final t = state.list.where((x) => x.hash == hash).firstOrNull;
    if (t != null) {
      _upsert(t.copyWith(stat: 5, statString: 'Torrent in db', activePeers: 0, downloadSpeed: 0, uploadSpeed: 0));
    }
  }

  Future<void> remove(String hash) async {
    await _api.remove(hash);
    _set(state.list.where((x) => x.hash != hash).toList());
  }

  @override
  Future<void> close() {
    _timer?.cancel();
    return super.close();
  }
}

/// The file worth preloading: the last watched one of this torrent, else the first video.
TorrentFile? preloadTarget(Torrent t, Iterable<WatchEntry> history) {
  final watched = history.where((e) => e.hash == t.hash).toList()..sort((a, b) => b.updatedAt.compareTo(a.updatedAt));
  for (final e in watched) {
    final f = t.files.where((f) => f.id == e.fileId).firstOrNull;
    if (f != null) return f;
  }
  return t.videos.firstOrNull;
}
