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

  const txt$ = TxtFileLoader();

  function loadBook(content, title) {
    var chapters = txt$.parseBookContent(content);

    book_title_.as(title);
    chapters_.as(chapters);
  }

  function openFile() {
    invoke("/api/file/open", { method: "GET" }).then(function (res) {
      if (res && res.code === 0 && res.data) {
        loadBook(res.data.content, res.data.title);
      }
    });
  }

  return View(
    {
      class: "reader-device velo-drag",
    },
    [
      View(
        {
          class: "screen-container",
          onMounted(el) {
            // var lastScrollTop = 0;
            // var header = el.querySelector(".header");
            // el.addEventListener("scroll", function () {
            //   var st = el.scrollTop;
            //   if (st > lastScrollTop && st > 60) {
            //     header.classList.add("header-hidden");
            //   }
            //   lastScrollTop = st;
            // });
            // el.querySelector(".content-area").addEventListener("click", function () {
            //   header.classList.toggle("header-hidden");
            // });
          },
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
              class: "content-area",
              id: "content-area",
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
                              return View({ class: "paragraph velo-no-drag" }, [paragraph]);
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
});
