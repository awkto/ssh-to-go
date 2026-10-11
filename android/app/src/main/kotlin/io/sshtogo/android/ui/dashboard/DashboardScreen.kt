package io.sshtogo.android.ui.dashboard

import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
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
import androidx.compose.material.icons.filled.ExpandLess
import androidx.compose.material.icons.filled.ExpandMore
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
import androidx.compose.ui.text.style.TextOverflow
import android.widget.Toast
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import io.sshtogo.android.SshToGoApplication
import io.sshtogo.android.appVersionName
import io.sshtogo.android.net.CreateSessionRequest
import io.sshtogo.android.net.HostSession
import io.sshtogo.android.net.HostState
import io.sshtogo.android.net.MissingSession
import io.sshtogo.android.ui.tags.TagChips
import io.sshtogo.android.ui.tags.TagEditorDialog
import io.sshtogo.android.ui.tags.TagFilterRow
import io.sshtogo.android.ui.tags.UNTAGGED

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
                    recentCommands = state.recentCommands,
                    serverNewSessionDir = state.serverNewSessionDir,
                    sessionTags = state.sessionTags,
                    tagColors = state.tagColors,
                    onOpenSession = onOpenSession,
                    onCreateSession = { host, req, cb -> vm.createSession(host, req, cb) },
                    onResumeSession = { host, name, cb -> vm.recreateSession(host, name, cb) },
                    onSaveTags = { host, name, tags, cb -> vm.saveTags(host, name, tags, cb) },
                    onSetTagColor = { tag, hue, onErr -> vm.setTagColor(tag, hue, onErr) },
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
    recentCommands: List<String>,
    serverNewSessionDir: String,
    sessionTags: Map<String, List<String>>,
    tagColors: Map<String, Int>,
    onOpenSession: (hostName: String, sessionName: String) -> Unit,
    onCreateSession: (host: String, req: CreateSessionRequest, onResult: (String?, String?) -> Unit) -> Unit,
    onResumeSession: (host: String, name: String, onResult: (String?, String?) -> Unit) -> Unit,
    onSaveTags: (host: String, name: String, tags: List<String>, onResult: (String?) -> Unit) -> Unit,
    onSetTagColor: (tag: String, hue: Int, onError: (String) -> Unit) -> Unit,
) {
    val context = LocalContext.current
    val prefs = SshToGoApplication.instance.prefs
    val sessionsByHost = sessions.groupBy { it.host }
    // Reading this state-backed pref here re-sorts the list when toggled.
    val sortByRecent = prefs.sortSessionsByRecent

    // Tag filter over live and offloaded sessions alike. A remembered tag
    // nobody carries any more falls back to "all" rather than hiding
    // everything.
    val tagsOf = { host: String, name: String -> sessionTags[tagKey(host, name)].orEmpty() }
    val everySession = sessions.map { it.host to it.name } +
        hosts.flatMap { h -> h.missingSessions.map { h.name to it.name } }
    val counts = everySession.flatMap { (h, n) -> tagsOf(h, n) }
        .groupingBy { it }.eachCount().toSortedMap().toList()
    val untaggedCount = everySession.count { (h, n) -> tagsOf(h, n).isEmpty() }
    val filter = prefs.tagFilter.takeIf { it.isEmpty() || it == UNTAGGED || counts.any { c -> c.first == it } } ?: ""
    val matches = { tags: List<String> ->
        when (filter) {
            "" -> true
            UNTAGGED -> tags.isEmpty()
            else -> filter in tags
        }
    }

    // With a filter on, a host with nothing matching is just noise.
    val visibleHosts = if (filter.isEmpty()) hosts else hosts.filter { h ->
        sessionsByHost[h.name].orEmpty().any { matches(it.tags) } ||
            h.missingSessions.any { matches(tagsOf(h.name, it.name)) }
    }

    // Session whose tags are being edited: (host, name, current tags).
    var editing by remember { mutableStateOf<Triple<String, String, List<String>>?>(null) }
    editing?.let { (host, name, current) ->
        TagEditorDialog(
            sessionName = name,
            current = current,
            allTags = counts.map { it.first },
            colors = tagColors,
            onSave = { tags ->
                editing = null
                onSaveTags(host, name, tags) { err ->
                    if (err != null) Toast.makeText(context, "Saving tags failed: $err", Toast.LENGTH_LONG).show()
                }
            },
            onSetColor = { tag, hue ->
                onSetTagColor(tag, hue) { err ->
                    Toast.makeText(context, "Colour change failed: $err", Toast.LENGTH_LONG).show()
                }
            },
            onDismiss = { editing = null },
        )
    }
    val onEditTags = { host: String, name: String -> editing = Triple(host, name, tagsOf(host, name)) }

    LazyColumn(
        modifier = modifier.fillMaxSize(),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        item(key = "tag-filter") {
            if (counts.isEmpty()) {
                Text(
                    "Long-press a session to tag it (work, personal, …).",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.outline,
                )
            } else {
                TagFilterRow(
                    counts = counts,
                    untaggedCount = untaggedCount,
                    colors = tagColors,
                    selected = filter,
                    onSelect = { prefs.tagFilter = it },
                )
            }
        }
        items(visibleHosts, key = { it.name }) { host ->
            val hostSessions = sessionsByHost[host.name].orEmpty().filter { matches(it.tags) }
            val offloaded = host.missingSessions.filter { matches(tagsOf(host.name, it.name)) }
            HostCard(
                host = host,
                // Stable sort: most-recently-used first, ties keep server order.
                sessions = if (sortByRecent)
                    hostSessions.sortedByDescending { it.activityEpochMs }
                else hostSessions,
                offloaded = offloaded,
                tagColors = tagColors,
                tagsOf = tagsOf,
                recentCommands = recentCommands,
                serverNewSessionDir = serverNewSessionDir,
                onOpenSession = onOpenSession,
                onCreateSession = onCreateSession,
                onResumeSession = onResumeSession,
                onEditTags = onEditTags,
            )
        }
    }
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun HostCard(
    host: HostState,
    sessions: List<HostSession>,
    offloaded: List<MissingSession>,
    tagColors: Map<String, Int>,
    tagsOf: (host: String, name: String) -> List<String>,
    recentCommands: List<String>,
    serverNewSessionDir: String,
    onOpenSession: (hostName: String, sessionName: String) -> Unit,
    onCreateSession: (host: String, req: CreateSessionRequest, onResult: (String?, String?) -> Unit) -> Unit,
    onResumeSession: (host: String, name: String, onResult: (String?, String?) -> Unit) -> Unit,
    onEditTags: (host: String, name: String) -> Unit,
) {
    var showCreate by remember { mutableStateOf(false) }

    if (showCreate) {
        NewSessionDialog(
            hostName = host.name,
            recentCommands = recentCommands,
            serverNewSessionDir = serverNewSessionDir,
            onCreate = { req, cb -> onCreateSession(host.name, req, cb) },
            onCreated = { createdName ->
                showCreate = false
                // Straight into the terminal — the mobile analog of the web
                // form's "then attach".
                onOpenSession(host.name, createdName)
            },
            onDismiss = { showCreate = false },
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
                IconButton(onClick = { showCreate = true }) {
                    Icon(Icons.Default.Add, contentDescription = "New session on ${host.name}")
                }
            }
            val subtitle = buildString {
                if (host.user.isNotBlank()) append(host.user).append('@')
                append(host.address.ifBlank { "—" })
                if (host.os.isNotBlank()) append("  ·  ").append(host.os)
            }
            Text(subtitle, style = MaterialTheme.typography.bodySmall)

            if (sessions.isEmpty() && offloaded.isEmpty()) {
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
                                .combinedClickable(
                                    onClick = { onOpenSession(host.name, s.name) },
                                    onLongClick = { onEditTags(host.name, s.name) },
                                    onLongClickLabel = "Edit tags",
                                )
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
                            Column(modifier = Modifier.weight(1f)) {
                                Text(
                                    "›  ${s.name}",
                                    style = MaterialTheme.typography.titleMedium,
                                    maxLines = 1,
                                    overflow = TextOverflow.Ellipsis,
                                )
                                TagChips(s.tags, tagColors, Modifier.padding(start = 18.dp, top = 3.dp))
                            }
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

            if (offloaded.isNotEmpty()) {
                OffloadedSessions(
                    hostName = host.name,
                    // Most recently seen first — the one you just offloaded is on top.
                    offloaded = offloaded.sortedByDescending { it.lastSeenEpochMs },
                    tagColors = tagColors,
                    tagsOf = { name -> tagsOf(host.name, name) },
                    onResume = { name, cb -> onResumeSession(host.name, name, cb) },
                    onOpenSession = onOpenSession,
                    onEditTags = { name -> onEditTags(host.name, name) },
                )
            }
        }
    }
}

/**
 * Sessions the server still tracks but tmux no longer runs (offloaded by
 * hand or by auto-sleep). Collapsed by default so they don't crowd the live
 * list; tapping one recreates it with its recorded directory and launch
 * command, then opens the terminal — the app's version of the web
 * dashboard's Recreate button and `stogo resume`.
 */
@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun OffloadedSessions(
    hostName: String,
    offloaded: List<MissingSession>,
    tagColors: Map<String, Int>,
    tagsOf: (name: String) -> List<String>,
    onResume: (name: String, onResult: (String?, String?) -> Unit) -> Unit,
    onOpenSession: (hostName: String, sessionName: String) -> Unit,
    onEditTags: (name: String) -> Unit,
) {
    val context = LocalContext.current
    var expanded by remember(hostName) { mutableStateOf(false) }
    // Name currently being resumed; blocks double taps while the server works.
    var resuming by remember(hostName) { mutableStateOf<String?>(null) }

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { expanded = !expanded }
            .padding(horizontal = 4.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            "Offloaded (${offloaded.size})",
            style = MaterialTheme.typography.labelLarge,
            color = MaterialTheme.colorScheme.outline,
            modifier = Modifier.weight(1f),
        )
        Icon(
            if (expanded) Icons.Default.ExpandLess else Icons.Default.ExpandMore,
            contentDescription = if (expanded) "Hide offloaded sessions" else "Show offloaded sessions",
            tint = MaterialTheme.colorScheme.outline,
        )
    }
    if (!expanded) return

    Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
        offloaded.forEach { m ->
            val busy = resuming == m.name
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .combinedClickable(
                        enabled = resuming == null,
                        onLongClick = { onEditTags(m.name) },
                        onLongClickLabel = "Edit tags",
                    ) {
                        resuming = m.name
                        onResume(m.name) { error, name ->
                            resuming = null
                            if (error != null) {
                                Toast.makeText(context, "Resume failed: $error", Toast.LENGTH_LONG).show()
                            } else if (name != null) {
                                onOpenSession(hostName, name)
                            }
                        }
                    }
                    .padding(horizontal = 4.dp, vertical = 10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Spacer(modifier = Modifier.size(16.dp))
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        m.name,
                        style = MaterialTheme.typography.titleMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    val detail = listOfNotNull(
                        m.workingDir.ifBlank { null },
                        m.command.ifBlank { null },
                        if (m.autoOffloaded) "auto-slept" else null,
                    ).joinToString("  ·  ")
                    if (detail.isNotEmpty()) {
                        Text(
                            detail,
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.outline,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                    TagChips(tagsOf(m.name), tagColors, Modifier.padding(top = 3.dp))
                }
                if (busy) {
                    CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Text(
                        "resume",
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.primary,
                    )
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
