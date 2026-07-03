package io.sshtogo.android.ui.dashboard

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.SwapHoriz
import androidx.compose.material.icons.filled.Palette
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import io.sshtogo.android.SshToGoApplication
import io.sshtogo.android.appVersionName
import io.sshtogo.android.net.HostSession
import io.sshtogo.android.net.HostState

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DashboardScreen(
    profileId: String,
    onSwitchServer: () -> Unit,
    onAddServer: () -> Unit,
    onOpenSession: (hostName: String, sessionName: String) -> Unit,
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
    val version = LocalContext.current.appVersionName()

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text(profile.name, style = MaterialTheme.typography.titleMedium)
                        Text(
                            "${profile.baseUrl}  ·  v$version",
                            style = MaterialTheme.typography.bodySmall,
                        )
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
                        HorizontalDivider()
                        val app = io.sshtogo.android.SshToGoApplication.instance
                        // Toggle: keep the menu open so the check updates in place.
                        DropdownMenuItem(
                            leadingIcon = if (app.prefs.sortSessionsByRecent)
                                { { Icon(Icons.Default.Check, contentDescription = null) } } else null,
                            text = { Text("Sort by recently used") },
                            onClick = {
                                app.prefs.sortSessionsByRecent = !app.prefs.sortSessionsByRecent
                            },
                        )
                        HorizontalDivider()
                        Text(
                            "App theme",
                            style = MaterialTheme.typography.labelSmall,
                            modifier = Modifier.padding(horizontal = 16.dp, vertical = 6.dp),
                        )
                        io.sshtogo.android.data.ThemeMode.entries.forEach { mode ->
                            DropdownMenuItem(
                                leadingIcon = if (mode == app.prefs.themeMode)
                                    { { Icon(Icons.Default.Palette, contentDescription = null) } } else null,
                                text = { Text(mode.name.lowercase().replaceFirstChar { it.uppercase() }) },
                                onClick = {
                                    menuOpen = false
                                    app.prefs.themeMode = mode
                                },
                            )
                        }
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
                    onOpenSession = onOpenSession,
                    onCreateSession = { host, name, cwd, cb -> vm.createSession(host, name, cwd, cb) },
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
    onOpenSession: (hostName: String, sessionName: String) -> Unit,
    onCreateSession: (host: String, name: String, cwd: String, onResult: (String?) -> Unit) -> Unit,
) {
    val sessionsByHost = sessions.groupBy { it.host }
    // Reading this state-backed pref here re-sorts the list when toggled.
    val sortByRecent = SshToGoApplication.instance.prefs.sortSessionsByRecent

    LazyColumn(
        modifier = modifier.fillMaxSize(),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        items(hosts, key = { it.name }) { host ->
            val hostSessions = sessionsByHost[host.name].orEmpty()
            HostCard(
                host = host,
                // Stable sort: most-recently-used first, ties keep server order.
                sessions = if (sortByRecent)
                    hostSessions.sortedByDescending { it.activityEpochMs }
                else hostSessions,
                onOpenSession = onOpenSession,
                onCreateSession = onCreateSession,
            )
        }
    }
}

@Composable
private fun HostCard(
    host: HostState,
    sessions: List<HostSession>,
    onOpenSession: (hostName: String, sessionName: String) -> Unit,
    onCreateSession: (host: String, name: String, cwd: String, onResult: (String?) -> Unit) -> Unit,
) {
    var showCreate by remember { mutableStateOf(false) }
    var newName by remember { mutableStateOf("") }
    var newCwd by remember { mutableStateOf("") }
    var creating by remember { mutableStateOf(false) }
    var createError by remember { mutableStateOf<String?>(null) }

    if (showCreate) {
        AlertDialog(
            onDismissRequest = { if (!creating) showCreate = false },
            title = { Text("New session on ${host.name}") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(
                        value = newName,
                        onValueChange = { newName = it; createError = null },
                        label = { Text("Session name") },
                        singleLine = true,
                        enabled = !creating,
                    )
                    OutlinedTextField(
                        value = newCwd,
                        onValueChange = { newCwd = it },
                        label = { Text("Working directory (optional)") },
                        singleLine = true,
                        enabled = !creating,
                    )
                    createError?.let {
                        Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                    }
                }
            },
            confirmButton = {
                TextButton(
                    enabled = newName.isNotBlank() && !creating,
                    onClick = {
                        creating = true
                        createError = null
                        onCreateSession(host.name, newName, newCwd) { err ->
                            creating = false
                            if (err == null) {
                                showCreate = false; newName = ""; newCwd = ""
                            } else {
                                createError = err
                            }
                        }
                    },
                ) { Text(if (creating) "Creating…" else "Create") }
            },
            dismissButton = {
                TextButton(enabled = !creating, onClick = { showCreate = false }) { Text("Cancel") }
            },
        )
    }

    Card(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                StatusDot(online = host.online)
                Text(
                    text = host.name,
                    style = MaterialTheme.typography.titleMedium,
                    modifier = Modifier.padding(start = 8.dp),
                )
                Spacer(modifier = Modifier.weight(1f))
                IconButton(onClick = { newName = ""; newCwd = ""; createError = null; showCreate = true }) {
                    Icon(Icons.Default.Add, contentDescription = "New session on ${host.name}")
                }
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
                    modifier = Modifier.padding(top = 8.dp),
                    verticalArrangement = Arrangement.spacedBy(2.dp),
                ) {
                    sessions.forEach { s ->
                        // Reactive: dot shows when this session is open in the app
                        // (in the swipe carousel for this run).
                        val isOpen = SshToGoApplication.instance.openedSessions.isOpen(host.name, s.name)
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable { onOpenSession(host.name, s.name) }
                                .padding(horizontal = 4.dp, vertical = 12.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            // Fixed-width leading slot keeps names aligned whether
                            // or not the open-dot is shown.
                            Box(modifier = Modifier.size(8.dp), contentAlignment = Alignment.Center) {
                                if (isOpen) {
                                    Box(
                                        modifier = Modifier
                                            .size(8.dp)
                                            .clip(CircleShape)
                                            .background(MaterialTheme.colorScheme.primary),
                                    )
                                }
                            }
                            Spacer(modifier = Modifier.size(8.dp))
                            Text(
                                "›  ${s.name}",
                                style = MaterialTheme.typography.titleMedium,
                                modifier = Modifier.weight(1f),
                            )
                            if (s.attached) {
                                Text(
                                    "attached",
                                    style = MaterialTheme.typography.labelMedium,
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
