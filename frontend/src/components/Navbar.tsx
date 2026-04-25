import { Disclosure, DisclosureButton, DisclosurePanel, Menu, MenuButton, MenuItem, MenuItems } from "@headlessui/react"
import { Bars3Icon, XMarkIcon } from "@heroicons/react/24/outline"
import { Link, useLocation, useNavigate } from "react-router-dom"
import { useAuth } from "../auth/AuthContext"

const navigation = [
	{ name: "Home", href: "/" },
	{ name: "Analysis", href: "/analysis" },
	{ name: "Vocabulary", href: "/vocabulary" },
	{ name: "Dictionary", href: "/dictionary" },
	{ name: "jWiki", href: "https://ja.wikipedia.org/wiki/%E3%83%A1%E3%82%A4%E3%83%B3%E3%83%9A%E3%83%BC%E3%82%B8", external: true }
]

function classNames(...classes: string[]) {
	return classes.filter(Boolean).join(" ")
}

export default function Navbar() {
	const { user, logout } = useAuth()
	const navigate = useNavigate()
	const location = useLocation()

	const handleLogout = async () => {
		await logout()
		navigate("/login")
	}

	return (
		<Disclosure
			as="nav"
			className="bg-gray-100 text-gray-900">
			<div className="mx-auto max-w-6xl px-4">
				<div className="flex h-14 items-center gap-2">
					{/* Hamburger - mobile only */}
					<div className="sm:hidden">
						<DisclosureButton className="group inline-flex items-center justify-center rounded p-2 hover:bg-gray-200">
							<span className="sr-only">Open main menu</span>
							<Bars3Icon aria-hidden="true" className="block size-6 group-data-[open]:hidden" />
							<XMarkIcon aria-hidden="true" className="hidden size-6 group-data-[open]:block" />
						</DisclosureButton>
					</div>
					<div className="flex flex-1 items-center">
						<div className="flex shrink-0 items-center">
							<img alt="ObuCon" src="/android-chrome-192x192.png" className="h-7 w-auto" />
						</div>
						<div className="hidden sm:ml-6 sm:flex sm:items-center sm:gap-4">
							<div className="flex items-center gap-4">
								{navigation.map((item) => {
								const isCurrent = !item.external && (item.href === "/"
									? location.pathname === item.href
									: location.pathname.startsWith(item.href))
								if (item.external) {
									return (
										<a
											key={item.name}
											href={item.href}
											target="_blank"
											rel="noopener noreferrer"
											className="text-sm font-medium text-gray-600 hover:text-gray-800"
										>
											{item.name}
										</a>
									)
								}
									return (
										<Link
											key={item.name}
											to={item.href}
											aria-current={isCurrent ? "page" : undefined}
											className={classNames(
												isCurrent ? "text-black" : "text-gray-600 hover:text-gray-800",
												"text-sm font-medium"
											)}
										>
											{item.name}
										</Link>
									)
								})}
							</div>
						</div>
					</div>
					<div className="hidden sm:flex items-center sm:ml-6">
						{user ? (
							<Menu as="div" className="relative ml-3">
								<MenuButton className="relative flex rounded-full">
									<span className="absolute -inset-1.5" />
									<span className="sr-only">Open user menu</span>
									<img alt="" src="user.png" className="size-8 rounded-full" />
								</MenuButton>

								<MenuItems transition className="absolute right-0 z-10 mt-2 w-48 origin-top-right rounded-md bg-gray-800 py-1">
									<MenuItem>
										<Link to="/profile" className="block px-4 py-2 text-sm text-gray-100 hover:bg-gray-700">
											Your profile
										</Link>
									</MenuItem>
									<MenuItem>
										<Link to="/settings" className="block px-4 py-2 text-sm text-gray-100 hover:bg-gray-700">
											Settings
										</Link>
									</MenuItem>
									<MenuItem>
										<button
											type="button"
											onClick={handleLogout}
											className="block w-full px-4 py-2 text-left text-sm text-gray-100 hover:bg-gray-700"
										>
											Sign out
										</button>
									</MenuItem>
								</MenuItems>
							</Menu>
						) : (
							<div className="flex items-center gap-2">
								<Link to="/login" className="text-sm font-semibold text-gray-700 hover:text-gray-900">
									Login
								</Link>
								<Link
									to="/register"
									className="rounded-full border border-[#55F] px-3 py-1 text-sm font-semibold text-[#55F] hover:bg-[#55F] hover:text-white"
								>
									Register
								</Link>
							</div>
						)}
					</div>
				</div>
			</div>

			<DisclosurePanel className="sm:hidden">
				<div className="space-y-1 px-4 pb-3 pt-2">
					{navigation.map((item) => {
						const isCurrent = !item.external && (item.href === "/"
							? location.pathname === item.href
							: location.pathname.startsWith(item.href))
						if (item.external) {
							return (
								<a
									key={item.name}
									href={item.href}
									target="_blank"
									rel="noopener noreferrer"
									className="block py-2 text-base font-medium text-gray-900 hover:text-gray-800"
								>
									{item.name}
								</a>
							)
						}
						return (
							<DisclosureButton
								key={item.name}
								as={Link}
								to={item.href}
								aria-current={isCurrent ? "page" : undefined}
								className={classNames(
									isCurrent ? "text-black" : "text-gray-900 hover:text-gray-800",
									"block py-2 text-base font-medium"
								)}
							>
								{item.name}
							</DisclosureButton>
						)
					})}
					<div className="border-t border-gray-200 mt-2 pt-3">
						{user ? (
							<>
								<DisclosureButton
									as={Link}
									to="/profile"
									className="block py-2 text-base font-medium text-gray-900 hover:text-gray-800"
								>
									Your profile
								</DisclosureButton>
								<DisclosureButton
									as="button"
									type="button"
									onClick={handleLogout}
									className="block w-full text-left py-2 text-base font-medium text-gray-900 hover:text-gray-800"
								>
									Sign out
								</DisclosureButton>
							</>
						) : (
							<>
								<DisclosureButton
									as={Link}
									to="/login"
									className="block py-2 text-base font-medium text-gray-900 hover:text-gray-800"
								>
									Login
								</DisclosureButton>
								<DisclosureButton
									as={Link}
									to="/register"
									className="block py-2 text-base font-medium text-[#55F] hover:text-[#44E]"
								>
									Register
								</DisclosureButton>
							</>
						)}
					</div>
				</div>
			</DisclosurePanel>
		</Disclosure>
	)
}
