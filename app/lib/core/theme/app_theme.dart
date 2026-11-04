import 'package:flutter/material.dart';
import 'colors.dart';

/// Type scale from DESIGN_MANIFEST.md § 3.2. Pick a table row; don't invent in-between sizes.
/// Fonts live in assets/fonts — nothing is downloaded.
abstract final class AppText {
  static const String mono = 'JetBrainsMono';
  static const String display = 'Syne';
  // Syne has no Cyrillic — Russian headings render in JetBrains Mono.
  static const List<String> _displayFallback = [mono];

  /// 22 — page title, logo.
  static const pageTitle = TextStyle(
    fontFamily: display,
    fontFamilyFallback: _displayFallback,
    fontSize: 22,
    fontWeight: FontWeight.w800,
    letterSpacing: -0.5,
    height: 1.2,
  );

  /// 20 — large value (rating, counter).
  static const statValue = TextStyle(
    fontFamily: display,
    fontFamilyFallback: _displayFallback,
    fontSize: 20,
    fontWeight: FontWeight.w800,
  );

  /// 15 — dialog / card title.
  static const modalTitle = TextStyle(
    fontFamily: display,
    fontFamilyFallback: _displayFallback,
    fontSize: 15,
    fontWeight: FontWeight.w700,
    color: AppColors.text,
  );

  /// 14 — text field value.
  static const input = TextStyle(fontFamily: mono, fontSize: 14, fontWeight: FontWeight.w400);

  /// 13 — body text.
  static const body = TextStyle(fontFamily: mono, fontSize: 13, fontWeight: FontWeight.w400, height: 1.45);

  /// 12 — secondary text, field label (labels are uppercase).
  static const small = TextStyle(fontFamily: mono, fontSize: 12, fontWeight: FontWeight.w400, color: AppColors.muted);
  static const label = TextStyle(
    fontFamily: mono,
    fontSize: 12,
    fontWeight: FontWeight.w600,
    letterSpacing: 0.5,
    color: AppColors.muted,
  );

  /// 11 — buttons, navigation.
  static const button = TextStyle(fontFamily: mono, fontSize: 11, fontWeight: FontWeight.w600, letterSpacing: 0.5);

  /// 11 — section heading (uppercase, Syne).
  static const section = TextStyle(
    fontFamily: display,
    fontFamilyFallback: _displayFallback,
    fontSize: 11,
    fontWeight: FontWeight.w700,
    letterSpacing: 1.5,
    color: AppColors.muted,
  );

  /// 10 — badge.
  static const badge = TextStyle(fontFamily: mono, fontSize: 10, fontWeight: FontWeight.w600, letterSpacing: 0.5);
}

/// Radii § 7–14: badge 4, button/field 8, table 10, card 12, dialog 14.
abstract final class AppRadius {
  static const double badge = 4;
  static const double segment = 6;
  static const double control = 8;
  static const double card = 12;
  static const double modal = 14;
}

/// Button variants § 7.3.
enum ButtonVariant { primary, cyan, red, neutral }

abstract final class AppButtons {
  static const double height = 30;

  static ButtonStyle of(ButtonVariant v) => switch (v) {
    ButtonVariant.primary => _primary(),
    ButtonVariant.cyan => _ghost(AppColors.accent),
    ButtonVariant.red => _ghost(AppColors.error),
    ButtonVariant.neutral => _neutral(),
  };

  static bool _lit(Set<WidgetState> s) =>
      s.contains(WidgetState.focused) || s.contains(WidgetState.hovered) || s.contains(WidgetState.pressed);

