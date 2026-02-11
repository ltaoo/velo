#ifndef TRAY_WINDOWS_H
#define TRAY_WINDOWS_H

#ifdef __cplusplus
extern "C" {
#endif

void init_tray_win();
void set_icon_win(const char* data, int length);
void set_tooltip_win(const char* tooltip);
void add_menu_item_win(int id, const char* title, const char* shortcut, int disabled, int checked, int parentId, int isSubmenu);
void add_separator_win(int parentId);
void run_loop_win();
void quit_app_win();

void set_item_label_win(int id, const char* label);
void set_item_tooltip_win(int id, const char* tooltip);
void set_item_checked_win(int id, int checked);
void set_item_disabled_win(int id, int disabled);

#ifdef __cplusplus
}
#endif

#endif
