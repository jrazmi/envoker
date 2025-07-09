// Import the generated route tree
import { routeTree } from "./routeTree.gen";
import { createRouter } from "@tanstack/react-router";
import { QueryClient } from "@tanstack/react-query";

export const queryClient = new QueryClient();

export const router = createRouter({
  routeTree,
  scrollRestoration: true,
  basepath: "/admin",
  context: {
    queryClient: queryClient!,
  },
  defaultPreload: "intent",
  // Since we're using React Query, we don't want loader calls to ever be stale
  // This will ensure that the loader is always called when the route is preloaded or visited
  defaultPreloadStaleTime: 0,

  //   defaultNotFoundComponent: NotFound,
});
// Register the router instance for type safety
declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
