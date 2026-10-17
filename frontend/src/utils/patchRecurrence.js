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

export const DEFAULT_RECURRENCE = {
	mode: "none",
	time: "01:00",
	weekday: "0",
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
	if (r.mode === "weekly") return `${m} ${h} * * ${Number(r.weekday)}`;
	if (r.mode === "monthly") return `${m} ${h} ${Number(r.monthday)} * *`;
	return null;
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
		if (dom === "*" && mon === "*" && /^[0-6]$/.test(dow))
			return { ...DEFAULT_RECURRENCE, mode: "weekly", time, weekday: dow };
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
	if (r.mode === "weekly")
		return `Every ${WEEKDAYS[Number(r.weekday)]} at ${r.time}${tz}`;
	if (r.mode === "monthly")
		return `Monthly on day ${Number(r.monthday)} at ${r.time}${tz}`;
	return `Cron "${cron}"${tz}`;
}
