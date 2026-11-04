import 'package:flutter_test/flutter_test.dart';
import 'package:plenka/data/api/tmdb_api.dart';
import 'package:plenka/data/models/media.dart';
import 'package:plenka/data/models/trailer.dart';
import 'package:plenka/data/server/server.dart';

import 'helpers/fake_adapter.dart';

Map<String, dynamic> v(String key, String type, String lang, String date, {String site = 'YouTube'}) => {
  'key': key,
  'name': 'n-$key',
  'site': site,
  'type': type,
  'iso_639_1': lang,
  'published_at': date,
};

void main() {
  test('mergeTrailers: YouTube only, dedupe by key, catalog language first, then type, then newest', () {
    final ru = {
      'results': [
        v('r-clip', 'Clip', 'ru', '2024-01-01T00:00:00.000Z'),
        v('r-old', 'Trailer', 'ru', '2023-01-01T00:00:00.000Z'),
        v('r-new', 'Trailer', 'ru', '2024-06-01T00:00:00.000Z'),
        v('vimeo', 'Trailer', 'ru', '2025-01-01T00:00:00.000Z', site: 'Vimeo'),
      ],
    };
    final en = {
      'results': [
        v('e-bts', 'Behind the Scenes', 'en', '2024-01-01T00:00:00.000Z'),
        v('e-other', 'Bloopers', 'en', '2025-01-01T00:00:00.000Z'),
        v('r-new', 'Trailer', 'en', '2024-06-01T00:00:00.000Z'),
        v('e-teaser', 'Teaser', 'en', '2022-01-01T00:00:00.000Z'),
      ],
    };
    final list = mergeTrailers(ru, en);
    expect(list.map((t) => t.key), ['r-new', 'r-old', 'r-clip', 'e-teaser', 'e-bts', 'e-other']);
    expect(list.first.language, 'ru');
    expect(list.first.thumbnail, 'https://i.ytimg.com/vi/r-new/hqdefault.jpg');
    expect(list.first.watchUrl, 'https://www.youtube.com/watch?v=r-new');
  });

  test('mergeTrailers tolerates missing bodies', () {
    expect(mergeTrailers(null, null), isEmpty);
    expect(mergeTrailers({'results': []}, null), isEmpty);
  });

  test('videos queries the catalog language and en-US through the TMDB proxy', () async {
    final dio = fakeDio((o) {
      final lang = o.queryParameters['language'];
      return (
        200,
        {
          'results': [v(lang == 'en-US' ? 'e' : 'r', 'Trailer', lang == 'en-US' ? 'en' : 'ru', '2024-01-01')],
        },
      );
    });
    final c = ServerClient(dio: dio)..attach(const ServerInfo(baseUrl: 'http://srv:8095', tmdb: true, torrents: true));
    final list = await TmdbApi(c).videos(MediaType.tv, 1399);
    final reqs = adapterOf(dio).requests;
    expect(reqs.map((r) => r.path).toSet(), {'/api/tmdb/3/tv/1399/videos'});
    expect(reqs.map((r) => r.queryParameters['language']), contains('en-US'));
    // default catalog language (en-US before the UI starts) — no second request needed
    expect(list, isNotEmpty);
  });
}
