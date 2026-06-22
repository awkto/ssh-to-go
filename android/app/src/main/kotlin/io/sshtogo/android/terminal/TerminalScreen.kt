package io.sshtogo.android.terminal

import android.graphics.Typeface
import android.view.KeyEvent
import android.view.MotionEvent
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Palette
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import com.termux.terminal.TerminalSession
import com.termux.terminal.TerminalSessionClient
import com.termux.view.TerminalView
import com.termux.view.TerminalViewClient
import io.sshtogo.android.SshToGoApplication
import io.sshtogo.android.data.AppPreferences
import io.sshtogo.android.data.ServerProfile

@OptIn(ExperimentalMaterial3Api::class, ExperimentalFoundationApi::class)
@Composable
fun TerminalScreen(
    profileId: String,
    hostName: String,
    sessionName: String,
    onBack: () -> Unit,
) {
    val store = SshToGoApplication.instance.profileStore
    val profile = remember(profileId) { store.profiles.value.firstOrNull { it.id == profileId } }
    if (profile == null) {
        Text("Server profile not found")
        return
    }

    // Build the horizontal-swipe carousel from the sessions opened in this app
    // run on this host (recording the one just tapped first), rather than the
    // host's full session list — so swiping only steps between sessions we've
    // actually opened and never reconnects through ones we haven't.
    val list = remember(profile.id, hostName, sessionName) {
        val opened = SshToGoApplication.instance.openedSessions
        opened.markOpened(hostName, sessionName)
        opened.forHost(hostName).ifEmpty { listOf(sessionName) }
    }

    // Wrap-around carousel: swiping the terminal left/right switches between the
    // active sessions, and swiping past the last one loops back to the first.
    // With a single session there's nothing to switch to, so it's one fixed page.
    val initialIndex = list.indexOf(sessionName).let { if (it < 0) 0 else it }
    val pageCount = list.size
    val loop = pageCount > 1
    // A huge virtual page count fakes an endless carousel; the real session for
    // any page is page % pageCount. Start near the middle (aligned so the tapped
    // session shows first) to leave room to swipe both directions.
    val virtualCount = if (loop) Int.MAX_VALUE else 1
    val startPage = if (loop) (virtualCount / 2).let { it - it % pageCount + initialIndex } else 0
    val pagerState = rememberPagerState(initialPage = startPage) { virtualCount }
    fun sessionIndexFor(page: Int): Int = if (pageCount > 0) page % pageCount else 0
    val currentIndex = sessionIndexFor(pagerState.currentPage)
    val currentSession = list.getOrNull(currentIndex) ?: sessionName

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text(
                            "$currentSession @ $hostName",
                            style = MaterialTheme.typography.titleMedium,
                        )
                        if (loop) {
                            Text(
                                "${currentIndex + 1} / $pageCount  ·  swipe to switch",
                                style = MaterialTheme.typography.labelSmall,
                            )
                        }
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    val app = SshToGoApplication.instance

                    // Per-session palette control:
                    //  · single tap  → advance to the next palette
                    //  · long press → open the full palette list
                    // Each tmux session keeps its own colour scheme; changes
                    // apply to the active session only and persist across reconnects.
                    var paletteMenuOpen by remember { mutableStateOf(false) }
                    Box {
                        Box(
                            modifier = Modifier
                                .size(48.dp)
                                .combinedClickable(
                                    onClick = {
                                        val entries = TerminalPalette.entries
                                        val current = app.prefs.paletteFor(hostName, currentSession)
                                        val idx = entries.indexOfFirst { it.name == current }
                                            .let { if (it < 0) 0 else it }
                                        val next = entries[(idx + 1) % entries.size]
                                        app.prefs.setPaletteFor(hostName, currentSession, next.name)
                                    },
                                    onLongClick = { paletteMenuOpen = true },
                                ),
                            contentAlignment = Alignment.Center,
                        ) {
                            Icon(Icons.Default.Palette, contentDescription = "Theme — tap: next, hold: list")
                        }
                        DropdownMenu(
                            expanded = paletteMenuOpen,
                            onDismissRequest = { paletteMenuOpen = false },
                        ) {
                            val currentPaletteName = app.prefs.paletteFor(hostName, currentSession)
                            TerminalPalette.entries.forEach { palette ->
                                DropdownMenuItem(
                                    leadingIcon = if (palette.name == currentPaletteName)
                                        { { Icon(Icons.Default.Palette, contentDescription = null) } } else null,
                                    text = { Text(palette.displayName) },
                                    onClick = {
                                        paletteMenuOpen = false
                                        // paletteFor() reads from the state-backed map, so this
                                        // write triggers a recomposition of SessionTerminal and
                                        // its AndroidView's update lambda re-applies the colours
                                        // to the active emulator.
                                        app.prefs.setPaletteFor(hostName, currentSession, palette.name)
                                    },
                                )
                            }
                        }
                    }

                    // Close THIS session in the app. Drops it from the swipe
                    // carousel and returns to the list — the tmux session keeps
                    // running on the server (no kill is sent).
                    IconButton(onClick = {
                        app.openedSessions.close(hostName, currentSession)
                        onBack()
                    }) {
                        Icon(Icons.Default.Close, contentDescription = "Close session (tmux keeps running)")
                    }
                },
            )
        },
    ) { pad ->
        HorizontalPager(
            state = pagerState,
            modifier = Modifier.fillMaxSize().padding(pad).imePadding(),
            // Horizontal swipe on the terminal switches sessions; vertical drags
            // pass through to the terminal for scrollback. Disabled when there's
            // only one session to switch to.
            userScrollEnabled = loop,
            // Only the currently visible page keeps a live WebSocket +
            // emulator. Pre-warming neighbours made the app prone to freezes
            // on hosts with many sessions; the small ~1s reconnect on swipe
            // is a worthwhile trade for stability.
            beyondViewportPageCount = 0,
        ) { pageIndex ->
            SessionTerminal(profile = profile, hostName = hostName, sessionName = list[sessionIndexFor(pageIndex)])
        }
    }
}

