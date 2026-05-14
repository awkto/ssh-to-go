package io.sshtogo.android.ui.dashboard

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import io.sshtogo.android.data.ServerProfile
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
                _state.value = DashboardState(loading = false, hosts = hosts, sessions = sessions)
            } catch (t: Throwable) {
                _state.value = _state.value.copy(loading = false, error = t.message ?: "Failed to load")
            }
        }
    }
}
