import { useState } from "react";
import { followUser, unfollowUser } from "../../api/social";
import { useUiStore } from "../../store/uiStore";
import Button from "../ui/Button";

interface Props {
  userId: string;
  initialFollowing?: boolean;
}

export default function FollowButton({
  userId,
  initialFollowing = false,
}: Props) {
  const [following, setFollowing] = useState(initialFollowing);
  const [loading, setLoading] = useState(false);
  const addToast = useUiStore((s) => s.addToast);

  const handleClick = async () => {
    setLoading(true);
    try {
      if (following) {
        await unfollowUser(userId);
        setFollowing(false);
      } else {
        await followUser(userId);
        setFollowing(true);
      }
    } catch {
      addToast(
        following ? "Failed to disconnect" : "Failed to connect",
        "error",
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <Button
      variant={following ? "secondary" : "primary"}
      size="sm"
      loading={loading}
      onClick={handleClick}
    >
      {following ? "Connected" : "Connect"}
    </Button>
  );
}
