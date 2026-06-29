package io.sshtogo.android.net

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import retrofit2.http.GET

@Serializable
data class MeResponse(
    val authenticated: Boolean,
    @SerialName("no_auth") val noAuth: Boolean = false,
    val version: String = "",
)

/**
 * Subset of the ssh-to-go HTTP API that the Android app calls. Mirrors the routes in
 * internal/api/router.go. Phase 1 only needs read endpoints.
 *
 * Authentication is via a bearer API token created in the web UI under
 * Settings → API tokens. The app pastes the token into the Add Server screen and
 * validates it by calling /api/me before persisting the profile.
 */
interface SshToGoApi {
    @GET("api/me")
    suspend fun me(): MeResponse

    @GET("api/hosts")
    suspend fun hosts(): List<HostState>

    @GET("api/sessions")
    suspend fun sessions(): List<HostSessionEntry>
}

// ──────────────────────────────────────────────────────────────────────────────
// Wire format mirrors the Go shapes in internal/hub/hub.go and the
// hostResponse wrapper in internal/api/handlers.go (which embeds HostState
// and adds missing_sessions).
// ──────────────────────────────────────────────────────────────────────────────

@Serializable
data class HostConfig(
    val name: String,
    val address: String = "",
    val port: Int = 22,
    val user: String = "",
    val os: String = "",
    val icon: String = "",
    @SerialName("icon_color") val iconColor: String = "",
)

@Serializable
data class HostMetrics(
    val cpu: Double = 0.0,
    val mem: Double = 0.0,
    val disk: Double = 0.0,
    val load1: Double = 0.0,
)

@Serializable
data class HostState(
    val config: HostConfig,
    @SerialName("tmux_detected") val tmuxDetected: Boolean = false,
    @SerialName("tmux_version") val tmuxVersion: String = "",
    val sessions: List<TmuxSession>? = null,
    @SerialName("last_poll") val lastPoll: String = "",
    val error: String = "",
    val online: Boolean = false,
    @SerialName("detected_os") val detectedOs: String = "",
    val metrics: HostMetrics? = null,
    @SerialName("missing_sessions") val missingSessions: List<MissingSession> = emptyList(),
) {
    // Flat conveniences so UI code doesn't have to know the wire shape.
    val name: String get() = config.name
    val address: String get() = config.address
    val user: String get() = config.user
    val os: String get() = detectedOs.ifBlank { config.os }
}

@Serializable
data class TmuxSession(
    val name: String,
    val windows: Int = 1,
    val created: String = "",
    // Last user/program activity in the session (RFC3339), emitted by the
    // server from tmux's #{session_activity}. Drives "most recently used"
    // sorting, mirroring the desktop UI. Falls back to created when absent.
    val activity: String = "",
    val attached: Boolean = false,
    @SerialName("attached_clients") val attachedClients: Int = 0,
)

@Serializable
data class MissingSession(
    val host: String,
    val name: String,
    @SerialName("working_dir") val workingDir: String = "",
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("last_seen_at") val lastSeenAt: String = "",
)

// /api/sessions returns a flat list keyed by host_name.
@Serializable
data class HostSessionEntry(
    @SerialName("host_name") val hostName: String,
    val session: TmuxSession,
)

// UI-friendly flattened view used by DashboardScreen.
data class HostSession(
    val host: String,
    val name: String,
    val attached: Boolean,
    val attachedClients: Int,
    val windows: Int,
    // Epoch millis of last activity (or created as fallback); 0 if unknown.
    // Used to sort sessions most-recently-used first.
    val activityEpochMs: Long,
)

fun HostSessionEntry.toHostSession(): HostSession = HostSession(
    host = hostName,
    name = session.name,
    attached = session.attached,
    attachedClients = session.attachedClients,
    windows = session.windows,
    activityEpochMs = parseEpochMs(session.activity).takeIf { it > 0 }
        ?: parseEpochMs(session.created),
)

/**
 * Parse an RFC3339 timestamp to epoch millis, or 0 if blank/unparseable.
 * Uses OffsetDateTime, not Instant.parse: the Go server emits the zone-offset
 * form ("...+10:00") for non-UTC hosts, which Instant.parse rejects — that
 * would silently zero every timestamp and disable the most-recently-used sort.
 */
private fun parseEpochMs(s: String): Long =
    if (s.isBlank()) 0L
    else runCatching { java.time.OffsetDateTime.parse(s).toInstant().toEpochMilli() }.getOrDefault(0L)
