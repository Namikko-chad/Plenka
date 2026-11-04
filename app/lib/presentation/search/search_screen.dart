import 'dart:async';

import 'package:equatable/equatable.dart';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../../core/di/injection.dart';
import '../../data/api/tmdb_api.dart';
import '../../data/models/media.dart';
import '../common/widgets/common.dart';
import '../common/widgets/media_row.dart';
import '../../l10n/l10n.dart';

class SearchState extends Equatable {
  final String query;
  final bool loading;
  final List<MediaItem> items;
  final Object? error;

  const SearchState({this.query = '', this.loading = false, this.items = const [], this.error});

  @override
  List<Object?> get props => [query, loading, items, error];
}

class SearchCubit extends Cubit<SearchState> {
  final TmdbApi _tmdb;
  SearchCubit(this._tmdb) : super(const SearchState());

  Future<void> search(String query) async {
    final q = query.trim();
    if (q.isEmpty) return emit(const SearchState());
    emit(SearchState(query: q, loading: true, items: state.items));
    try {
      final items = await _tmdb.search(q);
      if (state.query == q) emit(SearchState(query: q, items: items));
    } catch (e) {
      if (state.query == q) emit(SearchState(query: q, error: e));
    }
  }
}

class SearchScreen extends StatefulWidget {
  const SearchScreen({super.key});

  @override
  State<SearchScreen> createState() => _SearchScreenState();
}

class _SearchScreenState extends State<SearchScreen> {
  final _cubit = SearchCubit(getIt());
  final _controller = TextEditingController();
  Timer? _debounce;

  void _onChanged(String v) {
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 500), () => _cubit.search(v));
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _controller.dispose();
    _cubit.close();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return BlocProvider.value(
      value: _cubit,
      child: Column(
        children: [
          Padding(
            padding: pagePadding(context).copyWith(top: 20),
            child: TextField(
              controller: _controller,
              autofocus: true,
              textInputAction: TextInputAction.search,
              onChanged: _onChanged,
              onSubmitted: _cubit.search,
              decoration: InputDecoration(prefixIcon: const Icon(Icons.search), hintText: context.l10n.searchHint),
            ),
          ),
          Expanded(
            child: BlocBuilder<SearchCubit, SearchState>(
              builder: (context, s) {
                if (s.error != null) return MessageView.error(s.error!, onRetry: () => _cubit.search(s.query));
                if (s.loading && s.items.isEmpty) return const LoadingIndicator();
                if (s.query.isEmpty) {
                  return MessageView(icon: Icons.movie_filter_outlined, message: context.l10n.searchEnterTitle);
                }
                if (s.items.isEmpty) {
                  return MessageView(icon: Icons.search_off, message: context.l10n.searchNothingFound);
                }
                return MediaGrid(items: s.items);
              },
            ),
          ),
        ],
      ),
    );
  }
}
