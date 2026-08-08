package io.sshtogo.android.terminal

import android.os.Handler
import android.os.Looper
import com.termux.terminal.TerminalSession
import com.termux.terminal.TerminalSessionClient
import io.sshtogo.android.data.ServerProfile
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import okio.ByteString
import org.json.JSONObject
import java.util.concurrent.TimeUnit

/**
 * A TerminalSession that pipes the emulator's I/O through ssh-to-go's
 * /ws/{host}/{session} relay instead of a local PTY.
 *
 * Wire protocol:
 *  - Binary frames are raw terminal bytes (server → client = tmux output,
 *    client → server = user input).
 *  - Text frames are JSON control messages:
 *      {"type":"tty",  "tty":"/dev/pts/N"}  (informational; sent on attach)
 *      {"type":"kicked"}                    (another client took over)
 *      {"type":"resize","cols":C,"rows":R}  (sent by us on size change)
 *  - mouse=off disables tmux mouse mode so swipes aren't forwarded into
 *    copy-mode (smooth scroll comes from the view layer in Phase 3).
 */
class RelayTerminalSession(
    private val profile: ServerProfile,
    private val hostName: String,
    private val sessionName: String,
    client: TerminalSessionClient,
) : TerminalSession(null, client) {

    private companion object {
        /**
         * Terminal-report replies the termux emulator auto-generates (never
         * typed by a user): primary/secondary device attributes (CSI ? … c,
         * CSI > … c), cursor-position reports incl. DECXCPR (CSI [?] r;c[;p] R
         * — digits required, so the bare F-key shape ESC[R passes through),
         * and DSR-ok (CSI 0 n). Kept in sync with reportReplyRegex in
         * web/static/js/terminal.js.
         */
        val REPORT_REPLY = Regex(
            "\u001B\\[\\?[0-9;]*c|\u001B\\[>[0-9;]*c|\u001B\\[\\??[0-9]+;[0-9]+(?:;[0-9]+)?R|\u001B\\[0n"
        )
    }

    private val main = Handler(Looper.getMainLooper())

    private val http: OkHttpClient = OkHttpClient.Builder()
        .pingInterval(20, TimeUnit.SECONDS)
        .readTimeout(0, TimeUnit.SECONDS) // no read timeout for long-lived ws
        .build()

    @Volatile
    private var ws: WebSocket? = null

    @Volatile
    private var hostTty: String? = null

    @Volatile
    private var lastCols: Int = 0

    @Volatile
    private var lastRows: Int = 0

    /** Set when finishIfRunning() is called — no reconnect after a user-initiated close. */
    @Volatile
    private var userClosed = false

    /** Number of consecutive failed reconnects; resets on a healthy onOpen. */
    @Volatile
    private var retryCount = 0

    /** Pending reconnect timer (so we can cancel it on user-initiated close). */
    @Volatile
    private var reconnectRunnable: Runnable? = null

    /**
     * True once the emulator has initialised and made its first connect().
     * Gates lifecycle-driven resume() so a spurious ON_RESUME during initial
     * layout (which fires before initializeEmulator) can't open a second socket.
     */
    @Volatile
    private var emulatorReady = false

    fun ttyPath(): String? = hostTty

    override fun initializeEmulator(columns: Int, rows: Int, cellWidthPixels: Int, cellHeightPixels: Int) {
        super.initializeEmulator(columns, rows, cellWidthPixels, cellHeightPixels)
        connect()
        emulatorReady = true
    }

    override fun onSizeChanged(columns: Int, rows: Int, cellWidthPixels: Int, cellHeightPixels: Int) {
        lastCols = columns
        lastRows = rows
        sendResize(columns, rows)
    }

    override fun write(data: ByteArray, offset: Int, count: Int) {
        // Skip mTerminalToProcessIOQueue entirely — send straight through the socket.
        val socket = ws ?: return
        var slice = if (offset == 0 && count == data.size) data else data.copyOfRange(offset, offset + count)
        // Strip terminal-report auto-replies the emulator generates when a query
        // reaches it — DA1/DA2 "who are you", DSR status, CPR cursor position
        // (plain and DECXCPR forms). Mirrors reportReplyRegex in
        // web/static/js/terminal.js (#79): no keyboard produces these byte
        // shapes, and forwarding them is the one injection path that's ours —
        // at the tmux end an unexpected reply gets typed into the pane as
        // garbage like "?61;4;...52c". ISO-8859-1 round-trips bytes losslessly.
        if (slice.contains(0x1b.toByte())) {
            val s = String(slice, Charsets.ISO_8859_1)
            val filtered = REPORT_REPLY.replace(s, "")
            if (filtered.length != s.length) {
                if (filtered.isEmpty()) return
                slice = filtered.toByteArray(Charsets.ISO_8859_1)
            }
        }
        socket.send(ByteString.of(*slice))
    }

    override fun finishIfRunning() {
        userClosed = true
        reconnectRunnable?.let { main.removeCallbacks(it) }
        reconnectRunnable = null
        ws?.close(1000, "client closed")
        ws = null
    }

    /**
     * Called when the screen turns off / the app is backgrounded. Closes the
     * relay socket cleanly (so the server detaches this client) and cancels any
     * pending reconnect. NOT a user-close — [resume] will reconnect. Nulling
     * [ws] first marks the old socket stale so its onClosed/onFailure are ignored.
     */
    fun pause() {
        if (userClosed) return
        reconnectRunnable?.let { main.removeCallbacks(it) }
        reconnectRunnable = null
        val old = ws
        ws = null
        old?.close(1000, "app backgrounded")
    }

    /**
     * Called when the app returns to the foreground (screen wakes) and when
     * this session's page becomes the visible one in the swipe carousel. The
     * OS tears WebSockets down while backgrounded and the failure often isn't
     * observed until pings resume — leaving the page dead with no reconnect in
     * flight. Reconnecting here makes wake-up recovery immediate, and the
     * page-change call guarantees a swipe never lands on a dead socket waiting
     * out a reconnect backoff. No-op if already connected, user-closed, or
     * before the first connect.
     */
    fun resume() {
        if (userClosed || !emulatorReady || ws != null) return
        reconnectRunnable?.let { main.removeCallbacks(it) }
        reconnectRunnable = null
        retryCount = 0
        connect()
    }

    /**
     * Reconnect with exponential backoff after an unexpected disconnect.
     * Caps at ~30s. Cancelled by finishIfRunning().
     */
    private fun scheduleReconnect() {
        if (userClosed) return
        val attempt = retryCount.coerceAtMost(5)
        val delayMs = (1000L shl attempt).coerceAtMost(30_000L) // 1, 2, 4, 8, 16, 30, 30, …
        retryCount++
        val notice = "\r\n[reconnecting in ${delayMs / 1000}s …]".toByteArray()
        emulatorAppend(notice, notice.size)
        val r = Runnable {
            if (userClosed) return@Runnable
            val msg = "\r\n[reconnecting…]".toByteArray()
            emulatorAppend(msg, msg.size)
            connect()
        }
        reconnectRunnable = r
        main.postDelayed(r, delayMs)
    }

    private fun connect() {
        val base = profile.baseUrl.trimEnd('/')
        val wsUrl = base
            .replaceFirst(Regex("^https://"), "wss://")
            .replaceFirst(Regex("^http://"), "ws://") +
            "/ws/" + encode(hostName) + "/" + encode(sessionName) + "?mouse=off"

        val req = Request.Builder()
            .url(wsUrl)
            .addHeader("Authorization", "Bearer ${profile.token}")
            .build()

        ws = http.newWebSocket(req, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                main.post { retryCount = 0 }
                if (lastCols > 0 && lastRows > 0) sendResize(lastCols, lastRows)
            }

            override fun onMessage(webSocket: WebSocket, bytes: ByteString) {
                val bytesArr = bytes.toByteArray()
                main.post { emulatorAppend(bytesArr, bytesArr.size) }
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                try {
                    val msg = JSONObject(text)
                    when (msg.optString("type")) {
                        "tty" -> hostTty = msg.optString("tty").takeIf { it.isNotBlank() }
                        "kicked" -> {
                            val notice = "\r\n[detached by another client]".toByteArray()
                            main.post { emulatorAppend(notice, notice.size) }
                        }
                    }
                } catch (_: Throwable) {
                    // Ignore malformed control frames.
                }
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                main.post {
                    if (webSocket !== ws) return@post // a socket we've already replaced (pause/reconnect)
                    ws = null // dead — lets resume() reconnect immediately instead of early-returning
                    val notice = "\r\n[connection closed${if (reason.isNotBlank()) " — $reason" else ""}]".toByteArray()
                    emulatorAppend(notice, notice.size)
                    if (!userClosed && code != 1000) scheduleReconnect()
                }
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                main.post {
                    if (webSocket !== ws) return@post // a socket we've already replaced (pause/reconnect)
                    ws = null // dead — lets resume() reconnect immediately instead of early-returning
                    val notice = "\r\n[connection error: ${t.message ?: t.javaClass.simpleName}]".toByteArray()
                    emulatorAppend(notice, notice.size)
                    if (!userClosed) scheduleReconnect()
                }
            }
        })
    }

    private fun sendResize(cols: Int, rows: Int) {
        if (cols <= 0 || rows <= 0) return
        val socket = ws ?: return
        val payload = JSONObject().apply {
            put("type", "resize")
            put("cols", cols)
            put("rows", rows)
        }.toString()
        socket.send(payload)
    }

    private fun encode(s: String): String = java.net.URLEncoder.encode(s, "UTF-8")
}
