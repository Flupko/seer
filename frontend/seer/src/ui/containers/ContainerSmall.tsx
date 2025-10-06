

export default function ContainerSmall({ children }: { children?: React.ReactNode }) {
    return (
        <div className="px-5.5 bg-gray-900 border border-gray-600 rounded-lg py-4.5">
            {children}
        </div>
    )
}