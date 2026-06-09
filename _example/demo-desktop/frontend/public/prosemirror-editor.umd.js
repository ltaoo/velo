(function (root, factory) {
  "use strict";

  if (typeof define === "function" && define.amd) {
    define([], function () {
      return factory(root);
    });
  } else if (typeof module === "object" && module.exports) {
    module.exports = factory(root);
  } else {
    root.ProsemirrorEditor = factory(root);
  }
})(
  typeof globalThis !== "undefined"
    ? globalThis
    : typeof window !== "undefined"
      ? window
      : this,
  function (root) {
    "use strict";

    const PM = root && root.ProsemirrorMod;
    if (!PM) {
      throw new Error("ProsemirrorMod was not loaded.");
    }

    let editorCounter = 0;
    let imageUploadCounter = 0;

    function normalizeTableAlign(value) {
      const align = String(value || "").trim().toLowerCase();
      return ["left", "center", "right"].includes(align) ? align : null;
    }

    function tableCellAlignFromDOM(dom) {
      if (!dom) return null;
      const styleAlign = dom.style && dom.style.textAlign;
      return normalizeTableAlign(dom.getAttribute("align") || styleAlign);
    }

    function tableCellDOMSpec(tagName, align) {
      const normalized = normalizeTableAlign(align);
      return [tagName, normalized ? { style: "text-align: " + normalized + ";" } : {}, 0];
    }

    function imageUploadStatus(status) {
      return ["uploading", "success", "error"].includes(status) ? status : "uploading";
    }

    function imageUploadText(attrs) {
      const status = imageUploadStatus(attrs && attrs.status);
      const message = String((attrs && attrs.message) || "").trim();
      const fileName = String((attrs && attrs.fileName) || "").trim();

      if (status === "success") return message || fileName || "Upload complete";
      if (status === "error") return "Upload failed: " + (message || "Unknown error");
      return "loading: Uploading " + (fileName || "image");
    }

    function numericPixel(value) {
      const parsed = parseFloat(value);
      return Number.isFinite(parsed) ? parsed : 0;
    }

    function selectionTextblockRect(viewInstance) {
      const selection = viewInstance && viewInstance.state && viewInstance.state.selection;
      const $head = selection && selection.$head;
      if (!$head) return null;

      for (let depth = $head.depth; depth > 0; depth -= 1) {
        const node = $head.node(depth);
        if (!node || !node.isTextblock) continue;
        const dom = viewInstance.nodeDOM($head.before(depth));
        if (dom && typeof dom.getBoundingClientRect === "function") {
          return dom.getBoundingClientRect();
        }
      }
      return null;
    }

    function selectionRect(viewInstance) {
      const selection = viewInstance && viewInstance.state && viewInstance.state.selection;
      if (!selection) return null;

      if (PM.NodeSelection && selection instanceof PM.NodeSelection) {
        const dom = viewInstance.nodeDOM(selection.from);
        if (dom && typeof dom.getBoundingClientRect === "function") {
          return dom.getBoundingClientRect();
        }
      }

      try {
        return viewInstance.coordsAtPos(selection.head, 1);
      } catch (_) {
        return null;
      }
    }

    function scrollSelectionIntoEditorPadding(viewInstance) {
      const scroller = viewInstance && viewInstance.dom;
      if (!scroller || typeof scroller.getBoundingClientRect !== "function") return false;
      if (scroller.scrollHeight <= scroller.clientHeight) return false;

      const rect = selectionRect(viewInstance);
      if (!rect) return false;

      const style = root.getComputedStyle ? root.getComputedStyle(scroller) : null;
      const paddingTop = style ? numericPixel(style.paddingTop) : 0;
      const paddingBottom = style ? numericPixel(style.paddingBottom) : 0;
      const scrollerRect = scroller.getBoundingClientRect();
      const topLimit = scrollerRect.top + paddingTop;
      const bottomLimit = scrollerRect.bottom - paddingBottom;

      let targetTop = rect.top;
      let targetBottom = rect.bottom;
      const blockRect = selectionTextblockRect(viewInstance);
      if (blockRect && blockRect.height <= bottomLimit - topLimit) {
        targetTop = Math.min(targetTop, blockRect.top);
        targetBottom = Math.max(targetBottom, blockRect.bottom);
      }

      if (targetTop < topLimit) {
        scroller.scrollTop -= topLimit - targetTop;
        return true;
      }
      if (targetBottom > bottomLimit) {
        scroller.scrollTop += targetBottom - bottomLimit;
        return true;
      }
      return true;
    }

    function markdownLinkPlainText(label, href, image) {
      const text = String(label || href || "").trim();
      const target = String(href || text).trim();
      return (image ? "!" : "") + "[" + text + "](" + target + ")";
    }

    function fileMarkdownPlainText(attrs) {
      return markdownLinkPlainText(attrs && attrs.name, attrs && attrs.href, false);
    }

    function imageMarkdownPlainText(attrs) {
      return markdownLinkPlainText(attrs && attrs.alt, attrs && attrs.src, true);
    }

    function imageLinkText(attrs) {
      return "IMG " + String((attrs && (attrs.alt || attrs.src)) || "image");
    }

    function createSchema() {
      const baseNodes = PM.addListNodes(PM.schema.spec.nodes, "paragraph block*", "block");
      const nodes = baseNodes.append({
        todo_item: {
          group: "block",
          content: "paragraph block*",
          attrs: { done: { default: false } },
          draggable: false,
          parseDOM: [
            {
              tag: "div.todo-item",
              getAttrs(dom) {
                return { done: dom.getAttribute("data-done") === "true" };
              },
              getContent(dom, schemaInstance) {
                const content = dom.querySelector(".todo-content");
                return PM.DOMParser.fromSchema(schemaInstance).parse(content || dom).content;
              },
            },
          ],
          toDOM(node) {
            return [
              "div",
              { class: "todo-item", "data-done": String(node.attrs.done) },
              [
                "input",
                {
                  class: "todo-checkbox",
                  type: "checkbox",
                  checked: node.attrs.done ? "checked" : null,
                  contenteditable: "false",
                },
              ],
              ["div", { class: "todo-content" }, 0],
            ];
          },
        },
        table: {
          group: "block",
          content: "table_row+",
          isolating: true,
          parseDOM: [{ tag: "table" }],
          toDOM() {
            return ["table", { class: "pm-table" }, ["tbody", 0]];
          },
        },
        table_row: {
          content: "(table_cell | table_header)+",
          parseDOM: [{ tag: "tr" }],
          toDOM() {
            return ["tr", 0];
          },
        },
        table_cell: {
          content: "block+",
          isolating: true,
          attrs: { align: { default: null } },
          parseDOM: [
            {
              tag: "td",
              getAttrs(dom) {
                return { align: tableCellAlignFromDOM(dom) };
              },
            },
          ],
          toDOM(node) {
            return tableCellDOMSpec("td", node.attrs.align);
          },
        },
        table_header: {
          content: "block+",
          isolating: true,
          attrs: { align: { default: null } },
          parseDOM: [
            {
              tag: "th",
              getAttrs(dom) {
                return { align: tableCellAlignFromDOM(dom) };
              },
            },
          ],
          toDOM(node) {
            return tableCellDOMSpec("th", node.attrs.align);
          },
        },
        time: {
          group: "inline",
          inline: true,
          atom: true,
          selectable: true,
          attrs: { value: { default: "" } },
          parseDOM: [
            {
              tag: "time[data-time-node]",
              getAttrs(dom) {
                return { value: dom.getAttribute("datetime") || dom.textContent };
              },
            },
          ],
          toDOM(node) {
            return [
              "time",
              {
                class: "time-node",
                "data-time-node": "true",
                datetime: node.attrs.value,
              },
              node.attrs.value,
            ];
          },
          leafText(node) {
            return node.attrs.value || "";
          },
        },
        file_link: {
          group: "inline",
          inline: true,
          atom: true,
          selectable: true,
          attrs: {
            href: { default: "" },
            name: { default: "" },
            syntax: { default: "" },
          },
          parseDOM: [
            {
              tag: "a[data-file-link]",
              getAttrs(dom) {
                return {
                  href: dom.getAttribute("href") || "",
                  name:
                    dom.getAttribute("data-file-name") ||
                    dom.textContent ||
                    dom.getAttribute("href") ||
                    "",
                  syntax: dom.getAttribute("data-file-link-syntax") || "",
                };
              },
            },
          ],
          leafText(node) {
            return node.attrs.syntax === "markdown"
              ? fileMarkdownPlainText(node.attrs)
              : "@" + (node.attrs.name || node.attrs.href || "");
          },
          toDOM(node) {
            const syntax = node.attrs.syntax || "";
            return [
              "a",
              {
                class:
                  "file-link-node" + (syntax === "markdown" ? " file-link-node-markdown" : ""),
                "data-file-link": "true",
                "data-file-link-syntax": syntax || null,
                "data-file-name": node.attrs.name,
                href: node.attrs.href,
                target: "_blank",
                rel: "noopener noreferrer",
                contenteditable: "false",
              },
              "FILE " + (node.attrs.name || node.attrs.href),
            ];
          },
        },
        image_link: {
          group: "inline",
          inline: true,
          atom: true,
          selectable: true,
          attrs: {
            src: { default: "" },
            alt: { default: "" },
          },
          parseDOM: [
            {
              tag: "span[data-image-link]",
              getAttrs(dom) {
                return {
                  src: dom.getAttribute("data-image-src") || "",
                  alt: dom.getAttribute("data-image-alt") || dom.textContent || "",
                };
              },
            },
          ],
          leafText(node) {
            return imageMarkdownPlainText(node.attrs);
          },
          toDOM(node) {
            const text = imageLinkText(node.attrs);

            return [
              "span",
              {
                class: "image-link-node",
                "data-image-link": "true",
                "data-image-src": node.attrs.src,
                "data-image-alt": node.attrs.alt,
                contenteditable: "false",
                title: node.attrs.src,
              },
              text,
            ];
          },
        },
        image_upload: {
          group: "inline",
          inline: true,
          atom: true,
          selectable: true,
          attrs: {
            id: { default: "" },
            status: { default: "uploading" },
            message: { default: "" },
            fileName: { default: "" },
          },
          parseDOM: [
            {
              tag: "span[data-image-upload]",
              getAttrs(dom) {
                return {
                  id: dom.getAttribute("data-image-upload-id") || "",
                  status: dom.getAttribute("data-status") || "success",
                  message: dom.getAttribute("data-message") || dom.textContent || "",
                  fileName: dom.getAttribute("data-file-name") || "",
                };
              },
            },
          ],
          leafText(node) {
            return imageUploadText(node.attrs);
          },
          toDOM(node) {
            const status = imageUploadStatus(node.attrs.status);
            const text = imageUploadText({ ...node.attrs, status });

            return [
              "span",
              {
                class: "image-upload-node image-upload-node-" + status,
                "data-image-upload": "true",
                "data-image-upload-id": node.attrs.id,
                "data-status": status,
                "data-message": node.attrs.message,
                "data-file-name": node.attrs.fileName,
                contenteditable: "false",
                title: text,
              },
              text,
            ];
          },
        },
      });

      const marks = PM.schema.spec.marks.append({
        strikethrough: {
          parseDOM: [
            { tag: "s" },
            { tag: "del" },
            { tag: "strike" },
            {
              style: "text-decoration",
              getAttrs(value) {
                return /\bline-through\b/i.test(value || "") ? {} : false;
              },
            },
          ],
          toDOM() {
            return ["s", 0];
          },
        },
        highlight: {
          parseDOM: [
            { tag: "mark" },
            {
              style: "background-color",
              getAttrs(value) {
                return value ? {} : false;
              },
            },
          ],
          toDOM() {
            return ["mark", { class: "highlight" }, 0];
          },
        },
        hashtag: {
          inclusive: false,
          parseDOM: [{ tag: "span.hashtag" }],
          toDOM() {
            return ["span", { class: "hashtag" }, 0];
          },
        },
      });

      return new PM.Schema({ nodes, marks });
    }

    const schema = createSchema();

    function textContent(text) {
      return text ? schema.text(text) : null;
    }

    function paragraphFromText(text) {
      return schema.nodes.paragraph.create(null, textContent(text));
    }

    function miniDocFromPlainText(text) {
      const lines = String(text || "").replace(/\r\n?/g, "\n").split("\n");
      const paragraphs = lines.map((line) => paragraphFromText(line));
      return schema.node("doc", null, paragraphs.length ? paragraphs : [paragraphFromText("")]);
    }

    function inlinePlainText(node) {
      const parts = [];
      node.forEach((child) => {
        if (child.isText) {
          parts.push(child.text || "");
          return;
        }

        if (child.type === schema.nodes.hard_break) {
          parts.push("\n");
          return;
        }

        if (child.type === schema.nodes.time) {
          parts.push(child.attrs.value || "");
          return;
        }

        if (child.type === schema.nodes.file_link) {
          parts.push(
            child.attrs.syntax === "markdown"
              ? fileMarkdownPlainText(child.attrs)
              : "@" + (child.attrs.name || child.attrs.href || ""),
          );
          return;
        }

        if (child.type === schema.nodes.image_link) {
          parts.push(imageMarkdownPlainText(child.attrs));
          return;
        }

        if (child.type === schema.nodes.image_upload) {
          parts.push(imageUploadText(child.attrs));
          return;
        }

        if (child.isInline && child.isAtom) {
          parts.push(child.textContent || "");
        }
      });

      return parts.join("");
    }

    function blockPlainText(node) {
      if (node.type === schema.nodes.paragraph) return inlinePlainText(node);
      if (node.type === schema.nodes.heading) {
        return "#".repeat(node.attrs.level || 1) + " " + inlinePlainText(node);
      }
      if (node.type === schema.nodes.code_block) return "```\n" + node.textContent + "\n```";
      if (node.type === schema.nodes.horizontal_rule) return "---";
      return node.textBetween(0, node.content.size, "\n", "");
    }

    function plainTextFromMiniDoc(doc) {
      const lines = [];
      doc.forEach((node) => {
        lines.push(node.isTextblock ? inlinePlainText(node) : blockPlainText(node));
      });
      return lines.join("\n");
    }

    function miniPlainTextBlocks(doc) {
      const blocks = [];
      let plainStart = 0;

      doc.forEach((node, offset) => {
        const text = node.isTextblock ? inlinePlainText(node) : blockPlainText(node);
        blocks.push({
          start: offset + 1,
          end: offset + 1 + node.content.size,
          plainStart,
          text,
        });
        plainStart += text.length + 1;
      });

      return blocks;
    }

    function trailingHardBreakRanges(doc) {
      const hardBreak = schema.nodes.hard_break;
      const ranges = [];
      if (!hardBreak) return ranges;

      doc.descendants((node, pos) => {
        if (!node.isTextblock || !node.childCount) return true;

        let end = node.content.size;
        for (let index = node.childCount - 1; index >= 0; index -= 1) {
          const child = node.child(index);
          if (child.type !== hardBreak) break;

          const from = pos + 1 + end - child.nodeSize;
          ranges.push({ from, to: from + child.nodeSize });
          end -= child.nodeSize;
        }

        return false;
      });

      return ranges;
    }

    function htmlFromDoc(doc) {
      const fragment = PM.DOMSerializer.fromSchema(schema).serializeFragment(doc.content);
      const wrap = document.createElement("div");
      wrap.appendChild(fragment);
      return wrap.innerHTML;
    }

    function normalizeFileItem(item) {
      if (item == null) return null;

      if (typeof item === "string") {
        const value = item.trim();
        return value ? { label: value, name: value, href: value, detail: "" } : null;
      }

      if (typeof item !== "object") {
        const value = String(item).trim();
        return value ? { label: value, name: value, href: value, detail: "" } : null;
      }

      const label = String(
        item.label ||
          item.name ||
          item.title ||
          item.path ||
          item.href ||
          item.url ||
          item.value ||
          "",
      ).trim();
      if (!label) return null;

      const href = String(item.href || item.url || item.path || item.value || label).trim();
      const name = String(item.name || item.label || item.title || label).trim();
      const detail = String(
        item.detail || item.description || item.path || item.href || item.url || "",
      ).trim();

      return {
        label,
        name: name || label,
        href: href || label,
        detail: detail && detail !== label ? detail : "",
      };
    }

    function normalizeFileItems(items) {
      return Array.isArray(items) ? items.map(normalizeFileItem).filter(Boolean) : [];
    }

    function fileItemIdentity(item) {
      return [item.href, item.name, item.label].join("\n");
    }

    function fileFromToken(token) {
      return normalizeFileItem({
        href: token.href,
        name: token.name,
        label: token.label || token.name,
        detail: token.detail,
      });
    }

    function tokenIdentity(token) {
      return [token.href || "", token.name || "", token.text || ""].join("\n");
    }

    function imageUploadExtension(type) {
      const value = String(type || "").toLowerCase();
      if (value === "image/jpeg") return ".jpg";
      if (value === "image/png") return ".png";
      if (value === "image/gif") return ".gif";
      if (value === "image/webp") return ".webp";
      if (value === "image/svg+xml") return ".svg";
      return "";
    }

    function imageUploadFileName(file, index) {
      const name = file && file.name ? String(file.name).trim() : "";
      if (name) return name;

      return "clipboard-image-" + (index + 1) + imageUploadExtension(file && file.type);
    }

    function nextImageUploadId() {
      imageUploadCounter += 1;
      return "image-upload-" + Date.now().toString(36) + "-" + imageUploadCounter.toString(36);
    }

    function imageUploadErrorText(error) {
      if (!error) return "Unknown error";
      if (typeof error === "string") return error;
      return error.message || String(error);
    }

    function imageFilesFromClipboard(clipboardData) {
      if (!clipboardData) return [];

      const itemFiles = [];
      Array.from(clipboardData.items || []).forEach((item) => {
        if (!item || item.kind !== "file") return;

        const file = item.getAsFile();
        if (file && /^image\//i.test(file.type || item.type || "")) itemFiles.push(file);
      });
      if (itemFiles.length) return itemFiles;

      return Array.from(clipboardData.files || []).filter((file) => {
        return file && /^image\//i.test(file.type || "");
      });
    }

    function readMarkdownLinkSyntax(text, start) {
      const value = String(text || "");
      const image = value.charAt(start) === "!";
      const open = image ? start + 1 : start;
      if (value.charAt(open) !== "[") return null;

      const closeLabel = value.indexOf("]", open + 1);
      if (closeLabel < 0 || value.charAt(closeLabel + 1) !== "(") return null;

      const closeHref = value.indexOf(")", closeLabel + 2);
      if (closeHref < 0) return null;

      const label = value.slice(open + 1, closeLabel);
      const href = value.slice(closeLabel + 2, closeHref).trim();
      if (!href || label.includes("\n") || href.includes("\n")) return null;

      return {
        type: image ? "image" : "file",
        from: start,
        to: closeHref + 1,
        label: label.trim() || href,
        href,
      };
    }

    function asciiWordChar(char) {
      return /[A-Za-z0-9_]/.test(char || "");
    }

    function markdownEmphasisDelimiter(text, start) {
      const value = String(text || "");
      if (value.startsWith("***", start)) return "***";
      if (value.startsWith("**", start)) return "**";
      return value.charAt(start) === "*" ? "*" : "";
    }

    function markdownEmphasisStartAllowed(text, index) {
      if (index > 0 && text.charAt(index - 1) === "*") return false;
      return index === 0 || !asciiWordChar(text.charAt(index - 1));
    }

    function markdownEmphasisEndAllowed(text, index) {
      if (text.charAt(index) === "*") return false;
      return index >= text.length || !asciiWordChar(text.charAt(index));
    }

    function readMarkdownEmphasisSyntax(text, start) {
      const value = String(text || "");
      const delimiter = markdownEmphasisDelimiter(value, start);
      if (!delimiter || !markdownEmphasisStartAllowed(value, start)) return null;

      const contentStart = start + delimiter.length;
      const first = value.charAt(contentStart);
      if (!first || first === "*" || /\s/.test(first)) return null;

      let searchFrom = contentStart + 1;
      while (searchFrom < value.length) {
        const close = value.indexOf(delimiter, searchFrom);
        if (close < 0) return null;

        const content = value.slice(contentStart, close);
        const last = value.charAt(close - 1);
        const after = close + delimiter.length;
        if (
          content &&
          last !== "*" &&
          !/\s/.test(last) &&
          !content.includes("\n") &&
          markdownEmphasisEndAllowed(value, after)
        ) {
          return {
            from: start,
            to: after,
            delimiter,
          };
        }

        searchFrom = close + 1;
      }

      return null;
    }

    function markdownStrikeStartAllowed(text, index) {
      if (index > 0 && text.charAt(index - 1) === "~") return false;
      return index === 0 || !asciiWordChar(text.charAt(index - 1));
    }

    function markdownStrikeEndAllowed(text, index) {
      if (text.charAt(index) === "~") return false;
      return index >= text.length || !asciiWordChar(text.charAt(index));
    }

    function readMarkdownStrikeSyntax(text, start) {
      const value = String(text || "");
      const delimiter = "~~";
      if (!value.startsWith(delimiter, start) || !markdownStrikeStartAllowed(value, start)) {
        return null;
      }

      const contentStart = start + delimiter.length;
      const first = value.charAt(contentStart);
      if (!first || first === "~" || /\s/.test(first)) return null;

      let searchFrom = contentStart + 1;
      while (searchFrom < value.length) {
        const close = value.indexOf(delimiter, searchFrom);
        if (close < 0) return null;

        const content = value.slice(contentStart, close);
        const last = value.charAt(close - 1);
        const after = close + delimiter.length;
        if (
          content &&
          last !== "~" &&
          !/\s/.test(last) &&
          !content.includes("\n") &&
          markdownStrikeEndAllowed(value, after)
        ) {
          return {
            from: start,
            to: after,
          };
        }

        searchFrom = close + 1;
      }

      return null;
    }

    function normalizeInlineCodeText(text, delimiter) {
      const value = String(text || "");
      if (
        delimiter.length > 1 &&
        value.length > 2 &&
        value.startsWith(" ") &&
        value.endsWith(" ") &&
        (value[1] === "`" || value[value.length - 2] === "`")
      ) {
        return value.slice(1, -1);
      }

      return value;
    }

    function readInlineCodeSpan(text, start) {
      const open = /^`+/.exec(text.slice(start));
      if (!open) return null;

      const delimiter = open[0];
      const contentStart = start + delimiter.length;
      const close = text.indexOf(delimiter, contentStart);
      if (close < 0) return null;

      const content = text.slice(contentStart, close);
      if (!content) return null;

      return {
        from: start,
        to: close + delimiter.length,
        text: normalizeInlineCodeText(content, delimiter),
      };
    }

    function markdownTablePipeIsEscaped(text, index) {
      let slashCount = 0;
      for (let cursor = index - 1; cursor >= 0 && text.charAt(cursor) === "\\"; cursor -= 1) {
        slashCount += 1;
      }
      return slashCount % 2 === 1;
    }

    function markdownTableCellFromRaw(line, from, to) {
      const raw = line.slice(from, to);
      const leading = (/^\s*/.exec(raw) || [""])[0].length;
      const trailing = (/\s*$/.exec(raw) || [""])[0].length;
      const end = raw.length - trailing;

      if (leading >= end) {
        return { text: "", offset: from + raw.length };
      }

      return {
        text: raw.slice(leading, end).replace(/\\\|/g, "|"),
        offset: from + leading,
      };
    }

    function splitMarkdownTableRow(line) {
      const value = String(line || "");
      const firstPipe = value.indexOf("|");
      if (firstPipe < 0 || value.slice(0, firstPipe).trim()) return null;

      let lastNonSpace = value.length - 1;
      while (lastNonSpace >= 0 && /\s/.test(value.charAt(lastNonSpace))) {
        lastNonSpace -= 1;
      }
      if (
        lastNonSpace <= firstPipe ||
        value.charAt(lastNonSpace) !== "|" ||
        markdownTablePipeIsEscaped(value, lastNonSpace)
      ) {
        return null;
      }

      const cells = [];
      let cellStart = firstPipe + 1;
      for (let index = cellStart; index <= lastNonSpace; index += 1) {
        if (value.charAt(index) !== "|" || markdownTablePipeIsEscaped(value, index)) continue;

        cells.push(markdownTableCellFromRaw(value, cellStart, index));
        cellStart = index + 1;
      }

      return cells.length ? { cells } : null;
    }

    function markdownTableSeparatorCell(text) {
      const value = String(text || "").trim().replace(/\s+/g, "");
      if (!/^:?-{3,}:?$/.test(value)) return null;

      const starts = value.startsWith(":");
      const ends = value.endsWith(":");
      return {
        align: starts && ends ? "center" : ends ? "right" : starts ? "left" : null,
      };
    }

    function miniLinkText(text) {
      return String(text || "").replace(/[)\].,;:!?]+$/u, "");
    }

    function miniMarkdownLinkRanges(text) {
      const ranges = [];
      const value = String(text || "");

      for (let index = 0; index < value.length; index += 1) {
        const token = readMarkdownLinkSyntax(value, index);
        if (!token) continue;

        ranges.push({
          from: token.from,
          to: token.to,
          image: token.type === "image",
        });
        index = token.to - 1;
      }

      return ranges;
    }

    function rangeOverlaps(ranges, from, to) {
      return ranges.some((range) => from < range.to && to > range.from);
    }

    function pushMiniInlineDecoration(decorations, from, to, className, attrs) {
      if (from >= to) return;

      decorations.push(
        PM.Decoration.inline(from, to, {
          class: className,
          ...(attrs || {}),
        }),
      );
    }

    function pushMiniLineDecoration(decorations, from, to, className) {
      if (from >= to) return;

      decorations.push(
        PM.Decoration.node(from, to, {
          class: className,
        }),
      );
    }

    function pushMiniInlineCodeDecorations(decorations, text, basePos, excludedRanges) {
      let searchFrom = 0;

      while (searchFrom < text.length) {
        const tickIndex = text.indexOf("`", searchFrom);
        if (tickIndex < 0) return;

        const codeSpan = readInlineCodeSpan(text, tickIndex);
        if (!codeSpan) {
          searchFrom = tickIndex + 1;
          continue;
        }

        if (rangeOverlaps(excludedRanges || [], codeSpan.from, codeSpan.to)) {
          searchFrom = codeSpan.to;
          continue;
        }

        pushMiniInlineDecoration(
          decorations,
          basePos + codeSpan.from,
          basePos + codeSpan.to,
          "mini-inline-code",
        );
        searchFrom = codeSpan.to;
      }
    }

    function pushMiniMarkdownEmphasisDecorations(decorations, text, basePos, excludedRanges) {
      let searchFrom = 0;

      while (searchFrom < text.length) {
        const starIndex = text.indexOf("*", searchFrom);
        if (starIndex < 0) return;

        const token = readMarkdownEmphasisSyntax(text, starIndex);
        if (!token) {
          searchFrom = starIndex + 1;
          continue;
        }

        if (rangeOverlaps(excludedRanges || [], token.from, token.to)) {
          searchFrom = token.to;
          continue;
        }

        const isStrong = token.delimiter.length >= 2;
        const isEm = token.delimiter.length === 1 || token.delimiter.length === 3;
        pushMiniInlineDecoration(
          decorations,
          basePos + token.from,
          basePos + token.to,
          "mini-markdown-emphasis-token" +
            (isStrong ? " mini-markdown-strong-token" : "") +
            (isEm ? " mini-markdown-em-token" : ""),
        );
        searchFrom = token.to;
      }
    }

    function pushMiniMarkdownStrikeDecorations(decorations, text, basePos, excludedRanges) {
      let searchFrom = 0;

      while (searchFrom < text.length) {
        const tildeIndex = text.indexOf("~", searchFrom);
        if (tildeIndex < 0) return;

        const token = readMarkdownStrikeSyntax(text, tildeIndex);
        if (!token) {
          searchFrom = tildeIndex + 1;
          continue;
        }

        if (rangeOverlaps(excludedRanges || [], token.from, token.to)) {
          searchFrom = token.to;
          continue;
        }

        pushMiniInlineDecoration(
          decorations,
          basePos + token.from,
          basePos + token.to,
          "mini-markdown-strike-token",
        );
        searchFrom = token.to;
      }
    }

    function pushMiniHashtagDecorations(decorations, text, basePos, excludedRanges) {
      const matcher = /(^|\s)(#[\p{L}\p{N}_-]+)/gu;
      let match;

      while ((match = matcher.exec(text))) {
        const relativeFrom = match.index + match[1].length;
        const relativeTo = relativeFrom + match[2].length;
        if (rangeOverlaps(excludedRanges || [], relativeFrom, relativeTo)) continue;

        pushMiniInlineDecoration(
          decorations,
          basePos + relativeFrom,
          basePos + relativeTo,
          "mini-hashtag-token",
        );
      }
    }

    function pushMiniMarkdownLinkDecorations(decorations, ranges, basePos) {
      ranges.forEach((range) => {
        pushMiniInlineDecoration(
          decorations,
          basePos + range.from,
          basePos + range.to,
          range.image ? "mini-markdown-image-token" : "mini-markdown-link-token",
        );
      });
    }

    class ProsemirrorEditor {
      constructor(options) {
        const editorOptions = options || {};
        const element = editorOptions.$el || editorOptions.el || editorOptions.element;

        if (!element || typeof element.appendChild !== "function") {
          throw new Error("ProsemirrorEditor requires a valid $el option.");
        }

        this.mode = editorOptions.mode || "mini";
        if (this.mode !== "mini") {
          throw new Error('Only mode: "mini" is supported by this UMD editor.');
        }

        editorCounter += 1;
        this.id = "prosemirror-editor-" + editorCounter;
        this.$el = element;
        this.options = editorOptions;
        this.destroyed = false;
        this.fileItems = normalizeFileItems(editorOptions.fileItems || editorOptions.files || []);
        this.knownFileItems = this.fileItems.slice();
        this.uploadTasks = new Map();
        this.callbacks = {
          selectFile: [],
          removeFile: [],
          uploadImage: [],
          fileQuery: [],
          change: [],
          save: [],
        };
        this.keys = {
          filePicker: new PM.PluginKey(this.id + "-filePicker"),
          imageUpload: new PM.PluginKey(this.id + "-imageUpload"),
          miniFileToken: new PM.PluginKey(this.id + "-miniFileToken"),
          miniLinkToken: new PM.PluginKey(this.id + "-miniLinkToken"),
          miniMarkdownDecoration: new PM.PluginKey(this.id + "-miniMarkdownDecoration"),
        };

        if (typeof editorOptions.onSelectFile === "function") {
          this.onSelectFile(editorOptions.onSelectFile);
        }
        if (typeof editorOptions.onRemoveFile === "function") {
          this.onRemoveFile(editorOptions.onRemoveFile);
        }
        if (typeof editorOptions.onUploadImage === "function") {
          this.onUploadImage(editorOptions.onUploadImage);
        }
        if (typeof editorOptions.onFileQuery === "function") {
          this.onFileQuery(editorOptions.onFileQuery);
        }
        if (typeof editorOptions.onChange === "function") {
          this.onChange(editorOptions.onChange);
        }
        if (typeof editorOptions.onSave === "function") {
          this.onSave(editorOptions.onSave);
        }

        this.$el.classList.add("mini-editor");
        this.view = new PM.EditorView(this.$el, {
          state: PM.EditorState.create({
            doc: this.initialDoc(editorOptions),
            plugins: this.buildMiniPlugins(),
          }),
          attributes: {
            autocapitalize: "off",
            autocomplete: "off",
            autocorrect: "off",
            "data-enable-grammarly": "false",
            "data-gramm": "false",
            "data-gramm_editor": "false",
            spellcheck: "false",
            writingsuggestions: "false",
          },
          handleScrollToSelection: scrollSelectionIntoEditorPadding,
          dispatchTransaction: (transaction) => {
            this.dispatchTransaction(transaction);
          },
        });
      }

      initialDoc(options) {
        if (options.doc) {
          try {
            const doc =
              typeof options.doc.toJSON === "function"
                ? options.doc
                : schema.nodeFromJSON(options.doc);
            return miniDocFromPlainText(plainTextFromMiniDoc(doc));
          } catch (error) {
            root.console && root.console.warn("Failed to load initial doc.", error);
          }
        }

        const value =
          options.value != null ? options.value : options.text != null ? options.text : "";
        return miniDocFromPlainText(String(value || ""));
      }

      addCallback(type, callback) {
        if (typeof callback !== "function") return function () {};

        this.callbacks[type].push(callback);
        return () => {
          this.callbacks[type] = this.callbacks[type].filter((item) => item !== callback);
        };
      }

      emit(type) {
        const args = Array.prototype.slice.call(arguments, 1);
        this.callbacks[type].slice().forEach((callback) => {
          try {
            callback.apply(null, args);
          } catch (error) {
            root.console && root.console.error("ProsemirrorEditor callback failed.", error);
          }
        });
      }

      onSelectFile(callback) {
        return this.addCallback("selectFile", callback);
      }

      onRemoveFile(callback) {
        return this.addCallback("removeFile", callback);
      }

      onUploadImage(callback) {
        return this.addCallback("uploadImage", callback);
      }

      onFileQuery(callback) {
        return this.addCallback("fileQuery", callback);
      }

      onChange(callback) {
        return this.addCallback("change", callback);
      }

      onSave(callback) {
        return this.addCallback("save", callback);
      }

      dispatchTransaction(transaction) {
        if (this.destroyed) return;

        const oldState = this.view.state;
        const nextState = oldState.apply(transaction);
        this.view.updateState(nextState);
        this.emitRemovedFileTokens(oldState, nextState);

        if (transaction.docChanged) {
          this.emit("change", this);
        }
      }

      emitRemovedFileTokens(oldState, newState) {
        const oldPluginState = this.keys.miniFileToken.getState(oldState);
        const newPluginState = this.keys.miniFileToken.getState(newState);
        const oldTokens = (oldPluginState && oldPluginState.tokens) || [];
        const newTokens = (newPluginState && newPluginState.tokens) || [];
        if (!oldTokens.length) return;

        const nextCounts = new Map();
        newTokens.forEach((token) => {
          const key = tokenIdentity(token);
          nextCounts.set(key, (nextCounts.get(key) || 0) + 1);
        });

        oldTokens.forEach((token) => {
          const key = tokenIdentity(token);
          const count = nextCounts.get(key) || 0;
          if (count > 0) {
            nextCounts.set(key, count - 1);
            return;
          }

          const file = fileFromToken(token);
          if (file) this.emit("removeFile", file);
        });
      }

      focus() {
        this.view.focus();
      }

      destroy() {
        if (this.destroyed) return;

        this.destroyed = true;
        this.uploadTasks.forEach((task) => {
          this.abortImageUploadTask(task, false);
        });
        this.uploadTasks.clear();
        this.view.destroy();
      }

      getText() {
        return plainTextFromMiniDoc(this.view.state.doc);
      }

      setText(text) {
        const doc = miniDocFromPlainText(text);
        this.replaceDoc(doc, [[this.keys.miniFileToken, { type: "setTokens", tokens: [] }]]);
      }

      getJSON() {
        return this.view.state.doc.toJSON();
      }

      setJSON(json) {
        const doc = typeof json.toJSON === "function" ? json : schema.nodeFromJSON(json);
        this.replaceDoc(miniDocFromPlainText(plainTextFromMiniDoc(doc)), [
          [this.keys.miniFileToken, { type: "setTokens", tokens: [] }],
        ]);
      }

      getHTML() {
        return htmlFromDoc(this.view.state.doc);
      }

      getDoc() {
        return this.view.state.doc;
      }

      replaceDoc(doc, metaEntries) {
        let transaction = this.view.state.tr
          .replaceWith(0, this.view.state.doc.content.size, doc.content)
          .setMeta("addToHistory", false);

        (metaEntries || []).forEach(([key, value]) => {
          transaction = transaction.setMeta(key, value);
        });

        this.view.dispatch(transaction);
      }

      rememberFileItems(items) {
        const normalized = normalizeFileItems(items);
        const indexed = new Map();

        normalizeFileItems(this.knownFileItems).forEach((item) => {
          indexed.set(fileItemIdentity(item), item);
        });
        normalized.forEach((item) => {
          indexed.set(fileItemIdentity(item), item);
        });

        this.knownFileItems = Array.from(indexed.values());
        return normalized;
      }

      setFileItems(items, query) {
        const normalized = this.rememberFileItems(items);
        this.fileItems = normalized;

        if (!this.view || this.destroyed) return normalized;

        const pickerState = this.keys.filePicker.getState(this.view.state);
        if (query != null) {
          const expected = String(query);
          if (!pickerState || !pickerState.active || pickerState.query !== expected) {
            return normalized;
          }
        }

        this.view.dispatch(
          this.view.state.tr.setMeta(this.keys.filePicker, {
            type: "setItems",
            items: normalized,
          }),
        );
        return normalized;
      }

      filterFileItems(query) {
        const keyword = String(query || "").trim().toLowerCase();
        return normalizeFileItems(this.knownFileItems)
          .filter((item) => {
            if (!keyword) return true;

            return [item.label, item.href, item.detail]
              .join("\n")
              .toLowerCase()
              .includes(keyword);
          })
          .slice(0, 8);
      }

      selectFile(file) {
        return this.insertFile(file);
      }

      insertFile(file) {
        const normalized = normalizeFileItem(file);
        if (!normalized || this.destroyed) return false;

        this.rememberFileItems([normalized]);
        const reference = "@" + normalized.name;
        const state = this.view.state;
        const tokenFrom = state.selection.from;
        let transaction = state.tr.insertText(reference, state.selection.from, state.selection.to);
        const tokenTo = tokenFrom + reference.length;
        let cursorPos = tokenTo;
        const nextText = state.doc.textBetween(
          state.selection.to,
          Math.min(state.selection.to + 1, state.doc.content.size),
          "\ufffc",
          "\ufffc",
        );

        if (!/\s/.test(nextText)) {
          transaction = transaction.insertText(" ", cursorPos, cursorPos);
          cursorPos += 1;
        }

        transaction = transaction
          .setSelection(PM.Selection.near(transaction.doc.resolve(cursorPos), 1))
          .setMeta(this.keys.miniFileToken, {
            type: "addToken",
            token: {
              from: tokenFrom,
              to: tokenTo,
              text: reference,
              href: normalized.href,
              name: normalized.name,
              label: normalized.label,
              detail: normalized.detail,
            },
          });

        this.view.dispatch(transaction.scrollIntoView());
        this.emit("selectFile", normalized);
        return true;
      }

      findFileTrigger(state) {
        const selection = state.selection;
        if (!selection.empty) return null;

        const $from = selection.$from;
        if (!$from.parent.isTextblock) return null;

        const before = $from.parent.textBetween(0, $from.parentOffset, "\ufffc", "\ufffc");
        const match = /(^|\s)@([^\s@]*)$/u.exec(before);
        if (!match) return null;

        const query = match[2];
        const from = selection.from - query.length - 1;
        if (from < $from.start()) return null;

        return {
          from,
          to: selection.from,
          query,
          key: this.fileTriggerKey({ from, to: selection.from, query }),
        };
      }

      fileTriggerKey(trigger) {
        if (!trigger || trigger.from == null || trigger.to == null) return null;
        return trigger.from + ":" + trigger.to + ":" + trigger.query;
      }

      insertedTextFromTransaction(transaction) {
        if (!transaction.steps) return "";

        return transaction.steps
          .map((step) => {
            if (!step.slice || !step.slice.content) return "";
            return step.slice.content.textBetween(0, step.slice.content.size, "\n", "\ufffc");
          })
          .join("");
      }

      transactionStartsFilePicker(transaction) {
        return this.insertedTextFromTransaction(transaction).includes("@");
      }

      clampPickerIndex(index, length) {
        if (!length) return 0;
        return Math.max(0, Math.min(index, length - 1));
      }

      emptyFilePickerState(dismissedKey) {
        return {
          active: false,
          from: null,
          to: null,
          query: "",
          items: [],
          selectedIndex: 0,
          dismissedKey: dismissedKey || null,
        };
      }

      applyFilePickerState(transaction, value, newState) {
        const meta = transaction.getMeta(this.keys.filePicker);
        let items = value.items;
        let selectedIndex = value.selectedIndex;
        let dismissedKey = value.dismissedKey;

        if (meta && meta.type === "close") {
          const trigger = this.findFileTrigger(newState);
          const dismissed = trigger ? trigger.key : this.fileTriggerKey(value) || dismissedKey;
          return trigger
            ? {
                ...this.emptyFilePickerState(dismissed),
                from: trigger.from,
                to: trigger.to,
                query: trigger.query,
              }
            : this.emptyFilePickerState(dismissed);
        }

        if (meta && meta.type === "setItems") {
          items = normalizeFileItems(meta.items);
          selectedIndex = 0;
        }

        if (meta && meta.type === "setSelectedIndex") {
          selectedIndex = Number(meta.selectedIndex) || 0;
        }

        const trigger = this.findFileTrigger(newState);
        const startsFileInput = this.transactionStartsFilePicker(transaction);
        if (!trigger) {
          return this.emptyFilePickerState(value.active ? this.fileTriggerKey(value) : dismissedKey);
        }
        if (trigger.key === dismissedKey && !startsFileInput) {
          return {
            ...this.emptyFilePickerState(dismissedKey),
            from: trigger.from,
            to: trigger.to,
            query: trigger.query,
            items,
            selectedIndex: this.clampPickerIndex(selectedIndex, items.length),
          };
        }

        const sameTrigger =
          value.active &&
          value.from === trigger.from &&
          value.to === trigger.to &&
          value.query === trigger.query;

        if (value.active && !transaction.docChanged && !sameTrigger) {
          return {
            ...this.emptyFilePickerState(this.fileTriggerKey(value)),
            from: trigger.from,
            to: trigger.to,
            query: trigger.query,
          };
        }

        if (!value.active && !startsFileInput) {
          return {
            ...this.emptyFilePickerState(dismissedKey),
            from: trigger.from,
            to: trigger.to,
            query: trigger.query,
          };
        }

        const nextItems = sameTrigger ? items : [];

        return {
          active: true,
          from: trigger.from,
          to: trigger.to,
          query: trigger.query,
          items: nextItems,
          selectedIndex: sameTrigger ? this.clampPickerIndex(selectedIndex, nextItems.length) : 0,
          dismissedKey: null,
        };
      }

      scheduleFilePickerChange(editorView, pickerState) {
        const query = pickerState.query;
        const from = pickerState.from;
        const to = pickerState.to;
        const notify = () => {
          if (this.destroyed) return;

          const latest = this.keys.filePicker.getState(editorView.state);
          if (
            !latest ||
            !latest.active ||
            latest.query !== query ||
            latest.from !== from ||
            latest.to !== to
          ) {
            return;
          }

          if (!this.callbacks.fileQuery.length) {
            this.setFileItems(this.filterFileItems(query), query);
          }
          this.emit("fileQuery", query, { from, to, view: editorView, editor: this });
        };

        if (root.queueMicrotask) {
          root.queueMicrotask(notify);
        } else {
          root.setTimeout(notify, 0);
        }
      }

      selectFilePickerItem(viewInstance, item) {
        const pickerState = this.keys.filePicker.getState(viewInstance.state);
        const file = normalizeFileItem(item);
        if (!pickerState || !pickerState.active || !file) return false;

        this.rememberFileItems([file]);
        const reference = "@" + file.name;
        let transaction = viewInstance.state.tr.insertText(
          reference,
          pickerState.from,
          pickerState.to,
        );
        const tokenFrom = pickerState.from;
        const tokenTo = tokenFrom + reference.length;
        let cursorPos = tokenTo;
        const nextText = viewInstance.state.doc.textBetween(
          pickerState.to,
          Math.min(pickerState.to + 1, viewInstance.state.doc.content.size),
          "\ufffc",
          "\ufffc",
        );

        if (!/\s/.test(nextText)) {
          transaction = transaction.insertText(" ", cursorPos, cursorPos);
          cursorPos += 1;
        }

        transaction = transaction
          .setSelection(PM.Selection.near(transaction.doc.resolve(cursorPos), 1))
          .setMeta(this.keys.filePicker, { type: "close" })
          .setMeta(this.keys.miniFileToken, {
            type: "addToken",
            token: {
              from: tokenFrom,
              to: tokenTo,
              text: reference,
              href: file.href,
              name: file.name,
              label: file.label,
              detail: file.detail,
            },
          });

        viewInstance.dispatch(transaction.scrollIntoView());
        this.emit("selectFile", file);
        return true;
      }

      renderFilePickerMenu(menu, pickerState) {
        const fragment = document.createDocumentFragment();

        if (!pickerState.items.length) {
          const empty = document.createElement("div");
          empty.className = "file-picker-empty";
          empty.textContent = pickerState.query ? "No matching files" : "Type to search files";
          fragment.appendChild(empty);
        } else {
          pickerState.items.forEach((item, index) => {
            const option = document.createElement("div");
            option.className =
              "file-picker-option" + (index === pickerState.selectedIndex ? " active" : "");
            option.dataset.filePickerIndex = String(index);
            option.setAttribute("role", "option");
            option.setAttribute("aria-selected", String(index === pickerState.selectedIndex));

            const label = document.createElement("span");
            label.className = "file-picker-label";
            label.textContent = item.label;
            option.appendChild(label);

            if (item.detail) {
              const detail = document.createElement("span");
              detail.className = "file-picker-detail";
              detail.textContent = item.detail;
              option.appendChild(detail);
            }

            fragment.appendChild(option);
          });
        }

        menu.replaceChildren(fragment);
      }

      positionFilePickerMenu(editorView, menu, pickerState) {
        if (!pickerState.active || menu.classList.contains("hidden")) return;

        try {
          const coords = editorView.coordsAtPos(pickerState.to);
          const rect = menu.getBoundingClientRect();
          const margin = 8;
          const maxLeft = Math.max(margin, root.innerWidth - rect.width - margin);
          const left = Math.min(Math.max(coords.left, margin), maxLeft);
          let top = coords.bottom + 6;

          if (top + rect.height > root.innerHeight - margin) {
            top = Math.max(margin, coords.top - rect.height - 6);
          }

          menu.style.left = left + "px";
          menu.style.top = top + "px";
        } catch (error) {
          menu.classList.add("hidden");
        }
      }

      createFilePickerView(editorView) {
        const menu = document.createElement("div");
        let positionFrame = 0;

        menu.className = "file-picker-menu hidden";
        menu.setAttribute("role", "listbox");
        menu.setAttribute("aria-label", "File picker");
        menu.addEventListener("mousedown", (event) => {
          const option = event.target.closest("[data-file-picker-index]");
          if (!option) return;

          event.preventDefault();
          const pickerState = this.keys.filePicker.getState(editorView.state);
          const item = pickerState.items[Number(option.dataset.filePickerIndex)];
          this.selectFilePickerItem(editorView, item);
          editorView.focus();
        });
        document.body.appendChild(menu);

        const schedulePosition = (pickerState) => {
          root.cancelAnimationFrame(positionFrame);
          positionFrame = root.requestAnimationFrame(() => {
            const current = pickerState || this.keys.filePicker.getState(editorView.state);
            if (current) this.positionFilePickerMenu(editorView, menu, current);
          });
        };

        const update = (viewInstance, prevState) => {
          const previous = prevState
            ? this.keys.filePicker.getState(prevState)
            : this.emptyFilePickerState();
          const current = this.keys.filePicker.getState(viewInstance.state);

          if (
            current.active &&
            (!previous.active || previous.query !== current.query || previous.from !== current.from)
          ) {
            this.scheduleFilePickerChange(viewInstance, current);
          }

          if (!current.active) {
            menu.classList.add("hidden");
            return;
          }

          this.renderFilePickerMenu(menu, current);
          menu.classList.remove("hidden");
          schedulePosition(current);
        };

        const reposition = () => schedulePosition();
        root.addEventListener("resize", reposition);
        root.addEventListener("scroll", reposition, true);
        update(editorView);

        return {
          update,
          destroy() {
            root.cancelAnimationFrame(positionFrame);
            root.removeEventListener("resize", reposition);
            root.removeEventListener("scroll", reposition, true);
            menu.remove();
          },
        };
      }

      handleFilePickerKeyDown(viewInstance, event) {
        const pickerState = this.keys.filePicker.getState(viewInstance.state);
        if (!pickerState || !pickerState.active) return false;

        if (event.key === "ArrowDown" || event.key === "ArrowUp") {
          const direction = event.key === "ArrowDown" ? 1 : -1;
          const length = pickerState.items.length;
          const selectedIndex = length
            ? (pickerState.selectedIndex + direction + length) % length
            : 0;
          event.preventDefault();
          viewInstance.dispatch(
            viewInstance.state.tr.setMeta(this.keys.filePicker, {
              type: "setSelectedIndex",
              selectedIndex,
            }),
          );
          return true;
        }

        if (event.key === "Enter" || event.key === "Tab") {
          if (!pickerState.items.length) return false;
          event.preventDefault();
          return this.selectFilePickerItem(viewInstance, pickerState.items[pickerState.selectedIndex]);
        }

        if (event.key === "Escape") {
          event.preventDefault();
          viewInstance.dispatch(
            viewInstance.state.tr.setMeta(this.keys.filePicker, { type: "close" }),
          );
          return true;
        }

        return false;
      }

      filePickerPlugin() {
        return new PM.Plugin({
          key: this.keys.filePicker,
          state: {
            init: () => {
              return this.emptyFilePickerState();
            },
            apply: (transaction, value, oldState, newState) => {
              return this.applyFilePickerState(transaction, value, newState);
            },
          },
          props: {
            decorations: (state) => {
              const pickerState = this.keys.filePicker.getState(state);
              if (!pickerState || !pickerState.active || pickerState.from >= pickerState.to) {
                return PM.DecorationSet.empty;
              }

              return PM.DecorationSet.create(state.doc, [
                PM.Decoration.inline(pickerState.from, pickerState.to, {
                  class: "file-query-range",
                }),
              ]);
            },
            handleKeyDown: (viewInstance, event) => {
              return this.handleFilePickerKeyDown(viewInstance, event);
            },
            handleDOMEvents: {
              blur: (viewInstance) => {
                const pickerState = this.keys.filePicker.getState(viewInstance.state);
                if (!pickerState || !pickerState.active) return false;

                viewInstance.dispatch(
                  viewInstance.state.tr.setMeta(this.keys.filePicker, { type: "close" }),
                );
                return false;
              },
            },
          },
          view: (viewInstance) => {
            return this.createFilePickerView(viewInstance);
          },
        });
      }

      miniPlainTextSlice(text) {
        const doc = miniDocFromPlainText(text);
        return doc.slice(0, doc.content.size);
      }

      miniPlainTextPlugin() {
        return new PM.Plugin({
          props: {
            clipboardTextParser: (text) => {
              return this.miniPlainTextSlice(text);
            },
            handlePaste: (viewInstance, event) => {
              if (!event.clipboardData) return false;

              let text = event.clipboardData.getData("text/plain");
              const html = event.clipboardData.getData("text/html");
              if (!text && html) {
                const wrap = document.createElement("div");
                wrap.innerHTML = html;
                text = wrap.textContent || "";
              }
              if (!text && !html) return false;

              event.preventDefault();
              const transaction = text.includes("\n")
                ? viewInstance.state.tr.replaceSelection(this.miniPlainTextSlice(text))
                : viewInstance.state.tr.insertText(text);
              viewInstance.dispatch(transaction.scrollIntoView());
              return true;
            },
          },
        });
      }

      miniHardBreakCleanupPlugin() {
        return new PM.Plugin({
          appendTransaction(transactions, oldState, newState) {
            if (!transactions.some((transaction) => transaction.docChanged)) return null;

            const ranges = trailingHardBreakRanges(newState.doc);
            if (!ranges.length) return null;

            let transaction = newState.tr;
            ranges
              .sort((left, right) => right.from - left.from)
              .forEach((range) => {
                transaction = transaction.delete(range.from, range.to);
              });
            return transaction.setMeta("addToHistory", false);
          },
        });
      }

      normalizeMiniFileToken(token, doc) {
        if (!token || token.from == null || token.to == null) return null;
        const from = Math.max(0, Math.min(token.from, doc.content.size));
        const to = Math.max(from, Math.min(token.to, doc.content.size));
        const text = token.text || doc.textBetween(from, to, "\n", "\ufffc");
        if (!text || from === to) return null;

        return {
          from,
          to,
          text,
          href: token.href || text.replace(/^@/, ""),
          name: token.name || text.replace(/^@/, ""),
          label: token.label || token.name || text.replace(/^@/, ""),
          detail: token.detail || "",
        };
      }

      tokenTextIsIntact(doc, token) {
        return doc.textBetween(token.from, token.to, "\n", "\ufffc") === token.text;
      }

      applyMiniFileTokenState(transaction, value, oldState, newState) {
        const meta = transaction.getMeta(this.keys.miniFileToken);
        let tokens = (value && value.tokens ? value.tokens : [])
          .map((token) => {
            return this.normalizeMiniFileToken(
              {
                ...token,
                from: transaction.mapping.map(token.from, 1),
                to: transaction.mapping.map(token.to, -1),
              },
              newState.doc,
            );
          })
          .filter((token) => token && this.tokenTextIsIntact(newState.doc, token));

        if (meta && meta.type === "setTokens") {
          tokens = (meta.tokens || [])
            .map((token) => this.normalizeMiniFileToken(token, newState.doc))
            .filter(Boolean);
        }

        if (meta && meta.type === "addToken") {
          const token = this.normalizeMiniFileToken(meta.token, newState.doc);
          if (token) {
            tokens = tokens.filter((item) => item.to <= token.from || item.from >= token.to);
            tokens.push(token);
          }
        }

        return { tokens };
      }

      miniFileTokenDecorations(state) {
        const tokenState = this.keys.miniFileToken.getState(state);
        const decorations = (tokenState && tokenState.tokens ? tokenState.tokens : []).map(
          (token) => {
            return PM.Decoration.inline(token.from, token.to, {
              class: "mini-file-token",
              title: token.detail || token.href,
            });
          },
        );
        return decorations.length
          ? PM.DecorationSet.create(state.doc, decorations)
          : PM.DecorationSet.empty;
      }

      miniFileTokenPlugin() {
        return new PM.Plugin({
          key: this.keys.miniFileToken,
          state: {
            init() {
              return { tokens: [] };
            },
            apply: (transaction, value, oldState, newState) => {
              return this.applyMiniFileTokenState(transaction, value, oldState, newState);
            },
          },
          props: {
            decorations: (state) => {
              return this.miniFileTokenDecorations(state);
            },
          },
        });
      }

      normalizeMiniLinkToken(token, doc) {
        if (!token || token.from == null || token.to == null) return null;
        const from = Math.max(0, Math.min(token.from, doc.content.size));
        const to = Math.max(from, Math.min(token.to, doc.content.size));
        const text = token.text || doc.textBetween(from, to, "\n", "\ufffc");
        if (!text || from === to) return null;

        return {
          from,
          to,
          text,
          href: token.href || text,
          title: token.title || token.href || text,
        };
      }

      applyMiniLinkTokenState(transaction, value, oldState, newState) {
        const meta = transaction.getMeta(this.keys.miniLinkToken);
        let tokens = (value && value.tokens ? value.tokens : [])
          .map((token) => {
            return this.normalizeMiniLinkToken(
              {
                ...token,
                from: transaction.mapping.map(token.from, 1),
                to: transaction.mapping.map(token.to, -1),
              },
              newState.doc,
            );
          })
          .filter((token) => token && this.tokenTextIsIntact(newState.doc, token));

        if (meta && meta.type === "setTokens") {
          tokens = (meta.tokens || [])
            .map((token) => this.normalizeMiniLinkToken(token, newState.doc))
            .filter(Boolean);
        }

        return { tokens };
      }

      miniLinkDecorations(state) {
        const decorations = [];
        const tokenState = this.keys.miniLinkToken.getState(state);
        const tokenRanges = [];
        const matcher = /https?:\/\/[^\s<>"'`]+/giu;

        (tokenState && tokenState.tokens ? tokenState.tokens : []).forEach((token) => {
          tokenRanges.push({ from: token.from, to: token.to });
          decorations.push(
            PM.Decoration.inline(token.from, token.to, {
              class: "mini-link-token",
              title: token.title || token.href,
            }),
          );
        });

        state.doc.descendants((node, pos) => {
          if (!node.isTextblock) return true;

          const text = node.textContent;
          let match;
          matcher.lastIndex = 0;
          const markdownLinkRanges = miniMarkdownLinkRanges(text).map((range) => {
            return {
              from: pos + 1 + range.from,
              to: pos + 1 + range.to,
            };
          });

          while ((match = matcher.exec(text))) {
            const href = miniLinkText(match[0]);
            if (!href) continue;

            const from = pos + 1 + match.index;
            const to = from + href.length;
            if (rangeOverlaps(tokenRanges, from, to)) continue;
            if (rangeOverlaps(markdownLinkRanges, from, to)) continue;

            decorations.push(
              PM.Decoration.inline(from, to, {
                class: "mini-link-token",
                title: href,
              }),
            );
          }

          return true;
        });

        return decorations.length
          ? PM.DecorationSet.create(state.doc, decorations)
          : PM.DecorationSet.empty;
      }

      miniLinkTokenPlugin() {
        return new PM.Plugin({
          key: this.keys.miniLinkToken,
          state: {
            init() {
              return { tokens: [] };
            },
            apply: (transaction, value, oldState, newState) => {
              return this.applyMiniLinkTokenState(transaction, value, oldState, newState);
            },
          },
          props: {
            decorations: (state) => {
              return this.miniLinkDecorations(state);
            },
          },
        });
      }

      miniMarkdownDecorations(state) {
        const decorations = [];
        let inCodeBlock = false;

        state.doc.forEach((node, offset) => {
          if (!node.isTextblock) return;

          const text = node.textContent;
          const lineFrom = offset;
          const lineTo = offset + node.nodeSize;
          const textFrom = offset + 1;
          const fence = /^```\S*\s*$/.test(text);

          if (fence) {
            pushMiniLineDecoration(decorations, lineFrom, lineTo, "mini-code-fence-line");
            pushMiniInlineDecoration(
              decorations,
              textFrom,
              textFrom + text.length,
              "mini-markdown-syntax",
            );
            inCodeBlock = !inCodeBlock;
            return;
          }

          if (inCodeBlock) {
            pushMiniLineDecoration(decorations, lineFrom, lineTo, "mini-code-block-line");
            return;
          }

          const heading = /^(#{1,6})(\s+.*)$/.exec(text);
          if (heading) {
            const level = Math.min(6, heading[1].length);
            pushMiniLineDecoration(
              decorations,
              lineFrom,
              lineTo,
              "mini-heading-line mini-heading-level-" + level,
            );
            pushMiniInlineDecoration(
              decorations,
              textFrom,
              textFrom + heading[1].length,
              "mini-markdown-syntax",
            );
          }

          const todo = /^(\s*[-+*]\s+\[([ xX])\]\s?)/.exec(text);
          if (todo) {
            const done = todo[2].toLowerCase() === "x";
            pushMiniLineDecoration(
              decorations,
              lineFrom,
              lineTo,
              "mini-todo-line" + (done ? " mini-todo-done" : ""),
            );
            pushMiniInlineDecoration(
              decorations,
              textFrom,
              textFrom + todo[1].length,
              "mini-markdown-syntax mini-todo-marker",
            );
          }

          if (!todo) {
            const list = /^(\s*(?:[-+*]|\d+\.)\s+)/.exec(text);
            if (list) {
              pushMiniLineDecoration(decorations, lineFrom, lineTo, "mini-list-line");
              pushMiniInlineDecoration(
                decorations,
                textFrom,
                textFrom + list[1].length,
                "mini-markdown-syntax",
              );
            }
          }

          const quote = /^(\s*>\s?)/.exec(text);
          if (quote) {
            pushMiniLineDecoration(decorations, lineFrom, lineTo, "mini-blockquote-line");
            pushMiniInlineDecoration(
              decorations,
              textFrom,
              textFrom + quote[1].length,
              "mini-markdown-syntax",
            );
          }

          const tableRow = splitMarkdownTableRow(text);
          if (tableRow) {
            const isSeparator = tableRow.cells.every((cell) => markdownTableSeparatorCell(cell.text));
            pushMiniLineDecoration(
              decorations,
              lineFrom,
              lineTo,
              "mini-table-line" + (isSeparator ? " mini-table-separator-line" : ""),
            );
          }

          const markdownLinkRanges = miniMarkdownLinkRanges(text);
          pushMiniMarkdownLinkDecorations(decorations, markdownLinkRanges, textFrom);
          pushMiniMarkdownEmphasisDecorations(decorations, text, textFrom, markdownLinkRanges);
          pushMiniMarkdownStrikeDecorations(decorations, text, textFrom, markdownLinkRanges);
          pushMiniInlineCodeDecorations(decorations, text, textFrom, markdownLinkRanges);
          pushMiniHashtagDecorations(decorations, text, textFrom, markdownLinkRanges);
        });

        return decorations.length
          ? PM.DecorationSet.create(state.doc, decorations)
          : PM.DecorationSet.empty;
      }

      miniMarkdownDecorationPlugin() {
        return new PM.Plugin({
          key: this.keys.miniMarkdownDecoration,
          props: {
            decorations: (state) => {
              return this.miniMarkdownDecorations(state);
            },
          },
        });
      }

      canInsertImageUploadNode(state) {
        const nodeType = schema.nodes.image_upload;
        const $from = state.selection.$from;

        for (let depth = $from.depth; depth >= 0; depth -= 1) {
          const parent = $from.node(depth);
          const index = $from.index(depth);
          if (parent.canReplaceWith(index, index, nodeType)) return true;
        }
        return false;
      }

      findImageUploadNode(viewInstance, id) {
        let found = null;
        viewInstance.state.doc.descendants((node, pos) => {
          if (node.type !== schema.nodes.image_upload || node.attrs.id !== id) return true;

          found = { node, pos };
          return false;
        });
        return found;
      }

      updateImageUploadNode(viewInstance, id, attrs) {
        if (!viewInstance || this.destroyed) return false;

        const found = this.findImageUploadNode(viewInstance, id);
        if (!found) return false;

        viewInstance.dispatch(
          viewInstance.state.tr
            .setNodeMarkup(found.pos, null, { ...found.node.attrs, ...attrs })
            .setMeta("addToHistory", false),
        );
        return true;
      }

      replaceImageUploadNode(viewInstance, id, node) {
        if (!viewInstance || this.destroyed) return false;

        const found = this.findImageUploadNode(viewInstance, id);
        if (!found) return false;

        viewInstance.dispatch(
          viewInstance.state.tr
            .replaceWith(found.pos, found.pos + found.node.nodeSize, node)
            .setMeta("addToHistory", false)
            .scrollIntoView(),
        );
        return true;
      }

      abortImageUploadTask(task, announce) {
        if (!task || task.cancelled) return;

        task.cancelled = true;
        if (task.controller) task.controller.abort();
        this.uploadTasks.delete(task.id);
        if (announce && task.image) task.image.setError("Upload cancelled");
      }

      createImageHandle(viewInstance, file, id, fileName, signal) {
        return {
          id,
          file,
          fileName,
          signal,
          setLoading: (loading) => {
            if (loading === false) {
              return this.updateImageUploadNode(viewInstance, id, {
                status: "success",
                message: fileName,
                fileName,
              });
            }

            return this.updateImageUploadNode(viewInstance, id, {
              status: "uploading",
              message: "",
              fileName,
            });
          },
          setContent: (content) => {
            const value = content || {};
            const url = String(value.url || value.src || value.href || "").trim();
            const name = String(
              value.name || value.alt || value.label || value.title || fileName || url,
            ).trim();

            this.uploadTasks.delete(id);
            if (!url) {
              return this.updateImageUploadNode(viewInstance, id, {
                status: "success",
                message: name || fileName,
                fileName,
              });
            }

            return this.replaceImageUploadNode(
              viewInstance,
              id,
              schema.text(imageMarkdownPlainText({ alt: name || url, src: url })),
            );
          },
          setError: (error) => {
            this.uploadTasks.delete(id);
            return this.updateImageUploadNode(viewInstance, id, {
              status: "error",
              message: imageUploadErrorText(error),
              fileName,
            });
          },
          remove: () => {
            this.uploadTasks.delete(id);
            const found = this.findImageUploadNode(viewInstance, id);
            if (!found) return false;

            viewInstance.dispatch(
              viewInstance.state.tr
                .delete(found.pos, found.pos + found.node.nodeSize)
                .setMeta("addToHistory", false)
                .scrollIntoView(),
            );
            return true;
          },
        };
      }

      emitUploadImage(image) {
        const handlers = this.callbacks.uploadImage.slice();
        if (!handlers.length) {
          image.setError(new Error("No image upload handler registered."));
          return;
        }

        handlers.forEach((callback) => {
          try {
            const result = callback(image);
            if (result && typeof result.then === "function") {
              result
                .then((content) => {
                  if (content != null) image.setContent(content);
                })
                .catch((error) => {
                  image.setError(error);
                });
            }
          } catch (error) {
            image.setError(error);
          }
        });
      }

      startImageUpload(viewInstance, file, id, fileName) {
        const controller =
          typeof root.AbortController === "function" ? new root.AbortController() : null;
        const image = this.createImageHandle(
          viewInstance,
          file,
          id,
          fileName,
          controller ? controller.signal : null,
        );
        const task = {
          id,
          view: viewInstance,
          controller,
          image,
          cancelled: false,
        };

        this.uploadTasks.set(id, task);
        this.emitUploadImage(image);
      }

      reconcileImageUploadTasks(viewInstance) {
        this.uploadTasks.forEach((task) => {
          if (task.view !== viewInstance) return;
          if (this.findImageUploadNode(viewInstance, task.id)) return;

          this.abortImageUploadTask(task, false);
        });
      }

      createImageUploadPluginView(viewInstance) {
        this.reconcileImageUploadTasks(viewInstance);

        return {
          update: (editorView) => {
            this.reconcileImageUploadTasks(editorView);
          },
          destroy: () => {
            this.uploadTasks.forEach((task) => {
              if (task.view === viewInstance) this.abortImageUploadTask(task, false);
            });
          },
        };
      }

      imageUploadPastePlugin() {
        return new PM.Plugin({
          key: this.keys.imageUpload,
          props: {
            handlePaste: (viewInstance, event) => {
              const files = imageFilesFromClipboard(event.clipboardData);
              if (!files.length) return false;
              if (!this.canInsertImageUploadNode(viewInstance.state)) return false;

              const uploads = files.map((file, index) => {
                return {
                  id: nextImageUploadId(),
                  file,
                  fileName: imageUploadFileName(file, index),
                };
              });
              const nodes = [];
              uploads.forEach((upload, index) => {
                if (index) nodes.push(schema.text(" "));
                nodes.push(
                  schema.nodes.image_upload.create({
                    id: upload.id,
                    status: "uploading",
                    fileName: upload.fileName,
                  }),
                );
              });

              const nextText = viewInstance.state.doc.textBetween(
                viewInstance.state.selection.to,
                Math.min(viewInstance.state.selection.to + 1, viewInstance.state.doc.content.size),
                "\ufffc",
                "\ufffc",
              );
              if (!/\s/.test(nextText)) nodes.push(schema.text(" "));

              const selectionFrom = viewInstance.state.selection.from;
              const slice = new PM.Slice(PM.Fragment.fromArray(nodes), 0, 0);
              let transaction = viewInstance.state.tr.replaceSelection(slice);
              const cursorPos = Math.min(
                transaction.mapping.map(selectionFrom, 1),
                transaction.doc.content.size,
              );
              transaction = transaction
                .setSelection(PM.Selection.near(transaction.doc.resolve(cursorPos), 1))
                .scrollIntoView();

              event.preventDefault();
              viewInstance.dispatch(transaction);
              uploads.forEach((upload) => {
                this.startImageUpload(viewInstance, upload.file, upload.id, upload.fileName);
              });
              return true;
            },
          },
          view: (viewInstance) => {
            return this.createImageUploadPluginView(viewInstance);
          },
        });
      }

      vimPluginsFor() {
        if (this.options.vim === false || typeof root.createVimPlugin !== "function") return [];

        return root.createVimPlugin({
          onSave: () => {
            this.emit("save", this);
          },
        });
      }

      saveKeymap() {
        return PM.keymap({
          "Mod-s": () => {
            this.emit("save", this);
            return true;
          },
        });
      }

      buildMiniPlugins() {
        return [
          ...this.vimPluginsFor(),
          this.filePickerPlugin(),
          this.imageUploadPastePlugin(),
          this.miniPlainTextPlugin(),
          this.miniHardBreakCleanupPlugin(),
          this.miniFileTokenPlugin(),
          this.miniLinkTokenPlugin(),
          this.miniMarkdownDecorationPlugin(),
          PM.history(),
          PM.keymap({
            "Mod-z": PM.undo,
            "Mod-y": PM.redo,
            "Shift-Mod-z": PM.redo,
          }),
          this.saveKeymap(),
          PM.keymap(PM.baseKeymap),
        ];
      }
    }

    ProsemirrorEditor.schema = schema;
    ProsemirrorEditor.normalizeFileItem = normalizeFileItem;
    ProsemirrorEditor.normalizeFileItems = normalizeFileItems;

    return ProsemirrorEditor;
  },
);
