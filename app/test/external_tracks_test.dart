import 'package:flutter_test/flutter_test.dart';
import 'package:plenka/data/models/torrent.dart';
import 'package:plenka/presentation/player/external_tracks.dart';

void main() {
  const video = TorrentFile(id: 2, path: 'Show/Show.S01E01.mkv', length: 1);
  const files = [
    TorrentFile(id: 1, path: 'Show/Subs/Other.S01E02.rus.srt', length: 1),
    video,
    TorrentFile(id: 3, path: 'Show/Subs/Show.S01E01.eng.srt', length: 1),
    TorrentFile(id: 4, path: 'Show/RUS Sound/Show.S01E01.mka', length: 1),
    TorrentFile(id: 5, path: 'Show/Show.S01E02.mkv', length: 1),
    TorrentFile(id: 6, path: 'Show/info.nfo', length: 1),
    TorrentFile(id: 7, path: 'Show/Subs/Show.S01E01.Forced.ASS', length: 1),
  ];

  test('subtitles: same base name first, all listed, language guessed', () {
    final subs = externalSubtitles(video, files);
    expect(subs.map((s) => s.file.id), [7, 3, 1]);
    expect(subs.map((s) => s.language), [null, 'en', 'ru']);
  });

  test('audio: external audio files, language from the folder', () {
    final audio = externalAudio(video, files);
    expect(audio.single.file.id, 4);
    expect(audio.single.language, 'ru');
  });

  test('guessLanguage prefers the file name over the folder', () {
    expect(guessLanguage('Rus Subs/Movie.eng.srt'), 'en');
    expect(guessLanguage('Movie.Русские.srt'), 'ru');
    expect(guessLanguage('Movie.srt'), isNull);
  });
}
