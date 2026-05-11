package io.sshtogo.android.net

import io.sshtogo.android.data.ServerProfile
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import java.util.concurrent.TimeUnit

/**
 * Builds Retrofit clients keyed on a server profile. Each profile gets its own client
 * with a fixed baseUrl + Authorization: Bearer interceptor.
 */
object SshToGoClient {

    private val json = Json {
        ignoreUnknownKeys = true
        coerceInputValues = true
    }

    private val loggingInterceptor = HttpLoggingInterceptor().apply {
        // Body logging would expose bearer tokens; keep at BASIC.
        level = HttpLoggingInterceptor.Level.BASIC
    }

    fun forProfile(profile: ServerProfile): SshToGoApi {
        val ok = OkHttpClient.Builder()
            .connectTimeout(15, TimeUnit.SECONDS)
            .readTimeout(60, TimeUnit.SECONDS)
            .addInterceptor(BearerInterceptor(profile.token))
            .addInterceptor(loggingInterceptor)
            .build()

        return Retrofit.Builder()
            .baseUrl(profile.baseUrl.trimEnd('/') + "/")
            .client(ok)
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()
            .create(SshToGoApi::class.java)
    }

    /**
     * Builds an *unauthenticated* client for the given base URL. Used during initial
     * login to call /api/auth/login before we have a token.
     */
    fun unauthenticated(baseUrl: String): SshToGoApi {
        val ok = OkHttpClient.Builder()
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(15, TimeUnit.SECONDS)
            .addInterceptor(loggingInterceptor)
            .build()

        return Retrofit.Builder()
            .baseUrl(baseUrl.trimEnd('/') + "/")
            .client(ok)
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()
            .create(SshToGoApi::class.java)
    }
}

private class BearerInterceptor(private val token: String) : okhttp3.Interceptor {
    override fun intercept(chain: okhttp3.Interceptor.Chain): okhttp3.Response {
        val req: Request = chain.request().newBuilder()
            .addHeader("Authorization", "Bearer $token")
            .build()
        return chain.proceed(req)
    }
}
