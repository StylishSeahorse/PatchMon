import { describe, expect, it } from "vitest";
import {
	buildRecurrenceCron,
	DEFAULT_RECURRENCE,
	describeRecurrence,
	parseRecurrenceCron,
} from "../../utils/patchRecurrence";

describe("buildRecurrenceCron", () => {
	it("returns null when the policy does not repeat", () => {
		expect(buildRecurrenceCron(DEFAULT_RECURRENCE)).toBeNull();
	});

	it("builds daily, weekly and monthly expressions", () => {
		const base = { ...DEFAULT_RECURRENCE, time: "03:30" };
		expect(buildRecurrenceCron({ ...base, mode: "daily" })).toBe("30 3 * * *");
		expect(
			buildRecurrenceCron({ ...base, mode: "weekly", weekdays: ["6"] }),
		).toBe("30 3 * * 6");
		expect(
			buildRecurrenceCron({ ...base, mode: "monthly", monthday: "15" }),
		).toBe("30 3 15 * *");
	});

	it("builds a weekly expression for several days, sorted and de-duplicated", () => {
		expect(
			buildRecurrenceCron({
				...DEFAULT_RECURRENCE,
				mode: "weekly",
				time: "02:00",
				weekdays: ["5", "1", "3", "1"],
			}),
		).toBe("0 2 * * 1,3,5");
	});

	it("returns null for a weekly schedule with no day selected", () => {
		expect(
			buildRecurrenceCron({ ...DEFAULT_RECURRENCE, mode: "weekly" }),
		).toBeNull();
	});

	it("passes a custom expression through trimmed, or null when empty", () => {
		const custom = { ...DEFAULT_RECURRENCE, mode: "custom" };
		expect(buildRecurrenceCron({ ...custom, cron: " 0 4 * * 1-5 " })).toBe(
			"0 4 * * 1-5",
		);
		expect(buildRecurrenceCron({ ...custom, cron: "  " })).toBeNull();
	});

	it("returns null for an unparseable time", () => {
		expect(
			buildRecurrenceCron({ ...DEFAULT_RECURRENCE, mode: "daily", time: "" }),
		).toBeNull();
	});
});

describe("parseRecurrenceCron", () => {
	it("round-trips the presets", () => {
		for (const r of [
			{ ...DEFAULT_RECURRENCE, mode: "daily", time: "01:00" },
			{
				...DEFAULT_RECURRENCE,
				mode: "weekly",
				time: "23:05",
				weekdays: ["1", "3", "5"],
			},
			{ ...DEFAULT_RECURRENCE, mode: "monthly", time: "00:00", monthday: "28" },
		]) {
			expect(parseRecurrenceCron(buildRecurrenceCron(r))).toEqual(r);
		}
	});

	it("reads day lists and ranges as weekly", () => {
		expect(parseRecurrenceCron("0 3 * * 1-5")).toEqual({
			...DEFAULT_RECURRENCE,
			mode: "weekly",
			time: "03:00",
			weekdays: ["1", "2", "3", "4", "5"],
		});
		expect(parseRecurrenceCron("0 3 * * 0,6").weekdays).toEqual(["0", "6"]);
	});

	it("treats anything else as custom", () => {
		expect(parseRecurrenceCron("0 4 * 1 1-5")).toEqual({
			...DEFAULT_RECURRENCE,
			mode: "custom",
			cron: "0 4 * 1 1-5",
		});
		expect(parseRecurrenceCron("0 4 * * 5-1").mode).toBe("custom");
	});

	it("maps no schedule to the default form", () => {
		expect(parseRecurrenceCron(null)).toEqual(DEFAULT_RECURRENCE);
	});
});

describe("describeRecurrence", () => {
	it("describes presets with the timezone", () => {
		expect(describeRecurrence("0 1 * * *", "Europe/London")).toBe(
			"Every day at 01:00 (Europe/London)",
		);
		expect(describeRecurrence("0 3 * * 0", "UTC")).toBe(
			"Every Sunday at 03:00 (UTC)",
		);
		expect(describeRecurrence("0 3 * * 1,3,5", "UTC")).toBe(
			"Every Mon, Wed, Fri at 03:00 (UTC)",
		);
	});

	it("returns null when there is no schedule", () => {
		expect(describeRecurrence(null, "UTC")).toBeNull();
	});
});
