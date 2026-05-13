function TxtFileLoader() {
  var chapter_title_pattern =
    /^={10,}\s*第([一二三四五六七八九十百千万\d]+)[回卷部章节]\s*(.*?)\s*={10,}$/;
  const methods = {
    parseBookContent(text) {
      var lines = text
        .split("\n")
        .map(function (line) {
          return line.trim();
        })
        .filter(function (line) {
          return line.length > 0;
        });
      var chapters = [];
      var cur_chapter = { title: "内容", paragraphs: [] };
      for (var i = 0; i < lines.length; i++) {
        var line = lines[i];
        var match = line.match(chapter_title_pattern);
        if (match) {
          var chapter_num = match[1];
          var chapter_title = match[2].trim();
          var full_title = "第" + chapter_num + "章 " + chapter_title;
          if (cur_chapter.paragraphs.length > 0) {
            chapters.push(cur_chapter);
          }
          cur_chapter = { title: full_title, paragraphs: [] };
        } else {
          cur_chapter.paragraphs.push(methods.formatContent(line));
        }
      }

      if (cur_chapter.paragraphs.length > 0) {
        chapters.push(cur_chapter);
      }

      if (chapters.length === 0) {
        chapters.push({ title: "内容", paragraphs: lines });
      }

      return chapters;
    },
    formatContent(text) {
      var paragraphs = text.split("\n\n").filter(function (p) {
        return p.trim();
      });
      return (
        paragraphs
          // .map(function (p) {
          //   return p.replace(/\n/g, "<br>");
          // })
          .join("")
      );
    },
  };
  return {
    parseBookContent: methods.parseBookContent,
  };
}

