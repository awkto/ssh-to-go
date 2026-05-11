package io.sshtogo.android.ui.add

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import io.sshtogo.android.SshToGoApplication
import io.sshtogo.android.data.ServerProfile
import io.sshtogo.android.net.SshToGoClient
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AddServerScreen(
    onAdded: (ServerProfile) -> Unit,
    onCancel: () -> Unit,
) {
    var name by remember { mutableStateOf("") }
    var baseUrl by remember { mutableStateOf("https://") }
    var token by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Add ssh-to-go server") },
                actions = {
                    TextButton(onClick = onCancel, enabled = !busy) { Text("Cancel") }
                },
            )
        },
    ) { pad ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(pad)
                .padding(horizontal = 20.dp, vertical = 16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(
                "Paste the URL of your ssh-to-go server and an API token created in " +
                    "Settings → API Tokens on the web UI.",
                style = MaterialTheme.typography.bodyMedium,
            )

            OutlinedTextField(
                value = name,
                onValueChange = { name = it },
                label = { Text("Display name (optional)") },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = baseUrl,
                onValueChange = { baseUrl = it.trim() },
                label = { Text("Base URL") },
                placeholder = { Text("https://ssh.example.com") },
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri),
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = token,
                onValueChange = { token = it.trim() },
                label = { Text("API token") },
                singleLine = true,
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
                modifier = Modifier.fillMaxWidth(),
            )

            error?.let {
                Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
            }

            Spacer(Modifier.height(8.dp))

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.End,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                if (busy) {
                    CircularProgressIndicator(
                        strokeWidth = 2.dp,
                        modifier = Modifier.padding(end = 12.dp).height(20.dp),
                    )
                }
                Button(
                    enabled = !busy && baseUrl.isNotBlank() && token.isNotBlank(),
                    onClick = {
                        busy = true
                        error = null
                        scope.launch {
                            val tentative = ServerProfile(
                                id = "",
                                name = name.ifBlank { baseUrl },
                                baseUrl = baseUrl,
                                token = token,
                            )
                            val api = SshToGoClient.forProfile(tentative)
                            try {
                                val me = api.me()
                                if (!me.authenticated) {
                                    error = "Server reports unauthenticated"
                                } else {
                                    val saved = SshToGoApplication.instance.profileStore.upsert(
                                        name = name.ifBlank { baseUrl },
                                        baseUrl = baseUrl,
                                        token = token,
                                    )
                                    onAdded(saved)
                                }
                            } catch (t: Throwable) {
                                error = t.message ?: "Failed to reach server"
                            } finally {
                                busy = false
                            }
                        }
                    },
                ) {
                    Text("Verify & save")
                }
            }
        }
    }
}
