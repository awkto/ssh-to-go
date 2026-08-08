package io.sshtogo.android.ui.dashboard

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import io.sshtogo.android.data.ServerProfile
import io.sshtogo.android.net.CreateSessionRequest
import io.sshtogo.android.net.HostSession
import io.sshtogo.android.net.HostState
import io.sshtogo.android.net.SshToGoClient
import io.sshtogo.android.net.toHostSession
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
)

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
                val sessions = api.sessions().map { it.toHostSession() }
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
}
