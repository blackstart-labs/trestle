import { fireEvent, render, screen } from "@testing-library/svelte";
import { expect, test, vi } from "vitest";
import MilestoneTimeline from "./MilestoneTimeline.svelte";
import type { Milestone } from "./MilestoneModal.svelte";

const mockMilestones: Milestone[] = [
  {
    id: "m-1",
    workspace_id: "ws-1",
    project_id: "proj-1",
    title: "Requirement Analysis & Architectural Blueprint",
    description: "Initial scoping and system architecture",
    start_date: "2026-10-01",
    due_date: "2026-11-15",
    end_date: "2026-11-15",
    color: "#9f302b",
    status: "open",
    total_tasks: 10,
    completed_tasks: 0,
    open_tasks: 10,
    created_at: "2026-10-01T00:00:00Z",
    updated_at: "2026-10-01T00:00:00Z",
  },
  {
    id: "m-2",
    workspace_id: "ws-1",
    project_id: "proj-1",
    title: "Database Schema (ERD) & Project Scaffolding",
    description: "PostgreSQL setup and migrations",
    start_date: "2026-11-10",
    due_date: "2026-12-31",
    end_date: "2026-12-31",
    color: "#9f302b",
    status: "open",
    total_tasks: 5,
    completed_tasks: 2,
    open_tasks: 3,
    created_at: "2026-10-15T00:00:00Z",
    updated_at: "2026-10-15T00:00:00Z",
  },
  {
    id: "m-3",
    workspace_id: "ws-1",
    project_id: "proj-1",
    title: "Authentication & Core Setup",
    description: "Argon2id and sessions",
    start_date: "2027-01-01",
    due_date: "2027-02-28",
    end_date: "2027-02-28",
    color: "#26633b",
    status: "open",
    total_tasks: 8,
    completed_tasks: 8,
    open_tasks: 0,
    created_at: "2026-11-01T00:00:00Z",
    updated_at: "2026-11-01T00:00:00Z",
  },
];

test("renders Gantt chart timeline with month headers and milestone bars", () => {
  const onedit = vi.fn();
  const onfilter = vi.fn();
  const oncreate = vi.fn();

  render(MilestoneTimeline, {
    milestones: mockMilestones,
    oneditmilestone: onedit,
    onfiltertasks: onfilter,
    oncreatemilestone: oncreate,
  });

  // Check month headers (October appears in 2026 and 2027)
  expect(screen.getAllByText("October").length).toBeGreaterThanOrEqual(1);
  expect(screen.getAllByText("November").length).toBeGreaterThanOrEqual(1);
  expect(screen.getByText("December")).toBeTruthy();
  expect(screen.getByText("January")).toBeTruthy();

  // Check milestone labels
  expect(
    screen.getAllByText("Requirement Analysis & Architectural Blueprint")
      .length,
  ).toBeGreaterThanOrEqual(1);
  expect(
    screen.getAllByText("Database Schema (ERD) & Project Scaffolding").length,
  ).toBeGreaterThanOrEqual(1);
  expect(
    screen.getAllByText("Authentication & Core Setup").length,
  ).toBeGreaterThanOrEqual(1);

  // Check completion percentages
  expect(screen.getByText("(0%) Done")).toBeTruthy();
  expect(screen.getByText("(40%) Done")).toBeTruthy();
  expect(screen.getByText("(100%) Done")).toBeTruthy();

  // Check count in toolbar
  expect(screen.getByText(/3 milestones/i)).toBeTruthy();
});

test("clicking on milestone bar opens edit modal", async () => {
  const onedit = vi.fn();
  const onfilter = vi.fn();
  const oncreate = vi.fn();

  render(MilestoneTimeline, {
    milestones: mockMilestones,
    oneditmilestone: onedit,
    onfiltertasks: onfilter,
    oncreatemilestone: oncreate,
  });

  const bar = screen.getByRole("button", {
    name: /milestone requirement analysis.*click to edit/i,
  });
  await fireEvent.click(bar);

  expect(onedit).toHaveBeenCalledWith(mockMilestones[0]);
});

test("clicking tasks button triggers task filter for milestone", async () => {
  const onedit = vi.fn();
  const onfilter = vi.fn();
  const oncreate = vi.fn();

  render(MilestoneTimeline, {
    milestones: mockMilestones,
    oneditmilestone: onedit,
    onfiltertasks: onfilter,
    oncreatemilestone: oncreate,
  });

  const taskBtn = screen.getByRole("button", {
    name: /view tasks for milestone requirement analysis/i,
  });
  await fireEvent.click(taskBtn);

  expect(onfilter).toHaveBeenCalledWith("m-1");
});

test("allows switching timeframe between month, quarter, and year", async () => {
  const onedit = vi.fn();
  const onfilter = vi.fn();
  const oncreate = vi.fn();

  render(MilestoneTimeline, {
    milestones: mockMilestones,
    oneditmilestone: onedit,
    onfiltertasks: onfilter,
    oncreatemilestone: oncreate,
  });

  const timeframeTrigger = screen.getByRole("button", {
    name: /timeframe/i,
  });
  await fireEvent.click(timeframeTrigger);

  const quarterOpt = screen.getByRole("option", {
    name: /timeframe: quarter/i,
  });
  await fireEvent.click(quarterOpt);

  // In quarter mode, labels show Q1, Q2, etc.
  expect(screen.getAllByText(/Q[1-4]/i).length).toBeGreaterThan(0);
});
