package io.sshtogo.android.terminal

import android.graphics.Typeface
import android.view.KeyEvent
import android.view.MotionEvent
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
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
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.input.pointer.PointerEventPass
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.input.pointer.positionChange
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import kotlin.math.abs
import kotlinx.coroutines.launch
import com.termux.terminal.TerminalSession
import com.termux.terminal.TerminalSessionClient
import com.termux.view.TerminalView
import com.termux.view.TerminalViewClient
import io.sshtogo.android.SshToGoApplication
import io.sshtogo.android.data.AppPreferences

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

    // Persistent per-session terminals: each session is created once and kept
    // alive (its WebSocket + emulator + scrollback) for the life of this screen,
    // cached by name. Swiping reuses the live session — no reconnect, no relay
    // replay — and sessions connect lazily the first time you swipe to one.
    val sessions = remember(profile.id, hostName) { mutableMapOf<String, SessionHolder>() }
    fun holderFor(name: String): SessionHolder = sessions.getOrPut(name) {
        val client = ViewBackedSessionClient()
        SessionHolder(RelayTerminalSession(profile, hostName, name, client), client)
    }
    DisposableEffect(profile.id, hostName) {
        onDispose { sessions.values.forEach { it.session.finishIfRunning() } }
    }

    // Endless wrap-around carousel: a huge virtual page count makes swiping past
    // the last session loop to the first with the same one-page slide animation
    // (the real session for a page is page % count). beyondViewportPageCount=0
    // keeps only the current + transitioning page composed, and adjacent virtual
    // pages are always distinct sessions, so a session is never hosted twice.
    val initialIndex = list.indexOf(sessionName).let { if (it < 0) 0 else it }
    val pageCount = list.size
    val loop = pageCount > 1
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
        val scope = rememberCoroutineScope()
        val switchThresholdPx = with(LocalDensity.current) { 56.dp.toPx() }

        // The pager is driven programmatically; we read touches ourselves on the
        // INITIAL pass (before the terminal view) and only claim a gesture once
        // it's clearly horizontal — biased toward vertical so terminal scrollback
        // stays smooth. Horizontal gestures get consumed (so the terminal ignores
        // them) and a past-threshold swipe flips to the next/previous session;
        // anything else is left untouched for the terminal to scroll.
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(pad)
                .imePadding()
                .then(
                    if (!loop) Modifier else Modifier.pointerInput(pageCount) {
                        awaitEachGesture {
                            val down = awaitFirstDown(requireUnconsumed = false, pass = PointerEventPass.Initial)
                            var dx = 0f
                            var dy = 0f
                            var axis = 0 // 0 = undecided, 1 = horizontal (ours), -1 = vertical (terminal's)
                            while (true) {
                                val event = awaitPointerEvent(PointerEventPass.Initial)
                                val change = event.changes.firstOrNull { it.id == down.id } ?: break
                                if (!change.pressed) break
                                val d = change.positionChange()
                                dx += d.x
                                dy += d.y
                                if (axis == 0 && (abs(dx) > viewConfiguration.touchSlop || abs(dy) > viewConfiguration.touchSlop)) {
                                    // Require clear horizontal dominance (1.25×) so
                                    // near-vertical and diagonal drags scroll the terminal.
                                    axis = if (abs(dx) > abs(dy) * 1.25f) 1 else -1
                                }
                                if (axis == 1) change.consume() else if (axis == -1) break
                            }
                            if (axis == 1 && abs(dx) > switchThresholdPx) {
                                // One virtual page either way; the modulo mapping makes
                                // the wrap (last→first) just another adjacent slide.
                                val target = pagerState.currentPage + if (dx < 0) 1 else -1
                                scope.launch { pagerState.animateScrollToPage(target) }
                            }
                        }
                    },
                ),
        ) {
            HorizontalPager(
                state = pagerState,
                modifier = Modifier.fillMaxSize(),
                // Gestures are handled by the axis-aware detector above, so the
                // pager's own (greedy) drag handling stays off.
                userScrollEnabled = false,
                // Only the current + transitioning page is composed; persistence
                // comes from the session cache above (not from keeping pages
                // alive), so wrap-around stays a smooth one-page slide.
                beyondViewportPageCount = 0,
            ) { pageIndex ->
                val name = list[sessionIndexFor(pageIndex)]
                SessionTerminal(holder = holderFor(name), hostName = hostName, sessionName = name)
            }
        }
    }
}

/**
 * Renders one tmux session from the screen-level cache. The session (its relay
 * WebSocket + emulator + scrollback) is owned by [holder] and outlives this
 * composable, so swiping away and back re-attaches a fresh TerminalView to the
 * still-connected session — TerminalSession.updateSize just resizes the existing
 * emulator, so the buffer re-renders instantly with no reconnect or replay.
 */
@Composable
private fun SessionTerminal(holder: SessionHolder, hostName: String, sessionName: String) {
    val session = holder.session
    val sessionClient = holder.client
    // Read the per-session palette name from the state-backed prefs map so
    // a change in the palette picker recomposes this view (and we'll push
    // the new colours into the live emulator via the update block below).
    val paletteName = SshToGoApplication.instance.prefs.paletteFor(hostName, sessionName)
    val palette = remember(paletteName) { TerminalPalette.fromName(paletteName) }

    // Detach the view on dispose but keep the session alive — it's owned by the
    // screen-level cache and finished only when the whole screen closes.
    DisposableEffect(session) {
        onDispose { sessionClient.view = null }
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
                    // Use the palette's own background so the whole terminal area
                    // (incl. padding) matches the theme — important for light/white
                    // and gray palettes, which otherwise show the app surface colour.
                    setBackgroundColor(palette.backgroundColor)
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
                // copies the new defaults, update the view background, and redraw.
                palette.apply()
                view.setBackgroundColor(palette.backgroundColor)
                view.currentSession?.emulator?.let { em ->
                    em.mColors.reset()
                    view.onScreenUpdated()
                }
            },
        )
    }
}

/**
 * Screen-scoped owner of a session's relay + client so it survives the pager
 * disposing and recreating its page during swipes.
 */
private class SessionHolder(
    val session: RelayTerminalSession,
    val client: ViewBackedSessionClient,
)

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
