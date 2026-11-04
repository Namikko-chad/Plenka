import 'package:flutter/material.dart';

/// Tokens from DESIGN_MANIFEST.md § 1. No other shades — take transparency from [AppAlpha].
abstract final class AppColors {
  // § 1.1 base palette
  static const Color bg = Color(0xFF080C10);
  static const Color surface = Color(0xFF0D1117);
  static const Color border = Color(0xFF1C2333);
  static const Color accent = Color(0xFF00D4FF);
  static const Color text = Color(0xFFE6EDF3);
  static const Color muted = Color(0xFF8B949E);

  // § 1.2 statuses
  static const Color ok = Color(0xFF10B981);
  static const Color error = Color(0xFFEF4444);
  static const Color paused = Color(0xFF6B7280);
  static const Color warning = Color(0xFFF59E0B);

  // § 1.1 secondary
  static const Color purple = Color(0xFF7C3AED);
  static const Color purpleLight = Color(0xFFA78BFA);

  /// Text on a solid primary button (§ 7.3).
  static const Color onAccent = Color(0xFF0A0E17);
}

/// Opacity scale (§ 1.3). Hover/focus is always +0.10 over rest.
abstract final class AppAlpha {
  static const double grid = 0.03;
  static const double ghost = 0.10;
  static const double ghostHover = 0.20;
  static const double tint = 0.08;
  static const double badge = 0.15;
  static const double segment = 0.12;
  static const double line = 0.30;
}
