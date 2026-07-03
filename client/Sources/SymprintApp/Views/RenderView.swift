import SwiftUI
import UniformTypeIdentifiers

struct RenderView: View {
    @State private var cliManager = CliManager.shared
    
    // Form Inputs
    @State private var selectedInputPath: String = ""
    @State private var selectedOutputPath: String = ""
    @State private var selectedProfile: String = "report"
    @State private var pdfStandard: String = ""
    @State private var fontPath: String = ""
    @State private var isReproducible: Bool = false
    
    // Status
    @State private var profiles: [CliProfile] = []
    @State private var isDragOver = false
    @State private var isRendering = false
    @State private var lastRenderedPdfUrl: URL?
    @State private var errorMessage: String?
    @State private var renderResult: RenderResult?
    @State private var showLogs = true
    
    var body: some View {
        HStack(spacing: 0) {
            // Configuration Panel (Left)
            VStack(alignment: .leading, spacing: 0) {
                ScrollView {
                    VStack(alignment: .leading, spacing: 20) {
                        // 1. File Selection
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Document")
                                .font(.system(size: 14, weight: .bold))
                                .foregroundStyle(Theme.textPrimary)
                            
                            FileDropZone(
                                path: $selectedInputPath,
                                isTargeted: $isDragOver
                            )
                        }
                        
                        // 2. Profile Selection
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Render Profile")
                                .font(.system(size: 14, weight: .bold))
                                .foregroundStyle(Theme.textPrimary)
                            
                            Picker("", selection: $selectedProfile) {
                                ForEach(profiles) { profile in
                                    Text(profile.title)
                                        .tag(profile.name)
                                }
                            }
                            .labelsHidden()
                            .pickerStyle(.menu)
                            
                            if let profile = profiles.first(where: { $0.name == selectedProfile }) {
                                Text(profile.description)
                                    .font(.caption)
                                    .foregroundStyle(Theme.textSecondary)
                                    .padding(.top, 2)
                            }
                        }
                        
                        // 3. Advanced Settings
                        DisclosureGroup("Advanced Parameters") {
                            VStack(alignment: .leading, spacing: 14) {
                                // Output Path Override
                                VStack(alignment: .leading, spacing: 4) {
                                    Text("Output Path (Optional)")
                                        .font(.caption.weight(.semibold))
                                        .foregroundStyle(Theme.textSecondary)
                                    HStack {
                                        TextField("Default: input.pdf", text: $selectedOutputPath)
                                            .textFieldStyle(.roundedBorder)
                                        Button("Browse…") {
                                            browseOutput()
                                        }
                                        .buttonStyle(.bordered)
                                    }
                                }
                                
                                // PDF Standard Override
                                VStack(alignment: .leading, spacing: 4) {
                                    Text("PDF Standard Override")
                                        .font(.caption.weight(.semibold))
                                        .foregroundStyle(Theme.textSecondary)
                                    TextField("e.g. a-2a,ua-1", text: $pdfStandard)
                                        .textFieldStyle(.roundedBorder)
                                }
                                
                                // Font Path Folder Picker
                                VStack(alignment: .leading, spacing: 4) {
                                    Text("Extra Fonts Directory")
                                        .font(.caption.weight(.semibold))
                                        .foregroundStyle(Theme.textSecondary)
                                    HStack {
                                        TextField("Path to font folder", text: $fontPath)
                                            .textFieldStyle(.roundedBorder)
                                        Button("Select…") {
                                            browseFonts()
                                        }
                                        .buttonStyle(.bordered)
                                    }
                                }
                                
                                // Reproducible checkbox
                                Toggle("Reproducible PDF output (SOURCE_DATE_EPOCH)", isOn: $isReproducible)
                                    .font(.caption)
                                    .foregroundStyle(Theme.textSecondary)
                            }
                            .padding(.top, 8)
                        }
                        .font(.system(size: 13, weight: .bold))
                        .foregroundStyle(Theme.textPrimary)
                        
                        // Render Button
                        VStack(spacing: 8) {
                            Button {
                                Task { await runRender() }
                            } label: {
                                HStack {
                                    if isRendering {
                                        ProgressView()
                                            .controlSize(.small)
                                            .padding(.trailing, 4)
                                    } else {
                                        Image(systemName: "bolt.fill")
                                    }
                                    Text("Render PDF")
                                        .fontWeight(.semibold)
                                }
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 8)
                            }
                            .buttonStyle(.borderedProminent)
                            .tint(Theme.goldPrimary)
                            .foregroundStyle(Theme.bgDarker)
                            .disabled(selectedInputPath.isEmpty || isRendering)
                            
                            if let error = errorMessage {
                                Text("Error: \(error)")
                                    .font(.caption)
                                    .foregroundStyle(.red)
                                    .multilineTextAlignment(.center)
                                    .padding(.top, 4)
                            }
                            
                            if let result = renderResult {
                                VStack(spacing: 2) {
                                    Text("✓ Render completed in \(result.durationMs)ms")
                                        .font(.caption.weight(.semibold))
                                        .foregroundStyle(.green)
                                    Text("\(String(format: "%.1f", Double(result.bytes)/1024.0)) kB · PDF standard \(result.pdfStandard?.joined(separator: ", ") ?? "Default")")
                                        .font(.system(size: 10))
                                        .foregroundStyle(Theme.textSecondary)
                                }
                                .padding(.top, 4)
                            }
                        }
                    }
                    .padding(20)
                }
                
