import 'package:equatable/equatable.dart';

enum MediaType { movie, tv }

/// A TMDB movie or TV show — a catalog card.
class MediaItem extends Equatable {
  final int id;
  final MediaType type;
  final String title;
  final String originalTitle;
  final int? year;
  final String? posterPath;
  final String? backdropPath;
  final String overview;
  final double rating;

  const MediaItem({
    required this.id,
    required this.type,
    required this.title,
    required this.originalTitle,
    this.year,
    this.posterPath,
    this.backdropPath,
    this.overview = '',
    this.rating = 0,
  });

  bool get isTv => type == MediaType.tv;

  /// Parses an item from TMDB lists. /trending and /search/multi carry the type in media_type,
  /// /movie/* and /tv/* don't — then [type] is passed.
  static MediaItem? fromTmdb(Map<String, dynamic> j, {MediaType? type}) {
    final mt = switch (j['media_type']) {
      'movie' => MediaType.movie,
      'tv' => MediaType.tv,
      null => type,
      _ => null, // person and others
    };
    if (mt == null) return null;
    final isTv = mt == MediaType.tv;
    final date = (isTv ? j['first_air_date'] : j['release_date']) as String?;
    return MediaItem(
      id: j['id'] as int,
      type: mt,
      title: (isTv ? j['name'] : j['title']) as String? ?? '',
      originalTitle: (isTv ? j['original_name'] : j['original_title']) as String? ?? '',
      year: (date != null && date.length >= 4) ? int.tryParse(date.substring(0, 4)) : null,
      posterPath: j['poster_path'] as String?,
      backdropPath: j['backdrop_path'] as String?,
      overview: j['overview'] as String? ?? '',
      rating: (j['vote_average'] as num?)?.toDouble() ?? 0,
    );
  }

  Map<String, dynamic> toJson() => {
    'id': id,
    'type': type.name,
    'title': title,
    'originalTitle': originalTitle,
    'year': year,
    'posterPath': posterPath,
    'backdropPath': backdropPath,
    'overview': overview,
    'rating': rating,
  };

  /// Watch history format (local cache and /api/history). null — the entry can't be parsed.
  static MediaItem? tryFromJson(Object? j) {
    if (j is! Map) return null;
    final type = MediaType.values.asNameMap()[j['type']];
    final id = (j['id'] as num?)?.toInt();
    if (type == null || id == null) return null;
    return MediaItem.fromJson({...j.cast<String, dynamic>(), 'id': id});
  }

  factory MediaItem.fromJson(Map<String, dynamic> j) => MediaItem(
    id: (j['id'] as num).toInt(),
    type: MediaType.values.byName(j['type'] as String),
    title: j['title'] as String? ?? '',
    originalTitle: j['originalTitle'] as String? ?? '',
    year: (j['year'] as num?)?.toInt(),
    posterPath: j['posterPath'] as String?,
    backdropPath: j['backdropPath'] as String?,
    overview: j['overview'] as String? ?? '',
    rating: (j['rating'] as num?)?.toDouble() ?? 0,
  );

  @override
  List<Object?> get props => [id, type];
}

/// Full card: what only /movie/{id} and /tv/{id} provide.
class MediaDetails {
  final MediaItem item;
  final List<String> genres;
  final int? runtime;
  final int? seasons;
  final String? tagline;
  final List<String> cast;
  final List<String> countries;

  const MediaDetails({
    required this.item,
    this.genres = const [],
    this.runtime,
    this.seasons,
    this.tagline,
    this.cast = const [],
    this.countries = const [],
  });

  factory MediaDetails.fromTmdb(Map<String, dynamic> j, MediaType type) {
    final runtimes = j['episode_run_time'] as List?;
    return MediaDetails(
      item: MediaItem.fromTmdb(j, type: type)!,
      genres: [for (final g in (j['genres'] as List? ?? [])) g['name'] as String],
      runtime: (j['runtime'] as int?) ?? (runtimes != null && runtimes.isNotEmpty ? runtimes.first as int : null),
      seasons: j['number_of_seasons'] as int?,
      tagline: (j['tagline'] as String?)?.trim().isEmpty ?? true ? null : j['tagline'] as String,
      cast: [for (final c in ((j['credits']?['cast'] as List?) ?? []).take(8)) c['name'] as String],
      countries: [for (final c in (j['production_countries'] as List? ?? [])) c['name'] as String],
    );
  }
}
