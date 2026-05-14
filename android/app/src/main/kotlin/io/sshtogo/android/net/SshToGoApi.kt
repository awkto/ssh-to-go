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
)

fun HostSessionEntry.toHostSession(): HostSession = HostSession(
    host = hostName,
    name = session.name,
    attached = session.attached,
    attachedClients = session.attachedClients,
    windows = session.windows,
)
