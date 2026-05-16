package io.sshtogo.android

import android.app.Application
import io.sshtogo.android.data.AppPreferences
import io.sshtogo.android.data.ServerProfileStore
import io.sshtogo.android.terminal.TerminalPalette

class SshToGoApplication : Application() {
    lateinit var profileStore: ServerProfileStore
        private set
    lateinit var prefs: AppPreferences
        private set

    override fun onCreate() {
        super.onCreate()
        instance = this
        profileStore = ServerProfileStore(this)
        prefs = AppPreferences(this)
        // Apply the saved palette to the static COLOR_SCHEME so any
        // TerminalEmulator created later picks it up by default.
        TerminalPalette.fromName(prefs.terminalPaletteName).apply()
    }

    companion object {
        lateinit var instance: SshToGoApplication
            private set
    }
}
