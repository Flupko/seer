export default function Tabs({ children }: { children: React.ReactNode }) {
    return (
        <div className="flex align-center justify-evenly h-8 border-b-[0.0625rem] border-gray-700">
            {children}
        </div>
    )
}