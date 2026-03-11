import { Disclosure, DisclosureButton, DisclosurePanel, Menu, MenuButton, MenuItem, MenuItems } from "@headlessui/react"
import { Bars3Icon, XMarkIcon } from "@heroicons/react/24/outline"
import { Link, useLocation, useNavigate } from "react-router-dom"
import { useAuth } from "../auth/AuthContext"

const navigation = [
	{ name: "Home", href: "/" },
	{ name: "Analysis", href: "/analysis" },
	{ name: "Vocabulary", href: "/vocabulary" },
	{ name: "jWiki", href: "https://ja.wikipedia.org/wiki/%E3%83%A1%E3%82%A4%E3%83%B3%E3%83%9A%E3%83%BC%E3%82%B8"}
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
				<div className="flex h-14 items-center justify-between">
					<div className="absolute inset-y-0 left-0 flex items-center sm:hidden">
						{/* Mobile menu button*/}
						<DisclosureButton className="group inline-flex items-center justify-center rounded p-2 hover:bg-gray-800">
							<span className="absolute -inset-0.5" />
							<span className="sr-only">Open main menu</span>
							<Bars3Icon aria-hidden="true" className="block size-6 group-data-[open]:hidden" />
							<XMarkIcon aria-hidden="true" className="hidden size-6 group-data-[open]:block" />
						</DisclosureButton>
					</div>
					<div className="flex flex-1 items-center justify-center sm:items-stretch sm:justify-start">
						<div className="flex shrink-0 items-center">
							<img alt="ObuCon" src="/android-chrome-192x192.png" className="h-7 w-auto" />
						</div>
						<div className="hidden sm:ml-6 sm:block">
							<div className="flex items-center gap-4">
								{navigation.map((item) => {
									const isCurrent = item.href === "/"
										? location.pathname === item.href
										: location.pathname.startsWith(item.href)
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
					<div className="absolute inset-y-0 right-0 flex items-center pr-2 sm:static sm:inset-auto sm:ml-6 sm:pr-0">
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
						const isCurrent = item.href === "/"
							? location.pathname === item.href
							: location.pathname.startsWith(item.href)
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
				</div>
			</DisclosurePanel>
		</Disclosure>
	)
}
