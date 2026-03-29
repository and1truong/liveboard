import SwiftUI

@main
struct LiveBoardApp: App {
    @StateObject private var server = ServerManager()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(server)
        }
    }
}
