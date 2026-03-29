import SwiftUI

@main
struct LiveBoardApp: App {
    @StateObject private var server = ServerManager()
    @Environment(\.scenePhase) private var scenePhase

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(server)
        }
        .onChange(of: scenePhase) { _, phase in
            if phase == .background {
                server.stop()
            }
        }
    }
}
