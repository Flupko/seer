import './loader.css';

export default function Loader({size = 1}: {size?: number}) {
    return (
        <div className="loader" 
        style={{width:(size * 20) + "px", 
            border: (size * 2.3) + "px solid",
            }}></div>
    )
}