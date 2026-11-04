import 'package:flutter/widgets.dart';

import 'app_localizations.dart';

export 'app_localizations.dart';

extension L10nContext on BuildContext {
  /// UI strings in the current language; the widget rebuilds when the language changes.
  AppLocalizations get l10n => AppLocalizations.of(this);
}

/// Strings for code without a BuildContext (errors, formatting, TMDB requests).
/// The app updates it on every language change; English until the UI starts.
AppLocalizations tr = lookupAppLocalizations(const Locale('en'));
