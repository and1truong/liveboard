import Foundation
import Gobridge

/// Manages the embedded Go LiveBoard server lifecycle.
@MainActor
final class ServerManager: ObservableObject {
    @Published var serverURL: URL?
    @Published var error: String?

    private var started = false

    func start() {
        guard !started else { return }
        started = true

        let workDir = Self.workspaceDirectory()

        // Ensure workspace directory exists.
        let fm = FileManager.default
        if !fm.fileExists(atPath: workDir) {
            try? fm.createDirectory(atPath: workDir, withIntermediateDirectories: true)
        }

        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "dev"

        Task.detached { [weak self] in
            var err: NSError?
            let urlString = GobridgeStart(workDir, version, &err)

            await MainActor.run {
                if let err = err {
                    self?.error = err.localizedDescription
                    return
                }
                if let url = URL(string: urlString) {
                    self?.serverURL = url
                }
            }
        }
    }

    func stop() {
        GobridgeStop()
        started = false
        serverURL = nil
    }

    /// Returns the default workspace path inside the app's Documents directory.
    private static func workspaceDirectory() -> String {
        let docs = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask).first!
        return docs.appendingPathComponent("Workspace").path
    }
}
