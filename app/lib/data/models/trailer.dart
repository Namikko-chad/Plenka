/// A movie/TV show video on YouTube (TMDB `/3/{movie|tv}/{id}/videos`).
class Trailer {
  /// YouTube video key.
  final String key;
  final String name;

  /// Trailer, Teaser, Clip, Featurette, Behind the Scenes…
  final String type;

  /// Language code (iso_639_1), may be empty.
  final String language;
  final DateTime? publishedAt;

  const Trailer({required this.key, required this.name, required this.type, this.language = '', this.publishedAt});

  String get thumbnail => 'https://i.ytimg.com/vi/$key/hqdefault.jpg';
  String get watchUrl => 'https://www.youtube.com/watch?v=$key';

  static const _typeOrder = ['Trailer', 'Teaser', 'Clip', 'Featurette', 'Behind the Scenes'];

  int get typeRank {
    final i = _typeOrder.indexOf(type);
    return i < 0 ? _typeOrder.length : i;
  }
}

/// Merges `/videos` responses in the catalog language ([primary]) and in English ([english]):
/// YouTube only, deduplicated by `key`; catalog-language videos first, then by type
/// (Trailer > Teaser > Clip > Featurette > Behind the Scenes > others), then newest first.
List<Trailer> mergeTrailers(Map<String, dynamic>? primary, Map<String, dynamic>? english) {
  final seen = <String>{};
  final out = <(int, Trailer)>[];
  for (final (rank, body) in [(0, primary), (1, english)]) {
    for (final raw in (body?['results'] as List?) ?? const []) {
      if (raw is! Map<String, dynamic>) continue;
      final key = raw['key'] as String? ?? '';
      if (raw['site'] != 'YouTube' || key.isEmpty || !seen.add(key)) continue;
      out.add((
        rank,
        Trailer(
          key: key,
          name: raw['name'] as String? ?? '',
          type: raw['type'] as String? ?? '',
          language: raw['iso_639_1'] as String? ?? '',
          publishedAt: DateTime.tryParse(raw['published_at'] as String? ?? ''),
        ),
      ));
    }
  }
  out.sort((a, b) {
    if (a.$1 != b.$1) return a.$1.compareTo(b.$1);
    final t = a.$2.typeRank.compareTo(b.$2.typeRank);
    if (t != 0) return t;
    return (b.$2.publishedAt ?? DateTime(0)).compareTo(a.$2.publishedAt ?? DateTime(0));
  });
  return [for (final (_, t) in out) t];
}