                // Logs Panel (Bottom of Config Panel)
                if showLogs {
                    Divider()
                        .background(Theme.borderGlass)
                    
                    VStack(alignment: .leading, spacing: 0) {
                        HStack {
                            Text("Engine Logs")
                                .font(.caption.weight(.bold))
                                .foregroundStyle(Theme.textMuted)
                            Spacer()
                            Button("Clear") {
                                cliManager.clearLogs()
                            }
                            .buttonStyle(.plain)
                            .font(.caption2)
                            .foregroundStyle(Theme.textSecondary)
                        }
                        .padding(.horizontal, 16)
                        .padding(.vertical, 6)
                        .background(Color.black.opacity(0.15))
                        
                        ScrollViewReader { proxy in
                            ScrollView {
                                LazyVStack(alignment: .leading, spacing: 2) {
                                    ForEach(Array(cliManager.logs.enumerated()), id: \.offset) { idx, log in
                                        Text(log)
                                            .font(.system(size: 10, design: .monospaced))
                                            .foregroundStyle(colorForLog(log))
                                            .textSelection(.enabled)
                                            .id(idx)
                                    }
                                }
                                .padding(8)
                            }
                            .onChange(of: cliManager.logs.count) {
                                if let last = cliManager.logs.indices.last {
                                    proxy.scrollTo(last)
                                }
                            }
                        }
                        .frame(height: 120)
                        .background(Color.black.opacity(0.25))
                    }
                }
            }
            .frame(width: 380)
            
            Divider()
                .background(Theme.borderGlass)
            
            // Live PDF Preview Panel (Right)
            ZStack {
                Theme.bgDark.ignoresSafeArea()
                
                if let pdfUrl = lastRenderedPdfUrl {
                    PDFKitRepresentedView(url: pdfUrl)
                        .id(pdfUrl) // Force refresh view on path change
                } else {
                    VStack(spacing: 16) {
                        Image(systemName: "pdf")
                            .font(.system(size: 64))
                            .foregroundStyle(Theme.textMuted)
                        Text("Live PDF Preview")
                            .font(.headline)
                            .foregroundStyle(Theme.textSecondary)
                        Text("Select a document and click 'Render PDF' to see output.")
                            .font(.caption)
                            .foregroundStyle(Theme.textMuted)
                            .multilineTextAlignment(.center)
                            .padding(.horizontal, 48)
                    }
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
        .task {
            profiles = await cliManager.getProfiles()
            if let first = profiles.first {
                selectedProfile = first.name
            }
        }
    }
    
    // MARK: - Actions
    private func browseOutput() {
        let panel = NSSavePanel()
        panel.allowedContentTypes = [UTType.pdf]
        panel.canCreateDirectories = true
        panel.title = "Save PDF output"
        panel.nameFieldStringValue = selectedInputPath.isEmpty ? "output.pdf" : (URL(fileURLWithPath: selectedInputPath).deletingPathExtension().lastPathComponent + ".pdf")
        if panel.runModal() == .OK, let url = panel.url {
            selectedOutputPath = url.path
        }
    }
    
    private func browseFonts() {
        let panel = NSOpenPanel()
        panel.canChooseFiles = false
        panel.canChooseDirectories = true
        panel.allowsMultipleSelection = false
        panel.title = "Choose Extra Fonts Directory"
        if panel.runModal() == .OK, let url = panel.url {
            fontPath = url.path
        }
    }
    
    private func runRender() async {
        guard !selectedInputPath.isEmpty else { return }
        isRendering = true
        errorMessage = nil
        renderResult = nil
        
        let inPath = selectedInputPath
        let outPath = selectedOutputPath.isEmpty ?
            URL(fileURLWithPath: inPath).deletingPathExtension().appendingPathExtension("pdf").path :
            selectedOutputPath
            
        do {
            let res = try await cliManager.render(
                inputPath: inPath,
                outputPath: outPath,
                profile: selectedProfile,
                standard: pdfStandard.isEmpty ? nil : pdfStandard,
                reproducible: isReproducible,
                fontPath: fontPath.isEmpty ? nil : fontPath
            )
            renderResult = res
            // Trigger PDFView to update by modifying the URL slightly or re-instantiating URL object
            lastRenderedPdfUrl = nil
            try? await Task.sleep(for: .milliseconds(50))
            lastRenderedPdfUrl = URL(fileURLWithPath: outPath)
        } catch {
            errorMessage = error.localizedDescription
        }
        isRendering = false
    }
    
    private func colorForLog(_ log: String) -> Color {
        if log.contains("[error]") || log.contains("[stderr]") {
            return .red
        }
        if log.contains("[symprint]") {
            return Theme.goldPrimary
        }
        return Theme.textSecondary
    }
}

