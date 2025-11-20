export default function Heading({ className, children }: { className?: string, children: React.ReactNode }) {
    return (
        <h2 className={`md:text-xl text-lg font-bold text-white ${className}`}>
            {children}
        </h2>
    );
}