declare namespace Dayjs {
  interface Dayjs {
    format(template?: string): string;
    add(value: number, unit: string): Dayjs;
    subtract(value: number, unit: string): Dayjs;
    isValid(): boolean;
    // 添加其他你需要的方法
  }

  function dayjs(date?: string | number | Date): Dayjs;
  function extend(plugin: any): void;
}

declare const dayjs: typeof Dayjs.dayjs;

declare function invoke(
  url: string,
  options: {
    method: string;
    headers?: Record<string, unknown[]>;
    args?: Record<string, unknown>;
  },
): Promise<any>;

declare interface Window {
  dayjs: typeof dayjs;
}
// Global Core Functions
declare const ref: typeof import("../src/components/ui/core").ref;
declare const computed: typeof import("../src/components/ui/core").computed;
declare const isRef: typeof import("../src/components/ui/core").isRef;
declare const classnames: typeof import("../src/components/ui/core").classnames;

declare const Show: typeof import("../src/components/ui/show").Show;
declare const For: typeof import("../src/components/ui/for").For;
declare const Match: typeof import("../src/components/ui/match").Match;
declare const Switch: typeof import("../src/components/ui/toggle").Toggle;
declare const Slider: typeof import("../src/components/ui/slider").Slider;
declare const Slide: typeof import("../src/components/ui/slider").Slider;
declare const Progress: typeof import("../src/components/ui/progress").Progress;
// Global Components
declare const View: typeof import("../src/components/ui/view").View;
declare const DangerouslyInnerHTML: typeof import("../src/components/ui/html").DangerouslyInnerHTML;
declare const Txt: typeof import("../src/components/ui/text").Txt;
declare const ScrollView: typeof import("../src/components/ui/scrollview").ScrollView;
declare const Flex: typeof import("../src/components/ui/flex").Flex;
declare const Button: typeof import("../src/components/ui/button").Button;
declare const Input: typeof import("../src/components/ui/input").Input;
declare const Checkbox: typeof import("../src/components/ui/checkbox").Checkbox;
declare const Select: typeof import("../src/components/ui/select").Select;
declare const Presence: typeof import("../src/components/ui/presence").Presence;
declare const Portal: typeof import("../src/components/ui/portal").Portal;
declare const Popper: typeof import("../src/components/ui/popper").Popper;
declare const Toggle: typeof import("../src/components/ui/toggle").Toggle;
declare const Switch: typeof import("../src/components/ui/toggle").Toggle;

declare const Menu: typeof import("../src/components/ui/menu").Menu;
declare const MenuItem: typeof import("../src/components/ui/menu").MenuItem;
declare const MenuLabel: typeof import("../src/components/ui/menu").MenuLabel;
declare const MenuSeparator: typeof import("../src/components/ui/menu").MenuSeparator;
declare const DropdownMenu: typeof import("../src/components/ui/menu").DropdownMenu;

declare const Tabs: typeof import("../src/components/ui/tabs").Tabs;
declare const Steps: typeof import("../src/components/ui/steps").Steps;

declare var TimelessWeb: {
  provide_http_client: (vm: any) => void;
  provide_ui_scroll_view_scroll: (vm: any, elm: HTMLDivElement) => void;
  provide_ui_scroll_view_indicator: (vm: any, elm: HTMLDivElement) => void;
};
