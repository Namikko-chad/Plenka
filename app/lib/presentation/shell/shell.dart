import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/theme/app_theme.dart';
import '../../l10n/l10n.dart';
import '../common/widgets/common.dart';

/// Navigation shell: side menu on TV/tablet, bottom bar on phone.
class AppShell extends StatefulWidget {
  final Widget child;
  const AppShell({super.key, required this.child});

  @override
  State<AppShell> createState() => _AppShellState();
}

class _AppShellState extends State<AppShell> {
  static const _tabs = ['/', '/search', '/torrents', '/settings'];

  /// Not focusable itself — used to find the side menu items.
  final _rail = FocusNode(debugLabel: 'rail', skipTraversal: true, canRequestFocus: false);

  @override
  void dispose() {
    _rail.dispose();
    super.dispose();
  }

  /// Tab content lives in a nested Navigator with its own FocusScope, and remote arrow keys
  /// don't leave its edge. We move "Left" from the content's left edge into the menu ourselves.
  void _moveFocus(DirectionalFocusIntent intent, int index) {
    final node = FocusManager.instance.primaryFocus;
    if (node == null) return;
    if (intent.ignoreTextFields && node.context?.widget is EditableText) return;
    if (node.focusInDirection(intent.direction) || intent.direction != TraversalDirection.left) return;
    final items = _rail.traversalDescendants.toList();
    if (items.isEmpty) return;
    items[index < items.length ? index : 0].requestFocus();
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final location = GoRouterState.of(context).matchedLocation;
    final index = _tabs.indexOf(location).clamp(0, _tabs.length - 1);
    void go(int i) => context.go(_tabs[i]);

    // "Back" on any tab but home goes home instead of closing the app.
    final body = PopScope(
      canPop: index == 0,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) go(0);
      },
      child: PageBackground(child: widget.child),
    );

    if (isWide(context)) {
      return Scaffold(
        body: Row(
          children: [
            Focus(
              focusNode: _rail,
              child: FocusTraversalGroup(
                // Menu styled like the manifest header (§ 4.2): a light cyan gradient from the edge.
                child: DecoratedBox(
                  decoration: const BoxDecoration(
                    gradient: LinearGradient(colors: [Color(0x0A00D4FF), Colors.transparent]), // --hm-accent @ 4 %
                  ),
                  child: NavigationRail(
                    selectedIndex: index,
                    onDestinationSelected: go,
                    labelType: NavigationRailLabelType.all,
                    minWidth: 96,
                    groupAlignment: -0.9,
                    leading: const Padding(
                      padding: EdgeInsets.only(top: 32, bottom: 20),
                      child: Text('Plenka', style: AppText.pageTitle),
                    ),
                    destinations: [
                      NavigationRailDestination(
                        icon: const Icon(Icons.home_outlined),
                        selectedIcon: const Icon(Icons.home),
                        label: Text(l.navHome),
                      ),
                      NavigationRailDestination(icon: const Icon(Icons.search), label: Text(l.navSearch)),
                      NavigationRailDestination(
                        icon: const Icon(Icons.download_outlined),
                        selectedIcon: const Icon(Icons.download),
                        label: Text(l.navTorrents),
                      ),
                      NavigationRailDestination(
                        icon: const Icon(Icons.settings_outlined),
                        selectedIcon: const Icon(Icons.settings),
                        label: Text(l.navSettings),
                      ),
                    ],
                  ),
                ),
              ),
            ),
            const VerticalDivider(width: 1),
            Expanded(
              child: Actions(
                actions: {
                  DirectionalFocusIntent: CallbackAction<DirectionalFocusIntent>(
                    onInvoke: (intent) => _moveFocus(intent, index),
                  ),
                },
                child: FocusTraversalGroup(child: body),
              ),
            ),
          ],
        ),
      );
    }
    return Scaffold(
      body: SafeArea(bottom: false, child: body),
      bottomNavigationBar: NavigationBar(
        selectedIndex: index,
        onDestinationSelected: go,
        height: 64,
        destinations: [
          NavigationDestination(
            icon: const Icon(Icons.home_outlined),
            selectedIcon: const Icon(Icons.home),
            label: l.navHome,
          ),
          NavigationDestination(icon: const Icon(Icons.search), label: l.navSearch),
          NavigationDestination(
            icon: const Icon(Icons.download_outlined),
            selectedIcon: const Icon(Icons.download),
            label: l.navTorrents,
          ),
          NavigationDestination(
            icon: const Icon(Icons.settings_outlined),
            selectedIcon: const Icon(Icons.settings),
            label: l.navSettings,
          ),
        ],
      ),
    );
  }
}
