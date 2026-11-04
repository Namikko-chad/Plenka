import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:get_it/get_it.dart';
import 'package:plenka/data/api/config_api.dart';
import 'package:plenka/data/api/engine_api.dart';
import 'package:plenka/data/api/indexers_api.dart';
import 'package:plenka/data/server/server.dart';
import 'package:plenka/l10n/l10n.dart';
import 'package:plenka/presentation/server_settings/catalog_screen.dart';
import 'package:plenka/presentation/server_settings/engine_screen.dart';
import 'package:plenka/presentation/server_settings/indexer_screen.dart';
import 'package:plenka/presentation/server_settings/indexers_screen.dart';

import 'helpers/fake_adapter.dart';

/// A server with one tracker and default settings.
FakeResponse _server(dynamic o) => switch (o.path as String) {
  '/api/config' => (200, {'tmdbApiKeySet': true, 'searchTimeoutSec': 30, 'flareSolverrUrl': ''}),
  '/api/engine/settings' => (200, {'cacheSize': 67108864, 'readerReadAhead': 95, 'retrackersMode': 1, 'useDisk': true}),
  '/api/indexers' => (
    200,
    [
      {'id': 'rutor', 'name': 'Rutor', 'type': 'public', 'enabled': true, 'configured': true, 'language': 'ru-RU'},
      {'id': 'rutracker', 'name': 'RuTracker.org', 'type': 'private', 'configured': false, 'lastError': 'login'},
    ],
  ),
  '/api/definitions' => (200, {'count': 2, 'updatedAt': '2026-11-03T06:00:00Z'}),
  _ => (
    200,
    {
      'id': 'rutracker',
      'name': 'RuTracker.org',
      'type': 'private',
      'links': ['https://rutracker.org/'],
      'lastError': 'login failed',
      'settings': [
        {'name': 'username', 'type': 'text', 'label': 'Username'},
        {'name': 'password', 'type': 'password', 'label': 'Password'},
        {'name': 'stripcyrillic', 'type': 'checkbox', 'label': 'Strip', 'default': false},
        {
          'name': 'sort',
          'type': 'select',
          'label': 'Sort',
          'default': '0',
          'options': [
            {'value': '0', 'label': 'created'},
          ],
        },
        {
          'name': 'cats',
          'type': 'multi-select',
          'label': 'Cats',
          'default': '1',
          'options': [
            {'value': '1', 'label': 'A'},
            {'value': '2', 'label': 'B'},
          ],
        },
        {'name': 'info', 'type': 'info', 'label': 'About', 'default': '<p>Hello <b>there</b></p>'},
      ],
      'config': {'username': 'me', 'password': '', 'password__set': '1'},
    },
  ),
};

void main() {
  setUp(() {
    final c = ServerClient(dio: fakeDio(_server))
      ..attach(const ServerInfo(baseUrl: 'http://srv:8095', tmdb: true, torrents: true));
    GetIt.I
      ..registerSingleton(ConfigApi(c))
      ..registerSingleton(EngineApi(c))
      ..registerSingleton(IndexersApi(c));
  });
  tearDown(GetIt.I.reset);

  Future<void> pump(WidgetTester t, Widget screen) async {
    t.view.physicalSize = const Size(1920, 1080);
    t.view.devicePixelRatio = 1;
    addTearDown(t.view.reset);
    await t.pumpWidget(
      MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: screen,
      ),
    );
    await t.pump();
    await t.pump(const Duration(milliseconds: 100));
  }

  testWidgets('catalog settings render', (t) async {
    await pump(t, const CatalogSettingsScreen());
    expect(find.text('Remove key'), findsOneWidget);
    expect(find.text('30'), findsOneWidget);
  });

  testWidgets('engine settings render with cache in MB', (t) async {
    await pump(t, const EngineSettingsScreen());
    expect(find.text('64'), findsOneWidget);
    expect(find.text('CACHE SIZE, MB'), findsOneWidget);
  });

  testWidgets('indexers list renders and filters', (t) async {
    await pump(t, const IndexersScreen());
    expect(find.text('Rutor'), findsOneWidget);
    expect(find.text('NEEDS SETUP'), findsOneWidget);
    await t.enterText(find.byType(TextField), 'rutr');
    await t.pump();
    expect(find.text('Rutor'), findsNothing);
    expect(find.text('RuTracker.org'), findsOneWidget);
  });

  testWidgets('indexer form renders every setting type', (t) async {
    await pump(t, const IndexerScreen(id: 'rutracker'));
    expect(find.text('me'), findsOneWidget);
    expect(find.text('set — leave empty to keep it'), findsOneWidget);
    expect(find.text('Strip'), findsOneWidget);
    expect(find.text('created'), findsOneWidget);
    expect(find.text('Hello there'), findsOneWidget);
    expect(find.textContaining('login failed'), findsOneWidget);
  });
}
