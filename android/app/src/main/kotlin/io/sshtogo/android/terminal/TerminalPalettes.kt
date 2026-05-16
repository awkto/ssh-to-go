package io.sshtogo.android.terminal

import com.termux.terminal.TextStyle

/**
 * Built-in colour palettes for the terminal. Each palette overrides the
 * 16 ANSI colours plus foreground/background/cursor. Apply via
 * {@link TerminalPalette#apply()} before opening a new TerminalView.
 */
enum class TerminalPalette(
    val displayName: String,
    private val background: Int,
    private val foreground: Int,
    private val cursor: Int,
    private val ansi16: IntArray,
) {
    DEFAULT(
        "Termux default",
        background = 0xff000000.toInt(),
        foreground = 0xffffffff.toInt(),
        cursor = 0xffffffff.toInt(),
        ansi16 = intArrayOf(
            0xff000000.toInt(), 0xffcd0000.toInt(), 0xff00cd00.toInt(), 0xffcdcd00.toInt(),
            0xff6495ed.toInt(), 0xffcd00cd.toInt(), 0xff00cdcd.toInt(), 0xffe5e5e5.toInt(),
            0xff7f7f7f.toInt(), 0xffff0000.toInt(), 0xff00ff00.toInt(), 0xffffff00.toInt(),
            0xff5c5cff.toInt(), 0xffff00ff.toInt(), 0xff00ffff.toInt(), 0xffffffff.toInt(),
        ),
    ),
    SOLARIZED_DARK(
        "Solarized dark",
        background = 0xff002b36.toInt(),
        foreground = 0xff839496.toInt(),
        cursor = 0xff93a1a1.toInt(),
        ansi16 = intArrayOf(
            0xff073642.toInt(), 0xffdc322f.toInt(), 0xff859900.toInt(), 0xffb58900.toInt(),
            0xff268bd2.toInt(), 0xffd33682.toInt(), 0xff2aa198.toInt(), 0xffeee8d5.toInt(),
            0xff002b36.toInt(), 0xffcb4b16.toInt(), 0xff586e75.toInt(), 0xff657b83.toInt(),
            0xff839496.toInt(), 0xff6c71c4.toInt(), 0xff93a1a1.toInt(), 0xfffdf6e3.toInt(),
        ),
    ),
    SOLARIZED_LIGHT(
        "Solarized light",
        background = 0xfffdf6e3.toInt(),
        foreground = 0xff657b83.toInt(),
        cursor = 0xff586e75.toInt(),
        ansi16 = intArrayOf(
            0xffeee8d5.toInt(), 0xffdc322f.toInt(), 0xff859900.toInt(), 0xffb58900.toInt(),
            0xff268bd2.toInt(), 0xffd33682.toInt(), 0xff2aa198.toInt(), 0xff073642.toInt(),
            0xfffdf6e3.toInt(), 0xffcb4b16.toInt(), 0xff93a1a1.toInt(), 0xff839496.toInt(),
            0xff657b83.toInt(), 0xff6c71c4.toInt(), 0xff586e75.toInt(), 0xff002b36.toInt(),
        ),
    ),
    DRACULA(
        "Dracula",
        background = 0xff282a36.toInt(),
        foreground = 0xfff8f8f2.toInt(),
        cursor = 0xfff8f8f0.toInt(),
        ansi16 = intArrayOf(
            0xff000000.toInt(), 0xffff5555.toInt(), 0xff50fa7b.toInt(), 0xfff1fa8c.toInt(),
            0xffbd93f9.toInt(), 0xffff79c6.toInt(), 0xff8be9fd.toInt(), 0xffbfbfbf.toInt(),
            0xff4d4d4d.toInt(), 0xffff6e67.toInt(), 0xff5af78e.toInt(), 0xfff4f99d.toInt(),
            0xffcaa9fa.toInt(), 0xffff92d0.toInt(), 0xff9aedfe.toInt(), 0xffe6e6e6.toInt(),
        ),
    ),
    NORD(
        "Nord",
        background = 0xff2e3440.toInt(),
        foreground = 0xffd8dee9.toInt(),
        cursor = 0xffd8dee9.toInt(),
        ansi16 = intArrayOf(
            0xff3b4252.toInt(), 0xffbf616a.toInt(), 0xffa3be8c.toInt(), 0xffebcb8b.toInt(),
            0xff81a1c1.toInt(), 0xffb48ead.toInt(), 0xff88c0d0.toInt(), 0xffe5e9f0.toInt(),
            0xff4c566a.toInt(), 0xffbf616a.toInt(), 0xffa3be8c.toInt(), 0xffebcb8b.toInt(),
            0xff81a1c1.toInt(), 0xffb48ead.toInt(), 0xff8fbcbb.toInt(), 0xffeceff4.toInt(),
        ),
    );

    /**
     * Apply this palette to the global [com.termux.terminal.TerminalColors.COLOR_SCHEME].
     * Any TerminalEmulator created AFTER this returns will use these colours.
     * Existing emulators keep their current palette until their mColors.reset() is called.
     */
    fun apply() {
        val defaults = com.termux.terminal.TerminalColors.COLOR_SCHEME.mDefaultColors
        for (i in 0 until 16) defaults[i] = ansi16[i]
        defaults[TextStyle.COLOR_INDEX_FOREGROUND] = foreground
        defaults[TextStyle.COLOR_INDEX_BACKGROUND] = background
        defaults[TextStyle.COLOR_INDEX_CURSOR] = cursor
    }

    companion object {
        fun fromName(name: String?): TerminalPalette =
            entries.firstOrNull { it.name == name } ?: DEFAULT
    }
}
