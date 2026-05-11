package io.sshtogo.android.ui.dashboard

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.SwapHoriz
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import io.sshtogo.android.SshToGoApplication
import io.sshtogo.android.net.HostSession
import io.sshtogo.android.net.HostState

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DashboardScreen(
    profileId: String,
    onSwitchServer: () -> Unit,
    onAddServer: () -> Unit,
) {
    val store = SshToGoApplication.instance.profileStore
    val profiles by store.profiles.collectAsState()
    val profile = profiles.firstOrNull { it.id == profileId }
        ?: run {
            // Profile was deleted out from under us — bounce back to picker.
            Text("Server profile not found")
            return
        }

    val vm: DashboardViewModel = viewModel(
        key = profile.id,
        factory = viewModelFactory {
            initializer { DashboardViewModel(profile) }
        },
    )
    val state by vm.state.collectAsState()
    var menuOpen by remember { mutableStateOf(false) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text(profile.name, style = MaterialTheme.typography.titleMedium)
                        Text(profile.baseUrl, style = MaterialTheme.typography.bodySmall)
                    }
                },
                actions = {
                    IconButton(onClick = { vm.refresh() }) {
                        Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                    }
                    IconButton(onClick = { menuOpen = true }) {
                        Icon(Icons.Default.SwapHoriz, contentDescription = "Server menu")
                    }
                    DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
                        DropdownMenuItem(
                            text = { Text("Switch server") },
                            onClick = { menuOpen = false; onSwitchServer() },
                        )
                        DropdownMenuItem(
                            text = { Text("Add server") },
                            leadingIcon = { Icon(Icons.Default.Add, contentDescription = null) },
                            onClick = { menuOpen = false; onAddServer() },
                        )
                    }
                },
            )
        },
    ) { pad ->
        when {
            state.loading && state.hosts.isEmpty() -> {
                Column(
                    modifier = Modifier.fillMaxSize().padding(pad),
                    verticalArrangement = Arrangement.Center,
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    CircularProgressIndicator()
                }
            }
            state.error != null && state.hosts.isEmpty() -> {
                Column(
                    modifier = Modifier.fillMaxSize().padding(pad).padding(24.dp),
                    verticalArrangement = Arrangement.Center,
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    Text("Couldn't reach server", style = MaterialTheme.typography.titleMedium)
                    Text(state.error!!, style = MaterialTheme.typography.bodyMedium)
                }
            }
            else -> {
                DashboardList(
                    modifier = Modifier.padding(pad),
                    hosts = state.hosts,
                    sessions = state.sessions,
                )
            }
        }
    }
}

@Composable
private fun DashboardList(
    modifier: Modifier,
    hosts: List<HostState>,
    sessions: List<HostSession>,
) {
    val sessionsByHost = sessions.groupBy { it.host }

    LazyColumn(
        modifier = modifier.fillMaxSize(),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        items(hosts, key = { it.name }) { host ->
            HostCard(host = host, sessions = sessionsByHost[host.name].orEmpty())
        }
    }
}

@Composable
private fun HostCard(host: HostState, sessions: List<HostSession>) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                StatusDot(online = host.online)
                Text(
                    text = host.name,
                    style = MaterialTheme.typography.titleMedium,
                    modifier = Modifier.padding(start = 8.dp),
                )
            }
            val subtitle = buildString {
                if (host.user.isNotBlank()) append(host.user).append('@')
                append(host.address.ifBlank { "—" })
                if (host.os.isNotBlank()) append("  ·  ").append(host.os)
            }
            Text(subtitle, style = MaterialTheme.typography.bodySmall)

            if (sessions.isEmpty()) {
                Text(
                    "No sessions",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.outline,
                )
            } else {
                Column(
                    modifier = Modifier.padding(top = 6.dp),
                    verticalArrangement = Arrangement.spacedBy(4.dp),
                ) {
                    sessions.forEach { s ->
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text(
                                "•  ${s.name}",
                                style = MaterialTheme.typography.bodyMedium,
                                modifier = Modifier.weight(1f),
                            )
                            if (s.attached) {
                                Text(
                                    "attached",
                                    style = MaterialTheme.typography.labelSmall,
                                    color = MaterialTheme.colorScheme.primary,
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun StatusDot(online: Boolean) {
    val color = if (online) Color(0xFF4CAF50) else Color(0xFF9E9E9E)
    Box(
        modifier = Modifier
            .size(10.dp)
            .clip(CircleShape)
            .background(color),
    )
}
