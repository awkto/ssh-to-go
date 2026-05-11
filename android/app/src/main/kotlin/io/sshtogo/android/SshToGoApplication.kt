package io.sshtogo.android

import android.app.Application
import io.sshtogo.android.data.ServerProfileStore

class SshToGoApplication : Application() {
    lateinit var profileStore: ServerProfileStore
        private set

    override fun onCreate() {
        super.onCreate()
        instance = this
        profileStore = ServerProfileStore(this)
    }

    companion object {
        lateinit var instance: SshToGoApplication
            private set
    }
}
