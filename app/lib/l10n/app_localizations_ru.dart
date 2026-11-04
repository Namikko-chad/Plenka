// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Russian (`ru`).
class AppLocalizationsRu extends AppLocalizations {
  AppLocalizationsRu([String locale = 'ru']) : super(locale);

  @override
  String get tmdbLanguage => 'ru-RU';

  @override
  String get navHome => 'ГЛАВНАЯ';

  @override
  String get navSearch => 'ПОИСК';

  @override
  String get navSettings => 'НАСТРОЙКИ';

  @override
  String get cancel => 'Отмена';

  @override
  String get retry => 'Повторить';

  @override
  String get remove => 'Убрать';

  @override
  String get statusOk => 'OK';

  @override
  String get statusError => 'Ошибка';

  @override
  String quoted(String text) {
    return '«$text»';
  }

  @override
  String get badgeSeries => 'Сериал';

  @override
  String get routerAddress => 'Адрес роутера';

  @override
  String get byteUnits => 'Б КБ МБ ГБ ТБ';

  @override
  String perSecond(String size) {
    return '$size/с';
  }

  @override
  String seconds(String value) {
    return '$value с';
  }

  @override
  String get errorTmdbNotConfigured =>
      'TMDB не настроен: укажите ключ в Настройки → Каталог и поиск';

  @override
  String get errorTmdbKeyRejected =>
      'TMDB отклонил ключ: проверьте его в Настройки → Каталог и поиск';

  @override
  String errorServerMessage(int code, String message) {
    return 'Ошибка сервера ($code): $message';
  }

  @override
  String errorServerCode(String code) {
    return 'Ошибка сервера: $code';
  }

  @override
  String get errorServerSlow => 'Сервер долго не отвечает';

  @override
  String get errorServerUnreachable => 'Нет связи с сервером';

  @override
  String get connectSearching => 'Ищу сервер на роутере…';

  @override
  String get connectNoResponse => 'По этому адресу сервер не отвечает';

  @override
  String get connectNotFound => 'Сервер не найден';

  @override
  String get connectNotFoundHint =>
      'Проверьте, что контейнер на роутере запущен и устройство в домашней сети. Или введите адрес роутера.';

  @override
  String get connectSearchAgain => 'Искать снова';

  @override
  String connectAddressHint(int port) {
    return '192.168.88.1 или https://… (порт $port по умолчанию)';
  }

  @override
  String get connectButton => 'Подключиться';

  @override
  String get homeTrending => 'В тренде';

  @override
  String get homeNowPlaying => 'Сейчас в кино';

  @override
  String get homePopularMovies => 'Популярные фильмы';

  @override
  String get homePopularTv => 'Популярные сериалы';

  @override
  String get homeTopMovies => 'Лучшие фильмы';

  @override
  String get homeTopTv => 'Лучшие сериалы';

  @override
  String get homeContinue => 'Продолжить просмотр';

  @override
  String get homeRemoveTitle => 'Убрать из «Продолжить»?';

  @override
  String get searchHint => 'Фильм или сериал';

  @override
  String get searchEnterTitle => 'Введите название';

  @override
  String get searchNothingFound => 'Ничего не найдено';

  @override
  String detailsMinutes(int count) {
    return '$count мин';
  }

