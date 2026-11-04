import 'package:flutter_test/flutter_test.dart';
import 'package:plenka/data/models/torrent.dart';

TorrentResult _r(String title) =>
    TorrentResult(title: title, tracker: 't', size: 1, seeders: 1, peers: 0, link: 'magnet:?');

void main() {
  test('HDR detection accepts HDR, HDR10, Dolby Vision and DV as words', () {
    expect(_r('Film 2160p HDR').isHdr, isTrue);
    expect(_r('Film 2160p HDR10').isHdr, isTrue);
    expect(_r('Film 2160p HDR10+').isHdr, isTrue);
    expect(_r('Film DolbyVision').isHdr, isTrue);
    expect(_r('Film [DV]').isHdr, isTrue);
    expect(_r('Film HDRip').isHdr, isFalse);
    expect(_r('DVDRip').isHdr, isFalse);
  });
}
