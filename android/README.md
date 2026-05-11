# ssh-to-go Android

Native Android client for [ssh-to-go](../README.md). Phase 1 (this commit): scaffold, login, server profiles, dashboard list. Terminal rendering lands in a later phase.

## Why a native app

The web UI works fine on desktop and is mostly OK on mobile *except* the terminal — touch IME on WebView is unreliable and tmux mouse-mode scroll feels jerky line-by-line. The native app fixes that by:

- Rendering its own scrollback with finger-tracking + momentum scroll (does **not** forward swipes to tmux).
- Custom `InputConnection` with no extracted-text mode, no autocorrect, plus an Esc/Ctrl/Tab/arrows accessory bar above the keyboard.
- Tabbed multi-terminal so you can flip between sessions without swapping browser tabs / native terminals.

The backend stays the same — the app talks to `/api/...` and `/ws/{host}/{session}?mouse=off` exactly like a second client to your existing ssh-to-go server.

## Building locally

Requires JDK 17 and Android SDK (Android Studio bundles both).

```bash
# One-time: generate the gradle wrapper (not committed)
cd android
gradle wrapper --gradle-version 8.9
./gradlew :app:assembleDebug

# APK lands at android/app/build/outputs/apk/debug/app-debug.apk
```

If you have Android Studio, just open the `android/` folder.

## CI

`.github/workflows/android.yml` builds a debug-signed APK on:
- push of an `android-v*` tag (attached to the corresponding GitHub release)
- manual `workflow_dispatch`

CI uses `gradle/actions/setup-gradle` so no wrapper jar is required in the repo.

## Phases

- **Phase 1** (current): Gradle scaffold, login, server profile storage, dashboard list (hosts + sessions). No terminal yet.
- **Phase 2**: terminal MVP with vanilla Termux view (baseline jerky scroll).
- **Phase 3**: forked terminal-view with smooth scroll + IME accessory bar — the make-or-break for the Termux-emulator route.
- **Phase 4**: multi-tab, kill/rename/handoff parity with the web dashboard.
- **Phase 5**: CI release polish.
