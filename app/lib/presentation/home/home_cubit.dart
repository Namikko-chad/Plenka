import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../../data/api/tmdb_api.dart';
import '../../data/models/media.dart';

class HomeRow {
  final HomeSection section;
  final List<MediaItem> items;
  const HomeRow(this.section, this.items);
}

class HomeState extends Equatable {
  final bool loading;
  final List<HomeRow> rows;
  final Object? error;

  const HomeState({this.loading = false, this.rows = const [], this.error});

  @override
  List<Object?> get props => [loading, rows, error];
}

class HomeCubit extends Cubit<HomeState> {
  final TmdbApi _tmdb;
  HomeCubit(this._tmdb) : super(const HomeState(loading: true));

  /// All collections load in parallel; a failed collection is simply not shown.
  Future<void> load() async {
    emit(const HomeState(loading: true));
    Object? firstError;
    final rows = await Future.wait(
      HomeSection.values.map((s) async {
        try {
          return HomeRow(s, await _tmdb.list(s.path, type: s.type));
        } catch (e) {
          firstError ??= e;
          return null;
        }
      }),
    );
    final ok = rows.whereType<HomeRow>().where((r) => r.items.isNotEmpty).toList();
    emit(HomeState(rows: ok, error: ok.isEmpty ? firstError : null));
  }
}
