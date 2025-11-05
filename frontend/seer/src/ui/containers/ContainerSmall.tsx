

export default function ContainerSmall({ children, className }: { children?: React.ReactNode, className?: string }) {
    return (
        <div className={`bg-gray-800 rounded-lg p-6 ${className}`}>
            {children}
        </div>
    )
}