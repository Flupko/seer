import { useInView } from "react-intersection-observer";


export default function InfiniteScrollContainer({ children, onBottomReached, className }: {
    children?: React.ReactNode,
    onBottomReached: () => void,
    className?: string,
}) {

    const { ref } = useInView({
        rootMargin: "50px",
        onChange: (inView) => {
            if (inView) {
                onBottomReached();
            }
        },
    });

    return (
        <div className={className}>
            {children}
            <div ref={ref} />
        </div>
    );

}