/**
 * Renders one tmux session: opens a RelayTerminalSession + TerminalView,
 * tears them down on dispose. Kept WS-per-page so swiping between sessions
 * is instant after the initial connect.
 */
@Composable
private fun SessionTerminal(profile: ServerProfile, hostName: String, sessionName: String) {
    val surfaceColor = MaterialTheme.colorScheme.surface.toArgb()
    val sessionClient = remember(profile.id, hostName, sessionName) { ViewBackedSessionClient() }
    val session = remember(profile.id, hostName, sessionName) {
        RelayTerminalSession(profile, hostName, sessionName, sessionClient)
    }
    // Read the per-session palette name from the state-backed prefs map so
    // a change in the palette picker recomposes this view (and we'll push
    // the new colours into the live emulator via the update block below).
    val paletteName = SshToGoApplication.instance.prefs.paletteFor(hostName, sessionName)
    val palette = remember(paletteName) { TerminalPalette.fromName(paletteName) }

    DisposableEffect(session) {
        onDispose {
            sessionClient.view = null
            session.finishIfRunning()
        }
    }

    // Reconnect the relay across screen-off/on. The OS drops the WebSocket
    // while backgrounded; close it on pause and re-establish on resume so the
    // visible page recovers on wake instead of sitting on a dead socket.
    val lifecycleOwner = LocalLifecycleOwner.current
    DisposableEffect(lifecycleOwner, session) {
        val observer = LifecycleEventObserver { _, event ->
            when (event) {
                Lifecycle.Event.ON_PAUSE -> session.pause()
                Lifecycle.Event.ON_RESUME -> session.resume()
                else -> {}
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }

    Box(modifier = Modifier.fillMaxSize()) {
        AndroidView(
            modifier = Modifier.fillMaxSize(),
            factory = { context ->
                // Apply the per-session palette to the global COLOR_SCHEME
                // BEFORE the emulator is created — the emulator copies from
                // COLOR_SCHEME at init, so this baked-in palette will stick.
                palette.apply()
                TerminalView(context, null).apply {
                    setBackgroundColor(surfaceColor)
                    val app = SshToGoApplication.instance
                    setTextSize(app.prefs.terminalFontSizeSp.scaledPx(context))
                    setTypeface(Typeface.MONOSPACE)
                    setTerminalViewClient(ComposeTerminalViewClient(this, session, app.prefs))
                    attachSession(session)
                    isFocusable = true
                    isFocusableInTouchMode = true
                    sessionClient.view = this
                }
            },
            update = { view ->
                // Palette change while the view is already alive: re-apply to
                // COLOR_SCHEME, reset the active emulator's mColors so it
                // copies the new defaults, and force a redraw.
                palette.apply()
                view.currentSession?.emulator?.let { em ->
                    em.mColors.reset()
                    view.onScreenUpdated()
                }
            },
        )
    }
}

private fun Float.scaledPx(context: android.content.Context): Int =
    (this * context.resources.displayMetrics.scaledDensity).toInt()

/**
 * Bridges TerminalView's input events back into the session, and
 * translates pinch-zoom scale changes into actual font-size changes.
 */
private class ComposeTerminalViewClient(
    private val view: TerminalView,
    private val session: TerminalSession,
    private val prefs: AppPreferences,
) : TerminalViewClient {

    // TerminalView accumulates pinch scale into mScaleFactor and reports the
    // running total here. We treat the report as "scale relative to last
    // call" and translate that delta into a font-size change, persisting it
    // only when the pixel-rounded size actually changes (so we don't rebuild
    // the renderer every frame at sub-pixel granularity).
    private var lastReportedScale: Float = 1f

    override fun onScale(scale: Float): Float {
        if (scale <= 0f) return scale
        val delta = scale / lastReportedScale
        lastReportedScale = scale

        val density = view.resources.displayMetrics.scaledDensity
        val currentSp = prefs.terminalFontSizeSp
        val currentPx = (currentSp * density).toInt()
        val newSp = (currentSp * delta).coerceIn(
            AppPreferences.TerminalFontSizeMinSp,
            AppPreferences.TerminalFontSizeMaxSp,
        )
        val newPx = (newSp * density).toInt()
        if (newPx != currentPx) {
            prefs.terminalFontSizeSp = newSp
            view.setTextSize(newPx)
        }
        return scale
    }

    override fun onSingleTapUp(e: MotionEvent) {
        // Bring up the soft keyboard.
        val imm = view.context.getSystemService(android.content.Context.INPUT_METHOD_SERVICE)
            as? android.view.inputmethod.InputMethodManager
        view.requestFocus()
        imm?.showSoftInput(view, android.view.inputmethod.InputMethodManager.SHOW_IMPLICIT)
    }

    override fun shouldBackButtonBeMappedToEscape(): Boolean = false
    override fun shouldEnforceCharBasedInput(): Boolean = true
    override fun shouldUseCtrlSpaceWorkaround(): Boolean = false
    override fun isTerminalViewSelected(): Boolean = true

    override fun copyModeChanged(copyMode: Boolean) {}

    override fun onKeyDown(keyCode: Int, e: KeyEvent, s: TerminalSession): Boolean = false
    override fun onKeyUp(keyCode: Int, e: KeyEvent): Boolean = false
    override fun onLongPress(event: MotionEvent): Boolean = false

    override fun readControlKey(): Boolean = false
    override fun readAltKey(): Boolean = false
    override fun readShiftKey(): Boolean = false
    override fun readFnKey(): Boolean = false

    override fun onCodePoint(codePoint: Int, ctrlDown: Boolean, s: TerminalSession): Boolean = false

    override fun onEmulatorSet() {}

    override fun logError(tag: String, message: String) { android.util.Log.e(tag, message) }
    override fun logWarn(tag: String, message: String) { android.util.Log.w(tag, message) }
    override fun logInfo(tag: String, message: String) { android.util.Log.i(tag, message) }
    override fun logDebug(tag: String, message: String) { android.util.Log.d(tag, message) }
    override fun logVerbose(tag: String, message: String) { android.util.Log.v(tag, message) }
    override fun logStackTraceWithMessage(tag: String, message: String, e: Exception) {
        android.util.Log.e(tag, message, e)
    }
    override fun logStackTrace(tag: String, e: Exception) { android.util.Log.e(tag, "", e) }
}

/**
 * A session client that forwards onTextChanged → view.onScreenUpdated()
 * so the AndroidView actually invalidates and repaints. Without this the
 * emulator parses bytes happily but nothing visible ever changes.
 */
private class ViewBackedSessionClient : TerminalSessionClient {
    @Volatile var view: TerminalView? = null

    override fun onTextChanged(s: TerminalSession) {
        view?.onScreenUpdated()
    }
    override fun onTitleChanged(s: TerminalSession) {}
    override fun onSessionFinished(s: TerminalSession) {}
    override fun onCopyTextToClipboard(s: TerminalSession, text: String) {}
    override fun onPasteTextFromClipboard(s: TerminalSession?) {}
    override fun onBell(s: TerminalSession) {}
    override fun onColorsChanged(s: TerminalSession) {
        view?.onScreenUpdated()
    }
    override fun onTerminalCursorStateChange(state: Boolean) {}
    override fun setTerminalShellPid(s: TerminalSession, pid: Int) {}
    override fun getTerminalCursorStyle(): Int? = null

    override fun logError(tag: String, message: String) { android.util.Log.e(tag, message) }
    override fun logWarn(tag: String, message: String) { android.util.Log.w(tag, message) }
    override fun logInfo(tag: String, message: String) { android.util.Log.i(tag, message) }
    override fun logDebug(tag: String, message: String) { android.util.Log.d(tag, message) }
    override fun logVerbose(tag: String, message: String) { android.util.Log.v(tag, message) }
    override fun logStackTraceWithMessage(tag: String, message: String, e: Exception) {
        android.util.Log.e(tag, message, e)
    }
    override fun logStackTrace(tag: String, e: Exception) { android.util.Log.e(tag, "", e) }
}
