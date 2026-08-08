package io.sshtogo.android.data

import android.content.Context
import android.content.SharedPreferences
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

enum class ThemeMode { SYSTEM, DARK, LIGHT }

/**
 * Non-sensitive device-wide preferences. Server profile tokens stay in
 * EncryptedSharedPreferences (ServerProfileStore); everything that's just
 * UX state lives here in plain SharedPreferences so it survives backups
 * and doesn't depend on a master key.
 *
 * State-observed values (themeMode, terminalPaletteName) are backed by
 * Compose mutableStateOf so callers that read them in a Composable
 * automatically recompose when they're updated.
 */
class AppPreferences(context: Context) {
    private val prefs: SharedPreferences =
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    /** App theme: follow system, force dark, or force light. Recomposes on change. */
    private val themeModeState = mutableStateOf(loadThemeMode())
    var themeMode: ThemeMode
        get() = themeModeState.value
        set(value) {
            themeModeState.value = value
            prefs.edit().putString(KEY_THEME_MODE, value.name).apply()
        }

    private fun loadThemeMode(): ThemeMode =
        runCatching { ThemeMode.valueOf(prefs.getString(KEY_THEME_MODE, "SYSTEM") ?: "SYSTEM") }
            .getOrDefault(ThemeMode.SYSTEM)

    /**
     * Sort dashboard sessions most-recently-used first (default on, matching
     * the desktop UI) instead of the server's natural order. State-backed so
     * toggling it recomposes the dashboard list.
     */
    private val sortByRecentState = mutableStateOf(prefs.getBoolean(KEY_SORT_BY_RECENT, true))
    var sortSessionsByRecent: Boolean
        get() = sortByRecentState.value
        set(value) {
            sortByRecentState.value = value
            prefs.edit().putBoolean(KEY_SORT_BY_RECENT, value).apply()
        }

    /**
     * New Session form: last-used values, persisted so a repeat create is
     * just tap-+, tap-Create — the mobile analog of the web form's
     * localStorage defaults and stogo's ~/.config defaults. The directory
     * and command are stored AS TYPED ($name and all): the template is the
     * useful default, its expansion is one session's worth.
     * An empty newSessionDir means "never set" — fall back to the server's
     * new_session_dir setting.
     */
    var newSessionDir: String
        get() = prefs.getString(KEY_NS_DIR, "") ?: ""
        set(value) { prefs.edit().putString(KEY_NS_DIR, value).apply() }

    var newSessionCommand: String
        get() = prefs.getString(KEY_NS_COMMAND, "") ?: ""
        set(value) { prefs.edit().putString(KEY_NS_COMMAND, value).apply() }

    var newSessionCreateDir: Boolean
        get() = prefs.getBoolean(KEY_NS_CREATE_DIR, true)
        set(value) { prefs.edit().putBoolean(KEY_NS_CREATE_DIR, value).apply() }

    /** Terminal font size in sp. Default 14, range [TerminalFontSizeMinSp, TerminalFontSizeMaxSp]. */
    var terminalFontSizeSp: Float
        get() = prefs.getFloat(KEY_FONT_SIZE_SP, DEFAULT_FONT_SIZE_SP)
            .coerceIn(TerminalFontSizeMinSp, TerminalFontSizeMaxSp)
        set(value) {
            prefs.edit()
                .putFloat(KEY_FONT_SIZE_SP, value.coerceIn(TerminalFontSizeMinSp, TerminalFontSizeMaxSp))
                .apply()
        }

    /**
     * Per-session terminal palette. Keyed by (host, sessionName) so each
     * tmux session keeps its own colour scheme.
     *
     * The internal map is state-backed so picker UIs reading it recompose
     * when palettes are reassigned.
     */
    private val paletteMap: MutableMap<String, String> = mutableStateMapOf<String, String>().apply {
        // Load any previously-persisted palette assignments from prefs.
        prefs.all.forEach { (k, v) ->
            if (k.startsWith(KEY_PALETTE_PREFIX) && v is String) {
                put(k.removePrefix(KEY_PALETTE_PREFIX), v)
            }
        }
    }

    fun paletteFor(host: String, session: String): String =
        paletteMap[paletteKey(host, session)] ?: DEFAULT_PALETTE_NAME

    fun setPaletteFor(host: String, session: String, name: String) {
        val k = paletteKey(host, session)
        paletteMap[k] = name
        prefs.edit().putString(KEY_PALETTE_PREFIX + k, name).apply()
    }

    private fun paletteKey(host: String, session: String): String =
        host + "\u0000" + session

    companion object {
        private const val PREFS_NAME = "ssh-to-go-app"
        private const val KEY_FONT_SIZE_SP = "terminal_font_size_sp"
        private const val KEY_PALETTE_PREFIX = "palette."
        private const val KEY_THEME_MODE = "app_theme_mode"
        private const val KEY_SORT_BY_RECENT = "sort_sessions_by_recent"
        private const val KEY_NS_DIR = "new_session_dir"
        private const val KEY_NS_COMMAND = "new_session_command"
        private const val KEY_NS_CREATE_DIR = "new_session_create_dir"
        const val DEFAULT_FONT_SIZE_SP = 14f
        const val DEFAULT_PALETTE_NAME = "DEFAULT"
        const val TerminalFontSizeMinSp = 8f
        const val TerminalFontSizeMaxSp = 28f
    }
}
