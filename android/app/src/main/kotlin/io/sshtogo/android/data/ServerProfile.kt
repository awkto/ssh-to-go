package io.sshtogo.android.data

import kotlinx.serialization.Serializable

@Serializable
data class ServerProfile(
    val id: String,
    val name: String,
    val baseUrl: String,
    val token: String,
)
