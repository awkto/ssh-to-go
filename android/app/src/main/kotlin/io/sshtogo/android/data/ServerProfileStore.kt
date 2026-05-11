package io.sshtogo.android.data

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.json.Json
import java.util.UUID

/**
 * Persists ssh-to-go server profiles (URL + bearer token) in EncryptedSharedPreferences
 * and exposes the list as a StateFlow.
 *
 * The active profile id is stored separately so the app can boot straight into the last
 * server used.
 */
class ServerProfileStore(context: Context) {

    private val prefs: SharedPreferences = EncryptedSharedPreferences.create(
        context,
        PREFS_NAME,
        MasterKey.Builder(context).setKeyScheme(MasterKey.KeyScheme.AES256_GCM).build(),
        EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
        EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
    )

    private val _profiles = MutableStateFlow(loadProfiles())
    val profiles: StateFlow<List<ServerProfile>> = _profiles.asStateFlow()

    private val _activeId = MutableStateFlow(prefs.getString(KEY_ACTIVE_ID, null))
    val activeId: StateFlow<String?> = _activeId.asStateFlow()

    val active: ServerProfile?
        get() = _activeId.value?.let { id -> _profiles.value.firstOrNull { it.id == id } }

    fun upsert(name: String, baseUrl: String, token: String, existingId: String? = null): ServerProfile {
        val id = existingId ?: UUID.randomUUID().toString()
        val profile = ServerProfile(
            id = id,
            name = name.ifBlank { baseUrl },
            baseUrl = baseUrl.trimEnd('/'),
            token = token,
        )
        val current = _profiles.value.toMutableList()
        val idx = current.indexOfFirst { it.id == id }
        if (idx >= 0) current[idx] = profile else current.add(profile)
        persist(current)
        if (_activeId.value == null) setActive(id)
        return profile
    }

    fun remove(id: String) {
        val current = _profiles.value.filterNot { it.id == id }
        persist(current)
        if (_activeId.value == id) setActive(current.firstOrNull()?.id)
    }

    fun setActive(id: String?) {
        prefs.edit().apply {
            if (id == null) remove(KEY_ACTIVE_ID) else putString(KEY_ACTIVE_ID, id)
            apply()
        }
        _activeId.value = id
    }

    private fun persist(list: List<ServerProfile>) {
        prefs.edit()
            .putString(KEY_PROFILES, json.encodeToString(LIST_SERIALIZER, list))
            .apply()
        _profiles.value = list
    }

    private fun loadProfiles(): List<ServerProfile> {
        val raw = prefs.getString(KEY_PROFILES, null) ?: return emptyList()
        return runCatching { json.decodeFromString(LIST_SERIALIZER, raw) }.getOrDefault(emptyList())
    }

    companion object {
        private const val PREFS_NAME = "ssh-to-go-profiles"
        private const val KEY_PROFILES = "profiles"
        private const val KEY_ACTIVE_ID = "active_id"

        private val json = Json { ignoreUnknownKeys = true }
        private val LIST_SERIALIZER = kotlinx.serialization.builtins.ListSerializer(ServerProfile.serializer())
    }
}
