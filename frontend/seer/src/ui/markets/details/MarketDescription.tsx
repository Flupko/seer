"use client"

import { ChevronDown, ChevronUp } from "lucide-react";
import { useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";

export function MarketDescription({ description }: { description: string }) {
    const [isExpanded, setIsExpanded] = useState(false);

    return (
        <div className="relative">
            <div className={`relative ${!isExpanded ? 'max-h-24 overflow-hidden' : ''}`}>
                <ReactMarkdown
                    remarkPlugins={[remarkBreaks]}
                    components={{
                        h1: ({ node, ...props }) => <h1 className="text-2xl font-bold text-white mt-6 mb-3" {...props} />,
                        h2: ({ node, ...props }) => <h2 className="text-xl font-bold text-white mt-5 mb-2" {...props} />,
                        h3: ({ node, ...props }) => <h3 className="text-lg font-semibold text-white mt-4 mb-2" {...props} />,
                        h4: ({ node, ...props }) => <h4 className="text-base font-semibold text-white mt-3 mb-1" {...props} />,
                        a: ({ node, ...props }) => <a className="text-primary-blue hover:underline" {...props} />,
                        p: ({ node, ...props }) => <p className="text-gray-300 mb-3 leading-relaxed" {...props} />,
                        em: ({ node, ...props }) => <em className="italic text-gray-200" {...props} />,
                        u: ({ node, ...props }) => <u className="underline" {...props} />,
                        strong: ({ node, ...props }) => <strong className="font-bold text-white" {...props} />,
                        pre: ({ node, ...props }) => <pre className="bg-gray-900 p-3 rounded mb-3 overflow-x-auto" {...props} />,
                        code: ({ node, ...props }) => <code className="bg-gray-800 px-2 py-0.5 rounded text-primary-blue text-sm" {...props} />,
                        ul: ({ node, ...props }) => <ul className="list-disc ml-6 mb-3 space-y-1" {...props} />,
                        ol: ({ node, ...props }) => <ol className="list-decimal ml-6 mb-3 space-y-1" {...props} />,
                        li: ({ node, ...props }) => <li className="text-gray-300 leading-relaxed" {...props} />,
                        hr: ({ node, ...props }) => <hr className="border-gray-700 my-4" {...props} />,
                        blockquote: ({ node, ...props }) => <blockquote className="border-l-4 border-primary-blue pl-4 italic text-gray-400 my-3" {...props} />,
                        br: ({ node, ...props }) => <br className="my-1" {...props} />,
                    }}
                >
                    {description}
                </ReactMarkdown>

                {/* Gradient overlay when collapsed - darker, faster fade */}
                {!isExpanded && (
                    <div className="absolute bottom-0 left-0 right-0 h-20 bg-gradient-to-t from-grayscale-black to-transparent pointer-events-none" />
                )}
            </div>

            {/* Show More/Less Button - positioned below content */}

            <div className={`flex ${isExpanded ? "mt-8" : "mt-4"}`}>
                <button
                    onClick={() => setIsExpanded(!isExpanded)}
                    className="text-sm font-bold text-primary-blue cursor-pointer flex items-center gap-1"
                >
                    {isExpanded ? <>
                        Show Less <ChevronUp size={16} strokeWidth={2.5} />
                    </>
                        :
                        <>
                            Show More <ChevronDown size={16} strokeWidth={2.5} />
                        </>}
                </button>
            </div>
        </div>
    );
}
