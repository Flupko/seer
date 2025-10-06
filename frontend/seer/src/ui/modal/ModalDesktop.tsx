import { AnimatePresence, LayoutGroup, motion } from "motion/react"
import ToolTip from "../ToolTip"
import { X } from "lucide-react"
import { modalComponents, useModal } from "./Modal";
import { div } from "motion/react-client";


export default function ModalDesktop() {

    const { currentModal, modalData, closeModal } = useModal();
    const ModalContent = currentModal ? modalComponents[currentModal].content : null;
    const height = currentModal ? modalComponents[currentModal].height : null
    const width = currentModal ? modalComponents[currentModal].desktopWidth : null

    return (
        <LayoutGroup id={"layout-desktop"}>
            <AnimatePresence>

                {currentModal &&
                    <motion.div
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        transition={{ duration: 0.17 }}
                        className="fixed left-0 top-0 bottom-0 right-0 z-40 bg-black/25 flex justify-center items-center"
                        onClick={closeModal}
                    >

                        <motion.div className={`flex flex-col relative max-h-[calc(100vh-20px)] w-full ${width}`} key={currentModal + 'desktop'}
                            initial={{ opacity: 0, scale: 0.9, y: 10 }}
                            animate={{ opacity: 1, scale: 1, y: 0 }}
                            exit={{ opacity: 0, scale: 0.9 }}
                            transition={{ duration: 0.08, ease: 'easeInOut' }}
                            onClick={(e) => e.stopPropagation()}>

                            <div className="justify-end align-center top-2 z-50 right-1 absolute flex">
                                <ToolTip Icon={X} onClick={closeModal} />
                            </div>

                            <div className="overflow-auto">
                                <div className={`${height}`}>
                                    <div>


                                        {/* Scroller fills viewport, no vertical centering */}

                                        {/* Center horizontally with mx-auto, no items-center */}
                                        <motion.div

                                            className={`bg-gray-900 w-full ${height} rounded-lg relative mx-auto`}
                                        >

                                            {ModalContent && <ModalContent data={modalData} />}
                                        </motion.div>
                                    </div>



                                </div>
                            </div>

                        </motion.div>


                    </motion.div>}
            </AnimatePresence>
        </LayoutGroup >
    );
}