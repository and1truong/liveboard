import SwiftUI

struct ContentView: View {
    @EnvironmentObject var server: ServerManager

    var body: some View {
        Group {
            if let url = server.serverURL {
                LiveBoardWebView(url: url)
                    .ignoresSafeArea()
            } else if let error = server.error {
                VStack(spacing: 16) {
                    Image(systemName: "exclamationmark.triangle")
                        .font(.system(size: 48))
                        .foregroundColor(.orange)
                    Text("Failed to Start Server")
                        .font(.title2.bold())
                    Text(error)
                        .font(.body)
                        .foregroundColor(.secondary)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal, 40)
                    Button("Retry") {
                        server.error = nil
                        server.start()
                    }
                    .buttonStyle(.borderedProminent)
                }
            } else {
                VStack(spacing: 12) {
                    ProgressView()
                        .scaleEffect(1.5)
                    Text("Starting LiveBoard...")
                        .font(.headline)
                        .foregroundColor(.secondary)
                }
            }
        }
        .onAppear {
            server.start()
        }
    }
}
