import {
  DEFAULT_VISIBILITY,
  VISIBILITY,
  buildMemoReferenceIndex,
  compactText,
  extractProjectDirective,
  memoTitle,
  stripProjectDirective,
} from "../../domain/memos.js";
import { normalizeProjectID } from "../../domain/projects.js";
import {
  errorMessage,
  updateMemoInVault,
} from "../../domain/memo-repository.js";
import {
  deleteMemoDraftInVault,
  memoEditDraftId,
  upsertMemoDraftInVault,
} from "../../domain/memo-drafts.js";
import { SVG } from "./memo-icons.js";
import {
  projectOptionsTemplate,
  visibilityOptionsTemplate,
} from "./memo-templates.js";
import { createMiniEditor } from "./memo-editor.js";
import { renderMemoMarkdown } from "./memo-markdown.js";
import { escapeHTML } from "./memo-utils.js";

export function mountMemoEditDialog(host, context) {
  if (!context || !context.memo) {
    throw new Error("mountMemoEditDialog requires context.memo");
  }

  var memo = context.memo;
  var initialDraft = context.initialDraft || null;
  var memos = Array.isArray(context.memos) ? context.memos : [];
  var projects = Array.isArray(context.projects) ? context.projects : [];
  var editorSettings = context.editorSettings || {};
  var tagItems = typeof context.tagItems === "function" ? context.tagItems : function () { return []; };
  var onSaveComplete = typeof context.onSaveComplete === "function" ? context.onSaveComplete : function () {};
  var onClose = typeof context.onClose === "function" ? context.onClose : function () {};
  var onDraftUpsert = typeof context.onDraftUpsert === "function" ? context.onDraftUpsert : function () {};
  var onDraftDelete = typeof context.onDraftDelete === "function" ? context.onDraftDelete : function () {};
  var showToast = typeof context.showToast === "function" ? context.showToast : function () {};
  var resolveOrCreateProject = typeof context.resolveOrCreateProject === "function"
    ? context.resolveOrCreateProject
    : function (name) { return Promise.resolve(""); };

  var draftContent = initialDraft || memo.content || "";
  var previewVisible = false;
  var saving = false;
  var projectId = normalizeProjectID(memo.projectId);
  var visibility = (function () {
    var v = memo.visibility || DEFAULT_VISIBILITY;
    var p = Boolean(memo.private);
    return p && v === "PRIVATE" ? "SECRET" : v;
  })();

  var editor = null;
  var dialogEl = null;
  var destroyed = false;

  render();
  mountEditor();
  window.requestAnimationFrame(function () {
    if (!destroyed && editor) editor.focus();
  });

  return {
    destroy: destroy,
    focus: focus,
    getDraft: getDraft,
  };

  function destroy() {
    destroyed = true;
    if (editor) {
      editor.destroy();
      editor = null;
    }
    if (dialogEl) {
      dialogEl.remove();
      dialogEl = null;
    }
  }

  function focus() {
    if (!destroyed && editor) editor.focus();
  }

  function getDraft() {
    if (!destroyed && editor) {
      draftContent = editor.getText();
    }
    return draftContent;
  }

  function render() {
    if (destroyed) return;

    if (editor) {
      editor.destroy();
      editor = null;
    }

    if (dialogEl) {
      dialogEl.remove();
      dialogEl = null;
    }

    var title = "修改 memo";
    var description = compactText(memoTitle(memo), 88);
    var saveLabel = "保存";
    var editControls = '<div class="memo-dialog-meta-controls">'
      + '<label class="memo-select-wrap is-compact">'
      + '<select data-memo-dialog-project aria-label="编辑 Project">'
      + projectOptionsTemplate(projects, projectId)
      + '</select>'
      + SVG.chevronDown
      + '</label>'
      + '<label class="memo-select-wrap is-compact">'
      + '<select data-memo-dialog-visibility aria-label="编辑可见性">'
      + visibilityOptionsTemplate(visibility)
      + '</select>'
      + SVG.chevronDown
      + '</label>'
      + '</div>';

    var html = ''
      + '<section class="memo-dialog-panel" role="dialog" aria-modal="true" aria-labelledby="memo-dialog-title">'
      + '<header class="memo-dialog-head">'
      + '<div>'
      + '<h2 id="memo-dialog-title">' + escapeHTML(title) + '</h2>'
      + '<p>' + escapeHTML(description) + '</p>'
      + '</div>'
      + '<button class="memo-action-button" type="button" data-memo-dialog-action="close" title="关闭" aria-label="关闭">' + SVG.x + '</button>'
      + '</header>'
      + '<div class="memo-dialog-body">'
      + '<div class="memo-editor-switch memo-dialog-editor-switch">'
      + '<div class="memo-editor-host memo-dialog-editor-host" data-memo-dialog-editor-host data-editor-switch-host></div>'
      + '<section class="memo-editor-preview memo-dialog-preview" data-memo-dialog-preview hidden></section>'
      + '</div>'
      + '</div>'
      + '<footer class="memo-dialog-actions">'
      + editControls
      + '<div class="memo-inline-status-line" data-memo-dialog-vim-status></div>'
      + '<button class="memo-secondary-button" type="button" data-memo-dialog-action="preview" aria-pressed="false">' + SVG.eye + '<span>预览</span></button>'
      + '<button class="memo-secondary-button" type="button" data-memo-dialog-action="cancel">' + SVG.x + '<span>取消</span></button>'
      + '<button class="memo-primary-button" type="button" data-memo-dialog-action="save">' + SVG.check + '<span>' + escapeHTML(saveLabel) + '</span></button>'
      + '</footer>'
      + '</section>';

    dialogEl = document.createElement("div");
    dialogEl.className = "memo-dialog";
    dialogEl.dataset.memoDialog = "true";
    dialogEl.dataset.memoId = memo.id || "";
    dialogEl.innerHTML = html;
    host.appendChild(dialogEl);

    dialogEl.addEventListener("click", handleClick);
    dialogEl.addEventListener("change", handleChange);

    renderPreview();
    renderSaving();
  }

  function mountEditor() {
    if (destroyed || !dialogEl) return;

    var editorHost = dialogEl.querySelector("[data-memo-dialog-editor-host]");
    var vimStatusHost = dialogEl.querySelector("[data-memo-dialog-vim-status]");
    if (!editorHost) return;

    editor = createMiniEditor(editorHost, {
      memoItems: function () {
        return memos;
      },
      tagItems: tagItems,
      onChange: function (value) {
        if (destroyed) return;
        draftContent = value;
        renderPreview();
      },
      onCommit: function () {
        return save({ source: "vim-wq" });
      },
      onDiscard: function () {
        return discardDraft({ exit: true, message: "草稿已丢弃" });
      },
      onQuit: function () {
        return exitEdit();
      },
      onSave: function () {
        return writeDraft();
      },
      onSubmit: function () {
        return save();
      },
      onWriteDraft: function () {
        return writeDraft();
      },
      placeholder: "编辑 memo...",
      sourceMemoId: memo.id || "",
      value: draftContent,
      vim: editorSettings.vimMode === true,
      vimStatusHost: vimStatusHost,
    });
  }

  function syncDraft() {
    if (!destroyed && editor) {
      draftContent = editor.getText();
    }
  }

  function handleClick(event) {
    if (destroyed || !dialogEl) return;

    var actionBtn = event.target.closest("[data-memo-dialog-action]");
    if (actionBtn && dialogEl.contains(actionBtn)) {
      event.preventDefault();
      runAction(actionBtn.dataset.memoDialogAction || "");
      return;
    }

    if (event.target === dialogEl && !saving) {
      cancel();
    }
  }

  function handleChange(event) {
    if (destroyed) return;

    if (event.target.matches("[data-memo-dialog-project]")) {
      projectId = normalizeProjectID(event.target.value);
      return;
    }

    if (event.target.matches("[data-memo-dialog-visibility]")) {
      visibility = event.target.value || DEFAULT_VISIBILITY;
      return;
    }
  }

  function runAction(action) {
    switch (action) {
      case "cancel":
        cancel();
        break;
      case "close":
        exitEdit().then(function () {
          onClose();
        });
        break;
      case "preview":
        togglePreview();
        break;
      case "save":
        save();
        break;
    }
  }

  function togglePreview() {
    syncDraft();
    previewVisible = !previewVisible;
    renderPreview();
    if (!previewVisible && !destroyed && editor) editor.focus();
  }

  function renderPreview() {
    if (destroyed || !dialogEl) return;
    var panel = dialogEl.querySelector("[data-memo-dialog-preview]");
    var button = dialogEl.querySelector('[data-memo-dialog-action="preview"]');
    updatePreviewButton(button, previewVisible);
    if (!panel) return;

    var switcher = panel.closest(".memo-editor-switch");
    var host = switcher && switcher.querySelector("[data-editor-switch-host]");
    if (host) host.hidden = previewVisible;
    panel.hidden = !previewVisible;
    panel.classList.toggle("is-visible", previewVisible);

    if (!previewVisible) {
      panel.innerHTML = "";
      return;
    }

    var text = String(draftContent || "");
    if (!text.trim()) {
      panel.innerHTML = '<div class="memo-editor-preview-empty">暂无预览内容</div>';
      return;
    }

    var memoRefIndex = buildMemoReferenceIndex(memos);
    var renderContext = {
      depth: 0,
      index: memoRefIndex,
      maxDepth: 2,
      readonly: true,
      editorSettings: editorSettings,
      showLineNumbers: true,
      sourceId: memo.id || "",
      stack: memo.id ? [memo.id] : [],
    };

    try {
      panel.innerHTML = '<div class="memo-content">' + renderMemoMarkdown(text, renderContext) + '</div>';
    } catch (_err) {
      panel.innerHTML = '<div class="memo-content"><p>' + escapeHTML(text) + '</p></div>';
    }
  }

  function updatePreviewButton(button, visible) {
    if (!button) return;
    button.setAttribute("aria-pressed", visible ? "true" : "false");
    button.title = visible ? "编辑" : "预览";
    button.setAttribute("aria-label", visible ? "编辑" : "预览");
    var label = button.querySelector("span");
    if (label) label.textContent = visible ? "编辑" : "预览";
  }

  function renderSaving() {
    if (destroyed || !dialogEl) return;
    dialogEl.classList.toggle("is-saving", saving);
    dialogEl.querySelectorAll("[data-memo-dialog-action], [data-memo-dialog-project], [data-memo-dialog-visibility]").forEach(function (control) {
      control.disabled = saving;
    });
  }

  function setSaving(value) {
    saving = Boolean(value);
    renderSaving();
  }

  function cancel() {
    if (destroyed) return Promise.resolve({ ok: true });
    return discardDraft({ exit: true, message: "编辑已取消" }).then(function () {
      onClose();
    });
  }

  function save(options) {
    if (destroyed) return Promise.resolve({ ok: false, message: "component destroyed" });
    if (saving) return Promise.resolve({ ok: false, message: "正在保存" });

    syncDraft();
    var content = String(draftContent || "");
    if (!content.trim()) {
      showToast("内容不能为空");
      if (editor) editor.focus();
      return Promise.resolve({ ok: false, message: "内容不能为空" });
    }

    var memoId = memo.id;
    setSaving(true);

    var projectRef = extractProjectDirective(content);
    var resolveProject = projectRef
      ? resolveOrCreateProject(projectRef)
      : Promise.resolve(null);

    return resolveProject.then(function (resolvedProjectId) {
      var finalContent = projectRef ? stripProjectDirective(content) : content;
      var finalProjectId = resolvedProjectId || projectId;
      var isSecret = visibility === "SECRET";
      return updateMemoInVault(memoId, {
        content: finalContent,
        private: isSecret,
        projectId: finalProjectId,
        updatedAt: new Date().toISOString(),
        visibility: isSecret ? "PRIVATE" : visibility,
      });
    }).then(
      function () {
        deleteMemoDraftInVault(memoEditDraftId(memoId)).catch(function () {});
        if (!destroyed) setSaving(false);
        if (options && options.source !== "vim-wq") showToast("已保存");
        onSaveComplete(memoId);
        return { ok: true, message: "已保存" };
      },
      function (err) {
        showToast("保存失败: " + errorMessage(err));
        if (!destroyed) setSaving(false);
        return { ok: false, message: "保存失败: " + errorMessage(err) };
      },
    );
  }

  function writeDraft() {
    if (destroyed) return Promise.resolve({ ok: false });
    syncDraft();
    var content = String(draftContent || "");
    if (!content.trim()) {
      return discardDraft({ exit: false, message: "空草稿已清理" });
    }
    var draftPayload = {
      baseUpdatedAt: memo.updatedAt || "",
      content: content,
      id: memoEditDraftId(memo.id),
      kind: "memo-edit",
      memoId: memo.id,
      projectId: projectId,
      visibility: visibility,
    };
    return upsertMemoDraftInVault(draftPayload).then(
      function (draft) {
        if (!destroyed) draftContent = content;
        showToast("草稿已保存");
        onDraftUpsert(draft);
        return { ok: true, message: "draft written" };
      },
      function (err) {
        showToast("保存草稿失败: " + errorMessage(err));
        return { ok: false, message: "保存草稿失败: " + errorMessage(err) };
      },
    );
  }

  function discardDraft(options) {
    if (destroyed) return Promise.resolve({ ok: false });
    var memoId = memo.id;
    if (!memoId) return Promise.resolve({ ok: false });
    var draftId = memoEditDraftId(memoId);
    onDraftDelete(draftId);
    return deleteMemoDraftInVault(draftId).then(
      function () {
        if (options && options.message) showToast(options.message);
        return { ok: true, message: (options && options.message) || "draft discarded" };
      },
      function (err) {
        showToast("删除草稿失败: " + errorMessage(err));
        return { ok: false, message: "删除草稿失败: " + errorMessage(err) };
      },
    );
  }

  function exitEdit() {
    if (destroyed) return Promise.resolve({ ok: true });
    syncDraft();
    var memoDisplayVis = memo.private && memo.visibility === "PRIVATE" ? "SECRET" : memo.visibility || DEFAULT_VISIBILITY;
    var changed =
      String(draftContent || "") !== (memo.content || "") ||
      normalizeProjectID(projectId) !== normalizeProjectID(memo.projectId) ||
      (visibility || DEFAULT_VISIBILITY) !== memoDisplayVis;
    if (!changed) return Promise.resolve({ ok: true });
    return writeDraft();
  }
}
