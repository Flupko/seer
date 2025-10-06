
export default function DrawerClose({ onClick }: { onClick: () => void }) {
    return (
        <div className="bg-gray-900 h-3 z-50 absolute flex justify-center items-center w-full cursor-pointer py-4 md:hidden">
            <span className="w-7.5 h-1 rounded-3xl cursor-grab bg-gray-600 active:bg-gray-500" onClick={onClick}></span>
        </div>
    )
}