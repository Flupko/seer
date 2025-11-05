
export default function MainWrapper({ children }: { children?: React.ReactNode }) {
    return (
        <div className="px-4 md:px-12 transition-all">
            <div className="max-w-7xl mx-auto">
                {children}
            </div>

        </div >
    )
}