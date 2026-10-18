// Recurring patch policies store a 5-field cron expression read in the
// organisation timezone. The form offers daily / weekly / monthly presets and
// falls back to a raw cron field for anything else the server accepts.

export const WEEKDAYS = [
	"Sunday",
	"Monday",
	"Tuesday",
	"Wednesday",
	"Thursday",
	"Friday",
	"Saturday",
];

const WEEKDAY_ABBREVIATIONS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

export const DEFAULT_RECURRENCE = {
	mode: "none",
	time: "01:00",
	weekdays: [],
	monthday: "1",
	cron: "",
};

export const WEEKDAY_PRESETS = {
	everyDay: ["0", "1", "2", "3", "4", "5", "6"],
	weekdays: ["1", "2", "3", "4", "5"],
	weekends: ["0", "6"],
};

/** True when two weekday lists hold the same days, in any order. */
export function sameWeekdays(a, b) {
	return a.length === b.length && a.every((d) => b.includes(d));
}

/** "HH:MM" (24-hour) as the 12-hour parts the time buttons show. */
export function splitTime(time) {
	const m = /^(\d{1,2}):(\d{2})$/.exec(time || "");
	const h24 = m ? Number(m[1]) : 1;
	const minute = m ? Number(m[2]) : 0;
	return { hour12: h24 % 12 === 0 ? 12 : h24 % 12, minute, pm: h24 >= 12 };
}

/** The 24-hour "HH:MM" for 12-hour parts; 12 AM is 00:00 and 12 PM is 12:00. */
export function joinTime(hour12, minute, pm) {
	const h24 = (hour12 % 12) + (pm ? 12 : 0);
	return `${String(h24).padStart(2, "0")}:${String(minute).padStart(2, "0")}`;
}

const pad = (n) => String(n).padStart(2, "0");

/** Cron expression for the recurrence form, or null when the policy does not repeat. */
export function buildRecurrenceCron(r) {
	if (!r || r.mode === "none") return null;
	if (r.mode === "custom") return r.cron.trim() || null;
	const [h, m] = r.time.split(":").map(Number);
	if (!Number.isInteger(h) || !Number.isInteger(m)) return null;
	if (r.mode === "daily") return `${m} ${h} * * *`;
	if (r.mode === "weekly") {
		const days = [...new Set(r.weekdays.map(Number))].sort((a, b) => a - b);
		return days.length > 0 ? `${m} ${h} * * ${days.join(",")}` : null;
	}
	if (r.mode === "monthly") return `${m} ${h} ${Number(r.monthday)} * *`;
	return null;
}

// Weekday numbers (as strings, ascending) for a cron day-of-week field made of
// single days, lists and ranges such as "1,3" or "1-5"; null for anything else.
function parseWeekdays(field) {
	const days = new Set();
	for (const part of field.split(",")) {
		const m = /^([0-6])(?:-([0-6]))?$/.exec(part);
		if (!m) return null;
		const from = Number(m[1]);
		const to = m[2] === undefined ? from : Number(m[2]);
		if (to < from) return null;
		for (let d = from; d <= to; d++) days.add(d);
	}
	return [...days].sort((a, b) => a - b).map(String);
}

/** Recurrence form values for a stored cron expression. */
export function parseRecurrenceCron(cron) {
	if (!cron) return { ...DEFAULT_RECURRENCE };
	const f = cron.trim().split(/\s+/);
	if (f.length === 5 && /^\d+$/.test(f[0]) && /^\d+$/.test(f[1])) {
		const time = `${pad(Number(f[1]))}:${pad(Number(f[0]))}`;
		const [dom, mon, dow] = f.slice(2);
		if (dom === "*" && mon === "*" && dow === "*")
			return { ...DEFAULT_RECURRENCE, mode: "daily", time };
		const weekdays = mon === "*" && dom === "*" ? parseWeekdays(dow) : null;
		if (weekdays)
			return { ...DEFAULT_RECURRENCE, mode: "weekly", time, weekdays };
		if (/^\d+$/.test(dom) && mon === "*" && dow === "*")
			return { ...DEFAULT_RECURRENCE, mode: "monthly", time, monthday: dom };
	}
	return { ...DEFAULT_RECURRENCE, mode: "custom", cron };
}

/** Short human description, e.g. "Every Sunday at 03:00 (Europe/London)". */
export function describeRecurrence(cron, timezone) {
	if (!cron) return null;
	const r = parseRecurrenceCron(cron);
	const tz = timezone ? ` (${timezone})` : "";
	if (r.mode === "daily") return `Every day at ${r.time}${tz}`;
	if (r.mode === "weekly") {
		const days =
			r.weekdays.length === 1
				? WEEKDAYS[Number(r.weekdays[0])]
				: r.weekdays.map((d) => WEEKDAY_ABBREVIATIONS[Number(d)]).join(", ");
		return `Every ${days} at ${r.time}${tz}`;
	}
	if (r.mode === "monthly")
		return `Monthly on day ${Number(r.monthday)} at ${r.time}${tz}`;
	return `Cron "${cron}"${tz}`;
}
