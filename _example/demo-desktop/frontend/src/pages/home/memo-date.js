export const CALENDAR_WEEKDAYS = ["日", "一", "二", "三", "四", "五", "六"];

function normalizeCalendarWeekStart(value) {
  return value === "sunday" ? "sunday" : "monday";
}

function calendarWeekStartIndex(value) {
  return normalizeCalendarWeekStart(value) === "sunday" ? 0 : 1;
}

function calendarWeekdays(value) {
  const startIndex = calendarWeekStartIndex(value);
  return CALENDAR_WEEKDAYS.slice(startIndex).concat(CALENDAR_WEEKDAYS.slice(0, startIndex));
}

function formatRelativeDate(value) {
  const date = new Date(value);
  const diff = Date.now() - date.getTime();
  const minute = 1000 * 60;
  const hour = minute * 60;
  const day = hour * 24;
  if (diff < minute) return "刚刚";
  if (diff < hour) return `${Math.floor(diff / minute)} 分钟前`;
  if (diff < day) return `${Math.floor(diff / hour)} 小时前`;
  if (diff < day * 7) return `${Math.floor(diff / day)} 天前`;
  return date.toLocaleDateString();
}

function formatShortDate(value) {
  const date = new Date(value);
  return `${date.getMonth() + 1}/${date.getDate()}`;
}

function addMonths(date, delta) {
  const value = new Date(date);
  return startOfMonth(new Date(value.getFullYear(), value.getMonth() + delta, 1));
}

function startOfMonth(date) {
  const value = new Date(date);
  return new Date(value.getFullYear(), value.getMonth(), 1);
}

function generateCalendarDays(monthDate, weekStart) {
  const month = startOfMonth(monthDate);
  const start = new Date(month);
  const startDay = calendarWeekStartIndex(weekStart);
  const offset = (start.getDay() - startDay + 7) % 7;
  start.setDate(start.getDate() - offset);

  return Array.from({ length: 42 }, function (_value, index) {
    const date = new Date(start);
    date.setDate(start.getDate() + index);
    return {
      date,
      inMonth: date.getMonth() === month.getMonth(),
      key: formatDateKey(date),
    };
  });
}

function memoDateCounts(memos) {
  const counts = new Map();
  memos.forEach((memo) => {
    if (memo.archived) return;
    const key = memoDateKey(memo);
    if (!key) return;
    counts.set(key, (counts.get(key) || 0) + 1);
  });
  return counts;
}

function memoDateKey(memo) {
  const value = memo && memo.createdAt;
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return formatDateKey(date);
}

function formatDateKey(date) {
  const value = new Date(date);
  const year = value.getFullYear();
  const month = String(value.getMonth() + 1).padStart(2, "0");
  const day = String(value.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function dateFromKey(value) {
  const match = String(value || "").match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (!match) return new Date();
  return new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]));
}

export {
  addMonths,
  calendarWeekdays,
  dateFromKey,
  formatDateKey,
  formatRelativeDate,
  formatShortDate,
  generateCalendarDays,
  memoDateCounts,
  memoDateKey,
  normalizeCalendarWeekStart,
  startOfMonth,
};
