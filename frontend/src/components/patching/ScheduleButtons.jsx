import {
	joinTime,
	sameWeekdays,
	splitTime,
	WEEKDAY_PRESETS,
	WEEKDAYS,
} from "../../utils/patchRecurrence";

const BUTTON =
	"rounded border px-2.5 py-1.5 text-sm transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500";
const ON =
	"border-primary-500 bg-primary-600 text-white dark:bg-primary-600 dark:text-white";
const OFF =
	"border-secondary-300 dark:border-secondary-600 bg-white dark:bg-secondary-700 text-secondary-700 dark:text-secondary-200 hover:bg-secondary-50 dark:hover:bg-secondary-600";

const MINUTES = [0, 15, 30, 45];
const HOURS = [12, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11];

function ToggleButton({ active, onClick, children, className = "" }) {
	return (
		<button
			type="button"
			aria-pressed={active}
			onClick={onClick}
			className={`${BUTTON} ${active ? ON : OFF} ${className}`}
		>
			{children}
		</button>
	);
}

function Legend({ children }) {
	return (
		<legend className="block text-sm font-medium text-secondary-700 dark:text-secondary-300 mb-1">
			{children}
		</legend>
	);
}

/** Day-of-week buttons with Every day / Weekdays / Weekends shortcuts. */
export function WeekdayButtons({ value, onChange }) {
	const toggle = (day) =>
		onChange(
			value.includes(day)
				? value.filter((d) => d !== day)
				: [...value, day].sort((a, b) => Number(a) - Number(b)),
		);
	const presets = [
		["Every day", WEEKDAY_PRESETS.everyDay],
		["Weekdays", WEEKDAY_PRESETS.weekdays],
		["Weekends", WEEKDAY_PRESETS.weekends],
	];
	return (
		<fieldset className="space-y-2">
			<Legend>Days</Legend>
			<div className="flex flex-wrap gap-2">
				{WEEKDAYS.map((name, i) => (
					<ToggleButton
						key={name}
						active={value.includes(String(i))}
						onClick={() => toggle(String(i))}
						className="min-w-[3rem]"
					>
						{name.slice(0, 3)}
					</ToggleButton>
				))}
			</div>
			<div className="flex flex-wrap gap-2">
				{presets.map(([label, days]) => (
					<ToggleButton
						key={label}
						active={sameWeekdays(value, days)}
						onClick={() => onChange([...days])}
						className="text-xs"
					>
						{label}
					</ToggleButton>
				))}
			</div>
		</fieldset>
	);
}

/** Day-of-month buttons, 1 to 28 so every month has the day. */
export function MonthdayButtons({ value, onChange }) {
	return (
		<fieldset>
			<Legend>Day of month</Legend>
			<div className="grid grid-cols-7 gap-1.5">
				{Array.from({ length: 28 }, (_, i) => String(i + 1)).map((day) => (
					<ToggleButton
						key={day}
						active={String(Number(value)) === day}
						onClick={() => onChange(day)}
						className="px-0"
					>
						{day}
					</ToggleButton>
				))}
			</div>
		</fieldset>
	);
}

/** Time-of-day buttons: hour, minute and AM/PM. value is 24-hour "HH:MM". */
export function TimeButtons({ value, onChange }) {
	const { hour12, minute, pm } = splitTime(value);
	// An existing schedule may use a minute the quick buttons do not offer.
	const minutes = MINUTES.includes(minute)
		? MINUTES
		: [...MINUTES, minute].sort((a, b) => a - b);
	return (
		<fieldset className="space-y-2">
			<Legend>Time</Legend>
			<div className="grid grid-cols-6 gap-1.5">
				{HOURS.map((h) => (
					<ToggleButton
						key={h}
						active={h === hour12}
						onClick={() => onChange(joinTime(h, minute, pm))}
						className="px-0"
					>
						{h}
					</ToggleButton>
				))}
			</div>
			<div className="flex flex-wrap items-center gap-2">
				{minutes.map((m) => (
					<ToggleButton
						key={m}
						active={m === minute}
						onClick={() => onChange(joinTime(hour12, m, pm))}
					>
						:{String(m).padStart(2, "0")}
					</ToggleButton>
				))}
				<span className="mx-1 h-5 w-px bg-secondary-300 dark:bg-secondary-600" />
				<ToggleButton
					active={!pm}
					onClick={() => onChange(joinTime(hour12, minute, false))}
				>
					AM
				</ToggleButton>
				<ToggleButton
					active={pm}
					onClick={() => onChange(joinTime(hour12, minute, true))}
				>
					PM
				</ToggleButton>
			</div>
		</fieldset>
	);
}
