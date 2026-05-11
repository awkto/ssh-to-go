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
 * Settings → API Tokens. The app pastes the token into the Add Server screen and
 * validates it by calling /api/me before persisting the profile.
 */
interface SshToGoApi {
    @GET("api/me")
    suspend fun me(): MeResponse

    @GET("api/hosts")
    suspend fun hosts(): List<HostState>

    @GET("api/sessions")
    suspend fun sessions(): List<HostSession>
}

@Serializable
data class HostState(
    val name: String,
    val address: String = "",
    val user: String = "",
    val online: Boolean = false,
    val os: String = "",
    @SerialName("session_count") val sessionCount: Int = 0,
    @SerialName("cpu_load") val cpuLoad: Double = 0.0,
    @SerialName("mem_used") val memUsed: Double = 0.0,
)

@Serializable
data class HostSession(
    val host: String,
    val name: String,
    val attached: Boolean = false,
    @SerialName("client_count") val clientCount: Int = 0,
    val windows: Int = 0,
    val created: Long = 0,
)
