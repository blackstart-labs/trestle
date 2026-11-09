import { render, fireEvent } from "@testing-library/svelte";
import { expect, test } from "vitest";
import Avatar from "./Avatar.svelte";

test("renders initials with identity hue when no avatar URL is provided", () => {
  const { container } = render(Avatar, {
    props: {
      id: "u-123",
      name: "Alice Smith",
    },
  });

  const avatar = container.querySelector(".avatar");
  expect(avatar).toBeTruthy();
  expect(avatar?.textContent?.trim()).toBe("AS");
  expect(container.querySelector("img")).toBeNull();
});

test("renders image with avatar URL when provided", () => {
  const { container } = render(Avatar, {
    props: {
      id: "u-123",
      name: "Alice Smith",
      avatarUrl: "/api/v1/users/u-123/avatar",
    },
  });

  const img = container.querySelector("img");
  expect(img).toBeTruthy();
  expect(img?.getAttribute("src")).toBe("/api/v1/users/u-123/avatar");
});

test("falls back cleanly to initials on image load failure", async () => {
  const { container } = render(Avatar, {
    props: {
      id: "u-123",
      name: "Alice Smith",
      avatarUrl: "/api/v1/users/u-123/avatar",
    },
  });

  const img = container.querySelector("img");
  expect(img).toBeTruthy();

  await fireEvent.error(img!);
  expect(container.querySelector("img")).toBeNull();
  expect(container.querySelector(".avatar-initials")?.textContent?.trim()).toBe(
    "AS",
  );
});

test("applies size variants correctly", () => {
  const { container: smContainer } = render(Avatar, {
    props: { id: "1", name: "Bob", size: "sm" },
  });
  expect(smContainer.querySelector(".avatar-sm")).toBeTruthy();

  const { container: lgContainer } = render(Avatar, {
    props: { id: "1", name: "Bob", size: "lg" },
  });
  expect(lgContainer.querySelector(".avatar-lg")).toBeTruthy();
});
