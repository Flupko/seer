import { X } from "lucide-react";
import Input from "../Input";
import ToolTip from "../ToolTip";
import { useModal } from "../modal/Modal";
import Tabs from "../Tabs/Tabs";
import Tab from "../Tabs/Tab";
import { useState } from "react";
import Button from "../Button";
import Register from "./Register";
import DrawerClose from "../DrawerClose";
import Login from "./Login";


export default function AuthModal({ data: seletedTab }: { data: 'login' | 'register' }) {

    const [activeTab, setActiveTab] = useState<'login' | 'register'>(seletedTab);

    return (
        <div className="flex flex-col gap-10 px-9 lg:px-9.5 pb-8 md:pt-11 pt-6 w-full">
            <Tabs>
                <Tab isSelected={activeTab == 'register'} onClick={() => { setActiveTab('register') }}>Sign Up</Tab>
                <Tab isSelected={activeTab == 'login'} onClick={() => { setActiveTab('login') }}>Login</Tab>
            </Tabs>

            {activeTab == "register" ? <Register /> : <Login />}
            
        </div>
    )
}


// import { X } from "lucide-react";
// import Input from "../Input";
// import ToolTip from "../ToolTip";
// import { useModal } from "@/app/Modal";
// import Tabs from "../Tabs/Tabs";
// import Tab from "../Tabs/Tab";
// import { useState } from "react";
// import Button from "../Button";
// import Register from "./Register";
// import DrawerClose from "../DrawerClose";


// export default function AuthModal({ data: seletedTab }: { data: 'login' | 'register' }) {

//     const { currentModal, modalData, closeModal } = useModal();
//     const [activeTab, setActiveTab] = useState<'login' | 'register'>(seletedTab);

//     return (
//         <div className="w-full md:w-screen md:max-w-lg max-h-full flex flex-col">
//             <div className="w-full bg-gray-900 md:rounded-lg shrink-0 relative flex flex-col max-h-full">

//                 <div className="justify-end align-center top-2 right-1 absolute hidden md:flex">
//                     <ToolTip Icon={X} onClick={closeModal} />
//                 </div>

//                 <DrawerClose onClick={closeModal}/>
 
//                 <div className="flex flex-col gap-10 px-10 pb-8 pt-11 overflow-y-scroll scrollbar-hide max-h-[">
//                     <Tabs>
//                         <Tab isSelected={activeTab == 'register'} onClick={() => { setActiveTab('register') }}>Sign Up</Tab>
//                         <Tab isSelected={activeTab == 'login'} onClick={() => { setActiveTab('login') }}>Login</Tab>
//                     </Tabs>

//                     <Register />
//                 </div>

//             </div>
//         </div>
//     )
// }
