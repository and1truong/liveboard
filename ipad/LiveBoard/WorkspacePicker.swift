import SwiftUI
import UniformTypeIdentifiers

/// Document picker for importing markdown workspace folders.
struct WorkspacePicker: UIViewControllerRepresentable {
    let onPick: (URL) -> Void
    var onError: ((String) -> Void)?

    func makeUIViewController(context: Context) -> UIDocumentPickerViewController {
        let picker = UIDocumentPickerViewController(forOpeningContentTypes: [.folder])
        picker.allowsMultipleSelection = false
        picker.delegate = context.coordinator
        return picker
    }

    func updateUIViewController(_ uiViewController: UIDocumentPickerViewController, context: Context) {}

    func makeCoordinator() -> Coordinator {
        Coordinator(onPick: onPick, onError: onError)
    }

    class Coordinator: NSObject, UIDocumentPickerDelegate {
        let onPick: (URL) -> Void
        let onError: ((String) -> Void)?

        init(onPick: @escaping (URL) -> Void, onError: ((String) -> Void)?) {
            self.onPick = onPick
            self.onError = onError
        }

        func documentPicker(_ controller: UIDocumentPickerViewController, didPickDocumentsAt urls: [URL]) {
            guard let url = urls.first else { return }
            guard url.startAccessingSecurityScopedResource() else {
                onError?("Unable to access the selected folder. Please try again.")
                return
            }
            // Pass URL to caller; caller is responsible for calling
            // url.stopAccessingSecurityScopedResource() when done.
            onPick(url)
        }
    }
}
