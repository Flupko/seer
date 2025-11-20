

export default function Container({ title, children }: { title: string, children?: React.ReactNode }) {
    return (
        <div className="border border-gray-600 rounded-lg p-6">
            <h1 className="text-md font-bold mb-4 ml-0.5">{title}</h1>
            {children}
        </div>
    );
}