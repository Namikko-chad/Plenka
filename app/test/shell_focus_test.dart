import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:plenka/l10n/l10n.dart';
import 'package:plenka/presentation/shell/shell.dart';

/// A tab like the home one: a horizontal row of cards.
class _Tab extends StatelessWidget {
  const _Tab();

  @override
  Widget build(BuildContext context) {
    return ListView(
      children: [
        const SizedBox(height: 200),
        SizedBox(
          height: 120,
          child: ListView(
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.symmetric(horizontal: 40),
            children: [
              for (var i = 0; i < 3; i++)
                Padding(
                  padding: const EdgeInsets.all(8),
                  child: ElevatedButton(autofocus: i == 0, onPressed: () {}, child: Text('card $i')),
                ),
            ],
          ),
        ),
      ],
    );
  }
}

bool _inRail() => FocusManager.instance.primaryFocus!.context!.findAncestorWidgetOfExactType<NavigationRail>() != null;

void main() {
  testWidgets('remote "left" from the content edge reaches the side menu, "right" comes back', (t) async {
    t.view.physicalSize = const Size(1920, 1080);
    t.view.devicePixelRatio = 1;
    addTearDown(t.view.reset);

    final router = GoRouter(
      routes: [
        ShellRoute(
          builder: (_, _, child) => AppShell(child: child),
          routes: [
            GoRoute(path: '/', builder: (_, _) => const _Tab()),
            GoRoute(path: '/search', builder: (_, _) => const _Tab()),
            GoRoute(path: '/torrents', builder: (_, _) => const _Tab()),
            GoRoute(path: '/settings', builder: (_, _) => const _Tab()),
          ],
        ),
      ],
    );
    addTearDown(router.dispose);
    await t.pumpWidget(
      MaterialApp.router(
        routerConfig: router,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
      ),
    );
    await t.pumpAndSettle();
    expect(_inRail(), isFalse);

    await t.sendKeyEvent(LogicalKeyboardKey.arrowRight); // right along the row
    await t.pumpAndSettle();
    expect(_inRail(), isFalse);

    await t.sendKeyEvent(LogicalKeyboardKey.arrowLeft); // back to the first card
    await t.pumpAndSettle();
    expect(_inRail(), isFalse);

    await t.sendKeyEvent(LogicalKeyboardKey.arrowLeft); // from the edge into the menu
    await t.pumpAndSettle();
    expect(_inRail(), isTrue);
    expect(find.text('HOME'), findsOneWidget);

    await t.sendKeyEvent(LogicalKeyboardKey.arrowRight); // from the menu back into the content
    await t.pumpAndSettle();
    expect(_inRail(), isFalse);
  });
}
