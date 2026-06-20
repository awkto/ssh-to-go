package io.sshtogo.android

import android.app.Application
import io.sshtogo.android.data.AppPreferences
import io.sshtogo.android.data.OpenedSessions
import io.sshtogo.android.data.ServerProfileStore

class SshToGoApplication : Application() {
    lateinit var profileStore: ServerProfileStore
        private set
    lateinit var prefs: AppPreferences
        private set

    /** Sessions opened during this app run, driving the terminal swipe carousel. */
    val openedSessions: OpenedSessions = OpenedSessions()

    override fun onCreate() {
        super.onCreate()
        instance = this
        profileStore = ServerProfileStore(this)
        prefs = AppPreferences(this)
    }

    companion object {
        lateinit var instance: SshToGoApplication
            private set
    }
}
