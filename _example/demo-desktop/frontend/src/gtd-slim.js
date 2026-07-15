import {
  completeTask,
  createTask,
  loadTasks,
  normalizeTaskSummary,
  updateTask,
} from "./domain/tasks.js";
import { errorMessage } from "./domain/memo-repository.js";
import { SVG } from "./pages/home/memo-icons.js";
import { closestElement, escapeAttr, escapeHTML } from "./pages/home/memo-utils.js";
import { registerWindowSession, setPersistedWindowFixed } from "./window-state.js";

const FILTER_STORAGE_KEY = "demo-desktop:gtd:task-filter:v1";
const FILTERS = [
  ["today", "今天"],
  ["overdue", "过期"],
  ["inbox", "Inbox"],
  ["next", "下一步"],
  ["scheduled", "计划"],
  ["all", "全部"],
  ["completed", "已完成"],
];
const KNOWN_FILTERS = new Set(FILTERS.map((item) => item[0]));

document.addEventListener("DOMContentLoaded", function () {
  const root = document.querySelector("#root");
  if (!root) {
    console.error("[GTDSlim] Root element not found");
    return;
  }
  mountGTDSlim(root);
});

function mountGTDSlim(root) {
  const params = new URLSearchParams(window.location.search);
  const state = {
    error: "",
    filter: normalizeFilter(localStorage.getItem(FILTER_STORAGE_KEY)),
    fixed: params.get("fixed") === "1",
    loading: false,
    saving: false,
    tasks: [],
    toastTimer: null,
  };

  registerWindowSession({
    entryPage: "gtd-slim.html",
    fixed: state.fixed,
    title: "Todos",
  });

  root.innerHTML = slimTemplate();

  const els = {
    count: root.querySelector("[data-gtd-slim-count]"),
    dueInput: root.querySelector("[data-gtd-slim-due]"),
    fixedButton: root.querySelector('[data-window-control="toggleFixed"]'),
    form: root.querySelector("[data-gtd-slim-form]"),
    list: root.querySelector("[data-gtd-slim-list]"),
    prioritySelect: root.querySelector("[data-gtd-slim-priority]"),
    submit: root.querySelector("[data-gtd-slim-submit]"),
    tabs: root.querySelector("[data-gtd-slim-tabs]"),
    titleInput: root.querySelector("[data-gtd-slim-title]"),
    toast: root.querySelector("[data-toast]"),
  };

  root.addEventListener("click", function (event) {
    const control = closestElement(event.target, "[data-window-control]");
    if (control && root.contains(control)) {
      runWindowControl(control.dataset.windowControl);
      return;
    }

    const filterButton = closestElement(event.target, "[data-gtd-slim-filter]");
    if (!filterButton || !root.contains(filterButton)) return;
    setFilter(filterButton.dataset.gtdSlimFilter);
  });

  root.addEventListener("change", function (event) {
    if (!event.target.matches("[data-gtd-slim-complete]")) return;
    const taskNode = closestElement(event.target, "[data-gtd-slim-task-id]");
    if (!taskNode) return;
    toggleTaskCompletion(taskNode.dataset.gtdSlimTaskId, event.target);
  });

  root.addEventListener("click", function (event) {
    const bellBtn = closestElement(event.target, "[data-gtd-slim-reminder-btn]");
    if (bellBtn && root.contains(bellBtn)) {
      const taskNode = closestElement(bellBtn, "[data-gtd-slim-task-id]");
      if (taskNode) toggleReminderPopover(taskNode.dataset.gtdSlimTaskId, bellBtn);
      return;
    }
    const quickOpt = closestElement(event.target, "[data-reminder-quick]");
    if (quickOpt && root.contains(quickOpt)) {
      const minutes = parseInt(quickOpt.dataset.reminderQuick, 10);
      const popover = closestElement(quickOpt, "[data-reminder-popover]");
      const taskId = popover ? popover.dataset.reminderPopover : "";
      if (taskId && !isNaN(minutes)) addRelativeReminder(taskId, minutes);
      return;
    }
    const deleteBtn = closestElement(event.target, "[data-reminder-delete]");
    if (deleteBtn && root.contains(deleteBtn)) {
      const popover = closestElement(deleteBtn, "[data-reminder-popover]");
      const taskId = popover ? popover.dataset.reminderPopover : "";
      const idx = parseInt(deleteBtn.dataset.reminderDelete, 10);
      if (taskId && !isNaN(idx)) deleteReminder(taskId, idx);
      return;
    }
    const absBtn = closestElement(event.target, "[data-reminder-abs-confirm]");
    if (absBtn && root.contains(absBtn)) {
      const popover = closestElement(absBtn, "[data-reminder-popover]");
      const taskId = popover ? popover.dataset.reminderPopover : "";
      const input = popover ? popover.querySelector("[data-reminder-abs-input]") : null;
      if (taskId && input && input.value) addAbsoluteReminder(taskId, input.value);
      return;
    }
    const editCompletedBtn = closestElement(event.target, "[data-gtd-slim-edit-completed]");
    if (editCompletedBtn && root.contains(editCompletedBtn)) {
      const taskNode = closestElement(editCompletedBtn, "[data-gtd-slim-task-id]");
      if (taskNode) editCompletedAtInline(taskNode.dataset.gtdSlimTaskId, editCompletedBtn);
      return;
    }
    // Close open popovers on outside click.
    const openPopover = root.querySelector("[data-reminder-popover]");
    if (openPopover && !openPopover.contains(event.target)) {
      openPopover.remove();
    }
  });

  function toggleReminderPopover(taskId, anchor) {
    const existing = root.querySelector("[data-reminder-popover]");
    if (existing) {
      existing.remove();
      if (existing.dataset.reminderPopover === taskId) return;
    }
    const task = state.tasks.find((t) => t.id === taskId);
    if (!task) return;
    const popover = document.createElement("div");
    popover.className = "gtd-slim-reminder-popover";
    popover.setAttribute("data-reminder-popover", taskId);
    popover.innerHTML = reminderPopoverContent(task);
    anchor.parentElement.appendChild(popover);
  }

  function addRelativeReminder(taskId, minutes) {
    const task = state.tasks.find((t) => t.id === taskId);
    if (!task) return;
    const reminders = (task.reminders || []).concat({
      type: "relative",
      base: "dueAt",
      offsetMinutes: minutes,
    });
    updateTask(taskId, { reminders }).then(function (updated) {
      const summary = normalizeTaskSummary(updated);
      if (summary) state.tasks = state.tasks.map((t) => t.id === taskId ? summary : t);
      closeReminderPopover();
      showToast("已添加提醒");
      render();
    }, function (err) {
      showToast("设置提醒失败: " + errorMessage(err));
    });
  }

  function addAbsoluteReminder(taskId, datetimeValue) {
    const task = state.tasks.find((t) => t.id === taskId);
    if (!task) return;
    const at = new Date(datetimeValue).toISOString();
    const reminders = (task.reminders || []).concat({
      type: "absolute",
      at,
    });
    updateTask(taskId, { reminders }).then(function (updated) {
      const summary = normalizeTaskSummary(updated);
      if (summary) state.tasks = state.tasks.map((t) => t.id === taskId ? summary : t);
      closeReminderPopover();
      showToast("已添加提醒");
      render();
    }, function (err) {
      showToast("设置提醒失败: " + errorMessage(err));
    });
  }

  function deleteReminder(taskId, index) {
    const task = state.tasks.find((t) => t.id === taskId);
    if (!task || !task.reminders) return;
    const reminders = task.reminders.filter((_, i) => i !== index);
    updateTask(taskId, { reminders }).then(function (updated) {
      const summary = normalizeTaskSummary(updated);
      if (summary) state.tasks = state.tasks.map((t) => t.id === taskId ? summary : t);
      closeReminderPopover();
      showToast("已删除提醒");
      render();
    }, function (err) {
      showToast("删除提醒失败: " + errorMessage(err));
    });
  }

  function closeReminderPopover() {
    const existing = root.querySelector("[data-reminder-popover]");
    if (existing) existing.remove();
  }

  els.form.addEventListener("submit", function (event) {
    event.preventDefault();
    createTodo();
  });

  els.titleInput.addEventListener("keydown", function (event) {
    if (event.key !== "Enter" || event.isComposing) return;
    event.preventDefault();
    createTodo();
  });

  window.addEventListener("focus", function () {
    refreshTasks({ silent: true });
  });

  applyFixedState();
  render();
  refreshTasks();
  window.requestAnimationFrame(function () {
    els.titleInput.focus();
  });

  function refreshTasks(options = {}) {
    state.loading = true;
    if (!options.silent) render();
    loadTasks().then(
      function (payload) {
        state.error = "";
        state.tasks = payload.tasks.map(normalizeTaskSummary).filter(Boolean);
      },
      function (err) {
        state.error = "读取代办失败: " + errorMessage(err);
      },
    ).finally(function () {
      state.loading = false;
      render();
    });
  }

  function createTodo() {
    if (state.saving) return;
    const title = String(els.titleInput.value || "").trim();
    if (!title) {
      els.titleInput.focus();
      return;
    }

    state.saving = true;
    els.submit.disabled = true;

    const dueAt = String(els.dueInput.value || "").trim() || defaultDueForFilter(state.filter);
    const payload = {
      dueAt,
      listId: state.filter === "inbox" ? "inbox" : "",
      priority: String(els.prioritySelect.value || "none").trim(),
      title,
    };

    createTask(payload).then(
      function (task) {
        const summary = normalizeTaskSummary(task);
        if (summary) state.tasks = [summary].concat(state.tasks.filter((item) => item.id !== summary.id));
        state.error = "";
        els.form.reset();
        render();
        window.requestAnimationFrame(function () {
          els.titleInput.focus();
        });
      },
      function (err) {
        state.error = "添加代办失败: " + errorMessage(err);
        render();
      },
    ).finally(function () {
      state.saving = false;
      els.submit.disabled = false;
    });
  }

  function toggleTaskCompletion(taskId, checkbox) {
    const id = String(taskId || "").trim();
    if (!id || !checkbox) return;
    const checked = checkbox.checked;
    checkbox.disabled = true;
    const request = checked
      ? completeTask(id)
      : updateTask(id, { completedAt: "", status: "open" });

    request.then(
      function (task) {
        const summary = normalizeTaskSummary(task);
        if (summary) {
          state.tasks = state.tasks.map((item) => item.id === id ? summary : item);
        }
        render();
      },
      function (err) {
        checkbox.checked = !checked;
        checkbox.disabled = false;
        showToast((checked ? "完成失败: " : "恢复失败: ") + errorMessage(err));
      },
    );
  }

  function editCompletedAtInline(taskId, button) {
    var currentValue = button.getAttribute("datetime") || "";
    var localValue = taskDateTimeLocalValue(currentValue);

    var wrapper = document.createElement("span");
    wrapper.className = "gtd-slim-completed-time-edit";
    wrapper.innerHTML = '<input type="datetime-local" class="gtd-slim-completed-time-input" value="' + escapeAttr(localValue) + '" />' +
      '<button type="button" class="gtd-slim-completed-time-confirm" title="确认">' + SVG.check + '</button>' +
      '<button type="button" class="gtd-slim-completed-time-cancel" title="取消">' + SVG.x + '</button>';

    button.replaceWith(wrapper);
    var input = wrapper.querySelector("input");
    input.focus();

    function save() {
      var newValue = input.value;
      if (!newValue) return;
      updateTask(taskId, { completedAt: new Date(newValue).toISOString() }).then(function (updated) {
        var summary = normalizeTaskSummary(updated);
        if (summary) state.tasks = state.tasks.map(function (t) { return t.id === taskId ? summary : t; });
        render();
        showToast("完成时间已更新");
      }, function (err) {
        showToast("更新失败: " + errorMessage(err));
        wrapper.replaceWith(button);
      });
    }

    function cancel() {
      wrapper.replaceWith(button);
    }

    wrapper.querySelector(".gtd-slim-completed-time-confirm").addEventListener("click", save);
    wrapper.querySelector(".gtd-slim-completed-time-cancel").addEventListener("click", cancel);

    input.addEventListener("keydown", function (e) {
      if (e.key === "Enter") { e.preventDefault(); save(); }
      if (e.key === "Escape") { e.preventDefault(); cancel(); }
    });

    input.addEventListener("blur", function () {
      setTimeout(function () {
        if (root.contains(wrapper)) cancel();
      }, 150);
    });
  }

  function setFilter(filter) {
    const nextFilter = normalizeFilter(filter);
    if (state.filter === nextFilter) return;
    state.filter = nextFilter;
    localStorage.setItem(FILTER_STORAGE_KEY, nextFilter);
    render();
  }

  function runWindowControl(control) {
    switch (control) {
      case "openFull":
        openFullGTD();
        break;
      case "refresh":
        refreshTasks();
        break;
      case "toggleFixed":
        state.fixed = !state.fixed;
        applyFixedState();
        setPersistedWindowFixed(state.fixed).catch(function () {});
        break;
      default:
        break;
    }
  }

  function openFullGTD() {
    if (typeof invoke !== "function") {
      window.open("/desktop", "_blank", "noopener");
      return;
    }

    invoke("/api/open_window?pathname=%2Fdesktop", { method: "GET" }).then(
      function (resp) {
        if (!resp || resp.code !== 0) {
          showToast((resp && resp.msg) || "打开完整版失败");
          return;
        }
        showToast("已打开完整版");
      },
      function (err) {
        showToast("打开完整版失败: " + err);
      },
    );
  }

  function applyFixedState() {
    renderFixedButton();
    document.body.classList.toggle("is-fixed-window", state.fixed);
    callNativeWindow("__velo/window/set_always_on_top", { onTop: state.fixed }).catch(function () {});
  }

  function renderFixedButton() {
    if (!els.fixedButton) return;
    els.fixedButton.classList.toggle("is-active", state.fixed);
    els.fixedButton.setAttribute("aria-pressed", state.fixed ? "true" : "false");
    els.fixedButton.setAttribute("title", state.fixed ? "取消固定" : "固定在所有窗口上方");
    els.fixedButton.setAttribute("aria-label", state.fixed ? "取消固定" : "固定在所有窗口上方");
  }

  function render() {
    renderTabs();
    renderCount();
    renderList();
  }

  function renderTabs() {
    if (!els.tabs) return;
    const counts = taskFilterCounts(state.tasks);
    els.tabs.innerHTML = FILTERS.map(function ([value, label]) {
      const count = counts[value] || 0;
      return `
        <button class="gtd-slim-tab ${state.filter === value ? "is-active" : ""}" type="button" data-gtd-slim-filter="${escapeAttr(value)}" aria-pressed="${state.filter === value ? "true" : "false"}">
          <span>${escapeHTML(label)}</span>
          <strong>${count ? escapeHTML(count) : ""}</strong>
        </button>
      `;
    }).join("");
  }

  function renderCount() {
    if (!els.count) return;
    if (state.loading) {
      els.count.textContent = "读取中";
      return;
    }
    const tasks = visibleTasksFor(state.tasks, state.filter);
    const open = tasks.filter((task) => task.status !== "completed").length;
    els.count.textContent = tasks.length ? `${open} / ${tasks.length}` : "0";
  }

  function renderList() {
    if (!els.list) return;
    if (state.error) {
      renderState(state.error);
      return;
    }
    if (state.loading && state.tasks.length === 0) {
      renderState("正在加载代办...");
      return;
    }

    const tasks = visibleTasksFor(state.tasks, state.filter);
    if (!tasks.length) {
      renderState(emptyLabelForFilter(state.filter));
      return;
    }

    els.list.innerHTML = groupedTasks(tasks, state.filter).map(function (group) {
      return `
        <section class="gtd-slim-group" aria-label="${escapeAttr(group.label)}">
          <div class="gtd-slim-group-head">
            <span>${escapeHTML(group.label)}</span>
            <strong>${group.tasks.length}</strong>
          </div>
          ${group.tasks.map(taskTemplate).join("")}
        </section>
      `;
    }).join("");
  }

  function renderState(message) {
    els.list.innerHTML = `<div class="gtd-slim-state">${escapeHTML(message || "")}</div>`;
  }

  function callNativeWindow(method, args) {
    if (typeof invoke !== "function") {
      return Promise.reject(new Error("go bridge not available"));
    }
    return invoke(method, { args: args || {} });
  }

  function showToast(message) {
    if (!els.toast) return;
    els.toast.textContent = message;
    els.toast.classList.add("is-visible");
    if (state.toastTimer) window.clearTimeout(state.toastTimer);
    state.toastTimer = window.setTimeout(function () {
      els.toast.classList.remove("is-visible");
    }, 1800);
  }
}

