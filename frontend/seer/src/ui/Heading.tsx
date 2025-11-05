export default function Heading({ className, children }: { className?: string, children: React.ReactNode }) {
    return (
        <h2 className={`md:text-[1.375rem] text-[1.125rem] font-bold text-white ${className}`}>
            {children}
        </h2>
    );
}