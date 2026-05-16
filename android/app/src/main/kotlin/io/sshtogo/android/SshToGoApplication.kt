package io.sshtogo.android

import android.app.Application
import io.sshtogo.android.data.AppPreferences
import io.sshtogo.android.data.ServerProfileStore

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
    }

    companion object {
        lateinit var instance: SshToGoApplication
            private set
    }
}
