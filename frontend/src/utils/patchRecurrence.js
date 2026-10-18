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
