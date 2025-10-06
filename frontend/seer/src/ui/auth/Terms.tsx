import Link from "next/link"

export default function Terms() {
    return (<p className="text-gray-100 text-xs mt-1">By signing up you agree to the
        <Link href="" className="font-extrabold"> Terms</Link> &
        <Link href="" className="font-extrabold"> Privacy Policy</Link>
    </p>)
}