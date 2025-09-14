import Chat from "@/ui/Chat";
import Link from "next/link";

export default function Home() {
  return (
    <>

      <div className="flex flex-col items-center justify-center min-h-screen py-2">
        <button className="py-2 px-1 border hover:bg-red-200">
          <a href="localhost:4000/auth/google">Login with Google</a>
          </button>
      </div>
    </>
  );
}
