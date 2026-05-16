package io.sshtogo.android.data

import android.content.Context
import android.content.SharedPreferences

/**
 * Non-sensitive device-wide preferences. Server profile tokens stay in
 * EncryptedSharedPreferences (ServerProfileStore); everything that's just
 * UX state lives here in plain SharedPreferences so it survives backups
 * and doesn't depend on a master key.
 */
class AppPreferences(context: Context) {
    private val prefs: SharedPreferences =
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    /** Terminal font size in sp. Default 14, range [TerminalFontSizeMinSp, TerminalFontSizeMaxSp]. */
    var terminalFontSizeSp: Float
        get() = prefs.getFloat(KEY_FONT_SIZE_SP, DEFAULT_FONT_SIZE_SP)
            .coerceIn(TerminalFontSizeMinSp, TerminalFontSizeMaxSp)
        set(value) {
            prefs.edit()
                .putFloat(KEY_FONT_SIZE_SP, value.coerceIn(TerminalFontSizeMinSp, TerminalFontSizeMaxSp))
                .apply()
        }

    /** Name of the active terminal palette. Maps to a TerminalPalette enum value. */
    var terminalPaletteName: String
        get() = prefs.getString(KEY_PALETTE, DEFAULT_PALETTE_NAME) ?: DEFAULT_PALETTE_NAME
        set(value) { prefs.edit().putString(KEY_PALETTE, value).apply() }

    companion object {
        private const val PREFS_NAME = "ssh-to-go-app"
        private const val KEY_FONT_SIZE_SP = "terminal_font_size_sp"
        private const val KEY_PALETTE = "terminal_palette"
        const val DEFAULT_FONT_SIZE_SP = 14f
        const val DEFAULT_PALETTE_NAME = "DEFAULT"
        const val TerminalFontSizeMinSp = 8f
        const val TerminalFontSizeMaxSp = 28f
    }
}