function slimTemplate() {
  return `
    <div class="memo-window-shell gtd-slim-shell velo-drag" data-velo-drag>
      <header class="memo-window-titlebar gtd-slim-titlebar velo-drag" data-velo-drag>
        <div class="memo-window-native-controls" aria-hidden="true"></div>
        <div class="memo-window-drag-region" aria-hidden="true"></div>
        <div class="memo-window-title-actions">
          <button class="memo-window-text-button velo-no-drag" type="button" data-window-control="openFull">完整版</button>
          <button class="memo-window-icon-button velo-no-drag" type="button" data-window-control="refresh" title="刷新" aria-label="刷新">
            ${SVG.restore}
          </button>
          <button class="memo-window-icon-button velo-no-drag" type="button" data-window-control="toggleFixed" title="固定在所有窗口上方" aria-label="固定在所有窗口上方" aria-pressed="false">
            ${SVG.pin}
          </button>
        </div>
      </header>
      <main class="memo-window-body gtd-slim-body velo-no-drag">
        <section class="gtd-slim-head">
          <div>
            <h1>代办</h1>
            <p>${escapeHTML(formatToday())}</p>
          </div>
          <strong data-gtd-slim-count>0</strong>
        </section>
        <form class="gtd-slim-form" data-gtd-slim-form>
          <input class="gtd-slim-title-input" data-gtd-slim-title name="title" type="text" placeholder="添加代办" autocomplete="off" />
          <button class="gtd-slim-submit" data-gtd-slim-submit type="submit" title="添加" aria-label="添加">
            ${SVG.plus}
          </button>
          <div class="gtd-slim-form-options">
            <input data-gtd-slim-due name="dueAt" type="date" aria-label="截止日期" />
            <select data-gtd-slim-priority name="priority" aria-label="优先级">
              <option value="none">无优先级</option>
              <option value="low">低</option>
              <option value="medium">中</option>
              <option value="high">高</option>
            </select>
          </div>
        </form>
        <nav class="gtd-slim-tabs" data-gtd-slim-tabs aria-label="代办过滤"></nav>
        <section class="gtd-slim-list" data-gtd-slim-list aria-label="代办列表"></section>
      </main>
      <div class="memo-toast" data-toast role="status"></div>
    </div>
  `;
}