// File drop zone component
struct FileDropZone: View {
    @Binding var path: String
    @Binding var isTargeted: Bool
    
    var body: some View {
        Button {
            browseInput()
        } label: {
            VStack(spacing: 12) {
                if path.isEmpty {
                    Image(systemName: "square.and.arrow.down")
                        .font(.system(size: 32))
                        .foregroundStyle(isTargeted ? Theme.goldPrimary : Theme.textMuted)
                    
                    Text("Drag & Drop Markdown file here")
                        .font(.caption)
                        .foregroundStyle(Theme.textSecondary)
                    
                    Text("or click to browse files")
                        .font(.caption2)
                        .foregroundStyle(Theme.textMuted)
                } else {
                    let fileUrl = URL(fileURLWithPath: path)
                    Image(systemName: "doc.text.fill")
                        .font(.system(size: 32))
                        .foregroundStyle(Theme.goldPrimary)
                    
                    Text(fileUrl.lastPathComponent)
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(Theme.textPrimary)
                        .lineLimit(1)
                    
                    Text(path)
                        .font(.system(size: 9, design: .monospaced))
                        .foregroundStyle(Theme.textMuted)
                        .lineLimit(1)
                        .truncationMode(.middle)
                }
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 24)
            .padding(.horizontal, 16)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .glassCard(isHovered: isTargeted || !path.isEmpty)
        .overlay(
            RoundedRectangle(cornerRadius: 12)
                .stroke(isTargeted ? Theme.goldPrimary : Color.clear, lineWidth: 1.5)
        )
        .onDrop(of: [.fileURL], isTargeted: $isTargeted) { providers in
            guard let provider = providers.first else { return false }
            _ = provider.loadObject(ofClass: URL.self) { url, _ in
                if let url = url {
                    if url.pathExtension == "md" {
                        DispatchQueue.main.async {
                            self.path = url.path
                        }
                    }
                }
            }
            return true
        }
    }
    
    private func browseInput() {
        let panel = NSOpenPanel()
        panel.canChooseFiles = true
        panel.canChooseDirectories = false
        panel.allowsMultipleSelection = false
        panel.allowedContentTypes = [UTType(filenameExtension: "md")!]
        panel.title = "Select Markdown file"
        if panel.runModal() == .OK, let url = panel.url {
            path = url.path
        }
    }
}
