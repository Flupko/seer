

export default function ContainerSmall({ children }: { children?: React.ReactNode }) {
    return (
        <div className="bg-gray-800 rounded-md p-6">
            {children}
        </div>
    )
}