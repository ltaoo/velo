function calendarDayInfo(date) {
  const fallback = {
    festivalLabel: "",
    holidayStatus: "",
    holidayBadge: "",
    lunarLabel: "",
    title: "",
  };
  if (!(date instanceof Date) || Number.isNaN(date.getTime())) return fallback;

  const api = calendarApi();
  if (!api.Solar) return fallback;

  const solar = api.Solar.fromYmd(date.getFullYear(), date.getMonth() + 1, date.getDate());
  const lunar = solar.getLunar();
  const key = formatCalendarDateKey(date);
  const holiday = api.HolidayUtil ? api.HolidayUtil.getHoliday(key) : null;
  const lunarFestival = firstText(lunar.getFestivals && lunar.getFestivals());
  const solarFestival = firstText(solar.getFestivals && solar.getFestivals());
  const jieQi = safeText(lunar.getJieQi && lunar.getJieQi());
  const lunarLabel = lunarDayLabel(lunar);
  const holidayName = holiday ? safeText(holiday.getName()) : "";
  const isWorkday = holiday ? Boolean(holiday.isWork()) : false;
  const holidayStatus = holiday ? (isWorkday ? "workday" : "holiday") : "";
  const festivalLabel = holidayName
    ? holidayName + (isWorkday ? "调休" : "")
    : lunarFestival || solarFestival || jieQi;
  const holidayTarget = holiday ? safeText(holiday.getTarget()) : "";
  const title = [lunarLabel, festivalLabel, holidayTarget && holidayTarget !== key ? "对应 " + holidayTarget : ""]
    .filter(Boolean)
    .join(" · ");

  return {
    festivalLabel,
    holidayStatus,
    holidayBadge: holiday ? (isWorkday ? "班" : "休") : "",
    lunarLabel,
    title,
  };
}

function calendarApi() {
  if (typeof window === "undefined") return {};
  return {
    HolidayUtil: window.HolidayUtil,
    Solar: window.Solar,
  };
}

function lunarDayLabel(lunar) {
  if (!lunar) return "";
  const day = safeText(lunar.getDayInChinese && lunar.getDayInChinese());
  if (!day) return "";
  if (day === "初一") {
    const month = safeText(lunar.getMonthInChinese && lunar.getMonthInChinese());
    return month ? month + "月" : day;
  }
  return day;
}

function firstText(values) {
  if (!Array.isArray(values) || values.length < 1) return "";
  return safeText(values[0]);
}

function safeText(value) {
  return value == null ? "" : String(value);
}

function formatCalendarDateKey(date) {
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${date.getFullYear()}-${month}-${day}`;
}

export { calendarDayInfo };
