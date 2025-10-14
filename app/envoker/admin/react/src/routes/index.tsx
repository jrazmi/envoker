import { createFileRoute } from "@tanstack/react-router";
// import { useFoo } from "@/components/foo/foo";
import bar from "@/components/foo/foo";

export const Route = createFileRoute("/")({
  component: Index,
});

function Index() {
  const { foo } = bar();

  return (
    <div className="p-2">
      <h3>Welcome {foo}</h3>
    </div>
  );
}
