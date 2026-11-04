import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';

import 'core/di/injection.dart';
import 'core/theme/app_theme.dart';
import 'data/storage/settings_store.dart';
import 'l10n/l10n.dart';
import 'presentation/connect/connection_cubit.dart';
import 'presentation/router/app_router.dart';

class PlenkaApp extends StatefulWidget {
  const PlenkaApp({super.key});

  @override
  State<PlenkaApp> createState() => _PlenkaAppState();
}

class _PlenkaAppState extends State<PlenkaApp> {
  late final ConnectionCubit _connection;
  late final GoRouter _router;

  @override
  void initState() {
    super.initState();
    _connection = ConnectionCubit(getIt(), getIt(), getIt(), getIt())..connect();
    _router = createRouter(_connection);
  }

  @override
  void dispose() {
    _connection.close();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return BlocProvider.value(
      value: _connection,
      child: ValueListenableBuilder<AppLanguage>(
        valueListenable: getIt<SettingsStore>().language,
        builder: (context, language, _) => MaterialApp.router(
          title: 'Plenka',
          // null — device language; an unsupported one falls back to the first listed (English).
          locale: language.code == null ? null : Locale(language.code!),
          supportedLocales: AppLocalizations.supportedLocales,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          builder: (context, child) {
            tr = AppLocalizations.of(context);
            return Shortcuts(shortcuts: _remoteArrows, child: child!);
          },
          debugShowCheckedModeBanner: false,
          theme: AppTheme.dark,
          darkTheme: AppTheme.dark,
          themeMode: ThemeMode.dark,
          routerConfig: _router,
          // The remote's OK button activates the focused widget.
          shortcuts: {
            ...WidgetsApp.defaultShortcuts,
            const SingleActivator(LogicalKeyboardKey.select): const ActivateIntent(),
            const SingleActivator(LogicalKeyboardKey.gameButtonA): const ActivateIntent(),
            const SingleActivator(LogicalKeyboardKey.numpadEnter): const ActivateIntent(),
          },
        ),
      ),
    );
  }
}

/// Up/down on the remote always move focus — including out of a single-line text field:
/// otherwise Flutter moves the cursor within it and the form field can't be left. There are no
/// multi-line fields.
const _remoteArrows = <ShortcutActivator, Intent>{
  SingleActivator(LogicalKeyboardKey.arrowUp): DirectionalFocusIntent(TraversalDirection.up, ignoreTextFields: false),
  SingleActivator(LogicalKeyboardKey.arrowDown): DirectionalFocusIntent(
    TraversalDirection.down,
    ignoreTextFields: false,
  ),
};