function ApplicationView() {
  const book_title_ = ref("请打开文件");
  const chapters_ = refarr([]);
  const font_size_ = ref(17);
  const auto_scroll_ = ref(false);
  const scroll_speed_ = ref(2);
  const font_size_label_ = ref("17px");
  const drag_over_ = ref(false);

  var scroll_raf = null;
  var scroll_last_time = 0;
  var scroll_accumulator = 0;
  var content_el = null;
  var current_path_ = null;
  var scroll_save_timer_ = null;

  const txt$ = TxtFileLoader();

  // Find which chapter is at the top of the viewport and the character
  // offset within that chapter. Returns { chapter, offset }.
  function getCurrentPosition() {
    if (!content_el) return { chapter: 0, offset: 0 };
    var blocks = content_el.querySelectorAll(".chapter-block");
    if (blocks.length === 0) return { chapter: 0, offset: 0 };
    var scrollTop = content_el.scrollTop;
    for (var i = 0; i < blocks.length; i++) {
      var block = blocks[i];
      var blockBottom = block.offsetTop + block.offsetHeight;
      if (blockBottom > scrollTop) {
        var relativeScroll = Math.max(0, scrollTop - block.offsetTop);
        var blockHeight = block.offsetHeight;
        var ratio = blockHeight > 0 ? relativeScroll / blockHeight : 0;
        var textLen = block.textContent ? block.textContent.length : 0;
        var offset = Math.floor(ratio * textLen);
        return { chapter: i, offset: offset };
      }
    }
    // Past the end — return last chapter at its end
    var lastIdx = blocks.length - 1;
    var lastBlock = blocks[lastIdx];
    var lastTextLen = lastBlock.textContent ? lastBlock.textContent.length : 0;
    return { chapter: lastIdx, offset: lastTextLen };
  }

  function saveScrollProgress() {
    if (!current_path_ || !content_el) {
      console.log("[progress] save skipped — no path or content_el", {
        current_path_,
        content_el,
      });
      return;
    }
    var pos = getCurrentPosition();
    console.log(
      "[progress] saving — ch:",
      pos.chapter,
      "off:",
      pos.offset,
      "path:",
      current_path_,
    );
    invoke("/api/book/progress", {
      method: "POST",
      args: {
        path: current_path_,
        current_chapter: pos.chapter,
        current_offset: pos.offset,
      },
    }).then(function (r) {
      console.log("[progress] save response:", r);
    });
  }

  // Retry via requestAnimationFrame until the target chapter block is
  // rendered, then scroll to the character-offset position within it.
  function restoreReadingPosition(chapter, offset) {
    console.log("[progress] restore requested — ch:", chapter, "off:", offset);
    var attempts = 0;
    function tryRestore() {
      if (!content_el) {
        console.log(
          "[progress] restore waiting — no content_el, attempt",
          attempts,
        );
        attempts++;
        if (attempts < 50) requestAnimationFrame(tryRestore);
        return;
      }
      var blocks = content_el.querySelectorAll(".chapter-block");
      console.log(
        "[progress] restore attempt",
        attempts,
        "— blocks:",
        blocks.length,
        "need ch:",
        chapter,
      );
      if (blocks.length > chapter) {
        var block = blocks[chapter];
        var textLen = block.textContent ? block.textContent.length : 1;
        var ratio = Math.min(1, offset / Math.max(1, textLen));
        var targetTop = block.offsetTop + ratio * block.offsetHeight;
        console.log(
          "[progress] restore done — scrollTop:",
          targetTop,
          "ratio:",
          ratio,
          "textLen:",
          textLen,
          "blockHeight:",
          block.offsetHeight,
        );
        content_el.scrollTop = targetTop;
        return;
      }
      attempts++;
      if (attempts < 50) {
        requestAnimationFrame(tryRestore);
      } else {
        console.log("[progress] restore FAILED — gave up after 50 attempts");
      }
    }
    requestAnimationFrame(tryRestore);
  }

  function debounceScrollSave() {
    if (scroll_save_timer_) {
      clearTimeout(scroll_save_timer_);
    }
    console.log("[progress] scroll detected — scheduling save in 2s");
    scroll_save_timer_ = setTimeout(saveScrollProgress, 2000);
  }

  function loadBook(content, title, path, file_size) {
    var chapters = txt$.parseBookContent(content);
    var word_count = content.length;

    book_title_.as(title);
    chapters_.as(chapters);
    current_path_ = path || null;

    if (path) {
      invoke("/api/book/load", {
        method: "POST",
        args: {
          name: title,
          path: path,
          word_count: word_count,
          file_size: file_size || 0,
        },
      }).then(function (res) {
        console.log(
          "[progress] loadBook /api/book/load response:",
          JSON.stringify(res),
        );
        if (
          res &&
          res.code === 0 &&
          res.data &&
          res.data.novel &&
          (res.data.novel.current_chapter > 0 ||
            res.data.novel.current_offset > 0)
        ) {
          restoreReadingPosition(
            res.data.novel.current_chapter,
            res.data.novel.current_offset,
          );
        } else {
          console.log("[progress] loadBook — no saved progress to restore");
        }
      });
    }
  }

  function openFile() {
    invoke("/api/file/open", { method: "GET" }).then(function (res) {
      if (res && res.code === 0 && res.data) {
        loadBook(
          res.data.content,
          res.data.title,
          res.data.path,
          res.data.file_size,
        );
      }
    });
  }

  function restoreLastBook() {
    invoke("/api/book/current", { method: "GET" }).then(function (res) {
      console.log(
        "[progress] restoreLastBook /api/book/current response:",
        JSON.stringify(res),
      );
      if (res && res.code === 0 && res.data && res.data.found) {
        var novel = res.data.novel;
        console.log(
          "[progress] restoreLastBook novel from DB:",
          JSON.stringify(novel),
        );
        invoke("/api/file/read?path=" + encodeURIComponent(novel.path), {
          method: "GET",
        }).then(function (fileRes) {
          if (fileRes && fileRes.code === 0 && fileRes.data) {
            var chapters = txt$.parseBookContent(fileRes.data.content);
            book_title_.as(fileRes.data.title);
            chapters_.as(chapters);
            current_path_ = novel.path;

            // Restore reading position after DOM render
            if (novel.current_chapter > 0 || novel.current_offset > 0) {
              restoreReadingPosition(
                novel.current_chapter,
                novel.current_offset,
              );
            }
          }
        });
      }
    });
  }

  // Restore last book on startup
  restoreLastBook();

  // Save reading progress immediately on window close / page unload
  window.addEventListener("beforeunload", function () {
    saveScrollProgress();
  });

  function applyFontSize(size) {
    font_size_.as(size);
    font_size_label_.as(size + "px");
    if (content_el) {
      content_el.style.fontSize = size + "px";
    }
  }

  function changeFontSize(delta) {
    var current = font_size_.value;
    var next = Math.min(32, Math.max(12, current + delta));
    applyFontSize(next);
    invoke("/api/storage/set?key=fontSize&value=" + next, { method: "GET" });
  }

  // Restore fontSize from config
  invoke("/api/storage/get?key=fontSize", { method: "GET" }).then(
    function (res) {
      if (res && res.code === 0 && res.data && res.data.found) {
        var size = Number(res.data.value);
        if (size >= 12 && size <= 32) {
          applyFontSize(size);
        }
      }
    },
  );

  // Restore scrollSpeed from config
  invoke("/api/storage/get?key=scrollSpeed", { method: "GET" }).then(
    function (res) {
      if (res && res.code === 0 && res.data && res.data.found) {
        var speed = Number(res.data.value);
        if (speed >= 1 && speed <= 10) {
          scroll_speed_.as(speed);
        }
      }
    },
  );

  function scrollFrame(timestamp) {
    if (!content_el) return;
    if (scroll_last_time) {
      var dt = timestamp - scroll_last_time;
      // Accumulate sub-pixel scroll to avoid truncation to 0
      scroll_accumulator += (scroll_speed_.value * 20 * dt) / 1000;
      var wholePx = Math.trunc(scroll_accumulator);
      if (wholePx !== 0) {
        scroll_accumulator -= wholePx;
        content_el.scrollTop += wholePx;
        if (
          content_el.scrollTop + content_el.clientHeight >=
          content_el.scrollHeight
        ) {
          toggleAutoScroll();
          return;
        }
      }
    }
    scroll_last_time = timestamp;
    scroll_raf = requestAnimationFrame(scrollFrame);
  }

  function startAutoScroll() {
    stopAutoScroll();
    if (!content_el) return;
    scroll_last_time = 0;
    scroll_accumulator = 0;
    scroll_raf = requestAnimationFrame(scrollFrame);
  }

  function stopAutoScroll() {
    if (scroll_raf) {
      cancelAnimationFrame(scroll_raf);
      scroll_raf = null;
      scroll_last_time = 0;
    }
  }

  function toggleAutoScroll() {
    var next = !auto_scroll_.value;
    auto_scroll_.as(next);
    if (next) {
      startAutoScroll();
    } else {
      stopAutoScroll();
    }
  }

  function changeScrollSpeed(delta) {
    var current = scroll_speed_.value;
    var next = Math.min(10, Math.max(1, current + delta));
    scroll_speed_.as(next);
    invoke("/api/storage/set?key=scrollSpeed&value=" + next, { method: "GET" });
  }

  function handleGoMessage(payload) {
    if (!payload || !payload.type) return;
    if (payload.type === "startAutoScroll") {
      if (!auto_scroll_.value) {
        toggleAutoScroll();
      }
    } else if (payload.type === "stopAutoScroll") {
      if (auto_scroll_.value) {
        toggleAutoScroll();
      }
    } else if (payload.type === "fileDrop") {
      loadBook(payload.content, payload.title, payload.path, payload.file_size);
    } else if (payload.type === "__velo_drag_enter") {
      drag_over_.as(true);
    } else if (payload.type === "__velo_drag_leave") {
      drag_over_.as(false);
    }
  }

  if (window.onGoMessage) {
    window.onGoMessage(handleGoMessage);
  }

  return View(
    {
      class: "reader-device velo-drag",
    },
    [
      View(
        {
          class: "drop-overlay",
          onMounted(event) {
            var el = event.target.get$elm();
            drag_over_.subscribe({
              onChange(v) {
                if (v) {
                  el.classList.add("active");
                } else {
                  el.classList.remove("active");
                }
              },
            });
          },
        },
        [
          View({ class: "drop-overlay-border" }),
          View({ class: "drop-overlay-icon" }, ["\u2193"]),
          View({ class: "drop-overlay-text" }, [
            "\u677E\u5F00\u4EE5\u52A0\u8F7D\u6587\u672C",
          ]),
          View({ class: "drop-overlay-hint" }, [
            "\u652F\u6301 .txt \u683C\u5F0F\u6587\u4EF6",
          ]),
        ],
      ),
      View(
        {
          class: "screen-container",
        },
        [
          View(
            {
              class: "header",
            },
            [
              View(
                {
                  class: "book-title",
                  id: "book-title",
                },
                [book_title_],
              ),
              View(
                {
                  style: {
                    display: "flex",
                    "align-items": "center",
                    gap: "12px",
                  },
                },
                [
                  Button(
                    {
                      class: "menu-btn",
                      onClick() {
                        openFile();
                      },
                    },
                    ["打开文件"],
                  ),
                  View(
                    {
                      class: "battery",
                    },
                    [
                      View(
                        {
                          class: "battery-icon",
                        },
                        [
                          View({
                            class: "battery-level",
                          }),
                        ],
                      ),
                    ],
                  ),
                ],
              ),
            ],
          ),
          View(
            {
              id: "content-area",
              class: "content-area",
              onMounted(event) {
                content_el = event.target.get$elm();
                // pause auto-scroll on manual interaction
                if (content_el) {
                  content_el.addEventListener("wheel", function () {
                    if (auto_scroll_.value) {
                      toggleAutoScroll();
                    }
                  });
                  content_el.addEventListener("touchstart", function () {
                    if (auto_scroll_.value) {
                      toggleAutoScroll();
                    }
                  });
                  // Save scroll progress on scroll (debounced)
                  content_el.addEventListener("scroll", debounceScrollSave);
                }
              },
            },
            [
              For({
                each: chapters_,
                render(chapter) {
                  return View(
                    {
                      class: "chapter-block",
                    },
                    [
                      Show({
                        when: chapter.title,
                        ok() {
                          return View(
                            {
                              class: "chapter-title",
                            },
                            [chapter.title],
                          );
                        },
                      }),
                      View(
                        {
                          class: "chapter-content",
                        },
                        [
                          For({
                            each: chapter.paragraphs,
                            render(paragraph) {
                              return View({ class: "paragraph velo-no-drag" }, [
                                paragraph,
                              ]);
                            },
                          }),
                        ],
                      ),
                    ],
                  );
                },
              }),
            ],
          ),
          View(
            {
              class: "toolbar velo-no-drag",
            },
            [
              View(
                {
                  class: "toolbar-group",
                },
                [
                  View({ class: "toolbar-label" }, ["字号"]),
                  Button(
                    {
                      class: "toolbar-btn",
                      onClick() {
                        changeFontSize(-1);
                      },
                    },
                    ["A-"],
                  ),
                  View({ class: "toolbar-value" }, [font_size_label_]),
                  Button(
                    {
                      class: "toolbar-btn",
                      onClick() {
                        changeFontSize(1);
                      },
                    },
                    ["A+"],
                  ),
                ],
              ),
              View({ class: "toolbar-divider" }),
              View(
                {
                  class: "toolbar-group",
                },
                [
                  View({ class: "toolbar-label" }, ["自动滚动"]),
                  Button(
                    {
                      class: "toolbar-btn",
                      onClick() {
                        changeScrollSpeed(-1);
                      },
                    },
                    ["-"],
                  ),
                  View({ class: "toolbar-value" }, [scroll_speed_]),
                  Button(
                    {
                      class: "toolbar-btn",
                      onClick() {
                        changeScrollSpeed(1);
                      },
                    },
                    ["+"],
                  ),
                  Button(
                    {
                      class: "toolbar-btn toolbar-btn-toggle",
                      onClick() {
                        toggleAutoScroll();
                      },
                    },
                    [
                      Show({
                        when: auto_scroll_,
                        ok() {
                          return "⏸";
                        },
                        else() {
                          return "▶";
                        },
                      }),
                    ],
                  ),
                ],
              ),
            ],
          ),
        ],
      ),
    ],
  );
}

document.addEventListener("DOMContentLoaded", function () {
  var $root = document.getElementById("root");
  if (!$root) {
    console.error("[Render] Root element not found");
    return;
  }
  Timeless.DOM.render(ApplicationView(), $root);
  // Show window after fonts are loaded so user doesn't see blank content
  document.fonts.ready.then(function () {
    invoke("/api/window/show", { method: "GET" });
  });

  // Periodically snapshot window position & size to data.json
  var _snapshotTimer = setInterval(function () {
    invoke("/api/window/state/snapshot?name=reader", { method: "GET" });
  }, 3000);

  // Save on window close / page unload
  window.addEventListener("beforeunload", function () {
    clearInterval(_snapshotTimer);
    // Use synchronous XHR to ensure the save completes before the page is torn down
    var xhr = new XMLHttpRequest();
    xhr.open("GET", "/api/window/state/snapshot?name=reader", false);
    xhr.send();
  });
});