function taskTemplate(task) {
  const complete = task.status === "completed";
  const priority = normalizePriority(task.priority);
  const due = task.dueAt ? taskDateLabel(task.dueAt) : "";
  const start = task.startAt ? taskDateLabel(task.startAt) : "";
  const dueState = taskDueState(task);
  const tags = (task.tags || []).slice(0, 2);
  const hasReminders = task.reminders && task.reminders.length > 0;
  return `
    <article class="gtd-slim-task ${complete ? "is-complete" : ""} is-priority-${escapeAttr(priority)}" data-gtd-slim-task-id="${escapeAttr(task.id)}">
      <label class="gtd-slim-check">
        <input type="checkbox" data-gtd-slim-complete ${complete ? "checked" : ""} />
        <span></span>
      </label>
      <div class="gtd-slim-task-body">
        <strong>${escapeHTML(task.title)}</strong>
        <div class="gtd-slim-task-meta">
          ${priority !== "none" ? `<span class="is-priority">${escapeHTML(priorityLabel(priority))}</span>` : ""}
          ${due ? `<time class="${escapeAttr(dueState)}" datetime="${escapeAttr(task.dueAt)}">截止 ${escapeHTML(due)}</time>` : ""}
          ${start ? `<time datetime="${escapeAttr(task.startAt)}">开始 ${escapeHTML(start)}</time>` : ""}
          ${task.listId && task.listId !== "inbox" ? `<span>${escapeHTML(task.listId)}</span>` : ""}
          ${tags.map((tag) => `<span>#${escapeHTML(tag)}</span>`).join("")}
          ${complete && task.completedAt ? `<button class="gtd-slim-completed-time" type="button" data-gtd-slim-edit-completed datetime="${escapeAttr(task.completedAt)}" title="点击编辑完成时间"><time datetime="${escapeAttr(task.completedAt)}">完成 ${escapeHTML(taskDateTimeLabel(task.completedAt))}</time></button>` : ""}
        </div>
      </div>
      <div class="gtd-slim-task-actions">
        <button class="gtd-slim-reminder-btn ${hasReminders ? "has-reminders" : ""}" type="button" data-gtd-slim-reminder-btn title="设置提醒" aria-label="设置提醒">
          ${SVG.bell || "🔔"}
        </button>
      </div>
    </article>
  `;
}

function visibleTasksFor(tasks, filter) {
  return tasks
    .filter((task) => task && task.status !== "archived" && task.status !== "cancelled")
    .filter((task) => taskMatchesFilter(task, filter))
    .sort(function (a, b) {
      return sortTasksForFilter(a, b, filter);
    });
}

function groupedTasks(tasks, filter) {
  if (normalizeFilter(filter) === "completed") return [{ label: "已完成", tasks }];
  if (normalizeFilter(filter) === "today") {
    return [
      { label: "已过期", tasks: tasks.filter(isTaskOverdue) },
      { label: "今天", tasks: tasks.filter((task) => !isTaskOverdue(task)) },
    ].filter((group) => group.tasks.length);
  }
  if (normalizeFilter(filter) === "overdue") return [{ label: "已过期", tasks }];
  if (normalizeFilter(filter) === "scheduled") {
    return [
      { label: "已过期", tasks: tasks.filter(isTaskOverdue) },
      { label: "今天", tasks: tasks.filter((task) => !isTaskOverdue(task) && isTaskToday(task)) },
      { label: "未来", tasks: tasks.filter((task) => !isTaskOverdue(task) && !isTaskToday(task)) },
    ].filter((group) => group.tasks.length);
  }
  if (normalizeFilter(filter) === "all") {
    return [
      { label: "未完成", tasks: tasks.filter((task) => task.status !== "completed") },
      { label: "已完成", tasks: tasks.filter((task) => task.status === "completed") },
    ].filter((group) => group.tasks.length);
  }
  return [{ label: filterLabel(filter), tasks }];
}

function taskMatchesFilter(task, filter) {
  switch (normalizeFilter(filter)) {
    case "all":
      return true;
    case "completed":
      return task.status === "completed";
    case "inbox":
      return task.status !== "completed" && (task.listId === "inbox" || !task.listId);
    case "next":
      return task.status !== "completed" && !task.parentId;
    case "overdue":
      return isTaskOverdue(task);
    case "scheduled":
      return task.status !== "completed" && Boolean(task.startAt || task.dueAt);
    case "today":
    default:
      return task.status !== "completed" && (isTaskToday(task) || isTaskOverdue(task));
  }
}

function taskFilterCounts(tasks) {
  const activeTasks = tasks.filter((task) => task && task.status !== "archived" && task.status !== "cancelled");
  return {
    all: activeTasks.length,
    completed: activeTasks.filter((task) => task.status === "completed").length,
    inbox: activeTasks.filter((task) => taskMatchesFilter(task, "inbox")).length,
    next: activeTasks.filter((task) => taskMatchesFilter(task, "next")).length,
    overdue: activeTasks.filter((task) => taskMatchesFilter(task, "overdue")).length,
    scheduled: activeTasks.filter((task) => taskMatchesFilter(task, "scheduled")).length,
    today: activeTasks.filter((task) => taskMatchesFilter(task, "today")).length,
  };
}

function sortTasksForFilter(a, b, filter) {
  if (normalizeFilter(filter) === "completed") {
    return taskTimeValue(b.completedAt || b.updatedAt || b.createdAt) - taskTimeValue(a.completedAt || a.updatedAt || a.createdAt);
  }
  if (a.status !== b.status) {
    if (a.status === "completed") return 1;
    if (b.status === "completed") return -1;
  }
  const created = taskTimeValue(b.createdAt) - taskTimeValue(a.createdAt);
  if (created !== 0) return created;
  const priority = taskPriorityWeight(b.priority) - taskPriorityWeight(a.priority);
  if (priority !== 0) return priority;
  return taskTimeValue(b.updatedAt) - taskTimeValue(a.updatedAt);
}

function defaultDueForFilter(filter) {
  return normalizeFilter(filter) === "today" ? dateKey(new Date()) : "";
}

function emptyLabelForFilter(filter) {
  switch (normalizeFilter(filter)) {
    case "completed":
      return "还没有已完成代办";
    case "overdue":
      return "没有过期代办";
    case "inbox":
      return "Inbox 为空";
    case "next":
      return "没有下一步代办";
    case "scheduled":
      return "没有计划代办";
    case "all":
      return "还没有代办";
    case "today":
    default:
      return "今天没有代办";
  }
}

function normalizeFilter(value) {
  const filter = String(value || "").trim().toLowerCase();
  return KNOWN_FILTERS.has(filter) ? filter : "today";
}

function filterLabel(filter) {
  const match = FILTERS.find((item) => item[0] === normalizeFilter(filter));
  return match ? match[1] : "今天";
}

function normalizePriority(value) {
  const priority = String(value || "").trim().toLowerCase();
  if (priority === "high" || priority === "medium" || priority === "low") return priority;
  return "none";
}

function priorityLabel(priority) {
  switch (normalizePriority(priority)) {
    case "high":
      return "高";
    case "medium":
      return "中";
    case "low":
      return "低";
    default:
      return "无";
  }
}

function taskDueState(task) {
  if (!task || task.status === "completed") return "";
  if (isTaskOverdue(task)) return "is-overdue";
  if (isTaskToday(task)) return "is-today";
  return "";
}

function isTaskToday(task) {
  const today = dateKey(new Date());
  return [task.startAt, task.dueAt].some(function (value) {
    return value && dateKey(taskDateValue(value)) === today;
  });
}

function isTaskOverdue(task) {
  if (!task || !task.dueAt || task.status === "completed") return false;
  const due = taskDateValue(task.dueAt);
  if (Number.isNaN(due.getTime())) return false;
  return dateKey(due) < dateKey(new Date());
}

function taskDateLabel(value) {
  const date = taskDateValue(value);
  if (Number.isNaN(date.getTime())) return String(value || "");
  const today = dateKey(new Date());
  if (dateKey(date) === today) return "今天";
  const tomorrow = new Date();
  tomorrow.setDate(tomorrow.getDate() + 1);
  if (dateKey(date) === dateKey(tomorrow)) return "明天";
  return date.toLocaleDateString([], {
    day: "numeric",
    month: "numeric",
  });
}

function taskDateTimeLabel(value) {
  const date = taskDateValue(value);
  if (Number.isNaN(date.getTime())) return String(value || "");
  return date.toLocaleString([], {
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    month: "numeric",
  });
}

function taskDateTimeLocalValue(value) {
  const date = taskDateValue(value);
  if (Number.isNaN(date.getTime())) return "";
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  const h = String(date.getHours()).padStart(2, "0");
  const min = String(date.getMinutes()).padStart(2, "0");
  return y + "-" + m + "-" + d + "T" + h + ":" + min;
}

function taskDateValue(value) {
  const raw = String(value || "").trim();
  const dateOnly = raw.match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (dateOnly) {
    return new Date(Number(dateOnly[1]), Number(dateOnly[2]) - 1, Number(dateOnly[3]));
  }
  return new Date(raw);
}

function taskTimeValue(value) {
  if (!value) return Number.MAX_SAFE_INTEGER;
  const date = taskDateValue(value);
  return Number.isNaN(date.getTime()) ? Number.MAX_SAFE_INTEGER : date.getTime();
}

function taskPriorityWeight(priority) {
  switch (normalizePriority(priority)) {
    case "high":
      return 3;
    case "medium":
      return 2;
    case "low":
      return 1;
    default:
      return 0;
  }
}

function dateKey(date) {
  if (!(date instanceof Date) || Number.isNaN(date.getTime())) return "";
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${date.getFullYear()}-${month}-${day}`;
}

