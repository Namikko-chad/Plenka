import 'package:get_it/get_it.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../data/api/config_api.dart';
import '../../data/api/engine_api.dart';
import '../../data/api/history_api.dart';
import '../../data/api/indexers_api.dart';
import '../../data/api/search_api.dart';
import '../../data/api/tmdb_api.dart';
import '../../data/api/torrent_api.dart';
import '../../data/server/server.dart';
import '../../data/storage/history_store.dart';
import '../../data/storage/settings_store.dart';

final getIt = GetIt.instance;

Future<void> configureDependencies() async {
  final prefs = await SharedPreferences.getInstance();
  final server = ServerClient();
  getIt.registerSingleton<SharedPreferences>(prefs);
  getIt.registerSingleton<SettingsStore>(SettingsStore(prefs));
  getIt.registerSingleton<ServerClient>(server);
  getIt.registerSingleton<HistoryStore>(HistoryStore(prefs, HistoryApi(server)));

  getIt.registerLazySingleton<ServerDiscovery>(() => ServerDiscovery());
  getIt.registerLazySingleton<TmdbApi>(() => TmdbApi(getIt()));
  getIt.registerLazySingleton<SearchApi>(() => SearchApi(getIt()));
  getIt.registerLazySingleton<TorrentApi>(() => TorrentApi(getIt()));
  getIt.registerLazySingleton<ConfigApi>(() => ConfigApi(getIt()));
  getIt.registerLazySingleton<IndexersApi>(() => IndexersApi(getIt()));
  getIt.registerLazySingleton<EngineApi>(() => EngineApi(getIt()));
}
