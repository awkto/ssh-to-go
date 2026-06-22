package io.sshtogo.android.terminal

import android.graphics.Typeface
import android.view.KeyEvent
import android.view.MotionEvent
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.layout.Arrangement
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
import androidx.compose.material.icons.filled.ChevronLeft
import androidx.compose.material.icons.filled.ChevronRight
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
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import kotlinx.coroutines.launch
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

    val initialIndex = list.indexOf(sessionName).let { if (it < 0) 0 else it }
    val pagerState = rememberPagerState(initialPage = initialIndex) { list.size }
    val currentSession = list.getOrNull(pagerState.currentPage) ?: sessionName
    val scope = rememberCoroutineScope()
    val swipeThresholdPx = with(LocalDensity.current) { 56.dp.toPx() }

    fun goPrev() {
        if (pagerState.currentPage > 0) {
            scope.launch { pagerState.animateScrollToPage(pagerState.currentPage - 1) }
        }
    }
    fun goNext() {
        if (pagerState.currentPage < list.size - 1) {
            scope.launch { pagerState.animateScrollToPage(pagerState.currentPage + 1) }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    // Swipe the title horizontally to switch sessions. The
                    // terminal body keeps all vertical+horizontal gestures
                    // for its own scrolling / selection.
                    var dragAccum = 0f
                    Box(
                        modifier = Modifier
                            .pointerInput(list.size) {
                                detectHorizontalDragGestures(
                                    onDragStart = { dragAccum = 0f },
                                    onDragEnd = {
                                        if (dragAccum > swipeThresholdPx) goPrev()
                                        else if (dragAccum < -swipeThresholdPx) goNext()
                                        dragAccum = 0f
                                    },
                                    onDragCancel = { dragAccum = 0f },
                                    onHorizontalDrag = { change, dragAmount ->
                                        dragAccum += dragAmount
                                        change.consume()
                                    },
                                )
                            },
                    ) {
                        Column {
                            Text(
                                "$currentSession @ $hostName",
                                style = MaterialTheme.typography.titleMedium,
                            )
                            if (list.size > 1) {
                                Text(
                                    "${pagerState.currentPage + 1} / ${list.size}  ·  swipe title or use arrows",
                                    style = MaterialTheme.typography.labelSmall,
                                )
                            }
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

                    // Per-session palette control, LEFT of the arrows:
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

                    if (list.size > 1) {
                        IconButton(onClick = ::goPrev, enabled = pagerState.currentPage > 0) {
                            Icon(Icons.Default.ChevronLeft, contentDescription = "Previous session")
                        }
                        IconButton(onClick = ::goNext, enabled = pagerState.currentPage < list.size - 1) {
                            Icon(Icons.Default.ChevronRight, contentDescription = "Next session")
                        }
                    }

                    // Close THIS session in the app (RIGHT of the arrows). Drops it
                    // from the swipe carousel and returns to the list — the tmux
                    // session keeps running on the server (no kill is sent).
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
            // Pager's own swipe gestures fight the terminal's vertical drag
            // (it greedy-grabs both axes), so we disable them and drive page
            // changes from the title/arrow controls.
            userScrollEnabled = false,
            // Only the currently visible page keeps a live WebSocket +
            // emulator. Pre-warming neighbours made the app prone to freezes
            // on hosts with many sessions; the small ~1s reconnect on swipe
            // is a worthwhile trade for stability.
            beyondViewportPageCount = 0,
        ) { pageIndex ->
            SessionTerminal(profile = profile, hostName = hostName, sessionName = list[pageIndex])
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