  @override
  String detailsSeasons(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count сезонов',
      few: '$count сезона',
      one: '$count сезон',
    );
    return '$_temp0';
  }

  @override
  String get detailsWatch => 'Смотреть';

  @override
  String detailsContinue(String name) {
    return 'Продолжить: $name';
  }

  @override
  String get detailsOverview => 'Описание';

  @override
  String get detailsCast => 'В ролях';

  @override
  String get releasesSortSeeders => 'По сидам';

  @override
  String get releasesSortSize => 'По размеру';

  @override
  String get releasesSortDate => 'По дате';

  @override
  String get releasesQueryHint => 'Свой запрос к трекерам';

  @override
  String get releasesSort => 'Сортировка';

  @override
  String get releasesQuality => 'Качество';

  @override
  String get releasesQualityAny => 'Любое';

  @override
  String get releasesSearching => 'Ищу раздачи на трекерах…';

  @override
  String get releasesNotFound =>
      'Раздач не найдено.\nПопробуйте изменить запрос или включить больше трекеров.';

  @override
  String releasesIndexersAnswered(int ok, int total) {
    return 'Трекеры: ответили $ok из $total';
  }

  @override
  String releasesIndexersFailed(String names) {
    return 'не ответили: $names';
  }

  @override
  String get filesTitle => 'Файлы';

  @override
  String get filesConnectingPeers => 'Подключаюсь к пирам…';

  @override
  String get filesFetchingList => 'Получаю список файлов от пиров…';

  @override
  String get filesAddFailed => 'Сервер не смог добавить раздачу';

  @override
  String get filesNoPeers => 'Не удалось получить список файлов: нет пиров';

  @override
  String get playOpenWith => 'Чем открыть?';

  @override
  String get playBuiltInPlayer => 'Встроенный плеер';

  @override
  String get playExternalPlayer => 'Внешний плеер';

  @override
  String get playerModeInternal => 'Встроенный';

  @override
  String get playerModeExternal => 'Внешний (VLC, MX Player…)';

  @override
  String get playerModeAsk => 'Спрашивать каждый раз';

  @override
  String get playerAudioTrack => 'Звуковая дорожка';

  @override
  String get playerSubtitles => 'Субтитры';

  @override
  String get playerSubtitlesOff => 'Выключены';

  @override
  String playerTrack(String id) {
    return 'Дорожка $id';
  }

  @override
  String get playerNoTracks => 'Нет дорожек';

  @override
  String playerPeers(String speed, String peers) {
    return '$speed · пиры $peers';
  }

  @override
  String get playerPause => 'Пауза';

  @override
  String get playerPlay => 'Играть';

  @override
  String get playerAudio => 'Звук';

  @override
  String get playerNext => 'Следующая';

  @override
  String get settingsTitle => 'Настройки';

  @override
  String get settingsServer => 'Сервер';

  @override
  String get settingsServerAddress => 'Адрес сервера';

  @override
  String get settingsConnect => 'Подключить';

  @override
  String get settingsNotConnected => 'не подключено';

  @override
  String get settingsTorrents => 'Торренты';

  @override
  String get settingsTorrentsOk => 'воспроизведение торрентов';

  @override
  String get settingsTorrentsOff => 'торрент-движок не запущен';

  @override
  String get settingsIndexers => 'Трекеры';

  @override
  String settingsIndexersCount(int enabled, int total) {
    return 'включено $enabled из $total';
  }

  @override
  String get settingsTmdbOk => 'каталог фильмов';

  @override
  String get settingsTmdbOff =>
      'ключ не задан — укажите его в «Каталог и поиск»';

  @override
  String get settingsPlayer => 'Плеер';

  @override
  String get settingsLanguage => 'Язык';

  @override
  String get languageSystem => 'Язык устройства';

  @override
  String get save => 'Сохранить';

  @override
  String get saved => 'Сохранено';

  @override
  String get on => 'Вкл.';

  @override
  String get off => 'Выкл.';

  @override
  String connectConnectingTo(String address) {
    return 'Подключаюсь к $address…';
  }

  @override
  String connectSavedNotResponding(String address) {
    return 'Сервер $address не отвечает. Проверьте, что он запущен и устройство в той же сети, или выберите другой сервер.';
  }

  @override
  String get connectSearchAuto => 'Искать автоматически';

  @override
  String get releasesNoIndexers => 'Нет включённых трекеров.';

  @override
  String get releasesSetUpIndexers => 'Настроить трекеры';

  @override
  String get settingsChangeServer => 'Сменить сервер';

  @override
  String get settingsChangeServerHint => 'Указать адрес другого plenka-server';

  @override
  String get settingsServerManual => 'задан вручную';

  @override
  String get settingsServerAuto => 'найден автоматически';

  @override
  String get settingsFindAuto => 'Найти сервер автоматически';

  @override
  String get settingsServerSettings => 'Настройки сервера';

  @override
  String get settingsCatalog => 'Каталог и поиск';

  @override
  String get settingsCatalogHint => 'Ключ TMDB, таймаут поиска, FlareSolverr';

  @override
  String get settingsEngine => 'Торрент-движок';

  @override
  String get settingsEngineHint =>
      'Кэш, соединения, лимиты скорости, протоколы';

  @override
  String get settingsIndexersHint => 'Включение трекеров, логины, определения';

  @override
  String get catalogTmdbKey => 'Ключ TMDB API';

  @override
  String get catalogTmdbKeyHint => 'v3 API key или v4 токен';

  @override
  String get catalogTmdbKeySet => 'Ключ задан. Введите новый, чтобы заменить.';

  @override
  String get catalogTmdbKeyRemove => 'Удалить ключ';

  @override
  String get catalogTmdbKeyRemoved => 'Ключ удалён';

  @override
  String get catalogSearchTimeout => 'Таймаут поиска по трекерам, с';

  @override
  String get catalogFlareSolverr => 'Адрес FlareSolverr';

  @override
  String get catalogFlareSolverrHint =>
      'Для трекеров за проверкой Cloudflare, например http://192.168.88.1:8191. Пусто — выключено.';

  @override
  String get engineCacheSize => 'Размер кэша, МБ';

  @override
  String get engineReadAhead => 'Упреждающее чтение, %';

  @override
  String get enginePreload => 'Предзагрузка на диск, % каждого файла';

  @override
  String get engineConnections => 'Лимит соединений';

  @override
  String get engineDownLimit => 'Лимит загрузки, КБ/с (0 — без лимита)';

  @override
  String get engineUpLimit => 'Лимит отдачи, КБ/с (0 — без лимита)';

  @override
  String get engineDisconnect => 'Выгружать неактивный торрент через, с';

  @override
  String get enginePort => 'Порт для входящих (0 — случайный)';

  @override
  String get engineRetrackers => 'Ретрекеры';

  @override
  String get engineRetrackersNone => 'Не трогать';

  @override
  String get engineRetrackersAdd => 'Добавлять';

  @override
  String get engineRetrackersRemove => 'Удалять';

  @override
  String get engineRetrackersReplace => 'Заменять';

  @override
  String get engineUseDisk => 'Хранить кэш на диске';

  @override
  String get engineSavePath => 'Папка для загрузок (пусто — в папке данных)';

  @override
  String get engineRemoveCacheOnDrop => 'Удалять кэш при выгрузке';

  @override
  String get engineUpload => 'Отдача (раздавать)';

  @override
  String get engineEncrypt => 'Только шифрованные соединения';

  @override
  String get engineResponsive =>
      'Отзывчивый режим (читать без проверки частей)';

  @override
  String get engineDebug => 'Отладочный лог';

  @override
  String get engineReset => 'Сбросить по умолчанию';

  @override
  String get engineResetConfirm =>
      'Сбросить настройки торрент-движка на значения по умолчанию?';

  @override
  String get engineResetDone => 'Настройки сброшены';

  @override
  String get engineRestartHint =>
      'После сохранения движок перезапускается, открытые торренты переподключаются.';

  @override
  String get indexersSearch => 'Поиск трекера';

  @override
  String get indexersAll => 'Все';

  @override
  String get indexersEnabled => 'Включённые';

  @override
  String get indexersPublic => 'Публичные';

  @override
  String get indexersPrivate => 'Приватные';

  @override
  String get indexersTypePublic => 'публичный';

  @override
  String get indexersTypeSemi => 'полуприватный';

  @override
  String get indexersTypePrivate => 'приватный';

  @override
  String get indexersNeedsSetup => 'нужна настройка';

  @override
  String get indexersError => 'ошибка';

  @override
  String get indexersEmpty => 'Трекеры не найдены';

  @override
  String indexersCounter(int enabled, int total) {
    return 'включено $enabled из $total';
  }

  @override
  String get indexersDefinitions => 'Определения трекеров';

  @override
  String indexersDefinitionsInfo(int count, String date) {
    return 'Загружено: $count · обновлены $date';
  }

  @override
  String indexersDefinitionsNever(int count) {
    return 'Загружено: $count · ещё не обновлялись';
  }

  @override
  String get indexersDefinitionsUpdate => 'Обновить определения';

  @override
  String get indexersDefinitionsUpdating => 'Определения загружаются…';

  @override
  String indexersDefinitionsUpdated(int count) {
    return 'Определения обновлены: $count';
  }

  @override
  String indexersDefinitionsFailed(String error) {
    return 'Последнее обновление не удалось: $error';
  }

  @override
  String get indexersDefinitionsEmpty =>
      'Определений трекеров пока нет. Нажмите «Обновить определения», чтобы загрузить их.';

  @override
  String get indexerEnabled => 'Трекер включён';

  @override
  String get indexerSiteLink => 'Адрес сайта';

  @override
  String get indexerCustomLink => 'Свой адрес';

  @override
  String get indexerSettings => 'Настройки';

  @override
  String get indexerNoSettings => 'У трекера нет настроек';

  @override
  String get indexerPasswordSet => 'задан — оставьте пустым, чтобы не менять';

  @override
  String get indexerTest => 'Проверить';

  @override
  String indexerTestOk(int results, int ms) {
    return 'Работает: $results результатов за $ms мс';
  }

  @override
  String indexerTestFail(String error) {
    return 'Ошибка: $error';
  }

  @override
  String get indexerLastError => 'Последняя ошибка';

  @override
  String get indexerSaved => 'Настройки трекера сохранены';

  @override
  String get settingsVideoOutput => 'Вывод видео';

  @override
  String get videoOutputHardware => 'Аппаратный — MediaCodec (HDR, быстрее)';

  @override
  String get videoOutputHardwareHint =>
      'Декодирует прямо на экран: 10 бит, HDR10, HDR10+ и Dolby Vision';

  @override
  String get videoOutputCompatible => 'Совместимый — OpenGL';

  @override
  String get videoOutputCompatibleHint =>
      'Если картинка чёрная, с артефактами или видео не запускается';

  @override
  String get playerVideoOutputHint =>
      'Проблемы с воспроизведением? Попробуйте Настройки → Вывод видео → Совместимый — OpenGL';

  @override
  String get watchTorrents => 'Торренты';

  @override
  String get watchTorrentsHint => 'Найти раздачу на трекерах';

  @override
  String get watchTrailers => 'Трейлеры';

  @override
  String get watchTrailersHint => 'Трейлеры и ролики на YouTube';

  @override
  String get trailersTitle => 'Трейлеры';

  @override
  String get trailersLoading => 'Загрузка роликов…';

  @override
  String get trailersEmpty => 'Трейлеров нет';

  @override
  String trailersOpenFailed(String error) {
    return 'Не удалось открыть YouTube: $error';
  }

  @override
  String get preload => 'Предзагрузка';

  @override
  String preloadStarted(String name) {
    return 'Предзагрузка запущена: $name';
  }

  @override
  String preloadProgress(String done, String total) {
    return 'Предзагрузка: $done / $total';
  }

  @override
  String get downloadAll => 'Загрузить целиком';

  @override
  String get downloadAllCancel => 'Отменить загрузку целиком';

  @override
  String get downloadAllStarted =>
      'Загрузка целиком началась — торрент останется на диске сервера';

  @override
  String get downloadAllCancelled => 'Загрузка целиком отменена';

  @override
  String get downloadAllNoRoom =>
      'Торрент не поместится: на диске сервера не хватит места';

  @override
  String downloadAllProgress(String done, String total) {
    return 'Целиком: $done / $total';
  }

  @override
  String downloadAllDone(String total) {
    return 'Загружен целиком: $total';
  }

  @override
  String get navTorrents => 'ТОРРЕНТЫ';

  @override
  String get torrentsTitle => 'Торренты';

  @override
  String get torrentsAdd => 'Добавить';

  @override
  String get torrentsAddTitle => 'Добавить торрент';

  @override
  String get torrentsLink => 'Magnet, ссылка на .torrent или хеш';

  @override
  String get torrentsTitleField => 'Название (необязательно)';

  @override
  String get torrentsAdded => 'Торрент добавлен';

  @override
  String get torrentsEmpty =>
      'Торрентов нет. Добавьте раздачу из поиска или по magnet-ссылке.';

  @override
  String get torrentsDrop => 'Выгрузить из памяти';

  @override
  String get torrentsDropped => 'Выгружен из памяти';

  @override
  String get torrentsDelete => 'Удалить';

  @override
  String torrentsDeleteConfirm(String title) {
    return 'Удалить «$title» из списка торрентов?';
  }

  @override
  String get torrentsDeleted => 'Торрент удалён';

  @override
  String get torrentsDetails => 'Подробнее';

  @override
  String get torrentsStatus => 'Статус';

  @override
  String get torrentsSize => 'Размер';

  @override
  String get torrentsLoaded => 'Загружено';

  @override
  String get torrentsSpeedDown => 'Загрузка';

  @override
  String get torrentsSpeedUp => 'Отдача';

  @override
  String get torrentsPeers => 'Пиры';

  @override
  String get torrentsSeeders => 'Сиды';

  @override
  String get torrentsFull => 'Целиком';

  @override
  String get torrentsFullDone => 'Загружен целиком';

  @override
  String get torrentsHash => 'Хеш';

  @override
  String get torrentsHashCopied => 'Хеш скопирован';

  @override
  String get torrentsAddedAt => 'Добавлен';

  @override
  String get torrentsCache => 'Кэш';

  @override
  String torrentsCacheSummary(
    String filled,
    String capacity,
    int count,
    String pieceSize,
  ) {
    return 'Заполнено $filled / $capacity · частей: $count · размер части $pieceSize';
  }

  @override
  String get torrentsNoCache => 'Кэш пуст — торрент не активен';

  @override
  String get torrentsLegendDone => 'загружено';

  @override
  String get torrentsLegendPartial => 'частично';

  @override
  String get torrentsLegendQueued => 'в очереди';

  @override
  String get torrentsLegendReader => 'читается';
}
