import './loader.css';

export default function Loader({ size = 1, color }: { size?: number, color?: string }) {
    return (
        <div className="loader"
            style={{
                width: (size * 20) + "px",
                border: (size * 2.3) + "px solid",
                borderColor: color ? color : "rgba(255, 255, 255, 0.5)",
            }}></div>
    )
}