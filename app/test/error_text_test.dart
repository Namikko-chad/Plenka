import 'package:dio/dio.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:plenka/l10n/l10n.dart';
import 'package:plenka/presentation/common/widgets/common.dart';

DioException _err(String path, int code, [Object? data]) {
  final o = RequestOptions(path: path);
  return DioException(
    requestOptions: o,
    response: Response(requestOptions: o, statusCode: code, data: data),
  );
}

void main() {
  test('TMDB 503 means the key is not configured', () {
    expect(errorText(_err('/api/tmdb/3/movie/popular', 503)), tr.errorTmdbNotConfigured);
  });

  test('native API errors show the server message', () {
    expect(errorText(_err('/api/search', 500, {'error': 'boom'})), 'Server error (500): boom');
    expect(errorText(_err('/api/torrents/h1', 404)), 'Server error: 404');
  });

  test('error texts follow the UI language', () {
    tr = lookupAppLocalizations(const Locale('ru'));
    addTearDown(() => tr = lookupAppLocalizations(const Locale('en')));
    expect(errorText(_err('/api/search', 500, {'error': 'boom'})), 'Ошибка сервера (500): boom');
  });
}
