
export default function MainWrapper({ children }: { children: React.ReactNode }) {
    return (
        <div className="bg-black min-h-screen px-4.5 pt-5 md:pt-10 md:px-12 transition-all">
            <div className="max-w-7xl mx-auto">
                {children}
            </div>
            
        </div>
    )
}