function formatToday() {
  return new Date().toLocaleDateString([], {
    day: "numeric",
    month: "long",
    weekday: "long",
  });
}

function reminderPopoverContent(task) {
  const reminders = task.reminders || [];
  const quickOptions = [
    [10, "到期前 10 分钟"],
    [30, "到期前 30 分钟"],
    [60, "到期前 1 小时"],
    [1440, "到期前 1 天"],
  ];
  let html = '<div class="gtd-slim-reminder-popover-inner">';
  html += '<div class="gtd-slim-reminder-popover-title">设置提醒</div>';
  html += '<div class="gtd-slim-reminder-quick">';
  for (const [minutes, label] of quickOptions) {
    html += `<button type="button" class="gtd-slim-reminder-quick-btn" data-reminder-quick="${minutes}">${escapeHTML(label)}</button>`;
  }
  html += '</div>';
  html += '<div class="gtd-slim-reminder-abs">';
  html += '<input type="datetime-local" data-reminder-abs-input class="gtd-slim-reminder-abs-input" />';
  html += '<button type="button" class="gtd-slim-reminder-abs-btn" data-reminder-abs-confirm>确定</button>';
  html += '</div>';
  if (reminders.length > 0) {
    html += '<div class="gtd-slim-reminder-list">';
    html += '<div class="gtd-slim-reminder-list-title">已设提醒</div>';
    reminders.forEach(function (r, i) {
      const label = reminderLabel(r);
      html += `<div class="gtd-slim-reminder-item"><span>${escapeHTML(label)}</span><button type="button" data-reminder-delete="${i}" class="gtd-slim-reminder-delete" title="删除">×</button></div>`;
    });
    html += '</div>';
  }
  html += '</div>';
  return html;
}

function reminderLabel(reminder) {
  if (reminder.type === "absolute" && reminder.at) {
    try {
      return new Date(reminder.at).toLocaleString([], { month: "numeric", day: "numeric", hour: "2-digit", minute: "2-digit" });
    } catch (_) {
      return reminder.at;
    }
  }
  if (reminder.type === "relative" && reminder.offsetMinutes) {
    const m = reminder.offsetMinutes;
    if (m >= 1440 && m % 1440 === 0) return `到期前 ${m / 1440} 天`;
    if (m >= 60 && m % 60 === 0) return `到期前 ${m / 60} 小时`;
    return `到期前 ${m} 分钟`;
  }
  return "提醒";
}
