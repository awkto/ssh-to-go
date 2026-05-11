package io.sshtogo.android.ui

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import io.sshtogo.android.SshToGoApplication
import io.sshtogo.android.ui.add.AddServerScreen
import io.sshtogo.android.ui.dashboard.DashboardScreen
import io.sshtogo.android.ui.servers.ServerListScreen

object Routes {
    const val ServerList = "servers"
    const val AddServer = "servers/add"
    const val Dashboard = "dashboard/{profileId}"
    fun dashboard(profileId: String) = "dashboard/$profileId"
}

@Composable
fun SshToGoApp() {
    val nav = rememberNavController()
    val store = SshToGoApplication.instance.profileStore
    val profiles by store.profiles.collectAsState()
    val activeId by store.activeId.collectAsState()

    val startDestination = when {
        profiles.isEmpty() -> Routes.AddServer
        activeId != null -> Routes.dashboard(activeId!!)
        else -> Routes.ServerList
    }

    Surface(modifier = Modifier.fillMaxSize()) {
        NavHost(navController = nav, startDestination = startDestination) {
            composable(Routes.ServerList) {
                ServerListScreen(
                    onSelect = { profile ->
                        store.setActive(profile.id)
                        nav.navigate(Routes.dashboard(profile.id)) {
                            popUpTo(Routes.ServerList) { inclusive = true }
                        }
                    },
                    onAdd = { nav.navigate(Routes.AddServer) },
                )
            }
            composable(Routes.AddServer) {
                AddServerScreen(
                    onAdded = { profile ->
                        store.setActive(profile.id)
                        nav.navigate(Routes.dashboard(profile.id)) {
                            popUpTo(0) { inclusive = true }
                        }
                    },
                    onCancel = {
                        if (profiles.isNotEmpty()) nav.popBackStack()
                    },
                )
            }
            composable(Routes.Dashboard) { backStack ->
                val profileId = backStack.arguments?.getString("profileId").orEmpty()
                DashboardScreen(
                    profileId = profileId,
                    onSwitchServer = {
                        nav.navigate(Routes.ServerList) {
                            popUpTo(0) { inclusive = true }
                        }
                    },
                    onAddServer = {
                        nav.navigate(Routes.AddServer)
                    },
                )
            }
        }
    }
}
