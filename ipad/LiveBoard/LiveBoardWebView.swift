import SwiftUI
import WebKit

/// WKWebView wrapper that loads the LiveBoard server UI.
struct LiveBoardWebView: UIViewRepresentable {
    let url: URL

    func makeCoordinator() -> Coordinator {
        Coordinator()
    }

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.allowsInlineMediaPlayback = true

        // Allow the web content to fill the viewport properly.
        let prefs = WKWebpagePreferences()
        prefs.allowsContentProcessUseThisPolicy = true
        config.defaultWebpagePreferences = prefs

        let webView = WKWebView(frame: .zero, configuration: config)
        webView.navigationDelegate = context.coordinator
        webView.allowsBackForwardNavigationGestures = true
        webView.scrollView.contentInsetAdjustmentBehavior = .never

        // Mark as iPad app so CSS can target it.
        webView.evaluateJavaScript(
            "document.documentElement.classList.add('ipad-app')",
            completionHandler: nil
        )

        // Support pointer (trackpad/mouse) and keyboard on iPad.
        webView.allowsLinkPreview = true

        webView.load(URLRequest(url: url))
        return webView
    }

    func updateUIView(_ webView: WKWebView, context: Context) {
        // If the server URL changes (workspace switch), reload.
        if webView.url?.host != url.host || webView.url?.port != url.port {
            webView.load(URLRequest(url: url))
        }
    }

    class Coordinator: NSObject, WKNavigationDelegate {
        func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
            // Add iPad-specific class after each navigation.
            webView.evaluateJavaScript(
                "document.documentElement.classList.add('ipad-app')",
                completionHandler: nil
            )
        }

        func webView(
            _ webView: WKWebView,
            decidePolicyFor navigationAction: WKNavigationAction,
            decisionHandler: @escaping (WKNavigationActionPolicy) -> Void
        ) {
            // Open external links in Safari.
            if let url = navigationAction.request.url,
               url.host != "127.0.0.1" && url.host != "localhost" {
                UIApplication.shared.open(url)
                decisionHandler(.cancel)
                return
            }
            decisionHandler(.allow)
        }
    }
}