  static ButtonStyle _base() => const ButtonStyle(
    minimumSize: WidgetStatePropertyAll(Size(0, height)),
    padding: WidgetStatePropertyAll(EdgeInsets.symmetric(horizontal: 14)),
    tapTargetSize: MaterialTapTargetSize.shrinkWrap,
    visualDensity: VisualDensity.standard,
    shape: WidgetStatePropertyAll(
      RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(AppRadius.control))),
    ),
    textStyle: WidgetStatePropertyAll(AppText.button),
    iconSize: WidgetStatePropertyAll(14),
    elevation: WidgetStatePropertyAll(0),
    overlayColor: WidgetStatePropertyAll(Colors.transparent),
    animationDuration: Duration(milliseconds: 150),
  );

  /// Solid cyan button: one per screen.
  static ButtonStyle _primary() => _base().copyWith(
    backgroundColor: WidgetStateProperty.resolveWith((s) {
      if (s.contains(WidgetState.disabled)) return AppColors.accent.withValues(alpha: 0.5);
      return _lit(s) ? AppColors.accent.withValues(alpha: 0.9) : AppColors.accent;
    }),
    foregroundColor: const WidgetStatePropertyAll(AppColors.onAccent),
    iconColor: const WidgetStatePropertyAll(AppColors.onAccent),
    // Remote focus: a light border around the solid button.
    side: WidgetStateProperty.resolveWith(
      (s) => s.contains(WidgetState.focused) ? const BorderSide(color: AppColors.text, width: 2) : BorderSide.none,
    ),
  );

  /// Transparent button tinted with a status color: background 10 %, border 30 %, +10 % on focus.
  static ButtonStyle _ghost(Color c) {
    return _base().copyWith(
      backgroundColor: WidgetStateProperty.resolveWith(
        (s) => c.withValues(alpha: _lit(s) && !s.contains(WidgetState.disabled) ? AppAlpha.ghostHover : AppAlpha.ghost),
      ),
      foregroundColor: WidgetStateProperty.resolveWith(
        (s) => s.contains(WidgetState.disabled) ? c.withValues(alpha: 0.5) : c,
      ),
      iconColor: WidgetStateProperty.resolveWith(
        (s) => s.contains(WidgetState.disabled) ? c.withValues(alpha: 0.5) : c,
      ),
      side: WidgetStateProperty.resolveWith(
        (s) => s.contains(WidgetState.focused)
            ? BorderSide(color: c, width: 2)
            : BorderSide(color: c.withValues(alpha: AppAlpha.line)),
      ),
    );
  }

  /// Cancel / close.
  static ButtonStyle _neutral() => _base().copyWith(
    backgroundColor: WidgetStateProperty.resolveWith(
      (s) => _lit(s) ? AppColors.muted.withValues(alpha: AppAlpha.ghost) : Colors.transparent,
    ),
    foregroundColor: WidgetStateProperty.resolveWith((s) {
      if (s.contains(WidgetState.disabled)) return AppColors.muted.withValues(alpha: 0.5);
      return _lit(s) ? AppColors.text : AppColors.muted;
    }),
    iconColor: WidgetStateProperty.resolveWith((s) => _lit(s) ? AppColors.text : AppColors.muted),
    side: WidgetStateProperty.resolveWith(
      (s) => s.contains(WidgetState.focused)
          ? const BorderSide(color: AppColors.accent, width: 2)
          : const BorderSide(color: AppColors.border),
    ),
  );
}

