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
            do {
                try fm.createDirectory(atPath: workDir, withIntermediateDirectories: true)
            } catch {
                self.error = "Failed to create workspace directory: \(error.localizedDescription)"
                started = false
                return
            }
        }

        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "dev"

        Task.detached { [weak self] in
            var err: NSError?
            let urlString = GobridgeStart(workDir, version, &err)

            await MainActor.run {
                if let err = err {
                    self?.error = err.localizedDescription
                    self?.started = false
                    return
                }
                guard let url = URL(string: urlString) else {
                    self?.error = "Server returned invalid URL: \(urlString)"
                    self?.started = false
                    return
                }
                self?.serverURL = url
            }
        }
    }

    func stop() {
        GobridgeStop()
        started = false
        serverURL = nil
    }

    deinit {
        GobridgeStop()
    }

    /// Returns the default workspace path inside the app's Documents directory.
    private static func workspaceDirectory() -> String {
        let docs = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask).first!
        return docs.appendingPathComponent("Workspace").path
    }
}
