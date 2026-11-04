import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';

/// Fake server response: status code and JSON body (null — empty).
typedef FakeResponse = (int status, Object? body);

/// Network-free HttpClientAdapter: each request goes to [handler] and is collected in [requests].
class FakeAdapter implements HttpClientAdapter {
  final FakeResponse Function(RequestOptions o) handler;
  final requests = <RequestOptions>[];

  FakeAdapter(this.handler);

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    final (status, body) = handler(options);
    return ResponseBody.fromString(
      body == null ? '' : jsonEncode(body),
      status,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

Dio fakeDio(FakeResponse Function(RequestOptions o) handler) =>
    Dio(BaseOptions(baseUrl: 'http://srv:8095'))..httpClientAdapter = FakeAdapter(handler);

FakeAdapter adapterOf(Dio dio) => dio.httpClientAdapter as FakeAdapter;
