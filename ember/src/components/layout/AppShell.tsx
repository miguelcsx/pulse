import { Outlet } from "react-router-dom";
import Header from "./Header";
import BottomNav from "./BottomNav";
import ConnectionStatus from "../ui/ConnectionStatus";

export default function AppShell() {
  return (
    <div className="flex h-dvh flex-col overflow-hidden">
      <Header />
      <ConnectionStatus />
      <main
        className="flex-1 overflow-y-auto overscroll-y-contain"
        style={{ paddingBottom: "calc(4.5rem + env(safe-area-inset-bottom))" }}
      >
        <div className="mx-auto w-full max-w-xl px-4 py-4">
          <Outlet />
        </div>
      </main>
      <BottomNav />
    </div>
  );
}
