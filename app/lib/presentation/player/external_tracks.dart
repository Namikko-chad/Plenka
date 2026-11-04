import '../../data/models/torrent.dart';

const _subtitleExt = {'srt', 'ass', 'ssa', 'vtt', 'sub'};
const _audioExt = {'mka', 'ac3', 'eac3', 'dts', 'aac', 'flac', 'mp3', 'm4a', 'opus', 'ogg', 'wav'};

String _ext(String name) => name.contains('.') ? name.split('.').last.toLowerCase() : '';
String _base(String name) {
  final i = name.lastIndexOf('.');
  return (i > 0 ? name.substring(0, i) : name).toLowerCase();
}

/// External track from the same release: the file and its language guessed from name/folder.
typedef ExternalTrack = ({TorrentFile file, String? language});

/// External subtitles of the release for [video]: files with the video's base name first.
List<ExternalTrack> externalSubtitles(TorrentFile video, List<TorrentFile> files) =>
    _external(video, files, _subtitleExt);

/// External audio tracks of the release (mka, ac3, …) for [video], name matches first.
List<ExternalTrack> externalAudio(TorrentFile video, List<TorrentFile> files) => _external(video, files, _audioExt);

List<ExternalTrack> _external(TorrentFile video, List<TorrentFile> files, Set<String> exts) {
  final base = _base(video.name);
  bool matches(TorrentFile f) {
    final b = _base(f.name);
    return b.startsWith(base) || base.startsWith(b);
  }

  final list = files.where((f) => f.id != video.id && exts.contains(_ext(f.name))).toList()
    ..sort((a, b) {
      final m = (matches(a) ? 0 : 1).compareTo(matches(b) ? 0 : 1);
      return m != 0 ? m : a.path.compareTo(b.path);
    });
  return [for (final f in list) (file: f, language: guessLanguage(f.path))];
}

const _languages = {
  'ru': 'ru',
  'rus': 'ru',
  'russian': 'ru',
  'рус': 'ru',
  'русские': 'ru',
  'русский': 'ru',
  'en': 'en',
  'eng': 'en',
  'english': 'en',
  'англ': 'en',
  'английские': 'en',
  'uk': 'uk',
  'ua': 'uk',
  'ukr': 'uk',
  'ukrainian': 'uk',
  'укр': 'uk',
};

/// Language code from the words of the file path (`…/Rus Subs/Show.S01E01.eng.srt` → `en`);
/// null — can't tell. The file name beats the folder: checked from the end.
String? guessLanguage(String path) {
  final words = path.toLowerCase().split(RegExp(r'[^a-zа-яё]+')).where((w) => w.isNotEmpty).toList();
  for (final w in words.reversed) {
    final code = _languages[w];
    if (code != null) return code;
  }
  return null;
}
