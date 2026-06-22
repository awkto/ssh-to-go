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
    ),
    GRAPHITE(
        "Graphite gray",
        background = 0xff303030.toInt(),
        foreground = 0xffd0d0d0.toInt(),
        cursor = 0xffd0d0d0.toInt(),
        ansi16 = intArrayOf(
            0xff303030.toInt(), 0xffd75f5f.toInt(), 0xff87af87.toInt(), 0xffd7af87.toInt(),
            0xff87afd7.toInt(), 0xffaf87d7.toInt(), 0xff87d7d7.toInt(), 0xffc0c0c0.toInt(),
            0xff5f5f5f.toInt(), 0xffff8787.toInt(), 0xffafd7af.toInt(), 0xffffd7af.toInt(),
            0xffafd7ff.toInt(), 0xffd7afff.toInt(), 0xffafffff.toInt(), 0xffffffff.toInt(),
        ),
    ),
    SLATE(
        "Slate gray",
        background = 0xff22272e.toInt(),
        foreground = 0xffadbac7.toInt(),
        cursor = 0xff539bf5.toInt(),
        ansi16 = intArrayOf(
            0xff2d333b.toInt(), 0xfff47067.toInt(), 0xff57ab5a.toInt(), 0xffc69026.toInt(),
            0xff539bf5.toInt(), 0xffb083f0.toInt(), 0xff39c5cf.toInt(), 0xffadbac7.toInt(),
            0xff636e7b.toInt(), 0xffff938a.toInt(), 0xff6bc46d.toInt(), 0xffdaaa3f.toInt(),
            0xff6cb6ff.toInt(), 0xffdcbdfb.toInt(), 0xff56d4dd.toInt(), 0xffcdd9e5.toInt(),
        ),
    ),
    GRUVBOX_DARK(
        "Gruvbox dark",
        background = 0xff282828.toInt(),
        foreground = 0xffebdbb2.toInt(),
        cursor = 0xffebdbb2.toInt(),
        ansi16 = intArrayOf(
            0xff282828.toInt(), 0xffcc241d.toInt(), 0xff98971a.toInt(), 0xffd79921.toInt(),
            0xff458588.toInt(), 0xffb16286.toInt(), 0xff689d6a.toInt(), 0xffa89984.toInt(),
            0xff928374.toInt(), 0xfffb4934.toInt(), 0xffb8bb26.toInt(), 0xfffabd2f.toInt(),
            0xff83a598.toInt(), 0xffd3869b.toInt(), 0xff8ec07c.toInt(), 0xffebdbb2.toInt(),
        ),
    ),
    SOLAR_FLARE(
        "Solar flare",
        background = 0xff1c1611.toInt(),
        foreground = 0xffe8d3a1.toInt(),
        cursor = 0xffffcc66.toInt(),
        ansi16 = intArrayOf(
            0xff2b2620.toInt(), 0xffff6f5e.toInt(), 0xffb5cc52.toInt(), 0xfff4bf75.toInt(),
            0xff6a9fb5.toInt(), 0xffaa759f.toInt(), 0xff75b5aa.toInt(), 0xffd0c8a0.toInt(),
            0xff5c5444.toInt(), 0xffff8a72.toInt(), 0xffcde071.toInt(), 0xffffd479.toInt(),
            0xff88b8cc.toInt(), 0xffc08fb0.toInt(), 0xff92c7bd.toInt(), 0xfff5eccf.toInt(),
        ),
    ),
    ZENBURN(
        "Zenburn gray",
        background = 0xff3f3f3f.toInt(),
        foreground = 0xffdcdccc.toInt(),
        cursor = 0xffdcdccc.toInt(),
        ansi16 = intArrayOf(
            0xff3f3f3f.toInt(), 0xffcc9393.toInt(), 0xff7f9f7f.toInt(), 0xffd0bf8f.toInt(),
            0xff6ca0a3.toInt(), 0xffdc8cc3.toInt(), 0xff93e0e3.toInt(), 0xffdcdccc.toInt(),
            0xff709080.toInt(), 0xffdca3a3.toInt(), 0xffbfebbf.toInt(), 0xfff0dfaf.toInt(),
            0xff8cd0d3.toInt(), 0xffec93d3.toInt(), 0xff93e0e3.toInt(), 0xffffffff.toInt(),
        ),
    ),
    PAPER(
        "Paper (white)",
        background = 0xffffffff.toInt(),
        foreground = 0xff2e2e2e.toInt(),
        cursor = 0xff2e2e2e.toInt(),
        ansi16 = intArrayOf(
            0xff2e2e2e.toInt(), 0xffc7254e.toInt(), 0xff2f8a2f.toInt(), 0xffb58900.toInt(),
            0xff2060c0.toInt(), 0xff9c27b0.toInt(), 0xff0a9b9b.toInt(), 0xffd0d0d0.toInt(),
            0xff5a5a5a.toInt(), 0xffe0405e.toInt(), 0xff36a836.toInt(), 0xffc79a1a.toInt(),
            0xff3a78d8.toInt(), 0xffb452c4.toInt(), 0xff14b3b3.toInt(), 0xffffffff.toInt(),
        ),
    );

    /** The palette's background colour (ARGB), for theming the whole view area. */
    val backgroundColor: Int get() = background

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
