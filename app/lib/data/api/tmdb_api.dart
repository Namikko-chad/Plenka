import '../models/media.dart';
import '../models/trailer.dart';
import '../server/server.dart';
import '../../l10n/l10n.dart';

/// A collection for the home screen.
enum HomeSection {
  trending('/3/trending/all/week'),
  nowPlaying('/3/movie/now_playing', MediaType.movie),
  popularMovies('/3/movie/popular', MediaType.movie),
  popularTv('/3/tv/popular', MediaType.tv),
  topMovies('/3/movie/top_rated', MediaType.movie),
  topTv('/3/tv/top_rated', MediaType.tv);

  final String path;
  final MediaType? type;
  const HomeSection(this.path, [this.type]);

  String title(AppLocalizations l) => switch (this) {
    trending => l.homeTrending,
    nowPlaying => l.homeNowPlaying,
    popularMovies => l.homePopularMovies,
    popularTv => l.homePopularTv,
    topMovies => l.homeTopMovies,
    topTv => l.homeTopTv,
  };
}

/// TMDB via the server proxy (/api/tmdb → api.themoviedb.org; the server injects the key,
/// the client sets the language from the UI language). 503 — no TMDB key set on the server.
class TmdbApi {
  static const prefix = '/api/tmdb';

  final ServerClient _server;
  TmdbApi(this._server);

  /// The catalog language is the UI language.
  static Map<String, dynamic> get _lang => {'language': tr.tmdbLanguage};

  Future<List<MediaItem>> list(String path, {MediaType? type, Map<String, dynamic>? query}) async {
    final r = await _server.dio.get<Map<String, dynamic>>(
      '$prefix$path',
      queryParameters: {..._lang, 'page': 1, ...?query},
    );
    final results = (r.data?['results'] as List?) ?? const [];
    return [for (final j in results) ?MediaItem.fromTmdb(j as Map<String, dynamic>, type: type)];
  }

  Future<List<MediaItem>> search(String query) =>
      list('/3/search/multi', query: {'query': query, 'include_adult': false});

  Future<MediaDetails> details(MediaItem item) async {
    final kind = item.isTv ? 'tv' : 'movie';
    final r = await _server.dio.get<Map<String, dynamic>>(
      '$prefix/3/$kind/${item.id}',
      queryParameters: {..._lang, 'append_to_response': 'credits'},
    );
    return MediaDetails.fromTmdb(r.data!, item.type);
  }

  /// YouTube videos: in the catalog language and in English in parallel, see [mergeTrailers].
  Future<List<Trailer>> videos(MediaType type, int id) async {
    final path = '$prefix/3/${type == MediaType.tv ? 'tv' : 'movie'}/$id/videos';
    final lang = _lang['language'] as String;
    Future<Map<String, dynamic>?> fetch(String language) async =>
        (await _server.dio.get<Map<String, dynamic>>(path, queryParameters: {'language': language})).data;
    final english = lang == 'en-US';
    final res = await Future.wait([fetch(lang), if (!english) fetch('en-US')]);
    return mergeTrailers(res[0], english ? null : res[1]);
  }
}
