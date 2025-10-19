
export default function MainWrapper({ children }: { children?: React.ReactNode }) {
    return (
        <div className="bg-grayscale-black h-[calc(100vh-4.75rem)] overflow-auto px-4 pt-5 md:pt-10 md:px-12 transition-all"
            style={{
                scrollbarColor: 'var(--color-gray-800) transparent',
                scrollbarWidth: 'thin',
            }}>
            <div className="max-w-7xl mx-auto">
                {children}
            </div>

        </div >
    )
}