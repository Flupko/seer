

export default function PreferenceContainer({ children }: { children?: React.ReactNode }) {
    return (
        <div className="border-b border-b-gray-700 py-5
        [&:first-of-type]:pt-2
        [&:last-of-type]:pb-0
        [&:last-of-type]:border-b-transparent
        flex items-center gap-6">
            {children}
        </div>
    )
}