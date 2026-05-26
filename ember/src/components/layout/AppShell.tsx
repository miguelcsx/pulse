import { Outlet, useLocation } from "react-router-dom";
import Header from "./Header";
import BottomNav from "./BottomNav";
import ConnectionStatus from "../ui/ConnectionStatus";
import ContextSheet from "../feed/ContextSheet";

export default function AppShell() {
  const location = useLocation();
  const isFeedRoute = location.pathname === "/moments";

  return (
    <div className="flex h-dvh flex-col overflow-hidden">
      <Header />
      <ConnectionStatus />
      <main
        className={`flex-1 overflow-y-auto overscroll-y-contain ${
          isFeedRoute ? "snap-y snap-mandatory" : ""
        }`}
        style={{ paddingBottom: "calc(4.5rem + env(safe-area-inset-bottom))" }}
      >
        <div
          className={`mx-auto w-full max-w-xl ${
            isFeedRoute ? "px-0 py-0 sm:px-4 sm:py-4" : "px-4 py-4"
          }`}
        >
          <Outlet />
        </div>
      </main>
      <ContextSheet />
      <BottomNav />
    </div>
  );
}
