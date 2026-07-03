import SwiftUI

enum Theme {
    static let bgDark = Color(red: 13/255, green: 12/255, blue: 10/255) // #0D0C0A
    static let bgDarker = Color(red: 7/255, green: 6/255, blue: 5/255) // #070605
    static let bgCard = Color(red: 18/255, green: 17/255, blue: 14/255).opacity(0.65) // rgba(18, 17, 14, 0.65)
    static let bgCardHover = Color(red: 26/255, green: 24/255, blue: 20/255).opacity(0.8) // rgba(26, 24, 20, 0.8)
    
    static let goldPrimary = Color(red: 229/255, green: 195/255, blue: 151/255) // #E5C397
    static let goldSecondary = Color(red: 248/255, green: 230/255, blue: 205/255) // #F8E6CD
    static let goldShadow = Color(red: 194/255, green: 153/255, blue: 101/255) // #C29965
    
    static let icePrimary = Color(red: 238/255, green: 220/255, blue: 196/255) // #EEDCC4
    static let iceSecondary = Color(red: 212/255, green: 178/255, blue: 133/255) // #D4B285
    
    static let textPrimary = Color(red: 245/255, green: 244/255, blue: 240/255) // #F5F4F0
    static let textSecondary = Color(red: 181/255, green: 174/255, blue: 165/255) // #B5AEA5
    static let textMuted = Color(red: 110/255, green: 104/255, blue: 96/255) // #6E6860
    
    static let borderGlass = Color.white.opacity(0.06)
    static let borderGlassHover = Color(red: 229/255, green: 195/255, blue: 151/255).opacity(0.22)
    
    // Ambient glows
    static let glowCyan = Color(red: 229/255, green: 195/255, blue: 151/255).opacity(0.04)
    static let glowCyanIntense = Color(red: 229/255, green: 195/255, blue: 151/255).opacity(0.12)
}

struct GlassCardModifier: ViewModifier {
    var isHovered: Bool = false
    
    func body(content: Content) -> some View {
        content
            .background(isHovered ? Theme.bgCardHover : Theme.bgCard)
            .cornerRadius(12)
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(isHovered ? Theme.borderGlassHover : Theme.borderGlass, lineWidth: 1.2)
            )
            .shadow(color: Color.black.opacity(0.4), radius: 10, x: 0, y: 5)
    }
}

extension View {
    func glassCard(isHovered: Bool = false) -> some View {
        self.modifier(GlassCardModifier(isHovered: isHovered))
    }
}

struct BlueprintBackground: View {
    var body: some View {
        ZStack {
            Theme.bgDark.ignoresSafeArea()
            
            // Grid background (24px cells matching symaira.com)
            GeometryReader { geo in
                Path { path in
                    let step: CGFloat = 24
                    
                    // Vertical lines
                    var x: CGFloat = 0
                    while x < geo.size.width {
                        path.move(to: CGPoint(x: x, y: 0))
                        path.addLine(to: CGPoint(x: x, y: geo.size.height))
                        x += step
                    }
                    
                    // Horizontal lines
                    var y: CGFloat = 0
                    while y < geo.size.height {
                        path.move(to: CGPoint(x: 0, y: y))
                        path.addLine(to: CGPoint(x: geo.size.width, y: y))
                        y += step
                    }
                }
                .stroke(Color.white.opacity(0.022), lineWidth: 1)
            }
            .ignoresSafeArea()
            
            // Glow Blobs
            VStack {
                HStack {
                    Circle()
                        .fill(Theme.glowCyanIntense)
                        .frame(width: 350, height: 350)
                        .blur(radius: 90)
                        .offset(x: -100, y: -100)
                    Spacer()
                }
                Spacer()
                HStack {
                    Spacer()
                    Circle()
                        .fill(Theme.glowCyan)
                        .frame(width: 450, height: 450)
                        .blur(radius: 120)
                        .offset(x: 150, y: 150)
                }
            }
            .ignoresSafeArea()
        }
    }
}
