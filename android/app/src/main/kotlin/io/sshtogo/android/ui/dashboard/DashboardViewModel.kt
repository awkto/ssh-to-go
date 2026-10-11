package io.sshtogo.android.ui.dashboard

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import io.sshtogo.android.data.ServerProfile
import io.sshtogo.android.net.CreateSessionRequest
import io.sshtogo.android.net.HostSession
import io.sshtogo.android.net.HostState
import io.sshtogo.android.net.SshToGoClient
import io.sshtogo.android.net.TagColorRequest
import io.sshtogo.android.net.toHostSession
import io.sshtogo.android.ui.tags.normalizeTags
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

data class DashboardState(
    val loading: Boolean = false,
    val error: String? = null,
    val hosts: List<HostState> = emptyList(),
    val sessions: List<HostSession> = emptyList(),
    // New Session form context, fetched alongside the dashboard data.
    // recentCommands feeds the command chips (server-side list, shared with
    // the web form); serverNewSessionDir is the directory prefill fallback
    // when the app has no persisted last value. Both are best-effort — a
    // failure just means an emptier form, never a failed dashboard.
    val recentCommands: List<String> = emptyList(),
    val serverNewSessionDir: String = "",
    // Tags per "host:session" (live and offloaded alike) and tag -> OKLCH
    // hue. Best-effort like the form context: an older server without
    // these endpoints just shows no tags.
    val sessionTags: Map<String, List<String>> = emptyMap(),
    val tagColors: Map<String, Int> = emptyMap(),
)

fun tagKey(host: String, session: String) = "$host:$session"

private fun JsonObject.tags(): List<String> =
    runCatching { this["tags"]?.jsonArray?.map { it.jsonPrimitive.content } }.getOrNull().orEmpty()

class DashboardViewModel(private val profile: ServerProfile) : ViewModel() {

    private val _state = MutableStateFlow(DashboardState(loading = true))
    val state: StateFlow<DashboardState> = _state.asStateFlow()

    init {
        refresh()
    }

    fun refresh() {
        _state.value = _state.value.copy(loading = true, error = null)
        viewModelScope.launch {
            val api = SshToGoClient.forProfile(profile)
            try {
                val hosts = api.hosts()
                val sessionTags = runCatching {
                    api.sessionIcons().mapValues { (_, v) -> v.tags() }.filterValues { it.isNotEmpty() }
                }.getOrDefault(_state.value.sessionTags)
                val tagColors = runCatching { api.tagColors() }.getOrDefault(_state.value.tagColors)
                val sessions = api.sessions().map {
                    it.toHostSession().let { s -> s.copy(tags = sessionTags[tagKey(s.host, s.name)].orEmpty()) }
                }
                val recents = runCatching { api.recentCommands().map { it.command } }
                    .getOrDefault(_state.value.recentCommands)
                val serverDir = runCatching { api.settings().newSessionDir }
                    .getOrDefault(_state.value.serverNewSessionDir)
                _state.value = DashboardState(
                    loading = false,
                    hosts = hosts,
                    sessions = sessions,
                    recentCommands = recents,
                    serverNewSessionDir = serverDir,
                    sessionTags = sessionTags,
                    tagColors = tagColors,
                )
            } catch (t: Throwable) {
                _state.value = _state.value.copy(loading = false, error = t.message ?: "Failed to load")
            }
        }
    }

    /**
     * Create a new tmux session on [hostName]. Invokes [onResult] with
     * (null, sanitizedName) on success — the server's sanitized name is the
     * one tmux really uses, so the terminal must be opened with it — or
     * (errorMessage, null) on failure (e.g. a 409 when the name collides
     * with a live or offloaded session).
     */
    fun createSession(
        hostName: String,
        req: CreateSessionRequest,
        onResult: (error: String?, createdName: String?) -> Unit,
    ) {
        viewModelScope.launch {
            val api = SshToGoClient.forProfile(profile)
            try {
                val resp = api.createSession(hostName, req)
                onResult(null, resp.name.ifBlank { req.name })
                refresh()
            } catch (t: Throwable) {
                onResult(t.message ?: "Failed to create session", null)
            }
        }
    }

    /**
     * Resume an offloaded session on [hostName] — the server recreates it
     * with its recorded directory and launch command. Invokes [onResult]
     * like [createSession]: (null, nameToOpen) or (errorMessage, null).
     */
    fun recreateSession(
        hostName: String,
        sessionName: String,
        onResult: (error: String?, name: String?) -> Unit,
    ) {
        viewModelScope.launch {
            val api = SshToGoClient.forProfile(profile)
            try {
                val resp = api.recreateSession(hostName, sessionName)
                onResult(null, resp.name.ifBlank { sessionName })
                refresh()
            } catch (t: Throwable) {
                onResult(t.message ?: "Failed to resume session", null)
            }
        }
    }

    /**
     * Replace a session's tags. The server PUT replaces the whole icon
     * entry, so the current entry is fetched fresh and only "tags" changes
     * — icon, colour, star, theme, keep-awake and anything newer survive.
     * Then refreshes, which also picks up the random colour the server
     * gave any brand-new tag.
     */
    fun saveTags(
        hostName: String,
        sessionName: String,
        tags: List<String>,
        onResult: (error: String?) -> Unit,
    ) {
        val clean = normalizeTags(tags)
        viewModelScope.launch {
            val api = SshToGoClient.forProfile(profile)
            try {
                val current = api.sessionIcons()[tagKey(hostName, sessionName)] ?: JsonObject(emptyMap())
                val next = JsonObject(current + ("tags" to JsonArray(clean.map { JsonPrimitive(it) })))
                api.setSessionIcon(hostName, sessionName, next)
                onResult(null)
                refresh()
            } catch (t: Throwable) {
                onResult(t.message ?: "Failed to save tags")
            }
        }
    }

    /** Recolour [tag] everywhere. Optimistic: the UI updates before the server answers. */
    fun setTagColor(tag: String, hue: Int, onError: (String) -> Unit) {
        val before = _state.value.tagColors
        _state.value = _state.value.copy(tagColors = before + (tag to hue))
        viewModelScope.launch {
            try {
                SshToGoClient.forProfile(profile).setTagColor(tag, TagColorRequest(hue))
            } catch (t: Throwable) {
                _state.value = _state.value.copy(tagColors = before)
                onError(t.message ?: "Failed to set colour")
            }
        }
    }
}