/// Dark theme per DESIGN_MANIFEST.md.
class AppTheme {
  static ThemeData get dark {
    const navShape = RoundedRectangleBorder(
      borderRadius: BorderRadius.all(Radius.circular(AppRadius.control)),
      side: BorderSide(color: Color(0x4D00D4FF)), // --hm-accent @ 30 %
    );
    final navTint = AppColors.accent.withValues(alpha: AppAlpha.tint);
    final inputBorder = OutlineInputBorder(
      borderRadius: BorderRadius.circular(AppRadius.control),
      borderSide: const BorderSide(color: AppColors.border),
    );

    return ThemeData(
      brightness: Brightness.dark,
      useMaterial3: true,
      fontFamily: AppText.mono,
      scaffoldBackgroundColor: AppColors.bg,
      canvasColor: AppColors.bg,
      colorScheme: const ColorScheme.dark(
        primary: AppColors.accent,
        onPrimary: AppColors.onAccent,
        secondary: AppColors.accent,
        onSecondary: AppColors.onAccent,
        surface: AppColors.surface,
        onSurface: AppColors.text,
        onSurfaceVariant: AppColors.muted,
        surfaceContainerLowest: AppColors.bg,
        surfaceContainerLow: AppColors.surface,
        surfaceContainer: AppColors.surface,
        surfaceContainerHigh: AppColors.surface,
        surfaceContainerHighest: AppColors.surface,
        error: AppColors.error,
        onError: AppColors.text,
        outline: AppColors.border,
        outlineVariant: AppColors.border,
      ),
      textTheme: const TextTheme(
        displayLarge: AppText.pageTitle,
        displayMedium: AppText.pageTitle,
        displaySmall: AppText.pageTitle,
        headlineLarge: AppText.pageTitle,
        headlineMedium: AppText.pageTitle,
        headlineSmall: AppText.pageTitle,
        titleLarge: AppText.pageTitle,
        titleMedium: AppText.modalTitle,
        titleSmall: TextStyle(fontFamily: AppText.mono, fontSize: 13, fontWeight: FontWeight.w600),
        bodyLarge: AppText.input,
        bodyMedium: AppText.body,
        bodySmall: AppText.small,
        labelLarge: AppText.button,
        labelMedium: AppText.button,
        labelSmall: AppText.badge,
      ).apply(bodyColor: AppColors.text, displayColor: AppColors.text),
      iconTheme: const IconThemeData(color: AppColors.text, size: 20),
      // Remote focus: 15 % tint (for ListTile and other standard widgets).
      focusColor: AppColors.accent.withValues(alpha: AppAlpha.badge),
      hoverColor: AppColors.accent.withValues(alpha: AppAlpha.tint),
      splashFactory: NoSplash.splashFactory,
      highlightColor: Colors.transparent,
      appBarTheme: const AppBarTheme(
        backgroundColor: AppColors.bg,
        foregroundColor: AppColors.text,
        elevation: 0,
        scrolledUnderElevation: 0,
        surfaceTintColor: Colors.transparent,
        titleTextStyle: AppText.pageTitle,
        toolbarHeight: 64,
        iconTheme: IconThemeData(color: AppColors.muted, size: 20),
        shape: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      dividerTheme: const DividerThemeData(color: AppColors.border, thickness: 1, space: 1),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: AppColors.bg,
        isDense: true,
        contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 11),
        border: inputBorder,
        enabledBorder: inputBorder,
        disabledBorder: inputBorder,
        focusedBorder: inputBorder.copyWith(borderSide: const BorderSide(color: AppColors.accent, width: 1.5)),
        errorBorder: inputBorder.copyWith(borderSide: const BorderSide(color: AppColors.error)),
        focusedErrorBorder: inputBorder.copyWith(borderSide: const BorderSide(color: AppColors.error, width: 1.5)),
        hintStyle: AppText.input.copyWith(color: AppColors.muted),
        errorStyle: const TextStyle(fontFamily: AppText.mono, fontSize: 11, color: AppColors.error),
        prefixIconColor: WidgetStateColor.resolveWith(
          (s) => s.contains(WidgetState.focused) ? AppColors.accent : AppColors.muted,
        ),
        prefixIconConstraints: const BoxConstraints(minWidth: 40, minHeight: 36),
      ),
      textSelectionTheme: TextSelectionThemeData(
        cursorColor: AppColors.accent,
        selectionColor: AppColors.accent.withValues(alpha: AppAlpha.line),
        selectionHandleColor: AppColors.accent,
      ),
      filledButtonTheme: FilledButtonThemeData(style: AppButtons.of(ButtonVariant.primary)),
      outlinedButtonTheme: OutlinedButtonThemeData(style: AppButtons.of(ButtonVariant.cyan)),
      textButtonTheme: TextButtonThemeData(style: AppButtons.of(ButtonVariant.neutral)),
      elevatedButtonTheme: ElevatedButtonThemeData(style: AppButtons.of(ButtonVariant.cyan)),
      iconButtonTheme: IconButtonThemeData(
        style: ButtonStyle(
          foregroundColor: WidgetStateProperty.resolveWith(
            (s) =>
                s.contains(WidgetState.focused) || s.contains(WidgetState.hovered) ? AppColors.accent : AppColors.muted,
          ),
          backgroundColor: WidgetStateProperty.resolveWith(
            (s) => s.contains(WidgetState.focused) || s.contains(WidgetState.hovered)
                ? AppColors.accent.withValues(alpha: AppAlpha.tint)
                : Colors.transparent,
          ),
          side: WidgetStateProperty.resolveWith(
            (s) =>
                s.contains(WidgetState.focused) ? const BorderSide(color: AppColors.accent, width: 2) : BorderSide.none,
          ),
          shape: const WidgetStatePropertyAll(
            RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(AppRadius.control))),
          ),
          overlayColor: const WidgetStatePropertyAll(Colors.transparent),
        ),
      ),
      dialogTheme: const DialogThemeData(
        backgroundColor: AppColors.surface,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        barrierColor: Color(0x99000000), // rgba(0,0,0,0.6)
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.all(Radius.circular(AppRadius.modal)),
          side: BorderSide(color: AppColors.border),
        ),
        titleTextStyle: AppText.modalTitle,
        contentTextStyle: TextStyle(fontFamily: AppText.mono, fontSize: 13, color: AppColors.text),
        insetPadding: EdgeInsets.all(24),
        actionsPadding: EdgeInsets.fromLTRB(24, 0, 24, 24),
      ),
      snackBarTheme: const SnackBarThemeData(
        behavior: SnackBarBehavior.floating,
        backgroundColor: AppColors.surface,
        elevation: 0,
        width: 420,
        contentTextStyle: TextStyle(fontFamily: AppText.mono, fontSize: 12, color: AppColors.text),
        actionTextColor: AppColors.accent,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.all(Radius.circular(AppRadius.control)),
          side: BorderSide(color: AppColors.border),
        ),
      ),
      progressIndicatorTheme: const ProgressIndicatorThemeData(
        color: AppColors.accent,
        linearTrackColor: AppColors.border,
        linearMinHeight: 3,
        circularTrackColor: Colors.transparent,
      ),
      sliderTheme: SliderThemeData(
        activeTrackColor: AppColors.accent,
        inactiveTrackColor: AppColors.border,
        thumbColor: AppColors.accent,
        overlayColor: AppColors.accent.withValues(alpha: AppAlpha.ghost),
        trackHeight: 3,
      ),
      radioTheme: RadioThemeData(
        fillColor: WidgetStateProperty.resolveWith((s) {
          if (s.contains(WidgetState.disabled)) return AppColors.muted.withValues(alpha: 0.5);
          return s.contains(WidgetState.selected) ? AppColors.accent : AppColors.muted;
        }),
      ),
      scrollbarTheme: const ScrollbarThemeData(
        thickness: WidgetStatePropertyAll(4),
        radius: Radius.circular(2),
        thumbColor: WidgetStatePropertyAll(AppColors.border),
      ),
      listTileTheme: const ListTileThemeData(
        iconColor: AppColors.muted,
        titleTextStyle: TextStyle(fontFamily: AppText.mono, fontSize: 13, fontWeight: FontWeight.w600),
        subtitleTextStyle: AppText.small,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(AppRadius.control))),
      ),
      navigationRailTheme: NavigationRailThemeData(
        backgroundColor: Colors.transparent,
        indicatorColor: navTint,
        indicatorShape: navShape,
        useIndicator: true,
        selectedIconTheme: const IconThemeData(color: AppColors.accent, size: 20),
        unselectedIconTheme: const IconThemeData(color: AppColors.muted, size: 20),
        selectedLabelTextStyle: AppText.button.copyWith(color: AppColors.accent),
        unselectedLabelTextStyle: AppText.button.copyWith(color: AppColors.muted),
      ),
      navigationBarTheme: NavigationBarThemeData(
        backgroundColor: AppColors.surface,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        indicatorColor: navTint,
        indicatorShape: navShape,
        iconTheme: WidgetStateProperty.resolveWith(
          (s) => IconThemeData(color: s.contains(WidgetState.selected) ? AppColors.accent : AppColors.muted, size: 20),
        ),
        labelTextStyle: WidgetStateProperty.resolveWith(
          (s) => AppText.button.copyWith(color: s.contains(WidgetState.selected) ? AppColors.accent : AppColors.muted),
        ),
      ),
      tooltipTheme: const TooltipThemeData(
        textStyle: TextStyle(fontFamily: AppText.mono, fontSize: 11, color: AppColors.text),
        decoration: BoxDecoration(
          color: AppColors.surface,
          borderRadius: BorderRadius.all(Radius.circular(AppRadius.control)),
          border: Border.fromBorderSide(BorderSide(color: AppColors.border)),
        ),
      ),
    );
  }
}
