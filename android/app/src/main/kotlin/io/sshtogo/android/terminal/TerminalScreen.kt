package io.sshtogo.android.terminal

import android.graphics.Typeface
import android.view.KeyEvent
import android.view.MotionEvent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.viewinterop.AndroidView
import com.termux.terminal.TerminalSession
import com.termux.terminal.TerminalSessionClient
import com.termux.view.TerminalView
import com.termux.view.TerminalViewClient
import io.sshtogo.android.SshToGoApplication
import io.sshtogo.android.data.AppPreferences
import io.sshtogo.android.data.ServerProfile
import io.sshtogo.android.net.SshToGoClient

@OptIn(ExperimentalMaterial3Api::class)
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

    // Fetch the host's full session list so we can offer horizontal-swipe
    // navigation between siblings. Falls back to the one tapped if the
    // API call fails or returns empty.
    var sessions by remember(profile.id, hostName) { mutableStateOf<List<String>?>(null) }
    LaunchedEffect(profile.id, hostName) {
        sessions = try {
            val api = SshToGoClient.forProfile(profile)
            api.sessions()
                .filter { it.hostName == hostName }
                .map { it.session.name }
                .ifEmpty { listOf(sessionName) }
        } catch (_: Throwable) {
            listOf(sessionName)
        }
    }

    val list = sessions
    if (list == null) {
        Scaffold(topBar = { TopAppBar(title = { Text("$sessionName @ $hostName") }) }) { pad ->
            Box(modifier = Modifier.fillMaxSize().padding(pad), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
        }
        return
    }

    val initialIndex = list.indexOf(sessionName).let { if (it < 0) 0 else it }
    val pagerState = rememberPagerState(initialPage = initialIndex) { list.size }
    val currentSession = list.getOrNull(pagerState.currentPage) ?: sessionName

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text(
                            "$currentSession @ $hostName",
                            style = MaterialTheme.typography.titleMedium,
                        )
                        if (list.size > 1) {
                            Text(
                                "${pagerState.currentPage + 1} / ${list.size}  ·  swipe for next",
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
            )
        },
    ) { pad ->
        HorizontalPager(
            state = pagerState,
            modifier = Modifier.fillMaxSize().padding(pad).imePadding(),
            // Keep adjacent pages composed so swiping reveals already-attached
            // terminals instead of a flash-of-loading. Costs one extra
            // WebSocket per neighbour, which is fine for a handful of sessions.
            beyondViewportPageCount = 1,
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

    DisposableEffect(session) {
        onDispose {
            sessionClient.view = null
            session.finishIfRunning()
        }
    }

    Box(modifier = Modifier.fillMaxSize()) {
        AndroidView(
            modifier = Modifier.fillMaxSize(),
            factory = { context ->
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
