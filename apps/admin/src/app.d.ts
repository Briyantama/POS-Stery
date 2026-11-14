import type { Session } from "@stery/kit/sveltekit/session";

declare global {
  namespace App {
    interface Locals {
      session: Session | null;
    }
  }
}

export {};
