import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import {
	MonthdayButtons,
	TimeButtons,
	WeekdayButtons,
} from "../../components/patching/ScheduleButtons";

const pressed = (name) =>
	screen.getByRole("button", { name }).getAttribute("aria-pressed") === "true";

describe("WeekdayButtons", () => {
	it("shows the selected days as pressed", () => {
		render(<WeekdayButtons value={["1", "3"]} onChange={() => {}} />);
		expect(pressed("Mon")).toBe(true);
		expect(pressed("Wed")).toBe(true);
		expect(pressed("Tue")).toBe(false);
	});

	it("adds a day in weekday order and removes a pressed one", () => {
		const onChange = vi.fn();
		const { rerender } = render(
			<WeekdayButtons value={["3", "5"]} onChange={onChange} />,
		);
		fireEvent.click(screen.getByRole("button", { name: "Mon" }));
		expect(onChange).toHaveBeenLastCalledWith(["1", "3", "5"]);

		rerender(<WeekdayButtons value={["3", "5"]} onChange={onChange} />);
		fireEvent.click(screen.getByRole("button", { name: "Wed" }));
		expect(onChange).toHaveBeenLastCalledWith(["5"]);
	});

	it("selects days with the shortcuts and highlights the matching one", () => {
		const onChange = vi.fn();
		render(
			<WeekdayButtons value={["1", "2", "3", "4", "5"]} onChange={onChange} />,
		);
		expect(pressed("Weekdays")).toBe(true);
		expect(pressed("Weekends")).toBe(false);
		fireEvent.click(screen.getByRole("button", { name: "Weekends" }));
		expect(onChange).toHaveBeenLastCalledWith(["0", "6"]);
		fireEvent.click(screen.getByRole("button", { name: "Every day" }));
		expect(onChange).toHaveBeenLastCalledWith([
			"0",
			"1",
			"2",
			"3",
			"4",
			"5",
			"6",
		]);
	});
});

describe("MonthdayButtons", () => {
	it("offers days 1 to 28 and reports the chosen one", () => {
		const onChange = vi.fn();
		render(<MonthdayButtons value="15" onChange={onChange} />);
		expect(screen.getAllByRole("button")).toHaveLength(28);
		expect(pressed("15")).toBe(true);
		fireEvent.click(screen.getByRole("button", { name: "28" }));
		expect(onChange).toHaveBeenLastCalledWith("28");
	});
});

describe("TimeButtons", () => {
	it("shows the current time as pressed buttons", () => {
		render(<TimeButtons value="13:15" onChange={() => {}} />);
		expect(pressed("1")).toBe(true);
		expect(pressed(":15")).toBe(true);
		expect(pressed("PM")).toBe(true);
		expect(pressed("AM")).toBe(false);
	});

	it("changes one part at a time, keeping the others", () => {
		const onChange = vi.fn();
		render(<TimeButtons value="01:00" onChange={onChange} />);
		fireEvent.click(screen.getByRole("button", { name: "3" }));
		expect(onChange).toHaveBeenLastCalledWith("03:00");
		fireEvent.click(screen.getByRole("button", { name: ":30" }));
		expect(onChange).toHaveBeenLastCalledWith("01:30");
		fireEvent.click(screen.getByRole("button", { name: "PM" }));
		expect(onChange).toHaveBeenLastCalledWith("13:00");
	});

	it("treats 12 AM as midnight and 12 PM as noon", () => {
		const onChange = vi.fn();
		render(<TimeButtons value="05:00" onChange={onChange} />);
		fireEvent.click(screen.getByRole("button", { name: "12" }));
		expect(onChange).toHaveBeenLastCalledWith("00:00");
	});

	it("keeps a stored minute the quick buttons do not offer", () => {
		render(<TimeButtons value="02:20" onChange={() => {}} />);
		expect(pressed(":20")).toBe(true);
	});
});
