import React from 'react'

export default function Footer() {
    return (
        <footer className="border-t border-gray-200 bg-white">
            <div className="mx-auto max-w-6xl px-6 py-12">
                <div className="grid gap-10 md:grid-cols-[2fr_1fr_1fr]">
                    <div>
                        <div className="flex items-center gap-3">
							<img alt="ObuCon" src="/android-chrome-192x192.png" className="h-7 w-auto" />
                            <div className="text-lg font-semibold text-gray-900">ObuCon</div>
                        </div>
                        <p className="mt-4 max-w-md text-sm text-gray-600">
                            Read the right resources with ObuCon
                        </p>
                        <div className="mt-6 flex items-center gap-4 text-gray-500">
                            <a href="https://ojhdev.pythonanywhere.com/" className="text-sm hover:text-gray-900">Korean Prototype site</a>
                            <a href="https://www.linkedin.com/in/ohdev" className="text-sm hover:text-gray-900">LinkedIn</a>
                        </div>
                    </div>
                    <div>
                        <div className="text-xs font-semibold tracking-wide text-gray-400">ObuCon</div>
                        <ul className="mt-4 space-y-3 text-sm text-gray-700">
                            <li><a href="/about" className="hover:text-gray-900">About</a></li>
                            <li><a href="/features" className="hover:text-gray-900">Features</a></li>
                            <li><a href="/blog" className="hover:text-gray-900">Blog</a></li>
                            <li><a href="/contact" className="hover:text-gray-900">Contact</a></li>
                        </ul>
                    </div>
                    <div>
                        <div className="text-xs font-semibold tracking-wide text-gray-400">Other</div>
                        <ul className="mt-4 space-y-3 text-sm text-gray-700">
                            <li><a href="/feedback" className="hover:text-gray-900">Feedback</a></li>
                            <li><a href="/terms" className="hover:text-gray-900">Terms</a></li>
                            <li><a href="/privacy" className="hover:text-gray-900">Privacy</a></li>
                            <li><a href="/cookies" className="hover:text-gray-900">Cookies</a></li>
                        </ul>
                    </div>
                </div>

                <div className="mt-10 border-t border-gray-200 pt-6 text-center text-xs text-gray-500">
                    2026 ObuCon. All rights reserved.
                </div>
            </div>
        </footer>
    )
}
