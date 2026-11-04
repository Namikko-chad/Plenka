import '../../l10n/l10n.dart';

String formatBytes(num bytes) {
  final units = tr.byteUnits.split(' ');
  var v = bytes.toDouble();
  var i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  return '${v.toStringAsFixed(v >= 100 || i == 0 ? 0 : 1)} ${units[i]}';
}

String formatSpeed(num bytesPerSec) => tr.perSecond(formatBytes(bytesPerSec));

String formatDuration(Duration d) {
  final h = d.inHours;
  final m = d.inMinutes.remainder(60).toString().padLeft(2, '0');
  final s = d.inSeconds.remainder(60).toString().padLeft(2, '0');
  return h > 0 ? '$h:$m:$s' : '$m:$s';
}

String formatDate(DateTime? d) {
  if (d == null) return '';
  return '${d.day.toString().padLeft(2, '0')}.${d.month.toString().padLeft(2, '0')}.${d.year}';
}

/// Natural string comparison with numbers: "Episode 2" < "Episode 10".
int naturalCompare(String a, String b) {
  final re = RegExp(r'(\d+|\D+)');
  final pa = re.allMatches(a.toLowerCase()).map((m) => m.group(0)!).toList();
  final pb = re.allMatches(b.toLowerCase()).map((m) => m.group(0)!).toList();
  for (var i = 0; i < pa.length && i < pb.length; i++) {
    final na = int.tryParse(pa[i]);
    final nb = int.tryParse(pb[i]);
    final c = (na != null && nb != null) ? na.compareTo(nb) : pa[i].compareTo(pb[i]);
    if (c != 0) return c;
  }
  return pa.length.compareTo(pb.length);
}

/// Display time: in the device's timezone, `YYYY-MM-DD HH:MM:SS`.
String formatDateTime(DateTime d) {
  final t = d.toLocal();
  String two(int v) => v.toString().padLeft(2, '0');
  return '${t.year}-${two(t.month)}-${two(t.day)} ${two(t.hour)}:${two(t.minute)}:${two(t.second)}';
}

/// Tracker HTML hint → plain text: line breaks in place of <br>, paragraphs and list items,
/// tags stripped, common entities decoded.
String htmlToText(String html) {
  var s = html
      .replaceAll(RegExp(r'<br\s*/?>', caseSensitive: false), '\n')
      .replaceAll(RegExp(r'</(p|div|h\d|li|ul|ol)>', caseSensitive: false), '\n')
      .replaceAll(RegExp(r'<li[^>]*>', caseSensitive: false), '\n• ')
      .replaceAll(RegExp(r'<[^>]*>'), '');
  const entities = {'&nbsp;': ' ', '&lt;': '<', '&gt;': '>', '&quot;': '"', '&#39;': "'", '&apos;': "'", '&amp;': '&'};
  entities.forEach((k, v) => s = s.replaceAll(k, v));
  return s.replaceAll(RegExp(r'[ \t]+'), ' ').replaceAll(RegExp(r'\s*\n\s*'), '\n').trim();
}
