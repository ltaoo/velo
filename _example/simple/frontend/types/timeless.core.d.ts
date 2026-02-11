declare namespace Timeless {

declare class BizError extends Error {
    messages: string[];
    code?: string | number;
    data: unknown | null;
    constructor(msg: string[], code?: string | number, data?: unknown);
}

type Resp<T> = {
    data: T extends null ? null : T;
    error: T extends null ? BizError : null;
};
type UnpackedResult<T> = NonNullable<T extends Resp<infer U> ? (U extends null ? U : U) : T>;
type Result<T> = Resp<T> | Resp<null>;
/** 构造一个结果对象 */
declare const Result: {
    /** 构造成功结果 */
    Ok: <T>(value: T) => Result<T>;
    /** 构造失败结果 */
    Err: <T>(message: string | string[] | BizError | Error | Result<null>, code?: string | number, data?: unknown) => Resp<null>;
};

declare type EventType = string | symbol;
declare type Handler<T = unknown> = (event: T) => void;
declare type WildcardHandler<T = Record<string, unknown>> = (type: keyof T, event: T[keyof T]) => void;
declare type EventHandlerList<T = unknown> = Array<Handler<T>>;
declare type WildCardEventHandlerList<T = Record<string, unknown>> = Array<WildcardHandler<T>>;
declare type EventHandlerMap<Events extends Record<EventType, unknown>> = Map<keyof Events | '*', EventHandlerList<Events[keyof Events]> | WildCardEventHandlerList<Events>>;
interface Emitter<Events extends Record<EventType, unknown>> {
    all: EventHandlerMap<Events>;
    on<Key extends keyof Events>(type: Key, handler: Handler<Events[Key]>): void;
    on(type: '*', handler: WildcardHandler<Events>): void;
    off<Key extends keyof Events>(type: Key, handler?: Handler<Events[Key]>): void;
    off(type: '*', handler: WildcardHandler<Events>): void;
    emit<Key extends keyof Events>(type: Key, event: Events[Key]): void;
    emit<Key extends keyof Events>(type: undefined extends Events[Key] ? Key : never): void;
}

declare enum BaseEvents {
    Loading = "__loading",
    Destroy = "__destroy"
}
type TheTypesOfBaseEvents = {
    [BaseEvents.Destroy]: void;
};
type BaseDomainEvents<E> = TheTypesOfBaseEvents & E;
declare function base<Events extends Record<EventType, unknown>>(): {
    off<Key extends keyof BaseDomainEvents<Events>>(event: Key, handler: Handler<BaseDomainEvents<Events>[Key]>): void;
    on<Key extends keyof BaseDomainEvents<Events>>(event: Key, handler: Handler<BaseDomainEvents<Events>[Key]>): () => void;
    uid: () => number;
    emit<Key extends keyof BaseDomainEvents<Events>>(event: Key, value?: BaseDomainEvents<Events>[Key]): void;
    destroy(): void;
};
declare class BaseDomain<Events extends Record<EventType, unknown>> {
    /** 用于自己区别同名 Domain 不同实例的标志 */
    unique_id: string;
    debug: boolean;
    _emitter: Emitter<BaseDomainEvents<Events>>;
    listeners: Record<keyof BaseDomainEvents<Events>, (() => void)[]>;
    constructor(props?: {});
    uid(): number;
    log(...args: unknown[]): unknown[];
    errorTip(...args: unknown[]): void;
    off<Key extends keyof BaseDomainEvents<Events>>(event: Key, handler: Handler<BaseDomainEvents<Events>[Key]>): void;
    offEvent<Key extends keyof BaseDomainEvents<Events>>(k: Key): void;
    on<Key extends keyof BaseDomainEvents<Events>>(event: Key, handler: Handler<BaseDomainEvents<Events>[Key]>): () => void;
    emit<Key extends keyof BaseDomainEvents<Events>>(event: Key, value?: BaseDomainEvents<Events>[Key]): void;
    /** 主动销毁所有的监听事件 */
    destroy(): void;
    onDestroy(handler: Handler<TheTypesOfBaseEvents[BaseEvents.Destroy]>): () => void;
    get [Symbol.toStringTag](): string;
}
declare function applyMixins(derivedCtor: any, constructors: any[]): void;

declare class UserCore extends BaseDomain<any> {
    constructor();
}

declare enum Events$y {
    StateChange = 0
}
type TheTypesOfEvents$C<T> = {
    [Events$y.StateChange]: StorageCoreState<T>;
};
type StorageCoreProps<T> = {
    key: string;
    values: T;
    defaultValues: T;
    client: {
        setItem: (key: string, value: string) => void;
        getItem: (key: string) => void;
    };
};
type StorageCoreState<T> = {
    values: T;
};
declare class StorageCore<T extends Record<string, unknown>> extends BaseDomain<TheTypesOfEvents$C<T>> {
    key: string;
    values: T;
    defaultValues: T;
    client: StorageCoreProps<T>["client"];
    get state(): {
        values: T;
    };
    constructor(props: Partial<{
        _name: string;
    }> & StorageCoreProps<T>);
    get<K extends keyof T>(key: K, defaultValue?: T[K]): T[K];
    set: (key: keyof T, value: unknown) => void;
    merge: <K extends keyof T>(key: K, values: Partial<T[K]>, extra?: Partial<{
        reverse: boolean;
        limit: number;
    }>) => {};
    clear<K extends keyof T>(key: K): any;
    remove<K extends keyof T>(key: K): void;
    onStateChange(handler: Handler<TheTypesOfEvents$C<T>[Events$y.StateChange]>): () => void;
}

type Unpacked<T> = T extends (infer U)[] ? U : T extends (...args: any[]) => infer U ? U : T extends Promise<infer U> ? U : T;
type MutableRecord2<U> = {
    [SubType in keyof U]: {
        type: SubType;
    } & U[SubType];
}[keyof U];
interface JSONArray extends Array<JSONValue> {
}
type JSONValue = string | number | boolean | JSONObject | JSONArray | null;
type JSONObject = {
    [Key in string]?: JSONValue;
};

type ThemeTypes = "dark" | "light" | "system";

declare enum OrientationTypes {
    Horizontal = "horizontal",
    Vertical = "vertical"
}
declare const mediaSizes: {
    sm: number;
    /** 中等设备宽度阈值 */
    md: number;
    /** 大设备宽度阈值 */
    lg: number;
    /** 特大设备宽度阈值 */
    xl: number;
    /** 特大设备宽度阈值 */
    "2xl": number;
};
type DeviceSizeTypes = keyof typeof mediaSizes;
declare enum Events$x {
    Tip = 0,
    Loading = 1,
    HideLoading = 2,
    Error = 3,
    Login = 4,
    Logout = 5,
    ForceUpdate = 6,
    DeviceSizeChange = 7,
    /** 生命周期 */
    Ready = 8,
    Show = 9,
    Hidden = 10,
    /** 平台相关 */
    Resize = 11,
    Blur = 12,
    Keydown = 13,
    OrientationChange = 14,
    EscapeKeyDown = 15,
    StateChange = 16
}
type TheTypesOfEvents$B = {
    [Events$x.Ready]: void;
    [Events$x.Error]: Error;
    [Events$x.Tip]: {
        icon?: unknown;
        text: string[];
    };
    [Events$x.Loading]: {
        text: string[];
    };
    [Events$x.HideLoading]: void;
    [Events$x.Login]: {};
    [Events$x.Logout]: void;
    [Events$x.ForceUpdate]: void;
    [Events$x.Resize]: {
        width: number;
        height: number;
    };
    [Events$x.DeviceSizeChange]: DeviceSizeTypes;
    [Events$x.Keydown]: {
        code: string;
        preventDefault: () => void;
    };
    [Events$x.EscapeKeyDown]: void;
    [Events$x.Blur]: void;
    [Events$x.Show]: void;
    [Events$x.Hidden]: void;
    [Events$x.OrientationChange]: "vertical" | "horizontal";
    [Events$x.StateChange]: ApplicationState;
};
type ApplicationState = {
    ready: boolean;
    env: JSONObject;
    theme: ThemeTypes;
    deviceSize: DeviceSizeTypes;
    height: number;
};
type ApplicationProps<T extends {
    storage: StorageCore<any>;
}> = {
    user: UserCore;
    storage: T["storage"];
    /**
     * 应用加载前的声明周期，只有返回 Result.Ok() 页面才会展示内容
     */
    beforeReady?: () => Promise<Result<null>>;
    onReady?: () => void;
};
declare class ApplicationModel<T extends {
    storage: StorageCore<any>;
}> extends BaseDomain<TheTypesOfEvents$B> {
    /** 用户 */
    $user: UserCore;
    $storage: T["storage"];
    lifetimes: Pick<ApplicationProps<T>, "beforeReady" | "onReady">;
    ready: boolean;
    screen: {
        statusBarHeight?: number;
        menuButton?: {
            width: number;
            left: number;
            right: number;
        };
        width: number;
        height: number;
    };
    env: {
        wechat: boolean;
        ios: boolean;
        android: boolean;
        pc: boolean;
        weapp: boolean;
        prod: "develop" | "trial" | "release";
    };
    orientation: OrientationTypes;
    curDeviceSize: DeviceSizeTypes;
    height: number;
    theme: ThemeTypes;
    safeArea: boolean;
    Events: typeof Events$x;
    get state(): ApplicationState;
    constructor(props: ApplicationProps<T>);
    /** 启动应用 */
    start(size: {
        width: number;
        height: number;
    }): Promise<Result<any>>;
    /** 应用指定主题 */
    setTheme(theme: ThemeTypes): Resp<null>;
    getTheme(): string;
    tipUpdate(): void;
    tip(arg: {
        icon?: unknown;
        text: string[];
    }): string;
    loading(arg: {
        text: string[];
    }): {
        hideLoading: () => void;
    };
    hideLoading(): void;
    /** 手机震动 */
    vibrate(): void;
    setSize(size: {
        width: number;
        height: number;
    }): void;
    /** 设置页面 title */
    setTitle(title: string): void;
    openWindow(url: string): void;
    setEnv(env: JSONObject): void;
    setHeight(v: number): void;
    /** 复制文本到粘贴板 */
    copy(text: string): void;
    getComputedStyle(el: unknown): {};
    /** 发送推送 */
    notify(msg: {
        title: string;
        body: string;
    }): void;
    disablePointer(): void;
    enablePointer(): void;
    /** 平台相关的全局事件 */
    keydown(event: {
        code: string;
        preventDefault: () => void;
    }): void;
    escape(): void;
    resize(size: {
        width: number;
        height: number;
    }): void;
    blur(): void;
    handleScreenOrientationChange(orientation: number): void;
    handleResize(size: {
        width: number;
        height: number;
    }): void;
    onReady(handler: Handler<TheTypesOfEvents$B[Events$x.Ready]>): () => void;
    onDeviceSizeChange(handler: Handler<TheTypesOfEvents$B[Events$x.DeviceSizeChange]>): () => void;
    onUpdate(handler: Handler<TheTypesOfEvents$B[Events$x.ForceUpdate]>): () => void;
    /** 平台相关全局事件 */
    onOrientationChange(handler: Handler<TheTypesOfEvents$B[Events$x.OrientationChange]>): () => void;
    onResize(handler: Handler<TheTypesOfEvents$B[Events$x.Resize]>): () => void;
    onBlur(handler: Handler<TheTypesOfEvents$B[Events$x.Blur]>): () => void;
    onShow(handler: Handler<TheTypesOfEvents$B[Events$x.Show]>): () => void;
    onHidden(handler: Handler<TheTypesOfEvents$B[Events$x.Hidden]>): () => void;
    onKeydown(handler: Handler<TheTypesOfEvents$B[Events$x.Keydown]>): () => void;
    onEscapeKeyDown(handler: Handler<TheTypesOfEvents$B[Events$x.EscapeKeyDown]>): () => void;
    onTip(handler: Handler<TheTypesOfEvents$B[Events$x.Tip]>): () => void;
    onLoading(handler: Handler<TheTypesOfEvents$B[Events$x.Loading]>): () => void;
    onHideLoading(handler: Handler<TheTypesOfEvents$B[Events$x.HideLoading]>): () => void;
    onStateChange(handler: Handler<TheTypesOfEvents$B[Events$x.StateChange]>): () => void;
    /**
     * ----------------
     * Event
     * ----------------
     */
    onError(handler: Handler<TheTypesOfEvents$B[Events$x.Error]>): () => void;
}

/**
 * @file 支持动画的 Popup
 */

declare enum Events$w {
    StateChange = 0,
    PresentChange = 1,
    Show = 2,
    TmpShow = 3,
    Hidden = 4,
    TmpHidden = 5,
    Unmounted = 6
}
type TheTypesOfEvents$A = {
    [Events$w.StateChange]: PresenceState;
    [Events$w.PresentChange]: boolean;
    [Events$w.Show]: void;
    [Events$w.TmpShow]: void;
    [Events$w.Hidden]: void;
    [Events$w.TmpHidden]: void;
    [Events$w.Unmounted]: void;
};
type PresenceState = {
    mounted: boolean;
    enter: boolean;
    visible: boolean;
    exit: boolean;
    text: string;
};
type PresenceProps = {
    mounted?: boolean;
    visible?: boolean;
};
declare class PresenceCore extends BaseDomain<TheTypesOfEvents$A> {
    name: string;
    debug: boolean;
    animationName: string;
    mounted: boolean;
    enter: boolean;
    visible: boolean;
    exit: boolean;
    get state(): PresenceState;
    constructor(props?: Partial<{
        _name: string;
    }> & PresenceProps);
    toggle(): void;
    show(): void;
    hide(options?: Partial<{
        reason: "show_sibling" | "back" | "forward";
        destroy: boolean;
    }>): void;
    /** 将 DOM 从页面卸载 */
    unmount(): void;
    reset(): void;
    onTmpShow(handler: Handler<TheTypesOfEvents$A[Events$w.TmpShow]>): () => void;
    onTmpHidden(handler: Handler<TheTypesOfEvents$A[Events$w.TmpHidden]>): () => void;
    onShow(handler: Handler<TheTypesOfEvents$A[Events$w.Show]>): () => void;
    onHidden(handler: Handler<TheTypesOfEvents$A[Events$w.Hidden]>): () => void;
    onUnmounted(handler: Handler<TheTypesOfEvents$A[Events$w.Unmounted]>): () => void;
    onStateChange(handler: Handler<TheTypesOfEvents$A[Events$w.StateChange]>): () => void;
    onPresentChange(handler: Handler<TheTypesOfEvents$A[Events$w.PresentChange]>): () => void;
    get [Symbol.toStringTag](): string;
}

/**
 * 将对象转成 search 字符串，前面不带 ?
 * @param query
 * @returns
 */
declare function query_stringify(query?: null | JSONObject): string;

declare function buildUrl(key: string, params?: JSONObject, query?: Parameters<typeof query_stringify>[0]): string;
type OriginalRouteConfigure = Record<PathnameKey, {
    title: string;
    pathname: string;
    options?: Partial<{
        keep_alive?: boolean;
        animation?: Partial<{
            in: string;
            out: string;
            show: string;
            hide: string;
        }>;
        require?: string[];
    }>;
    children?: OriginalRouteConfigure;
}>;
type PageKeysType<T extends OriginalRouteConfigure, K = keyof T> = K extends keyof T & (string | number) ? `${K}` | (T[K] extends object ? T[K]["children"] extends object ? `${K}.${PageKeysType<T[K]["children"]>}` : never : never) : never;
type PathnameKey = string;
type RouteConfig<T> = {
    /** 使用该值定位唯一 route/page */
    name: T;
    title: string;
    pathname: PathnameKey;
    /** 是否为布局 */
    layout?: boolean;
    parent: {
        name: string;
    };
    options?: Partial<{
        require?: string[];
        keep_alive?: boolean;
        animation?: {
            in: string;
            out: string;
            show: string;
            hide: string;
        };
    }>;
};
declare function build<T>(configure: OriginalRouteConfigure): {
    routes: Record<string, RouteConfig<T>>;
    routesWithPathname: Record<string, RouteConfig<T>>;
};

/**
 * @file 根据路由判断是否可见的视图块
 */

declare enum Events$v {
    SubViewChanged = 0,
    SubViewRemoved = 1,
    SubViewAppended = 2,
    /** 子视图改变（数量 */
    SubViewsChange = 3,
    /** 当前展示的子视图改变 */
    CurSubViewChange = 4,
    /** 有视图变为可见状态 */
    ViewShow = 5,
    /** 视图加载好 */
    Ready = 6,
    /** 当前视图载入页面 */
    Mounted = 7,
    BeforeShow = 8,
    /** 当前视图变为可见，稍晚于 Mounted 事件 */
    Show = 9,
    BeforeHide = 10,
    /** 当前视图变为隐藏 */
    Hidden = 11,
    /** 当前视图从页面卸载 */
    Unmounted = 12,
    /** 被其他视图覆盖 */
    Layered = 13,
    /** 覆盖自身的视图被移开 */
    Uncover = 14,
    Start = 15,
    StateChange = 16,
    /** 子视图匹配上了 */
    Match = 17,
    NotFound = 18
}
type TheTypesOfEvents$z = {
    [Events$v.SubViewChanged]: RouteViewCore;
    [Events$v.SubViewRemoved]: RouteViewCore;
    [Events$v.SubViewAppended]: RouteViewCore;
    [Events$v.SubViewsChange]: RouteViewCore[];
    [Events$v.CurSubViewChange]: RouteViewCore;
    [Events$v.Ready]: void;
    [Events$v.Mounted]: void;
    [Events$v.ViewShow]: RouteViewCore[];
    [Events$v.BeforeShow]: void;
    [Events$v.Show]: void;
    [Events$v.BeforeHide]: void;
    [Events$v.Hidden]: void;
    [Events$v.Layered]: void;
    [Events$v.Uncover]: void;
    [Events$v.Unmounted]: void;
    [Events$v.Start]: {
        pathname: string;
    };
    [Events$v.StateChange]: RouteViewCoreState;
    [Events$v.Match]: RouteViewCore;
    [Events$v.NotFound]: void;
};
type RouteViewCoreState = {
    /** 是否加载到页面上（如果有动画，在隐藏动画播放时该值仍为 true，在 animation end 后从视图上卸载后，该值被置为 false） */
    mounted: boolean;
    /** 是否可见，用于判断是「进入动画」还是「退出动画」 */
    visible: boolean;
    /** 被另一视图覆盖 */
    layered: boolean;
};
type RouteViewCoreProps = {
    /** 唯一标志 */
    name: string;
    pathname: string;
    title: string;
    parent?: RouteViewCore | null;
    query?: Record<string, string>;
    visible?: boolean;
    /** 该视图是布局视图 */
    layout?: boolean;
    animation?: Partial<{
        in: string;
        out: string;
        show: string;
        hide: string;
    }>;
    children?: RouteViewCore[];
    views?: RouteViewCore[];
};
declare class RouteViewCore extends BaseDomain<TheTypesOfEvents$z> {
    unique_id: string;
    debug: boolean;
    id: number;
    /** 一些配置项 */
    name: string;
    pathname: string;
    title: string;
    animation: Partial<{
        in: string;
        out: string;
        show: string;
        hide: string;
    }>;
    /** 当前视图的 query */
    query: Record<string, string>;
    /** 当前视图的 params */
    params: Record<string, string>;
    _showed: boolean;
    loaded: boolean;
    mounted: boolean;
    layered: boolean;
    isRoot: boolean;
    parent: RouteViewCore | null;
    /** 当前子视图 */
    curView: RouteViewCore | null;
    /** 当前所有的子视图 */
    subViews: RouteViewCore[];
    $presence: PresenceCore;
    get state(): RouteViewCoreState;
    get href(): string;
    get visible(): boolean;
    constructor(options: Partial<{
        _name: string;
    }> & RouteViewCoreProps);
    appendView(view: RouteViewCore): void;
    replaceViews(views: RouteViewCore[]): void;
    /** 移除（卸载）一个子视图 */
    removeView(view: RouteViewCore, options?: Partial<{
        reason: "show_sibling" | "back" | "forward";
        destroy: boolean;
        callback: () => void;
    }>): void;
    findCurView(): RouteViewCore | null;
    ready(): void;
    /** 让自身的一个子视图变为可见 */
    showView(sub_view: RouteViewCore, options?: Partial<{
        reason: "show_sibling" | "back";
        destroy: boolean;
    }>): void;
    /** 主动展示视图 */
    show(): void;
    /** 主动隐藏自身视图 */
    hide(options?: Partial<{
        reason: "show_sibling" | "back" | "forward";
        destroy: boolean;
    }>): void;
    /** 视图在页面上展示（变为可见） */
    setShow(): void;
    /** 视图在页面上隐藏（变为不可见） */
    setHidden(): void;
    mount(): void;
    /** 卸载自身 */
    unmount(): void;
    /** 视图被装载到页面 */
    setMounted(): void;
    /** 视图从页面被卸载 */
    setUnmounted(): void;
    /** 页面组件已加载 */
    setLoaded(): void;
    /** 页面组件未加载 */
    setUnload(): void;
    buildUrl(query: Record<string, string | number>): string;
    buildUrlWithPrefix(query: Record<string, string | number>): string;
    onStart(handler: Handler<TheTypesOfEvents$z[Events$v.Start]>): () => void;
    onReady(handler: Handler<TheTypesOfEvents$z[Events$v.Ready]>): () => void;
    onMounted(handler: Handler<TheTypesOfEvents$z[Events$v.Mounted]>): () => void;
    onViewShow(handler: Handler<TheTypesOfEvents$z[Events$v.ViewShow]>): () => void;
    onBeforeShow(handler: Handler<TheTypesOfEvents$z[Events$v.BeforeShow]>): () => void;
    onShow(handler: Handler<TheTypesOfEvents$z[Events$v.Show]>): () => void;
    onBeforeHide(handler: Handler<TheTypesOfEvents$z[Events$v.BeforeHide]>): () => void;
    onHidden(handler: Handler<TheTypesOfEvents$z[Events$v.Hidden]>): () => void;
    onLayered(handler: Handler<TheTypesOfEvents$z[Events$v.Layered]>): () => void;
    onUncover(handler: Handler<TheTypesOfEvents$z[Events$v.Uncover]>): () => void;
    onUnmounted(handler: Handler<TheTypesOfEvents$z[Events$v.Unmounted]>): () => void;
    onSubViewChanged(handler: Handler<TheTypesOfEvents$z[Events$v.SubViewChanged]>): () => void;
    onSubViewAppended(handler: Handler<TheTypesOfEvents$z[Events$v.SubViewAppended]>): () => void;
    onSubViewRemoved(handler: Handler<TheTypesOfEvents$z[Events$v.SubViewRemoved]>): () => void;
    onSubViewsChange(handler: Handler<TheTypesOfEvents$z[Events$v.SubViewsChange]>): () => void;
    onCurViewChange(handler: Handler<TheTypesOfEvents$z[Events$v.CurSubViewChange]>): () => void;
    onMatched(handler: Handler<TheTypesOfEvents$z[Events$v.Match]>): () => void;
    onNotFound(handler: Handler<TheTypesOfEvents$z[Events$v.NotFound]>): () => void;
    onStateChange(handler: Handler<TheTypesOfEvents$z[Events$v.StateChange]>): () => void;
}
declare function onViewCreated(fn: (views: RouteViewCore) => void): void;
declare function RouteMenusModel<T extends {
    title: string;
    url?: unknown;
    onClick?: (m: T) => void;
}>(props: {
    route: T["url"];
    menus: T[];
    $history: HistoryCore<any, any>;
}): {
    methods: {
        refresh(): void;
        setCurMenu(name: T["url"]): void;
    };
    ui: {};
    state: {
        readonly menus: T[];
        readonly route_name: T["url"];
    };
    ready(): void;
    destroy(): void;
    onStateChange(handler: Handler<{
        readonly menus: T[];
        readonly route_name: T["url"];
    }>): () => void;
    onError(handler: Handler<BizError>): () => void;
};

declare namespace URLParse {
    type URLPart =
        | "auth"
        | "hash"
        | "host"
        | "hostname"
        | "href"
        | "origin"
        | "password"
        | "pathname"
        | "port"
        | "protocol"
        | "query"
        | "slashes"
        | "username";

    type QueryParser<T = Record<string, string | undefined>> = (query: string) => T;

    type StringifyQuery = (query: object) => string;
}

interface URLParse<Query> {
    readonly auth: string;
    readonly hash: string;
    readonly host: string;
    readonly hostname: string;
    readonly href: string;
    readonly origin: string;
    readonly password: string;
    readonly pathname: string;
    readonly port: string;
    readonly protocol: string;
    readonly query: Query;
    readonly slashes: boolean;
    readonly username: string;
    set<Part extends URLParse.URLPart>(
        part: Part,
        value: URLParse<Query>[Part] | undefined,
        fn?: false,
    ): URLParse<Query>;
    set<Part extends URLParse.URLPart, T>(
        part: Part,
        value: URLParse<T>[Part] | undefined,
        fn?: URLParse.QueryParser<T>,
    ): URLParse<T>;
    toString(stringify?: URLParse.StringifyQuery): string;
}

declare const URLParse: {
    new(address: string, parser?: false): URLParse<string>;
    new(address: string, parser: true): URLParse<Record<string, string | undefined>>;
    new<T>(address: string, parser?: URLParse.QueryParser<T>): URLParse<T>;
    new(address: string, location?: string | object, parser?: false): URLParse<string>;
    new(
        address: string,
        location: string | object | undefined,
        parser: true,
    ): URLParse<Record<string, string | undefined>>;
    new<T>(address: string, location: string | object | undefined, parser: URLParse.QueryParser<T>): URLParse<T>;
    (address: string, parser?: false): URLParse<string>;
    (address: string, parser: true): URLParse<Record<string, string | undefined>>;
    <T>(address: string, parser: URLParse.QueryParser<T>): URLParse<T>;
    (address: string, location?: string | object, parser?: false): URLParse<string>;
    (
        address: string,
        location: string | object | undefined,
        parser: true,
    ): URLParse<Record<string, string | undefined>>;
    <T>(address: string, location: string | object | undefined, parser: URLParse.QueryParser<T>): URLParse<T>;

    extractProtocol(url: string): {
        slashes: boolean;
        protocol: string;
        rest: string;
    };
    location(url: string): object;
    qs: {
        parse: URLParse.QueryParser;
        stringify: URLParse.StringifyQuery;
    };
    trimLeft(url: string): string;
};

declare enum Events$u {
    PushState = 0,
    ReplaceState = 1,
    PopState = 2,
    Back = 3,
    Forward = 4,
    Reload = 5,
    Start = 6,
    PathnameChange = 7,
    /** 销毁所有页面并跳转至指定页面 */
    Relaunch = 8,
    /** ???? */
    RedirectToHome = 9,
    HistoriesChange = 10
}
type TheTypesOfEvents$y = {
    [Events$u.PathnameChange]: {
        pathname: string;
        search: string;
        type: RouteAction;
    };
    [Events$u.PushState]: {
        from: string | null;
        to: string | null;
        path: string;
        pathname: string;
    };
    [Events$u.ReplaceState]: {
        from: string | null;
        path: string;
        pathname: string;
    };
    [Events$u.PopState]: {
        type: string;
        href: string;
        pathname: string;
    };
    [Events$u.Back]: void;
    [Events$u.Forward]: void;
    [Events$u.Reload]: void;
    [Events$u.Start]: RouteLocation;
    [Events$u.Relaunch]: void;
    [Events$u.HistoriesChange]: {
        pathname: string;
    }[];
};
type RouteLocation = {
    host: string;
    protocol: string;
    origin: string;
    pathname: string;
    href: string;
    search: string;
};
declare class NavigatorCore extends BaseDomain<TheTypesOfEvents$y> {
    static prefix: string | null;
    static parse(url: string): {
        query: Record<string, string>;
        pathname: string;
        auth: string;
        hash: string;
        host: string;
        hostname: string;
        href: string;
        origin: string;
        password: string;
        port: string;
        protocol: string;
        slashes: boolean;
        username: string;
        set<Part extends URLParse.URLPart>(part: Part, value: URLParse<string>[Part], fn?: false): URLParse<string>;
        set<Part extends URLParse.URLPart, T>(part: Part, value: URLParse<T>[Part], fn?: URLParse.QueryParser<T>): URLParse<T>;
        toString(stringify?: URLParse.StringifyQuery): string;
    };
    unique_id: string;
    debug: boolean;
    name: string;
    /** 当前 pathname */
    pathname: string;
    /** 当前路由的 query */
    query: Record<string, string>;
    /** 当前路由的 params */
    params: Record<string, string>;
    /** 当前 URL */
    location: Partial<RouteLocation>;
    href: string;
    histories: {
        pathname: string;
    }[];
    prevHistories: {
        pathname: string;
    }[];
    /** 发生跳转前的 pathname */
    prevPathname: string | null;
    /** router 基础信息 */
    origin: string;
    host: string;
    _pending: {
        pathname: string;
        search: string;
        type: RouteAction;
    };
    get state(): {
        pathname: string;
        search: string;
        params: Record<string, string>;
        query: Record<string, string>;
        location: Partial<RouteLocation>;
    };
    /** 启动路由监听 */
    prepare(location: RouteLocation): Promise<void>;
    start(): void;
    private setPrevPathname;
    private setPathname;
    /** 调用该方法来「改变地址」 */
    pushState(url: string): void;
    replaceState(url: string): Promise<void>;
    /** 外部路由改变（点击浏览器前进、后退），作出响应 */
    handlePopState({ type, pathname, href, }: {
        type: string;
        href: string;
        pathname: string;
    }): void;
    onStart(handler: Handler<TheTypesOfEvents$y[Events$u.Start]>): () => void;
    onHistoryChange(handler: Handler<TheTypesOfEvents$y[Events$u.HistoriesChange]>): () => void;
    onPushState(handler: Handler<TheTypesOfEvents$y[Events$u.PushState]>): () => void;
    onReplaceState(handler: Handler<TheTypesOfEvents$y[Events$u.ReplaceState]>): () => void;
    onPopState(handler: Handler<TheTypesOfEvents$y[Events$u.PopState]>): () => void;
    onReload(handler: Handler<TheTypesOfEvents$y[Events$u.Reload]>): () => void;
    onPathnameChange(handler: Handler<TheTypesOfEvents$y[Events$u.PathnameChange]>): () => void;
    onBack(handler: Handler<TheTypesOfEvents$y[Events$u.Back]>): () => void;
    onForward(handler: Handler<TheTypesOfEvents$y[Events$u.Forward]>): () => void;
    onRelaunch(handler: Handler<TheTypesOfEvents$y[Events$u.Relaunch]>): () => void;
    onHistoriesChange(handler: Handler<TheTypesOfEvents$y[Events$u.HistoriesChange]>): () => void;
}
type RouteAction = "initialize" | "push" | "replace" | "back" | "forward";

declare enum Events$t {
    TopViewChange = 0,
    RouteChange = 1,
    ClickLink = 2,
    Back = 3,
    Forward = 4,
    StateChange = 5
}
type TheTypesOfEvents$x = {
    [Events$t.TopViewChange]: RouteViewCore;
    [Events$t.ClickLink]: {
        href: string;
        target: string | null;
    };
    [Events$t.Back]: void;
    [Events$t.Forward]: void;
    [Events$t.RouteChange]: {
        view: RouteViewCore;
        name: string;
        href: string;
        pathname: string;
        query: Record<string, string>;
        reason: "back" | "forward" | "push" | "replace";
        /** 用于在页面间传递标记、数据等 */
        data?: any;
        /** 调用方希望忽略这次 route change */
        ignore?: boolean;
    };
    [Events$t.StateChange]: HistoryCoreState;
};
type HistoryCoreProps<K extends string, R extends Record<string, any>> = {
    view: RouteViewCore;
    router: NavigatorCore;
    routes: Record<K, R>;
    views: Record<K, RouteViewCore>;
    /** 是否采用虚拟路由（不改变浏览器历史） */
    virtual?: boolean;
};
type HistoryCoreState = {
    href: string;
    stacks: {
        id: string;
        key: string;
        title: string;
        visible: boolean;
        query: string;
    }[];
    cursor: number;
};
declare class HistoryCore<K extends string, R extends Record<string, any>> extends BaseDomain<TheTypesOfEvents$x> {
    virtual: boolean;
    /** 路由配置 */
    routes: Record<K, R>;
    /** 加载的所有视图 */
    views: Record<string, RouteViewCore>;
    /** 按顺序依次 push 的视图 */
    stacks: RouteViewCore[];
    /** 栈指针 */
    cursor: number;
    /** 浏览器 url 管理 */
    $router: NavigatorCore;
    /** 根视图 */
    $view: RouteViewCore;
    get state(): HistoryCoreState;
    constructor(props: Partial<{
        _name: string;
    }> & HistoryCoreProps<K, R>);
    push(name: K, query?: Record<string, string>, options?: Partial<{
        /** 不变更 history stack */
        ignore: boolean;
    }>): any;
    replace(name: K, query?: Record<string, string>): any;
    back(opt?: Partial<{
        data: any;
    }>): void;
    forward(): void;
    reload(): void;
    /** 销毁所有页面，然后前往指定路由 */
    destroyAllAndPush(name: K, query?: Record<string, string>, options?: Partial<{
        /** 不变更 history stack */
        ignore: boolean;
    }>): any;
    /** 跳转到兄弟页面 */
    ensureParent(view: RouteViewCore): any;
    buildURL(name: K, query?: Record<string, string>): string;
    buildURLWithPrefix(name: K, query?: Record<string, string>): string;
    isLayout(name: K): any;
    handleClickLink(params: {
        href: string;
        target: null | string;
    }): void;
    onTopViewChange(handler: Handler<TheTypesOfEvents$x[Events$t.TopViewChange]>): () => void;
    onRouteChange(handler: Handler<TheTypesOfEvents$x[Events$t.RouteChange]>): () => void;
    onBack(handler: Handler<TheTypesOfEvents$x[Events$t.Back]>): () => void;
    onForward(handler: Handler<TheTypesOfEvents$x[Events$t.Forward]>): () => void;
    onClickLink(handler: Handler<TheTypesOfEvents$x[Events$t.ClickLink]>): () => void;
    onStateChange(handler: Handler<TheTypesOfEvents$x[Events$t.StateChange]>): () => void;
}

declare enum Events$s {
    StateChange = 0
}
type TheTypesOfEvents$w = {
    [Events$s.StateChange]: void;
};
type HttpClientCoreProps = {
    hostname?: string;
    headers?: Record<string, string>;
    debug?: boolean;
};
declare class HttpClientCore extends BaseDomain<TheTypesOfEvents$w> {
    hostname: string;
    headers: Record<string, string>;
    debug: boolean;
    constructor(props?: HttpClientCoreProps);
    get<T>(endpoint: unknown, query?: Record<string, string | number | undefined>, extra?: Partial<{
        headers: Record<string, string | number>;
        id: string;
    }>): Promise<Result<T>>;
    post<T>(endpoint: unknown, body?: JSONObject | FormData, extra?: Partial<{
        headers: Record<string, string | number>;
        id: string;
    }>): Promise<Result<T>>;
    fetch<T>(options: {
        url: unknown;
        method: "GET" | "POST";
        id?: string;
        data?: JSONObject | FormData;
        headers?: Record<string, string | number>;
    }): Promise<{
        data: T;
    }>;
    cancel(id: string): Resp<null>;
    setHeaders(headers: Record<string, string>): void;
    appendHeaders(headers: Record<string, string>): void;
    setDebug(debug: boolean): void;
    onStateChange(handler: Handler<TheTypesOfEvents$w[Events$s.StateChange]>): () => void;
}

/**
 * @file 一个缓存/当前值
 * 类似 useRef
 */

declare enum Events$r {
    StateChange = 0
}
type TheTypesOfEvents$v<T> = {
    [Events$r.StateChange]: T;
};
type RefProps<T> = {
    defaultValue?: T;
    onChange?: (v: T) => void;
};
declare class RefCore<T> extends BaseDomain<TheTypesOfEvents$v<T>> {
    value: T | null;
    get state(): T | null;
    constructor(options?: Partial<{
        _name: string;
    }> & RefProps<T>);
    /** 暂存一个值 */
    select(value: T): void;
    patch(value: Partial<T>): void;
    /** 暂存的值是否为空 */
    isEmpty(): boolean;
    /** 返回 select 方法保存的 value 并将 value 重置为 null */
    clear(): void;
    onStateChange(handler: Handler<TheTypesOfEvents$v<T>[Events$r.StateChange]>): () => void;
}

declare enum Events$q {
    Click = 0,
    StateChange = 1
}
type TheTypesOfEvents$u<T = unknown> = {
    [Events$q.Click]: T | null;
    [Events$q.StateChange]: ButtonState;
};
type ButtonState = {
    text: string;
    loading: boolean;
    disabled: boolean;
};
type ButtonProps<T = unknown> = {
    disabled?: boolean;
    onClick: (record: T | null) => void;
};
declare class ButtonCore<T = unknown> extends BaseDomain<TheTypesOfEvents$u<T>> {
    id: number;
    cur: RefCore<T>;
    state: ButtonState;
    constructor(props?: Partial<{
        _name: string;
    } & ButtonProps<T>>);
    /** 触发一次按钮点击事件 */
    click(): void;
    /** 禁用当前按钮 */
    disable(): void;
    /** 恢复按钮可用 */
    enable(): void;
    /** 当按钮处于列表中时，使用该方法保存所在列表记录 */
    bind(v: T): this;
    setLoading(loading: boolean): void;
    onClick(handler: Handler<TheTypesOfEvents$u<T>[Events$q.Click]>): void;
    onStateChange(handler: Handler<TheTypesOfEvents$u<T>[Events$q.StateChange]>): void;
}
type ButtonInListProps<T = unknown> = {
    onClick: (record: T) => void;
};
type TheTypesInListOfEvents$2<T> = {
    [Events$q.Click]: T;
    [Events$q.StateChange]: ButtonState;
};
declare class ButtonInListCore<T> extends BaseDomain<TheTypesInListOfEvents$2<T>> {
    /** 列表中一类多个按钮 */
    btns: ButtonCore<T>[];
    /** 按钮点击后，该值被设置为触发点击的那个按钮 */
    cur: ButtonCore<T> | null;
    constructor(options?: Partial<{
        _name: string;
    } & ButtonInListProps<T>>);
    /** 当按钮处于列表中时，使用该方法保存所在列表记录 */
    bind(v: T): ButtonCore<T>;
    /** 清空触发点击事件时保存的按钮 */
    clear(): void;
    setLoading(loading: boolean): void;
    click(): void;
    onClick(handler: Handler<TheTypesInListOfEvents$2<T>[Events$q.Click]>): void;
    onStateChange(handler: Handler<TheTypesInListOfEvents$2<T>[Events$q.StateChange]>): void;
}

declare enum Events$p {
    StateChange = 0,
    Change = 1
}
type TheTypesOfEvents$t = {
    [Events$p.StateChange]: CheckboxState;
    [Events$p.Change]: boolean;
};
type CheckboxProps = {
    label?: string;
    checked?: boolean;
    value?: boolean;
    disabled?: boolean;
    required?: boolean;
    onChange?: (checked: boolean) => void;
};
type CheckboxState = CheckboxProps & {};
declare class CheckboxCore extends BaseDomain<TheTypesOfEvents$t> {
    shape: "checkbox";
    label: string;
    disabled: CheckboxProps["disabled"];
    checked: boolean;
    defaultChecked: boolean;
    presence: PresenceCore;
    get state(): CheckboxState;
    get value(): boolean;
    get defaultValue(): boolean;
    prev_checked: boolean;
    constructor(props?: {
        _name?: string;
    } & CheckboxProps);
    /** 切换选中状态 */
    toggle(): void;
    check(): void;
    uncheck(): void;
    reset(): void;
    setValue(v: boolean): void;
    onChange(handler: Handler<TheTypesOfEvents$t[Events$p.Change]>): () => void;
    onStateChange(handler: Handler<TheTypesOfEvents$t[Events$p.StateChange]>): () => void;
}

type Alignment = "start" | "end";
type Side$1 = "top" | "right" | "bottom" | "left";
type AlignedPlacement = `${Side$1}-${Alignment}`;
type Placement = Side$1 | AlignedPlacement;
type Strategy = "absolute" | "fixed";
type Axis = "x" | "y";
type Length = "width" | "height";
type Promisable<T> = T | Promise<T>;
type Coords = {
    [key in Axis]: number;
};
type SideObject = {
    [key in Side$1]: number;
};
interface MiddlewareData {
    [key: string]: any;
    arrow?: Partial<Coords> & {
        centerOffset: number;
    };
    autoPlacement?: {
        index?: number;
        overflows: Array<{
            placement: Placement;
            overflows: Array<number>;
        }>;
    };
    flip?: {
        index?: number;
        overflows: Array<{
            placement: Placement;
            overflows: Array<number>;
        }>;
    };
    hide?: {
        referenceHidden?: boolean;
        escaped?: boolean;
        referenceHiddenOffsets?: SideObject;
        escapedOffsets?: SideObject;
    };
    offset?: Coords;
    shift?: Coords;
}
interface MiddlewareReturn extends Partial<Coords> {
    data?: {
        [key: string]: any;
    };
    reset?: true | {
        placement?: Placement;
        rects?: true | ElementRects;
    };
}
type Middleware = {
    name: string;
    options?: any;
    fn: (state: MiddlewareState) => Promisable<MiddlewareReturn>;
};
type Dimensions = {
    [key in Length]: number;
};
type Rect = Coords & Dimensions & SideObject;
interface ElementRects {
    reference: Rect;
    floating: Rect;
}
type ReferenceElement = any;
type FloatingElement = any;
interface Elements {
    reference: ReferenceElement;
    floating: FloatingElement;
}
interface MiddlewareState extends Coords {
    initialPlacement: Placement;
    placement: Placement;
    strategy: Strategy;
    middlewareData: MiddlewareData;
    elements: Elements;
    rects: ElementRects;
}

declare const SIDE_OPTIONS: readonly ["top", "right", "bottom", "left"];
declare const ALIGN_OPTIONS: readonly ["start", "center", "end"];
type Side = (typeof SIDE_OPTIONS)[number];
type Align = (typeof ALIGN_OPTIONS)[number];
declare enum Events$o {
    /** 参考原始被加载 */
    ReferenceMounted = 0,
    /** 内容元素被加载（可以获取宽高位置） */
    FloatingMounted = 1,
    /** 被放置（其实就是计算好了浮动元素位置） */
    Placed = 2,
    /** 鼠标进入内容区 */
    Enter = 3,
    /** 鼠标离开内容区 */
    Leave = 4,
    StateChange = 5,
    /** 父容器改变 */
    ContainerChange = 6
}
type TheTypesOfEvents$s = {
    [Events$o.FloatingMounted]: {
        getRect: () => Rect;
    };
    [Events$o.ReferenceMounted]: {
        getRect: () => Rect;
    };
    [Events$o.ContainerChange]: Node;
    [Events$o.Placed]: PopperState;
    [Events$o.Enter]: void;
    [Events$o.Leave]: void;
    [Events$o.StateChange]: PopperState;
};
type PopperProps = {
    side: Side;
    align: Align;
    strategy: "fixed" | "absolute";
    middleware: Middleware[];
};
type PopperState = {
    strategy: Strategy;
    x: number;
    y: number;
    isPlaced: boolean;
    placedSide: Side;
    placedAlign: Align;
    /** 是否设置了参考DOM */
    reference: boolean;
};
declare class PopperCore extends BaseDomain<TheTypesOfEvents$s> {
    unique_id: string;
    debug: boolean;
    placement: Placement;
    strategy: Strategy;
    middleware: Middleware[];
    reference: {
        getRect: () => Rect;
    } | null;
    floating: {
        getRect: () => Rect;
    } | null;
    container: Node | null;
    arrow: {
        width: number;
        height: number;
    } | null;
    state: PopperState;
    _enter: boolean;
    _focus: boolean;
    constructor(options?: Partial<{
        _name: string;
    }> & Partial<PopperProps>);
    /** 基准元素加载完成 */
    setReference(reference: {
        $el?: unknown;
        getRect: () => Rect;
    }, opt?: Partial<{
        force: boolean;
    }>): void;
    /** 更新基准元素（右键菜单时会用到这个方法） */
    updateReference(reference: {
        getRect: () => Rect;
    }): void;
    removeReference(): void;
    /** 内容元素加载完成 */
    setFloating(floating: PopperCore["floating"]): void;
    /** 箭头加载完成 */
    setArrow(arrow: PopperCore["arrow"]): void;
    setContainer(container: Node): void;
    setConfig(config: {
        placement?: Placement;
        strategy?: Strategy;
    }): void;
    setState(v: {
        x: number;
        y: number;
    }): void;
    place2(floating: {
        x: number;
        y: number;
        width: number;
        height: number;
    }): void;
    /** 计算浮动元素位置 */
    place(): Promise<void>;
    computePosition(): Promise<{
        x: number;
        y: number;
        placement: Placement;
        strategy: Strategy;
        middlewareData: MiddlewareData;
    }>;
    /** 根据放置位置，计算浮动元素坐标 */
    computeCoordsFromPlacement(elms: {
        reference: Rect;
        floating: Rect;
    }, placement: Placement, rtl?: boolean): Coords;
    handleEnter(): void;
    handleLeave(): void;
    reset(): void;
    onReferenceMounted(handler: Handler<TheTypesOfEvents$s[Events$o.ReferenceMounted]>): () => void;
    onFloatingMounted(handler: Handler<TheTypesOfEvents$s[Events$o.FloatingMounted]>): () => void;
    onContainerChange(handler: Handler<TheTypesOfEvents$s[Events$o.ContainerChange]>): () => void;
    onEnter(handler: Handler<TheTypesOfEvents$s[Events$o.Enter]>): () => void;
    onLeave(handler: Handler<TheTypesOfEvents$s[Events$o.Leave]>): () => void;
    onPlaced(handler: Handler<TheTypesOfEvents$s[Events$o.Placed]>): () => void;
    onStateChange(handler: Handler<TheTypesOfEvents$s[Events$o.StateChange]>): () => void;
    get [Symbol.toStringTag](): string;
}

type AbsNode = {};
declare enum Events$n {
    /** 遮罩消失 */
    Dismiss = 0,
    FocusOutside = 1,
    PointerDownOutside = 2,
    InteractOutside = 3
}
type TheTypesOfEvents$r = {
    [Events$n.Dismiss]: void;
    [Events$n.PointerDownOutside]: void;
    [Events$n.FocusOutside]: void;
    [Events$n.InteractOutside]: void;
};
type DismissableLayerState = {};
declare class DismissableLayerCore extends BaseDomain<TheTypesOfEvents$r> {
    name: string;
    layers: Set<unknown>;
    layersWithOutsidePointerEventsDisabled: Set<unknown>;
    branches: Set<AbsNode>;
    isPointerInside: boolean;
    state: DismissableLayerState;
    constructor(options?: Partial<{
        _name: string;
    }>);
    handlePointerOutside(branch: HTMLElement): void;
    /** 响应点击事件 */
    pointerDown(): void;
    /** 响应冒泡到最顶层时的点击事件 */
    handlePointerDownOnTop(absNode?: {}): void;
    onDismiss(handler: Handler<TheTypesOfEvents$r[Events$n.Dismiss]>): () => void;
}

/**
 * @file 菜单项
 */

declare enum Events$m {
    Enter = 0,
    Leave = 1,
    Focus = 2,
    Blur = 3,
    Click = 4,
    Change = 5
}
type TheTypesOfEvents$q = {
    [Events$m.Enter]: void;
    [Events$m.Leave]: void;
    [Events$m.Focus]: void;
    [Events$m.Blur]: void;
    [Events$m.Click]: void;
    [Events$m.Change]: MenuItemCoreState;
};
type MenuItemCoreProps = {
    /** 菜单文案 */
    label: string;
    /** hover 时的提示 */
    tooltip?: string;
    /** 菜单图标 */
    icon?: unknown;
    /** 菜单快捷键/或者说额外内容? */
    shortcut?: string;
    /** 菜单是否禁用 */
    disabled?: boolean;
    /** 是否隐藏 */
    hidden?: boolean;
    /** 子菜单 */
    menu?: MenuCore;
    /** 点击后的回调 */
    onClick?: () => void;
};
type MenuItemCoreState = MenuItemCoreProps & {
    /** 有子菜单并且子菜单展示了 */
    open: boolean;
    /** 是否聚焦 */
    focused: boolean;
};
declare class MenuItemCore extends BaseDomain<TheTypesOfEvents$q> {
    _name: string;
    debug: boolean;
    label: string;
    tooltip?: string;
    icon?: unknown;
    shortcut?: string;
    /** 子菜单 */
    menu: MenuCore | null;
    /** 子菜单是否展示 */
    _open: boolean;
    _hidden: boolean;
    _enter: boolean;
    _focused: boolean;
    _disabled: boolean;
    get state(): MenuItemCoreState;
    get hidden(): boolean;
    constructor(options: Partial<{
        _name: string;
    }> & MenuItemCoreProps);
    setIcon(icon: unknown): void;
    /** 禁用指定菜单项 */
    disable(): void;
    /** 启用指定菜单项 */
    enable(): void;
    /** 鼠标进入菜单项 */
    handlePointerEnter(): void;
    handlePointerMove(): void;
    /** 鼠标离开菜单项 */
    handlePointerLeave(): void;
    handleFocus(): void;
    handleBlur(): void;
    handleClick(): void;
    blur(): void;
    reset(): void;
    hide(): void;
    show(): void;
    unmount(): void;
    onEnter(handler: Handler<TheTypesOfEvents$q[Events$m.Enter]>): () => void;
    onLeave(handler: Handler<TheTypesOfEvents$q[Events$m.Leave]>): () => void;
    onFocus(handler: Handler<TheTypesOfEvents$q[Events$m.Focus]>): () => void;
    onBlur(handler: Handler<TheTypesOfEvents$q[Events$m.Blur]>): () => void;
    onClick(handler: Handler<TheTypesOfEvents$q[Events$m.Click]>): () => void;
    onStateChange(handler: Handler<TheTypesOfEvents$q[Events$m.Change]>): () => void;
    get [Symbol.toStringTag](): string;
}

/**
 * @file 菜单 组件
 */

declare enum Events$l {
    Show = 0,
    Hidden = 1,
    EnterItem = 2,
    LeaveItem = 3,
    EnterMenu = 4,
    LeaveMenu = 5,
    StateChange = 6
}
type TheTypesOfEvents$p = {
    [Events$l.Show]: void;
    [Events$l.Hidden]: void;
    [Events$l.EnterItem]: MenuItemCore;
    [Events$l.LeaveItem]: MenuItemCore;
    [Events$l.EnterMenu]: void;
    [Events$l.LeaveMenu]: void;
    [Events$l.StateChange]: MenuCoreState;
};
type MenuCoreState = {
    /** 是否是展开状态 */
    open: boolean;
    hover: boolean;
    /** 所有选项 */
    items: MenuItemCore[];
};
type MenuCoreProps = {
    side: Side;
    align: Align;
    strategy: "fixed" | "absolute";
    items: MenuItemCore[];
};
declare class MenuCore extends BaseDomain<TheTypesOfEvents$p> {
    _name: string;
    debug: boolean;
    popper: PopperCore;
    presence: PresenceCore;
    layer: DismissableLayerCore;
    open_timer: NodeJS.Timeout | null;
    state: MenuCoreState;
    constructor(options?: Partial<{
        _name: string;
    } & MenuCoreProps>);
    items: MenuItemCore[];
    cur_sub: MenuCore | null;
    cur_item: MenuItemCore | null;
    inside: boolean;
    /** 鼠标是否处于子菜单中 */
    in_sub_menu: boolean;
    /** 鼠标离开 item 时，可能要隐藏子菜单，但是如果从有子菜单的 item 离开前往子菜单，就不用隐藏 */
    maybe_hide_sub: boolean;
    hide_sub_timer: NodeJS.Timeout | null;
    toggle(): void;
    show(): void;
    hide(): void;
    /** 处理选项 */
    listen_item(item: MenuItemCore): void;
    listen_items(items: MenuItemCore[]): void;
    setItems(items: MenuItemCore[]): void;
    checkNeedHideSubMenu(item: MenuItemCore): void;
    reset(): void;
    unmount(): void;
    onShow(handler: Handler<TheTypesOfEvents$p[Events$l.Show]>): () => void;
    onHide(handler: Handler<TheTypesOfEvents$p[Events$l.Hidden]>): () => void;
    onEnterItem(handler: Handler<TheTypesOfEvents$p[Events$l.EnterItem]>): () => void;
    onLeaveItem(handler: Handler<TheTypesOfEvents$p[Events$l.LeaveItem]>): () => void;
    onEnter(handler: Handler<TheTypesOfEvents$p[Events$l.EnterMenu]>): () => void;
    onLeave(handler: Handler<TheTypesOfEvents$p[Events$l.LeaveMenu]>): () => void;
    onStateChange(handler: Handler<TheTypesOfEvents$p[Events$l.StateChange]>): () => void;
    get [Symbol.toStringTag](): string;
}

declare enum Events$k {
    StateChange = 0,
    Show = 1,
    Hidden = 2
}
type TheTypeOfEvent = {
    [Events$k.StateChange]: ContextMenuState;
    [Events$k.Show]: void;
    [Events$k.Hidden]: void;
};
type ContextMenuState = {
    items: MenuItemCore[];
};
type ContextMenuProps = {
    items: MenuItemCore[];
};
declare class ContextMenuCore extends BaseDomain<TheTypeOfEvent> {
    menu: MenuCore;
    state: ContextMenuState;
    constructor(options: Partial<{
        _name: string;
    } & ContextMenuProps>);
    show(position?: Partial<{
        x: number;
        y: number;
    }>): void;
    hide(): void;
    setReference(reference: {
        getRect: () => Rect;
    }): void;
    updateReference(reference: {
        getRect: () => Rect;
    }): void;
    setItems(items: MenuItemCore[]): void;
    onStateChange(handler: Handler<TheTypeOfEvent[Events$k.StateChange]>): void;
    onShow(handler: Handler<TheTypeOfEvent[Events$k.Show]>): void;
    onHide(handler: Handler<TheTypeOfEvent[Events$k.Hidden]>): void;
}

/**
 * @file 弹窗核心类
 */

declare enum Events$j {
    BeforeShow = 0,
    Show = 1,
    BeforeHidden = 2,
    Hidden = 3,
    Unmounted = 4,
    VisibleChange = 5,
    Cancel = 6,
    OK = 7,
    AnimationStart = 8,
    AnimationEnd = 9,
    StateChange = 10
}
type TheTypesOfEvents$o = {
    [Events$j.BeforeShow]: void;
    [Events$j.Show]: void;
    [Events$j.BeforeHidden]: void;
    [Events$j.Hidden]: void;
    [Events$j.Unmounted]: void;
    [Events$j.VisibleChange]: boolean;
    [Events$j.OK]: void;
    [Events$j.Cancel]: void;
    [Events$j.AnimationStart]: void;
    [Events$j.AnimationEnd]: void;
    [Events$j.StateChange]: DialogState;
};
type DialogProps = {
    title?: string;
    footer?: boolean;
    closeable?: boolean;
    mask?: boolean;
    open?: boolean;
    onCancel?: () => void;
    onOk?: () => void;
    onUnmounted?: () => void;
};
type DialogState = {
    open: boolean;
    title: string;
    footer: boolean;
    /** 能否手动关闭 */
    closeable: boolean;
    mask: boolean;
    enter: boolean;
    visible: boolean;
    exit: boolean;
};
declare class DialogCore extends BaseDomain<TheTypesOfEvents$o> {
    open: boolean;
    title: string;
    footer: boolean;
    closeable: boolean;
    mask: boolean;
    present: PresenceCore;
    okBtn: ButtonCore<unknown>;
    cancelBtn: ButtonCore<unknown>;
    get state(): DialogState;
    constructor(props?: Partial<{
        _name: string;
    }> & DialogProps);
    toggle(): void;
    /** 显示弹窗 */
    show(): void;
    /** 隐藏弹窗 */
    hide(opt?: Partial<{
        destroy: boolean;
    }>): void;
    ok(): void;
    cancel(): void;
    setTitle(title: string): void;
    onShow(handler: Handler<TheTypesOfEvents$o[Events$j.Show]>): () => void;
    onHidden(handler: Handler<TheTypesOfEvents$o[Events$j.Hidden]>): () => void;
    onUnmounted(handler: Handler<TheTypesOfEvents$o[Events$j.Unmounted]>): () => void;
    onVisibleChange(handler: Handler<TheTypesOfEvents$o[Events$j.VisibleChange]>): () => void;
    onOk(handler: Handler<TheTypesOfEvents$o[Events$j.OK]>): () => void;
    onCancel(handler: Handler<TheTypesOfEvents$o[Events$j.Cancel]>): () => void;
    onStateChange(handler: Handler<TheTypesOfEvents$o[Events$j.StateChange]>): () => void;
    get [Symbol.toStringTag](): string;
}

type Direction = "ltr" | "rtl";
type Orientation = "horizontal" | "vertical";

declare enum Events$i {
    StateChange = 0
}
type TheTypesOfEvents$n = {
    [Events$i.StateChange]: DropdownMenuState;
};
type DropdownMenuProps = {
    side?: Side$1;
    align?: Align;
    items?: MenuItemCore[];
    onHidden?: () => void;
};
type DropdownMenuState = {
    items: MenuItemCore[];
    open: boolean;
    disabled: boolean;
    enter: boolean;
    visible: boolean;
    exit: boolean;
};
declare class DropdownMenuCore extends BaseDomain<TheTypesOfEvents$n> {
    open: boolean;
    disabled: boolean;
    get state(): DropdownMenuState;
    menu: MenuCore;
    subs: MenuCore[];
    items: MenuItemCore[];
    constructor(props?: {
        _name?: string;
    } & DropdownMenuProps);
    listenItems(items: MenuItemCore[]): void;
    setItems(items: MenuItemCore[]): void;
    showMenuItem(label: string): void;
    toggle(position?: Partial<{
        x: number;
        y: number;
        width: number;
        height: number;
    }>): void;
    hide(): void;
    unmount(): void;
    onStateChange(handler: Handler<TheTypesOfEvents$n[Events$i.StateChange]>): void;
}

declare enum Events$h {
    StateChange = 0,
    Focusin = 1,
    Focusout = 2
}
type TheTypesOfEvents$m = {
    [Events$h.StateChange]: FocusScopeState;
    [Events$h.Focusin]: void;
    [Events$h.Focusout]: void;
};
type FocusScopeState = {
    paused: boolean;
};
declare class FocusScopeCore extends BaseDomain<TheTypesOfEvents$m> {
    name: string;
    state: FocusScopeState;
    constructor(options?: Partial<{
        _name: string;
    }>);
    pause(): void;
    resume(): void;
    focusin(): void;
    focusout(): void;
    onFocusin(handler: Handler<TheTypesOfEvents$m[Events$h.Focusin]>): void;
    onFocusout(handler: Handler<TheTypesOfEvents$m[Events$h.Focusout]>): void;
    onStateChange(handler: Handler<TheTypesOfEvents$m[Events$h.StateChange]>): void;
}

type ValueInputInterface<T> = {
    shape: "select" | "input" | "drag-upload" | "image-upload" | "upload" | "date-picker" | "list" | "form";
    value: T;
    setValue: (v: T, extra?: Partial<{
        silence: boolean;
    }>) => void;
    onChange: (fn: (v: T) => void) => void;
};

declare enum Events$g {
    Show = 0,
    Hide = 1,
    StateChange = 2
}
type TheTypesOfEvents$l = {
    [Events$g.Show]: void;
    [Events$g.Hide]: void;
    [Events$g.StateChange]: FormFieldCoreState;
};
type FormFieldCoreState = {
    label: string;
    name: string;
    required: boolean;
    hidden: boolean;
};
declare class FormFieldCore<T extends {
    label: string;
    name: string;
    required?: boolean;
    input: ValueInputInterface<any>;
}> extends BaseDomain<TheTypesOfEvents$l> {
    _label: string;
    _name: string;
    _required: boolean;
    _hidden: boolean;
    $input: T["input"];
    get state(): FormFieldCoreState;
    get label(): string;
    get name(): string;
    constructor(props: Partial<{
        _name: string;
    }> & T);
    setLabel(label: string): void;
    setValue(...args: Parameters<typeof this$1.$input.setValue>): void;
    hide(): void;
    show(): void;
    onShow(handler: Handler<TheTypesOfEvents$l[Events$g.Show]>): () => void;
    onHide(handler: Handler<TheTypesOfEvents$l[Events$g.Hide]>): () => void;
    onStateChange(handler: Handler<TheTypesOfEvents$l[Events$g.StateChange]>): () => void;
}

/**
 * @file 多字段 Input
 */

type FormProps<F extends Record<string, FormFieldCore<any>>> = {
    fields: F;
};
declare function FormCore<F extends Record<string, FormFieldCore<{
    label: string;
    name: string;
    input: ValueInputInterface<any>;
}>> = {}>(props: FormProps<F>): {
    symbol: "FormCore";
    shape: "form";
    state: {
        readonly value: { [K in keyof F]: F[K]["$input"]["value"]; };
        readonly fields: FormFieldCore<{
            label: string;
            name: string;
            input: ValueInputInterface<any>;
        }>[];
        readonly inline: boolean;
    };
    readonly value: { [K in keyof F]: F[K]["$input"]["value"]; };
    readonly fields: F;
    setValue(v: { [K in keyof F]: F[K]["$input"]["value"]; }, extra?: {
        silence?: boolean;
    }): void;
    setInline(v: boolean): void;
    input<Key extends keyof F>(key: Key, value: { [K in keyof F]: F[K]["$input"]["value"]; }[Key]): void;
    submit(): void;
    validate(): Result<{ [K in keyof F]: F[K]["$input"]["value"]; }>;
    onSubmit(handler: Handler<{ [K in keyof F]: F[K]["$input"]["value"]; }>): void;
    onInput(handler: Handler<{ [K in keyof F]: F[K]["$input"]["value"]; }>): void;
    onChange(handler: Handler<{ [K in keyof F]: F[K]["$input"]["value"]; }>): () => void;
    onStateChange(handler: Handler<{
        readonly value: { [K in keyof F]: F[K]["$input"]["value"]; };
        readonly fields: FormFieldCore<{
            label: string;
            name: string;
            input: ValueInputInterface<any>;
        }>[];
        readonly inline: boolean;
    }>): () => void;
};
type FormCore<F extends Record<string, FormFieldCore<{
    label: string;
    name: string;
    input: ValueInputInterface<any>;
}>> = {}> = ReturnType<typeof FormCore<F>>;

type FormInputInterface<T> = {
    shape: "number" | "string" | "textarea" | "boolean" | "select" | "multiple-select" | "tag-input" | "custom" | "switch" | "checkbox" | "input" | "drag-upload" | "image-upload" | "upload" | "date-picker" | "list" | "form" | "drag-select";
    value: T;
    defaultValue: T;
    setValue: (v: T, extra?: Partial<{
        silence: boolean;
    }>) => void;
    destroy?: () => void;
    onChange: (fn: (v: T) => void) => void;
};

type CommonRuleCore = {
    required: boolean;
};
type NumberRuleCore = {
    min: number;
    max: number;
};
type StringRuleCore = {
    minLength: number;
    maxLength: number;
    mode: "email" | "number";
};
type FieldRuleCore = Partial<CommonRuleCore & NumberRuleCore & StringRuleCore & {
    custom(v: any): Result<null>;
}>;
type FormFieldCoreProps = {
    label?: string;
    /** @deprecated */
    name?: string;
    rules?: FieldRuleCore[];
};
type FieldStatus = "normal" | "focus" | "warning" | "error" | "success";
declare enum SingleFieldEvents {
    Change = 0,
    StateChange = 1
}
type TheSingleFieldCoreEvents<T extends FormInputInterface<any>["value"]> = {
    [SingleFieldEvents.Change]: T;
    [SingleFieldEvents.StateChange]: SingleFieldCoreState<T>;
};
type SingleFieldCoreProps<T> = FormFieldCoreProps & {
    input: T;
    hidden?: boolean;
};
type SingleFieldCoreState<T> = {
    symbol: string;
    label: string;
    hidden: boolean;
    focus: boolean;
    error: BizError | null;
    status: FieldStatus;
    input: {
        shape: string;
        value: T;
        type: any;
        options?: any[];
    };
};
declare class SingleFieldCore<T extends FormInputInterface<any>> {
    symbol: "SingleFieldCore";
    _label: string;
    _hidden: boolean;
    _error: BizError | null;
    _status: FieldStatus;
    _focus: boolean;
    _input: T;
    _rules: FieldRuleCore[];
    _dirty: boolean;
    _bus: {
        off<Key extends BaseEvents.Destroy | keyof TheSingleFieldCoreEvents<T>>(event: Key, handler: Handler<({
            __destroy: void;
        } & TheSingleFieldCoreEvents<T>)[Key]>): void;
        on<Key extends BaseEvents.Destroy | keyof TheSingleFieldCoreEvents<T>>(event: Key, handler: Handler<({
            __destroy: void;
        } & TheSingleFieldCoreEvents<T>)[Key]>): () => void;
        uid: () => number;
        emit<Key extends BaseEvents.Destroy | keyof TheSingleFieldCoreEvents<T>>(event: Key, value?: ({
            __destroy: void;
        } & TheSingleFieldCoreEvents<T>)[Key]): void;
        destroy(): void;
    };
    get state(): SingleFieldCoreState<T>;
    constructor(props: SingleFieldCoreProps<T>);
    get label(): string;
    get hidden(): boolean;
    get dirty(): boolean;
    get input(): T;
    get value(): T["value"];
    hide(): void;
    show(): void;
    showField(key: string): void;
    hideField(key: string): void;
    setFieldValue(key: string, v: any): void;
    clear(): void;
    validate(): Promise<Result<any>>;
    setValue(value: T["value"], extra?: Partial<{
        key: string;
        idx: number;
        silence: boolean;
    }>): void;
    setStatus(status: FieldStatus): void;
    setFocus(v: boolean): void;
    handleValueChange(value: T["value"]): void;
    ready(): void;
    destroy(): void;
    onChange(handler: Handler<TheSingleFieldCoreEvents<T>[SingleFieldEvents.Change]>): () => void;
    onStateChange(handler: Handler<TheSingleFieldCoreEvents<T>[SingleFieldEvents.StateChange]>): () => void;
}
type ArrayFieldCoreProps<T extends (count: number) => SingleFieldCore<any> | ArrayFieldCore<any> | ObjectFieldCore<any>> = FormFieldCoreProps & {
    field: T;
    hidden?: boolean;
};
type ArrayFieldValue<T extends (count: number) => SingleFieldCore<any> | ArrayFieldCore<any> | ObjectFieldCore<any>> = ReturnType<T>["value"];
type ArrayFieldCoreState = {
    label: string;
    hidden: boolean;
    fields: {
        id: number;
        label: string;
    }[];
};
declare enum ArrayFieldEvents {
    Change = 0,
    StateChange = 1
}
type TheArrayFieldCoreEvents<T extends (count: number) => SingleFieldCore<any> | ArrayFieldCore<any> | ObjectFieldCore<any>> = {
    [ArrayFieldEvents.Change]: {
        idx: number;
        id: number;
    };
    [ArrayFieldEvents.StateChange]: ArrayFieldValue<T>;
};
declare class ArrayFieldCore<T extends (count: number) => SingleFieldCore<any> | ArrayFieldCore<any> | ObjectFieldCore<any>> {
    symbol: "ArrayFieldCore";
    _label: string;
    _hidden: boolean;
    fields: {
        id: number;
        idx: number;
        field: ReturnType<T>;
    }[];
    _field: T;
    _bus: {
        off<Key extends BaseEvents.Destroy | keyof TheArrayFieldCoreEvents<T>>(event: Key, handler: Handler<({
            __destroy: void;
        } & TheArrayFieldCoreEvents<T>)[Key]>): void;
        on<Key extends BaseEvents.Destroy | keyof TheArrayFieldCoreEvents<T>>(event: Key, handler: Handler<({
            __destroy: void;
        } & TheArrayFieldCoreEvents<T>)[Key]>): () => void;
        uid: () => number;
        emit<Key extends BaseEvents.Destroy | keyof TheArrayFieldCoreEvents<T>>(event: Key, value?: ({
            __destroy: void;
        } & TheArrayFieldCoreEvents<T>)[Key]): void;
        destroy(): void;
    };
    get state(): ArrayFieldCoreState;
    constructor(props: ArrayFieldCoreProps<T>);
    mapFieldWithIndex(index: number): {
        id: number;
        idx: number;
        field: ReturnType<T>;
    };
    getFieldWithId(id: number): {
        id: number;
        idx: number;
        field: ReturnType<T>;
    };
    showField(key: string): void;
    hideField(key: string): void;
    setFieldValue(key: string, v: any): void;
    get label(): string;
    get hidden(): boolean;
    get value(): ArrayFieldValue<T>[];
    refresh(): void;
    hide(): void;
    show(): void;
    setValue(values: any[], extra?: Partial<{
        key: string;
        idx: number;
        silence: boolean;
    }>): void;
    clear(): void;
    validate(): Promise<Result<ArrayFieldValue<T>[]>>;
    insertBefore(id: number): ReturnType<T>;
    insertAfter(id: number): ReturnType<T>;
    append(opt?: Partial<{
        silence: boolean;
    }>): ReturnType<T>;
    remove(id: number): void;
    removeByIndex(idx: number): void;
    /** 将指定的元素，向前移动一个位置 */
    upIdx(id: number): void;
    /** 将指定的元素，向后移动一个位置 */
    downIdx(id: number): void;
    ready(): void;
    destroy(): void;
    onChange(handler: Handler<TheArrayFieldCoreEvents<T>[ArrayFieldEvents.Change]>): () => void;
    onStateChange(handler: Handler<TheArrayFieldCoreEvents<T>[ArrayFieldEvents.StateChange]>): () => void;
}
type ObjectValue<O extends Record<string, SingleFieldCore<any> | ArrayFieldCore<any> | ObjectFieldCore<any>>> = {
    [K in keyof O]: O[K] extends SingleFieldCore<any> ? O[K]["value"] : O[K] extends ArrayFieldCore<any> ? O[K]["value"] : O[K] extends ObjectFieldCore<any> ? O[K]["value"] : never;
};
type ObjectFieldCoreProps<T> = FormFieldCoreProps & {
    fields: T;
    hidden?: boolean;
};
type ObjectFieldCoreState = {
    label: string;
    hidden: boolean;
    fields: {
        symbol: string;
        label: string;
        name: string;
        hidden: boolean;
    }[];
};
declare enum ObjectFieldEvents {
    Change = 0,
    StateChange = 1
}
type TheObjectFieldCoreEvents<T extends Record<string, SingleFieldCore<any> | ArrayFieldCore<any> | ObjectFieldCore<any>>> = {
    [ObjectFieldEvents.Change]: ObjectValue<T>;
    [ObjectFieldEvents.StateChange]: ObjectFieldCoreState;
};
declare class ObjectFieldCore<T extends Record<string, SingleFieldCore<any> | ArrayFieldCore<any> | ObjectFieldCore<any>>> {
    symbol: "ObjectFieldCore";
    _label: string;
    _hidden: boolean;
    _dirty: boolean;
    fields: T;
    rules: FieldRuleCore[];
    _bus: {
        off<Key extends BaseEvents.Destroy | keyof TheObjectFieldCoreEvents<T>>(event: Key, handler: Handler<({
            __destroy: void;
        } & TheObjectFieldCoreEvents<T>)[Key]>): void;
        on<Key extends BaseEvents.Destroy | keyof TheObjectFieldCoreEvents<T>>(event: Key, handler: Handler<({
            __destroy: void;
        } & TheObjectFieldCoreEvents<T>)[Key]>): () => void;
        uid: () => number;
        emit<Key extends BaseEvents.Destroy | keyof TheObjectFieldCoreEvents<T>>(event: Key, value?: ({
            __destroy: void;
        } & TheObjectFieldCoreEvents<T>)[Key]): void;
        destroy(): void;
    };
    get state(): ObjectFieldCoreState;
    constructor(props: ObjectFieldCoreProps<T>);
    get label(): string;
    get hidden(): boolean;
    get dirty(): boolean;
    get value(): ObjectValue<T>;
    mapFieldWithName(name: string): SingleFieldCore<any> | ArrayFieldCore<any> | ObjectFieldCore<any>;
    setField(name: string, field: SingleFieldCore<any> | ArrayFieldCore<any> | ObjectFieldCore<any>): void;
    showField(name: string): void;
    hideField(name: string): void;
    setFieldValue(key: string, v: any): void;
    hide(): void;
    show(): void;
    setValue(values: Partial<Record<keyof T, any>>, extra?: Partial<{
        key: keyof T;
        idx: number;
        silence: boolean;
    }>): void;
    refresh(): void;
    clear(): void;
    validate(): Promise<Result<ObjectValue<T>>>;
    handleValueChange(path: string, value: any): void;
    toJSON(): {
        [x: string]: any;
    };
    ready(): void;
    destroy(): void;
    onChange(handler: Handler<TheObjectFieldCoreEvents<T>[ObjectFieldEvents.Change]>): () => void;
    onStateChange(handler: Handler<TheObjectFieldCoreEvents<T>[ObjectFieldEvents.StateChange]>): () => void;
}

declare enum Events$f {
    StateChange = 0,
    StartLoad = 1,
    Loaded = 2,
    Error = 3
}
type TheTypesOfEvents$k = {
    [Events$f.StateChange]: ImageState;
    [Events$f.StartLoad]: void;
    [Events$f.Loaded]: void;
    [Events$f.Error]: void;
};
declare enum ImageStep {
    Pending = 0,
    Loading = 1,
    Loaded = 2,
    Failed = 3
}
type ImageProps = {
    /** 图片宽度 */
    width?: number;
    /** 图片高度 */
    height?: number;
    /** 图片地址 */
    src?: string;
    /** 说明 */
    alt?: string;
    scale?: number;
    /** 模式 */
    fit?: "cover" | "contain";
    unique_id?: unknown;
};
type ImageState = Omit<ImageProps, "scale"> & {
    step: ImageStep;
    scale: number | null;
};
declare class ImageCore extends BaseDomain<TheTypesOfEvents$k> {
    static prefix: string;
    static url(url?: string | null): string;
    unique_uid: unknown;
    src: string;
    width: number;
    height: number;
    scale: null | number;
    fit: "cover" | "contain";
    step: ImageStep;
    realSrc?: string;
    get state(): ImageState;
    constructor(props: Partial<{}> & ImageProps);
    setURL(src?: string | null): void;
    setLoaded(): void;
    /** 图片进入可视区域 */
    handleShow(): void;
    /** 图片加载完成 */
    handleLoaded(): void;
    /** 图片加载失败 */
    handleError(): void;
    onStateChange(handler: Handler<TheTypesOfEvents$k[Events$f.StateChange]>): () => void;
    onStartLoad(handler: Handler<TheTypesOfEvents$k[Events$f.StartLoad]>): () => void;
    onLoad(handler: Handler<TheTypesOfEvents$k[Events$f.Loaded]>): () => void;
    onError(handler: Handler<TheTypesOfEvents$k[Events$f.Error]>): () => void;
}
declare class ImageInListCore extends BaseDomain<TheTypesOfEvents$k> {
    /** 列表中一类多个按钮 */
    btns: ImageCore[];
    /** 按钮点击后，该值被设置为触发点击的那个按钮 */
    cur: ImageCore | null;
    scale: number | null;
    constructor(props?: Partial<{
        _name: string;
    } & ImageCore>);
    /** 当按钮处于列表中时，使用该方法保存所在列表记录 */
    bind(unique_id?: string): ImageCore;
    select(unique_id: unknown): void;
    /** 清空触发点击事件时保存的按钮 */
    clear(): void;
    onStateChange(handler: Handler<TheTypesOfEvents$k[Events$f.StateChange]>): void;
}

declare enum Events$e {
    Change = 10,
    StateChange = 11,
    Mounted = 12,
    Focus = 13,
    Blur = 14,
    Enter = 15,
    KeyDown = 16,
    Clear = 17,
    Click = 18
}
type TheTypesOfEvents$j<T> = {
    [Events$e.Mounted]: void;
    [Events$e.Change]: T;
    [Events$e.Blur]: T;
    [Events$e.Enter]: T;
    [Events$e.KeyDown]: {
        key: string;
        preventDefault: () => void;
    };
    [Events$e.Focus]: void;
    [Events$e.Clear]: void;
    [Events$e.Click]: {
        x: number;
        y: number;
    };
    [Events$e.StateChange]: InputState<T>;
};
type InputProps<T> = {
    /** 字段键 */
    name?: string;
    disabled?: boolean;
    defaultValue: T;
    placeholder?: string;
    type?: string;
    autoFocus?: boolean;
    autoComplete?: boolean;
    ignoreEnterEvent?: boolean;
    onChange?: (v: T) => void;
    onKeyDown?: (v: {
        key: string;
        preventDefault: () => void;
    }) => void;
    onEnter?: (v: T) => void;
    onBlur?: (v: T) => void;
    onClear?: () => void;
    onMounted?: () => void;
};
type InputState<T> = {
    value: T;
    placeholder: string;
    disabled: boolean;
    loading: boolean;
    focus: boolean;
    type: string;
    tmpType: string;
    allowClear: boolean;
    autoFocus: boolean;
    autoComplete: boolean;
};
declare class InputCore<T> extends BaseDomain<TheTypesOfEvents$j<T>> implements ValueInputInterface<T> {
    shape: "input";
    defaultValue: T;
    value: T;
    placeholder: string;
    disabled: boolean;
    allowClear: boolean;
    autoComplete: boolean;
    autoFocus: boolean;
    ignoreEnterEvent: boolean;
    isFocus: boolean;
    type: string;
    loading: boolean;
    /** 被消费过的值，用于做比较判断 input 值是否发生改变 */
    valueUsed: unknown;
    tmpType: string;
    get state(): InputState<T>;
    constructor(props: {
        unique_id?: string;
    } & InputProps<T>);
    setMounted(): void;
    handleKeyDown(event: {
        key: string;
        preventDefault: () => void;
    }): void;
    handleEnter(): void;
    handleFocus(): void;
    handleBlur(): void;
    handleClick(event: {
        x: number;
        y: number;
    }): void;
    handleChange(event: unknown): void;
    setValue(value: T, extra?: Partial<{
        silence: boolean;
    }>): void;
    setPlaceholder(v: string): void;
    setLoading(loading: boolean): void;
    setFocus(): void;
    focus(): void;
    enable(): void;
    disable(): void;
    showText(): void;
    hideText(): void;
    clear(): void;
    reset(): void;
    enter(): void;
    onChange(handler: Handler<TheTypesOfEvents$j<T>[Events$e.Change]>): () => void;
    onStateChange(handler: Handler<TheTypesOfEvents$j<T>[Events$e.StateChange]>): () => void;
    onMounted(handler: Handler<TheTypesOfEvents$j<T>[Events$e.Mounted]>): () => void;
    onFocus(handler: Handler<TheTypesOfEvents$j<T>[Events$e.Focus]>): () => void;
    onBlur(handler: Handler<TheTypesOfEvents$j<T>[Events$e.Blur]>): () => void;
    onKeyDown(handler: Handler<TheTypesOfEvents$j<T>[Events$e.KeyDown]>): () => void;
    onEnter(handler: Handler<TheTypesOfEvents$j<T>[Events$e.Enter]>): () => void;
    onClick(handler: Handler<TheTypesOfEvents$j<T>[Events$e.Click]>): () => void;
    onClear(handler: Handler<TheTypesOfEvents$j<T>[Events$e.Clear]>): () => void;
}
type InputInListProps<T = unknown> = {
    onChange?: (record: T) => void;
} & InputProps<T>;
type TheTypesInListOfEvents$1<K extends string, T> = {
    [Events$e.Change]: [K, T];
    [Events$e.StateChange]: InputProps<T>;
};
declare class InputInListCore<K extends string, T> extends BaseDomain<TheTypesInListOfEvents$1<K, T>> {
    defaultValue: T;
    list: InputCore<T>[];
    cached: Record<K, InputCore<T>>;
    values: Map<K, T | null>;
    constructor(props: Partial<{
        unique_id: string;
    }> & InputInListProps<T>);
    bind(unique_id: K, options?: {
        defaultValue?: T;
    }): InputCore<T>;
    getCur(unique_id: K): Record<K, InputCore<T>>[K];
    setValue(v: T): void;
    clear(): void;
    getValueByUniqueId(key: K): T;
    toJson<R>(handler: (value: [K, T | null]) => R): R[];
    /** 清空触发点击事件时保存的按钮 */
    onChange(handler: Handler<TheTypesInListOfEvents$1<K, T>[Events$e.Change]>): void;
    onStateChange(handler: Handler<TheTypesInListOfEvents$1<K, T>[Events$e.StateChange]>): void;
}

declare enum Events$d {
    Click = 0,
    ContextMenu = 1,
    Mounted = 2,
    EnterViewport = 3
}
type TheTypesOfEvents$i = {
    [Events$d.EnterViewport]: void;
    [Events$d.Mounted]: void;
    [Events$d.Click]: Events$d & {
        target: HTMLElement;
    };
};
declare class NodeCore extends BaseDomain<TheTypesOfEvents$i> {
    handleShow(): void;
    onVisible(handler: Handler<TheTypesOfEvents$i[Events$d.EnterViewport]>): () => void;
    onClick(handler: Handler<TheTypesOfEvents$i[Events$d.Click]>): () => void;
}

/**
 * @file 气泡
 */

declare enum Events$c {
    Show = 0,
    Hidden = 1,
    StateChange = 2
}
type TheTypesOfEvents$h = {
    [Events$c.Show]: void;
    [Events$c.Hidden]: void;
    [Events$c.StateChange]: PopoverState;
};
type PopoverState = {
    isPlaced: boolean;
    closeable: boolean;
    x: number;
    y: number;
    visible: boolean;
    enter: boolean;
    exit: boolean;
};
type PopoverProps = {
    side?: Side;
    align?: Align;
    strategy?: "fixed" | "absolute";
    closeable?: boolean;
};
declare class PopoverCore extends BaseDomain<TheTypesOfEvents$h> {
    popper: PopperCore;
    present: PresenceCore;
    layer: DismissableLayerCore;
    _side: Side;
    _align: Align;
    _closeable: boolean;
    visible: boolean;
    enter: boolean;
    exit: boolean;
    get state(): PopoverState;
    constructor(props?: {
        _name?: string;
    } & PopoverProps);
    ready(): void;
    destroy(): void;
    toggle(position?: Partial<{
        x: number;
        y: number;
        width: number;
        height: number;
    }>): void;
    show(position?: {
        x?: number;
        y?: number;
        width?: number;
        height?: number;
        left?: number;
        top?: number;
        right?: number;
        bottom?: number;
    }): void;
    hide(): void;
    unmount(): void;
    onShow(handler: Handler<TheTypesOfEvents$h[Events$c.Show]>): () => void;
    onHide(handler: Handler<TheTypesOfEvents$h[Events$c.Hidden]>): () => void;
    onStateChange(handler: Handler<TheTypesOfEvents$h[Events$c.StateChange]>): () => void;
    get [Symbol.toStringTag](): string;
}

type ProgressState = "indeterminate" | "complete" | "loading";
declare enum Events$b {
    ValueChange = 0,
    StateChange = 1
}
type TheTypesOfEvents$g = {
    [Events$b.ValueChange]: number;
    [Events$b.StateChange]: ProgressCore["state"];
};
declare class ProgressCore extends BaseDomain<TheTypesOfEvents$g> {
    _value: number | null;
    _label: string | undefined;
    _max: number;
    constructor(options: {
        value?: number | null | undefined;
        max?: number;
        getValueLabel?: (value: number, max: number) => string;
    });
    get state(): {
        state: ProgressState;
        value: number;
        max: number;
        label: string;
    };
    setValue(v: number): void;
    update(v: number): void;
    onValueChange(handler: Handler<TheTypesOfEvents$g[Events$b.ValueChange]>): void;
    onStateChange(handler: Handler<TheTypesOfEvents$g[Events$b.StateChange]>): void;
}

type TheTypesOfEvents$f = {};
declare class CollectionCore extends BaseDomain<TheTypesOfEvents$f> {
    itemMap: Map<unknown, unknown>;
    setWrap(wrap: unknown): void;
    add(key: unknown, v: unknown): void;
    remove(key: unknown): void;
    getItems(): unknown[];
}

declare enum Events$a {
    ItemFocus = 0,
    ItemShiftTab = 1,
    FocusableItemAdd = 2,
    FocusableItemRemove = 3,
    StateChange = 4
}
type TheTypesOfEvents$e = {
    [Events$a.ItemFocus]: number;
    [Events$a.ItemShiftTab]: void;
    [Events$a.FocusableItemAdd]: void;
    [Events$a.FocusableItemRemove]: void;
    [Events$a.StateChange]: RovingFocusState;
};
type RovingFocusState = {
    currentTabStopId: number | null;
    orientation?: Orientation;
    dir?: Direction;
    loop?: boolean;
};
declare class RovingFocusCore extends BaseDomain<TheTypesOfEvents$e> {
    collection: CollectionCore;
    state: RovingFocusState;
    constructor(options?: Partial<{
        _name: string;
    }>);
    focusItem(id: number): void;
    shiftTab(): void;
    addFocusableItem(): void;
    removeFocusableItem(): void;
    onStateChange(handler: Handler<TheTypesOfEvents$e[Events$a.StateChange]>): void;
    onItemFocus(handler: Handler<TheTypesOfEvents$e[Events$a.ItemFocus]>): void;
    onItemShiftTab(handler: Handler<TheTypesOfEvents$e[Events$a.ItemShiftTab]>): void;
    onFocusableItemAdd(handler: Handler<TheTypesOfEvents$e[Events$a.FocusableItemAdd]>): void;
    onFocusableItemRemove(handler: Handler<TheTypesOfEvents$e[Events$a.FocusableItemRemove]>): void;
}

/**
 * 根据点击滑动事件获取第一个手指的坐标
 */
declare function getPoint(e: {
    touches?: {
        pageX: number;
        pageY: number;
    }[];
    clientX?: number;
    clientY?: number;
}): {
    x: number;
    y: number;
};
/**
 * 阻止浏览器默认事件
 */
declare function preventDefault(e: {
    cancelable?: boolean;
    defaultPrevented?: boolean;
    preventDefault?: () => void;
}): void;
/**
 * 阻尼效果
 * 代码来自 https://www.jianshu.com/p/3e3aeab63555
 */
declare function damping(x: number, max: number): number;
declare function getAngleByPoints(lastPoint: {
    x: number;
    y: number;
}, curPoint: {
    x: number;
    y: number;
}): number;

declare function onCreateScrollView(h: (v: ScrollViewCore) => void): void;
type PullToRefreshStep = "pending" | "pulling" | "refreshing" | "releasing";
type PointEvent = {
    touches?: {
        pageX: number;
        pageY: number;
    }[];
    clientX?: number;
    clientY?: number;
    cancelable?: boolean;
    preventDefault?: () => void;
};
type PullToDownOptions = {
    /** 在列表顶部，松手即可触发下拉刷新回调的移动距离 */
    offset: number;
    /**
     * 是否锁定下拉刷新
     * 默认 false
     */
    isLock: boolean;
    /**
     * 当手指 touchmove 位置在距离 body 底部指定范围内的时候结束上拉刷新，避免 Webview 嵌套导致 touchend 事件不执行
     */
    bottomOffset: number;
    /**
     * 向下滑动最少偏移的角度，取值区间[0,90]
     * 默认45度，即向下滑动的角度大于45度则触发下拉。而小于45度，将不触发下拉，避免与左右滑动的轮播等组件冲突
     */
    minAngle: number;
};
declare enum Events$9 {
    InDownOffset = 0,
    OutDownOffset = 1,
    Pulling = 2,
    PullToRefresh = 3,
    PullToRefreshFinished = 4,
    InUpOffset = 5,
    OutUpOffset = 6,
    Scrolling = 7,
    ReachBottom = 8,
    Mounted = 9
}
type TheTypesOfEvents$d = {
    [Events$9.InDownOffset]: void;
    [Events$9.OutDownOffset]: void;
    [Events$9.Pulling]: {
        instance: number;
    };
    [Events$9.PullToRefresh]: void;
    [Events$9.PullToRefreshFinished]: void;
    [Events$9.InUpOffset]: void;
    [Events$9.OutUpOffset]: void;
    [Events$9.Scrolling]: {
        scrollTop: number;
    };
    [Events$9.ReachBottom]: void;
    [Events$9.Mounted]: void;
};
type EnvNeeded = {
    android: boolean;
    pc: boolean;
    ios: boolean;
    wechat: boolean;
};
type ScrollViewProps = {
    os?: EnvNeeded;
    /** 下拉多少距离后刷新 */
    offset?: number;
    disabled?: boolean;
    onScroll?: (pos: {
        scrollTop: number;
    }) => void;
    onReachBottom?: () => void;
    onPullToRefresh?: () => void;
    onPullToBack?: () => void;
};
type ScrollViewCoreState = {
    top: number;
    left: number;
    /** 当前滚动距离顶部的距离 */
    scrollTop: number;
    scrollable: boolean;
    /** 是否支持下拉刷新 */
    pullToRefresh: boolean;
    /** 下拉刷新的阶段 */
    step: PullToRefreshStep;
};
declare class ScrollViewCore extends BaseDomain<TheTypesOfEvents$d> {
    os: EnvNeeded;
    /** 尺寸信息 */
    rect: Partial<{
        /** 宽度 */
        width: number;
        /** 高度 */
        height: number;
        /** 在 y 轴方向滚动的距离 */
        scrollTop: number;
        /** 内容高度 */
        contentHeight: number;
    }>;
    disabled: boolean;
    canPullToRefresh: boolean;
    canReachBottom: boolean;
    /** 隐藏下拉刷新指示器 */
    needHideIndicator: boolean;
    scrollable: boolean;
    /** 下拉刷新相关的状态信息 */
    pullToRefresh: {
        step: PullToRefreshStep;
        /** 开始拖动的起点 y */
        pullStartY: number;
        /** 开始拖动的起点 x */
        pullStartX: number;
        /** 拖动过程中的 y */
        pullMoveY: number;
        /** 拖动过程中的 x */
        pullMoveX: number;
        /** 拖动过程 x 方向上移动的距离 */
        distX: number;
        /** 拖动过程 y 方向上移动的距离 */
        distY: number;
        /** 实际移动的距离？ */
        distResisted: number;
    };
    /** 滚动到底部的阈值 */
    threshold: number;
    options: ScrollViewProps;
    pullToRefreshOptions: PullToDownOptions;
    isPullToRefreshing: boolean;
    isLoadingMore: boolean;
    startPoint: {
        x: number;
        y: number;
    } | null;
    lastPoint: {
        x: number;
        y: number;
    };
    downHight: number;
    upHight: number;
    maxTouchMoveInstanceY: number;
    inTouchEnd: boolean;
    inTopWhenPointDown: boolean;
    inBottomWhenPointDown: boolean;
    isMoveDown: boolean;
    isMoveUp: boolean;
    isScrollTo: boolean;
    /**
     * 为了让 StartPullToRefresh、OutOffset 等事件在拖动过程中仅触发一次的标记
     */
    movetype: PullToRefreshStep;
    preScrollY: number;
    /** 标记上拉已经自动执行过，避免初始化时多次触发上拉回调 */
    isUpAutoLoad: boolean;
    get state(): ScrollViewCoreState;
    constructor(props?: ScrollViewProps);
    setReady(): void;
    setRect(rect: Partial<{
        width: number;
        height: number;
        contentHeight: number;
    }>): void;
    /** 显示下拉进度布局 */
    startPullToRefresh: () => void;
    /** 结束下拉刷新 */
    finishPullToRefresh: () => void;
    disablePullToRefresh: () => void;
    enablePullToRefresh: () => void;
    handleMouseDown: (event: MouseEvent) => void;
    handleMouseMove: (event: MouseEvent) => void;
    handleTouchStart: (event: TouchEvent) => void;
    handleTouchMove: (event: TouchEvent) => void;
    /** 鼠标/手指按下 */
    handlePointDown: (e: PointEvent) => void;
    /** 鼠标/手指移动 */
    handlePointMove: (e: PointEvent) => void;
    handleTouchEnd: () => void;
    handleScrolling: () => void;
    finishLoadingMore(): void;
    setMounted(): void;
    refreshRect(): void;
    setBounce: (isBounce: boolean) => void;
    changeIndicatorHeight(height: number): void;
    setIndicatorHeightTransition(set: boolean): void;
    optimizeScroll(optimize: boolean): void;
    hideIndicator: () => void;
    /**
     * 滑动列表到指定位置
     * 带缓冲效果 (y=0 回到顶部；如果要滚动到底部可以传一个较大的值，比如 99999)
     */
    scrollTo: (position: Partial<{
        left: number;
        top: number;
    }>, duration?: number) => void;
    getToBottom(): number;
    getOffsetTop(dom: unknown): number;
    getScrollHeight(): number;
    /** 获取滚动容器的高度 */
    getScrollClientHeight(): number;
    getScrollTop(): number;
    addScrollTop(difference: number): void;
    setScrollTop(y: number): void;
    getBodyHeight(): number;
    destroy: () => void;
    inDownOffset(handler: Handler<TheTypesOfEvents$d[Events$9.InDownOffset]>): () => void;
    outDownOffset(handler: Handler<TheTypesOfEvents$d[Events$9.OutDownOffset]>): () => void;
    onPulling(handler: Handler<TheTypesOfEvents$d[Events$9.Pulling]>): () => void;
    onScroll(handler: Handler<TheTypesOfEvents$d[Events$9.Scrolling]>): () => void;
    onReachBottom(handler: Handler<TheTypesOfEvents$d[Events$9.ReachBottom]>): () => void;
    onPullToRefresh(handler: Handler<TheTypesOfEvents$d[Events$9.PullToRefresh]>): () => void;
    onMounted(handler: Handler<TheTypesOfEvents$d[Events$9.Mounted]>): () => void;
}

type TheTypesOfEvents$c = {};
type SelectContentProps = {
    $node: () => HTMLElement;
    getStyles: () => CSSStyleDeclaration;
    getRect: () => DOMRect;
};
declare class SelectContentCore extends BaseDomain<TheTypesOfEvents$c> {
    constructor(options?: Partial<{
        _name: string;
    }> & Partial<SelectContentProps>);
    $node(): HTMLElement | null;
    getRect(): DOMRect;
    getStyles(): CSSStyleDeclaration;
    get clientHeight(): number;
}

type TheTypesOfEvents$b = {};
declare class SelectViewportCore extends BaseDomain<TheTypesOfEvents$b> {
    constructor(options?: Partial<{
        _name: string;
        $node: () => HTMLElement;
        getStyles: () => CSSStyleDeclaration;
        getRect: () => DOMRect;
    }>);
    $node(): HTMLElement | null;
    getRect(): DOMRect;
    getStyles(): CSSStyleDeclaration;
    get clientHeight(): number;
    get scrollHeight(): number;
    get offsetTop(): number;
    get offsetHeight(): number;
}

type TheTypesOfEvents$a = {};
declare class SelectTriggerCore extends BaseDomain<TheTypesOfEvents$a> {
    constructor(options?: Partial<{
        name: string;
        $node: () => HTMLElement;
        getStyles: () => CSSStyleDeclaration;
        getRect: () => DOMRect;
    }>);
    $node(): any;
    getRect(): DOMRect;
    getStyles(): CSSStyleDeclaration;
}

type TheTypesOfEvents$9 = {};
declare class SelectWrapCore extends BaseDomain<TheTypesOfEvents$9> {
    constructor(options?: Partial<{
        _name: string;
        $node: () => HTMLElement;
        getStyles: () => CSSStyleDeclaration;
        getRect: () => DOMRect;
    }>);
    $node(): HTMLElement | null;
    getRect(): DOMRect;
    getStyles(): CSSStyleDeclaration;
}

/**
 * @file Select 选项
 */

declare enum Events$8 {
    StateChange = 0,
    Select = 1,
    Leave = 2,
    Enter = 3,
    Move = 4,
    Focus = 5,
    Blur = 6
}
type TheTypesOfEvents$8<T> = {
    [Events$8.StateChange]: SelectItemState<T>;
    [Events$8.Select]: void;
    [Events$8.Leave]: void;
    [Events$8.Enter]: void;
    [Events$8.Focus]: void;
    [Events$8.Blur]: void;
};
type SelectItemState<T> = {
    /** 标志唯一值 */
    value: T | null;
    selected: boolean;
    focused: boolean;
    disabled: boolean;
};
type SelectItemProps<T> = {
    name?: string;
    label: string;
    value: T;
    selected?: boolean;
    focused?: boolean;
    disabled?: boolean;
    $node?: () => HTMLElement;
    getRect?: () => DOMRect;
    getStyles?: () => CSSStyleDeclaration;
};
declare class SelectItemCore<T> extends BaseDomain<TheTypesOfEvents$8<T>> {
    name: string;
    debug: boolean;
    text: string;
    value: T | null;
    selected: boolean;
    focused: boolean;
    disabled: boolean;
    _leave: boolean;
    _enter: boolean;
    get state(): SelectItemState<T>;
    constructor(options: Partial<{
        _name: string;
    }> & SelectItemProps<T>);
    $node(): HTMLElement | null;
    getRect(): DOMRect;
    getStyles(): CSSStyleDeclaration;
    get offsetHeight(): number;
    get offsetTop(): number;
    setText(text: SelectItemCore<T>["text"]): void;
    select(): void;
    unselect(): void;
    focus(): void;
    blur(): void;
    leave(): void;
    move(pos: {
        x: number;
        y: number;
    }): void;
    enter(): void;
    onStateChange(handler: Handler<TheTypesOfEvents$8<T>[Events$8.StateChange]>): () => void;
    onLeave(handler: Handler<TheTypesOfEvents$8<T>[Events$8.Leave]>): () => void;
    onEnter(handler: Handler<TheTypesOfEvents$8<T>[Events$8.Enter]>): () => void;
    onFocus(handler: Handler<TheTypesOfEvents$8<T>[Events$8.Focus]>): () => void;
    onBlur(handler: Handler<TheTypesOfEvents$8<T>[Events$8.Blur]>): () => void;
}

declare function clamp(value: number, [min, max]: [number, number]): number;

declare enum Events$7 {
    StateChange = 0,
    Change = 1,
    Focus = 2,
    Placed = 3
}
type TheTypesOfEvents$7<T> = {
    [Events$7.StateChange]: SelectState<T>;
    [Events$7.Change]: T | null;
    [Events$7.Focus]: void;
    [Events$7.Placed]: void;
};
type SelectProps<T> = {
    defaultValue: T | null;
    placeholder?: string;
    options?: {
        value: T;
        label: string;
    }[];
    onChange?: (v: T | null) => void;
};
type SelectState<T> = {
    options: {
        value: T;
        label: string;
        selected: boolean;
    }[];
    value: T | null;
    value2: {
        value: T;
        label: string;
    } | null;
    /** 菜单是否展开 */
    open: boolean;
    /** 提示 */
    placeholder: string;
    /** 禁用 */
    disabled: boolean;
    /** 是否必填 */
    required: boolean;
    dir: Direction;
    styles: Partial<CSSStyleDeclaration>;
    enter: boolean;
    visible: boolean;
    exit: boolean;
};
declare class SelectCore<T> extends BaseDomain<TheTypesOfEvents$7<T>> {
    shape: "select";
    name: string;
    debug: boolean;
    placeholder: string;
    options: {
        value: T;
        label: string;
        selected: boolean;
    }[];
    defaultValue: T | null;
    value: T | null;
    disabled: boolean;
    open: boolean;
    popper: PopperCore;
    popover: PopoverCore;
    presence: PresenceCore;
    collection: CollectionCore;
    layer: DismissableLayerCore;
    position: "popper" | "item-aligned";
    /** 参考点位置 */
    triggerPos: {
        x: number;
        y: number;
    };
    reference: Rect | null;
    /** 触发按钮 */
    trigger: SelectTriggerCore | null;
    wrap: SelectWrapCore | null;
    /** 下拉列表 */
    content: SelectContentCore | null;
    /** 下拉列表容器 */
    viewport: SelectViewportCore | null;
    /** 选中的 item */
    selectedItem: SelectItemCore<T> | null;
    _findFirstValidItem: boolean;
    get state(): SelectState<T>;
    constructor(props: Partial<{
        _name: string;
    }> & SelectProps<T>);
    mapViewModelWithIndex(index: number): {
        value: T;
        label: string;
        selected: boolean;
    };
    setTriggerPointerDownPos(pos: {
        x: number;
        y: number;
    }): void;
    setTrigger(trigger: SelectTriggerCore): void;
    setWrap(wrap: SelectWrapCore): void;
    setContent(content: SelectContentCore): void;
    setViewport(viewport: SelectViewportCore): void;
    setSelectedItem(item: SelectItemCore<T>): void;
    show(): Promise<void>;
    hide(): void;
    addNativeOption(): void;
    removeNativeOption(): void;
    /** 选择 item */
    select(value: T): void;
    focus(): void;
    setOptions(options: NonNullable<SelectProps<T>["options"]>): void;
    setValue(v: T | null): void;
    clear(): void;
    setPosition(): void;
    onStateChange(handler: Handler<TheTypesOfEvents$7<T>[Events$7.StateChange]>): () => void;
    onValueChange(handler: Handler<TheTypesOfEvents$7<T>[Events$7.Change]>): () => void;
    onChange(handler: Handler<TheTypesOfEvents$7<T>[Events$7.Change]>): () => void;
    onFocus(handler: Handler<TheTypesOfEvents$7<T>[Events$7.Focus]>): () => void;
}
type SelectInListProps<T = unknown> = {
    onChange: (record: T) => void;
} & SelectProps<T>;
type TheTypesInListOfEvents<K extends string, T> = {
    [Events$7.Change]: [K, T | null];
    [Events$7.StateChange]: SelectProps<T>;
};
declare class SelectInListCore<K extends string, T> extends BaseDomain<TheTypesInListOfEvents<K, T>> {
    options: SelectProps<T>["options"];
    list: SelectCore<T>[];
    cached: Map<K, SelectCore<T>>;
    values: Map<K, T | null>;
    constructor(props?: Partial<{
        _name: string;
    } & SelectInListProps<T>>);
    bind(unique_id: K, extra?: {
        defaultValue: T | null;
    }): SelectCore<T>;
    setOptions(options: NonNullable<SelectProps<T>["options"]>): void;
    setValue(v: T | null): void;
    getValue(key: K): T;
    clear(): void;
    toJson<R>(handler: (value: [K, T | null]) => R): R[];
    /** 清空触发点击事件时保存的按钮 */
    onChange(handler: Handler<TheTypesInListOfEvents<K, T>[Events$7.Change]>): void;
    onStateChange(handler: Handler<TheTypesInListOfEvents<K, T>[Events$7.StateChange]>): void;
}

declare enum Events$6 {
    StateChange = 0,
    ValueChange = 1
}
type TheTypesOfEvents$6 = {
    [Events$6.StateChange]: TabsState;
    [Events$6.ValueChange]: string;
};
type TabsState = {
    curValue: string | null;
    orientation: Orientation;
    dir: Direction;
};
declare class TabsCore extends BaseDomain<TheTypesOfEvents$6> {
    roving: RovingFocusCore;
    prevContent: {
        id: number;
        value: string;
        presence: PresenceCore;
    } | null;
    contents: {
        id: number;
        value: string;
        presence: PresenceCore;
    }[];
    state: TabsState;
    constructor(options?: Partial<{
        _name: string;
    }>);
    selectTab(value: string): void;
    appendContent(content: {
        id: number;
        value: string;
        presence: PresenceCore;
    }): void;
    onStateChange(handler: Handler<TheTypesOfEvents$6[Events$6.StateChange]>): void;
    onValueChange(handler: Handler<TheTypesOfEvents$6[Events$6.ValueChange]>): void;
}

/**
 * @file 弹窗核心类
 */

declare enum Events$5 {
    BeforeShow = 0,
    Show = 1,
    BeforeHidden = 2,
    Hidden = 3,
    OpenChange = 4,
    AnimationStart = 5,
    AnimationEnd = 6,
    StateChange = 7
}
type TheTypesOfEvents$5 = {
    [Events$5.BeforeShow]: void;
    [Events$5.Show]: void;
    [Events$5.BeforeHidden]: void;
    [Events$5.Hidden]: void;
    [Events$5.OpenChange]: boolean;
    [Events$5.AnimationStart]: void;
    [Events$5.AnimationEnd]: void;
    [Events$5.StateChange]: ToastState;
};
type ToastProps = {
    delay: number;
};
type ToastState = {
    mask: boolean;
    icon?: unknown;
    texts: string[];
    enter: boolean;
    visible: boolean;
    exit: boolean;
};
declare class ToastCore extends BaseDomain<TheTypesOfEvents$5> {
    name: string;
    present: PresenceCore;
    delay: number;
    timer: NodeJS.Timeout | null;
    open: boolean;
    _mask: boolean;
    _icon: unknown;
    _texts: string[];
    get state(): ToastState;
    constructor(options?: Partial<{
        _name: string;
    } & ToastProps>);
    /** 显示弹窗 */
    show(params: {
        mask?: boolean;
        icon?: unknown;
        texts: string[];
    }): Promise<void>;
    clearTimer(): void;
    /** 隐藏弹窗 */
    hide(): void;
    onShow(handler: Handler<TheTypesOfEvents$5[Events$5.Show]>): () => void;
    onHide(handler: Handler<TheTypesOfEvents$5[Events$5.Hidden]>): () => void;
    onOpenChange(handler: Handler<TheTypesOfEvents$5[Events$5.OpenChange]>): () => void;
    onStateChange(handler: Handler<TheTypesOfEvents$5[Events$5.StateChange]>): () => void;
    get [Symbol.toStringTag](): string;
}

declare enum TARGET_POSITION_TYPE {
    TOP = 1,
    BOTTOM = -1,
    CONTENT = 0
}

type SourceNode = {
    key: string;
    title: string;
    children?: Array<SourceNode>;
};

declare function noop$1(): void;
/**
 * type NodeLevel = string; // like 0、0-0、0-1、0-0-0
 * interface SourceNode {
 *  key: string;
 *  title: string;
 *  children?: Array<SourceNode>;
 * }
 * interface FormattedSourceNode {
 *  key: string;
 *  title: string;
 *  pos: string;
 *  children?: Array<FormattedSourceNode>;
 *  [propsName: string]: any;
 * }
 */
/**
 * add some key to sourceNode
 * @param {Array<SourceNode>} data
 * @param {string} [level='0'] - level at tree
 * @return {Array<FormattedSourceNode>}
 */
declare const formatSourceNodes: (sourceNodes: SourceNode[], level?: number, parentPos?: string) => {
    key: string;
    title: string;
    pos: string;
    children?: Array<SourceNode>;
}[];
/**
 * collect node key and its children keys
 * @param {Array<VueComponent>} treeNodes
 * @param {function} callback
 */
declare function traverseTreeNodes(treeNodes: any[], callback: any): void;
/**
 *
 * @param {*} smallArray
 * @param {*} bigArray
 */
declare function isInclude(smallArray: any, bigArray: any): any;
/**
 * get key and children's key of dragging node
 * @param {VueComponent} treeNode - dragging node
 * @return {Array<>}
 */
declare function getDraggingNodesKey(treeNode: any): any[];
/**
 * get node position info
 * @param {Element} ele
 */
declare function getOffset(ele: any): any;
/**
 * type TargetPositionType = -1 | 0 | 1;
 */
/**
 * @param {Event} e
 * @param {VueComponent} treeNode - entered node
 * @return {TargetPostionType}
 * TARGET_POSITION_TYPE.BOTTOM
 * |TARGET_POSITION_TYPE.CONTENT
 * |TARGET_POSITION_TYPE.TOP
 */
declare function calcDropPosition(e: any, treeNode: any): TARGET_POSITION_TYPE;
interface FindSourceCallback {
    (sourceNode: SourceNode, index: number, arr: Array<SourceNode>): void;
}
/**
 *  interface FindSourceCallback {
 *      (sourceNode: SourceNode, index: number, arr: Array<SourceNode>): void;
 *  }
 */
/**
 * @param {Array<SourceNode>} data
 * @param {string} key
 * @param {FindSourceCallback} callback
 */
declare const findSourceNodeByKey: (sourceNodes: SourceNode[], key: string, callback: FindSourceCallback) => void;
/**
 * get last sourceNodes and move type
 * @param {Array<SourceNode>} sourceNodes
 * @param {any} draggingNodeKey
 * @param {any} targetNodeKey
 * @param {TargetPostionType} targetPosition
 * @return {SourceNode | undefined} targetNode
 * @return {number | undefined} targetNodeIndex
 * @return {Array<SourceNode> | undefined} targetNodes
 * @return {SourceNode} originSourceNode
 * @return {number} originSourceNodeIndex
 * @return {Array<SourceNode>} originSourceNodes
 */
declare function computeMoveNeededParams(sourceNodes: SourceNode[], draggingNodeKey: string, targetNodeKey: string, targetPosition: TARGET_POSITION_TYPE): {
    targetSourceNode: any;
    originSourceNode: any;
    originSourceNodeIndex: any;
    originSourceNodes: any;
    targetSourceNodes?: undefined;
    targetSourceNodeIndex?: undefined;
} | {
    targetSourceNodes: any;
    targetSourceNodeIndex: any;
    originSourceNode: any;
    originSourceNodeIndex: any;
    originSourceNodes: any;
    targetSourceNode?: undefined;
};
/**
 * param reassign, no return
 * @param {number} targetSourceNodeIndex
 * @param {Array<SourceNode>} targetSourceNodes
 * @param {SourceNode} originSourceNode
 * @param {number} originSourceNodeIndex
 * @param {Array<SourceNode>} originSourceNodes
 */
declare function insertToTop(targetSourceNodeIndex: number, targetSourceNodes: SourceNode[], originSourceNode: SourceNode, originSourceNodeIndex: number, originSourceNodes: SourceNode[]): {
    targetSourceNodes: SourceNode[];
    originSourceNodes: SourceNode[];
};
/**
 * param reassign, no return
 * @param {number} targetSourceNodeIndex
 * @param {Array<SourceNode>} targetSourceNodes
 * @param {SourceNode} originSourceNode
 * @param {number} originSourceNodeIndex
 * @param {Array<SourceNode>} originSourceNodes
 */
declare function insertToBottom(targetSourceNodeIndex: number, targetSourceNodes: SourceNode[], originSourceNode: SourceNode, originSourceNodeIndex: number, originSourceNodes: SourceNode[]): {
    targetSourceNodes: SourceNode[];
    originSourceNodes: SourceNode[];
};

declare const utils$1_calcDropPosition: typeof calcDropPosition;
declare const utils$1_computeMoveNeededParams: typeof computeMoveNeededParams;
declare const utils$1_findSourceNodeByKey: typeof findSourceNodeByKey;
declare const utils$1_formatSourceNodes: typeof formatSourceNodes;
declare const utils$1_getDraggingNodesKey: typeof getDraggingNodesKey;
declare const utils$1_getOffset: typeof getOffset;
declare const utils$1_insertToBottom: typeof insertToBottom;
declare const utils$1_insertToTop: typeof insertToTop;
declare const utils$1_isInclude: typeof isInclude;
declare const utils$1_traverseTreeNodes: typeof traverseTreeNodes;
declare namespace utils$1 {
  export {
    utils$1_calcDropPosition as calcDropPosition,
    utils$1_computeMoveNeededParams as computeMoveNeededParams,
    utils$1_findSourceNodeByKey as findSourceNodeByKey,
    utils$1_formatSourceNodes as formatSourceNodes,
    utils$1_getDraggingNodesKey as getDraggingNodesKey,
    utils$1_getOffset as getOffset,
    utils$1_insertToBottom as insertToBottom,
    utils$1_insertToTop as insertToTop,
    utils$1_isInclude as isInclude,
    noop$1 as noop,
    utils$1_traverseTreeNodes as traverseTreeNodes,
  };
}

declare enum Events$4 {
    StateChange = 0
}
type TheTypesOfEvents$4 = {
    [Events$4.StateChange]: TreeState;
};
type TreeState = {};
declare class TreeCore extends BaseDomain<TheTypesOfEvents$4> {
    onStateChange(handler: Handler<TheTypesOfEvents$4[Events$4.StateChange]>): () => void;
}

/**
 * @file 多选
 */

declare enum Events$3 {
    StateChange = 0,
    Change = 1
}
type TheTypesOfEvents$3<T> = {
    [Events$3.StateChange]: CheckboxGroupState<T>;
    [Events$3.Change]: T[];
};
type CheckboxGroupOption<T> = {
    value: T;
    label: string;
    checked?: boolean;
    disabled?: boolean;
};
type CheckboxGroupProps<T> = {
    options?: CheckboxGroupOption<T>[];
    checked?: boolean;
    disabled?: boolean;
    required?: boolean;
    onChange?: (options: T[]) => void;
};
type CheckboxGroupState<T> = Omit<CheckboxGroupProps<T>, "options"> & {
    options: {
        label: string;
        value: T;
        core: CheckboxCore;
    }[];
    values: T[];
    indeterminate: boolean;
};
declare class CheckboxGroupCore<T extends any> extends BaseDomain<TheTypesOfEvents$3<T>> {
    shape: "checkbox-group";
    options: {
        label: string;
        value: T;
        core: CheckboxCore;
    }[];
    disabled: CheckboxGroupProps<T>["disabled"];
    values: T[];
    get indeterminate(): boolean;
    get state(): CheckboxGroupState<T>;
    prevChecked: boolean;
    constructor(props?: {
        _name?: string;
    } & CheckboxGroupProps<T>);
    checkOption(value: T): void;
    uncheckOption(value: T): void;
    reset(): void;
    setOptions(options: CheckboxGroupOption<T>[]): void;
    onChange(handler: Handler<TheTypesOfEvents$3<T>[Events$3.Change]>): () => void;
    onStateChange(handler: Handler<TheTypesOfEvents$3<T>[Events$3.StateChange]>): () => void;
}

/**
 * @name toDate
 * @category Common Helpers
 * @summary Convert the given argument to an instance of Date.
 *
 * @description
 * Convert the given argument to an instance of Date.
 *
 * If the argument is an instance of Date, the function returns its clone.
 *
 * If the argument is a number, it is treated as a timestamp.
 *
 * If the argument is none of the above, the function returns Invalid Date.
 *
 * **Note**: *all* Date arguments passed to any *date-fns* function is processed by `toDate`.
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param argument - The value to convert
 *
 * @returns The parsed date in the local time zone
 *
 * @example
 * // Clone the date:
 * const result = toDate(new Date(2014, 1, 11, 11, 30, 30))
 * //=> Tue Feb 11 2014 11:30:30
 *
 * @example
 * // Convert the timestamp to date:
 * const result = toDate(1392098430000)
 * //=> Tue Feb 11 2014 11:30:30
 */
declare function toDate<DateType extends Date = Date>(argument: DateType | number): DateType;
/**
 * @name constructFrom
 * @category Generic Helpers
 * @summary Constructs a date using the reference date and the value
 *
 * @description
 * The function constructs a new date using the constructor from the reference
 * date and the given value. It helps to build generic functions that accept
 * date extensions.
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param date - The reference date to take constructor from
 * @param value - The value to create the date
 *
 * @returns Date initialized using the given date and value
 *
 * @example
 * import { constructFrom } from 'date-fns'
 *
 * // A function that clones a date preserving the original type
 * function cloneDate<DateType extends Date(date: DateType): DateType {
 *   return constructFrom(
 *     date, // Use contrustor from the given date
 *     date.getTime() // Use the date value to create a new date
 *   )
 * }
 */
declare function constructFrom<DateType extends Date>(date: DateType | number, value: Date | number): DateType;
/**
 * @name startOfMonth
 * @category Month Helpers
 * @summary Return the start of a month for the given date.
 *
 * @description
 * Return the start of a month for the given date.
 * The result will be in the local timezone.
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param date - The original date
 *
 * @returns The start of a month
 *
 * @example
 * // The start of a month for 2 September 2014 11:55:00:
 * const result = startOfMonth(new Date(2014, 8, 2, 11, 55, 0))
 * //=> Mon Sep 01 2014 00:00:00
 */
declare function startOfMonth<DateType extends Date>(date: DateType | number): DateType;
/**
 * @name differenceInCalendarMonths
 * @category Month Helpers
 * @summary Get the number of calendar months between the given dates.
 *
 * @description
 * Get the number of calendar months between the given dates.
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param dateLeft - The later date
 * @param dateRight - The earlier date
 *
 * @returns The number of calendar months
 *
 * @example
 * // How many calendar months are between 31 January 2014 and 1 September 2014?
 * const result = differenceInCalendarMonths(
 *   new Date(2014, 8, 1),
 *   new Date(2014, 0, 31)
 * )
 * //=> 8
 */
declare function differenceInCalendarMonths<DateType extends Date>(dateLeft: DateType | number, dateRight: DateType | number): number;
/**
 * The era:
 * - 0 - Anno Domini (AD)
 * - 1 - Before Christ (BC)
 */
type Era = 0 | 1;
/**
 * The year quarter. Goes from 1 to 4.
 */
type Quarter = 1 | 2 | 3 | 4;
/**
 * The day of the week type alias. Unlike the date (the number of days since
 * the beginning of the month), which begins with 1 and is dynamic (can go up to
 * 28, 30, or 31), the day starts with 0 and static (always ends at 6). Look at
 * it as an index in an array where Sunday is the first element and Saturday
 * is the last.
 */
type Day = 0 | 1 | 2 | 3 | 4 | 5 | 6;
/**
 * The month type alias. Goes from 0 to 11, where 0 is January and 11 is
 * December.
 */
type Month = 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11;
/**
 * The locale object with all functions and data needed to parse and format
 * dates. This is what each locale implements and exports.
 */
interface Locale {
    /** The locale code (ISO 639-1 + optional country code) */
    code: string;
    /** The function to format distance */
    /** The function to relative time */
    /** The object with functions used to localize various values */
    /** The object with functions that return localized formats */
    /** The object with functions used to match and parse various localized values */
    /** An object with locale options */
    options?: LocaleOptions;
}
/**
 * The week function options. Used to build function options.
 */
interface WeekOptions {
    /** Which day the week starts on. */
    weekStartsOn?: Day;
}
/**
 * FirstWeekContainsDate is used to determine which week is the first week of
 * the year, based on what day the January, 1 is in that week.
 *
 * The day in that week can only be 1 (Monday) or 4 (Thursday).
 *
 * Please see https://en.wikipedia.org/wiki/Week#Week_numbering for more information.
 */
type FirstWeekContainsDate = 1 | 4;
/**
 * The first week contains date options. Used to build function options.
 */
interface FirstWeekContainsDateOptions {
    /** See {@link FirstWeekContainsDate} for more details. */
    firstWeekContainsDate?: FirstWeekContainsDate;
}
/**
 * The locale options.
 */
interface LocaleOptions extends WeekOptions, FirstWeekContainsDateOptions {
}
/** Represents a week in the month.*/
type MonthWeek = {
    /** The week number from the start of the year. */
    weekNumber: number;
    /** The dates in the week. */
    dates: Date[];
};
/** Return the weeks between two dates.  */
declare function daysToMonthWeeks(fromDate: Date, toDate: Date, options?: {
    ISOWeek?: boolean;
    locale?: Locale;
    weekStartsOn?: 0 | 1 | 2 | 3 | 4 | 5 | 6;
    firstWeekContainsDate?: 1 | 2 | 3 | 4 | 5 | 6 | 7;
}): MonthWeek[];
/**
 * The {@link getWeek} function options.
 */
interface GetWeekOptions extends LocalizedOptions<"options">, WeekOptions, FirstWeekContainsDateOptions {
}
/**
 * @name getWeek
 * @category Week Helpers
 * @summary Get the local week index of the given date.
 *
 * @description
 * Get the local week index of the given date.
 * The exact calculation depends on the values of
 * `options.weekStartsOn` (which is the index of the first day of the week)
 * and `options.firstWeekContainsDate` (which is the day of January, which is always in
 * the first week of the week-numbering year)
 *
 * Week numbering: https://en.wikipedia.org/wiki/Week#Week_numbering
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param date - The given date
 * @param options - An object with options
 *
 * @returns The week
 *
 * @example
 * // Which week of the local week numbering year is 2 January 2005 with default options?
 * const result = getWeek(new Date(2005, 0, 2))
 * //=> 2
 *
 * @example
 * // Which week of the local week numbering year is 2 January 2005,
 * // if Monday is the first day of the week,
 * // and the first week of the year always contains 4 January?
 * const result = getWeek(new Date(2005, 0, 2), {
 *   weekStartsOn: 1,
 *   firstWeekContainsDate: 4
 * })
 * //=> 53
 */
declare function getWeek<DateType extends Date>(date: DateType | number, options?: GetWeekOptions): number;
/**
 * The {@link startOfWeekYear} function options.
 */
interface StartOfWeekYearOptions extends LocalizedOptions<"options">, FirstWeekContainsDateOptions, WeekOptions {
}
/**
 * @name startOfWeekYear
 * @category Week-Numbering Year Helpers
 * @summary Return the start of a local week-numbering year for the given date.
 *
 * @description
 * Return the start of a local week-numbering year.
 * The exact calculation depends on the values of
 * `options.weekStartsOn` (which is the index of the first day of the week)
 * and `options.firstWeekContainsDate` (which is the day of January, which is always in
 * the first week of the week-numbering year)
 *
 * Week numbering: https://en.wikipedia.org/wiki/Week#Week_numbering
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param date - The original date
 * @param options - An object with options
 *
 * @returns The start of a week-numbering year
 *
 * @example
 * // The start of an a week-numbering year for 2 July 2005 with default settings:
 * const result = startOfWeekYear(new Date(2005, 6, 2))
 * //=> Sun Dec 26 2004 00:00:00
 *
 * @example
 * // The start of a week-numbering year for 2 July 2005
 * // if Monday is the first day of week
 * // and 4 January is always in the first week of the year:
 * const result = startOfWeekYear(new Date(2005, 6, 2), {
 *   weekStartsOn: 1,
 *   firstWeekContainsDate: 4
 * })
 * //=> Mon Jan 03 2005 00:00:00
 */
declare function startOfWeekYear<DateType extends Date>(date: DateType | number, options?: StartOfWeekYearOptions): DateType;
/**
 * The {@link getWeekYear} function options.
 */
interface GetWeekYearOptions extends LocalizedOptions<"options">, WeekOptions, FirstWeekContainsDateOptions {
}
/**
 * @name getWeekYear
 * @category Week-Numbering Year Helpers
 * @summary Get the local week-numbering year of the given date.
 *
 * @description
 * Get the local week-numbering year of the given date.
 * The exact calculation depends on the values of
 * `options.weekStartsOn` (which is the index of the first day of the week)
 * and `options.firstWeekContainsDate` (which is the day of January, which is always in
 * the first week of the week-numbering year)
 *
 * Week numbering: https://en.wikipedia.org/wiki/Week#Week_numbering
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param date - The given date
 * @param options - An object with options.
 *
 * @returns The local week-numbering year
 *
 * @example
 * // Which week numbering year is 26 December 2004 with the default settings?
 * const result = getWeekYear(new Date(2004, 11, 26))
 * //=> 2005
 *
 * @example
 * // Which week numbering year is 26 December 2004 if week starts on Saturday?
 * const result = getWeekYear(new Date(2004, 11, 26), { weekStartsOn: 6 })
 * //=> 2004
 *
 * @example
 * // Which week numbering year is 26 December 2004 if the first week contains 4 January?
 * const result = getWeekYear(new Date(2004, 11, 26), { firstWeekContainsDate: 4 })
 * //=> 2004
 */
declare function getWeekYear<DateType extends Date>(date: DateType | number, options?: GetWeekYearOptions): number;
/**
 * @name startOfISOWeek
 * @category ISO Week Helpers
 * @summary Return the start of an ISO week for the given date.
 *
 * @description
 * Return the start of an ISO week for the given date.
 * The result will be in the local timezone.
 *
 * ISO week-numbering year: http://en.wikipedia.org/wiki/ISO_week_date
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param date - The original date
 *
 * @returns The start of an ISO week
 *
 * @example
 * // The start of an ISO week for 2 September 2014 11:55:00:
 * const result = startOfISOWeek(new Date(2014, 8, 2, 11, 55, 0))
 * //=> Mon Sep 01 2014 00:00:00
 */
declare function startOfISOWeek<DateType extends Date>(date: DateType | number): DateType;
/**
 * @name getISOWeek
 * @category ISO Week Helpers
 * @summary Get the ISO week of the given date.
 *
 * @description
 * Get the ISO week of the given date.
 *
 * ISO week-numbering year: http://en.wikipedia.org/wiki/ISO_week_date
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param date - The given date
 *
 * @returns The ISO week
 *
 * @example
 * // Which week of the ISO-week numbering year is 2 January 2005?
 * const result = getISOWeek(new Date(2005, 0, 2))
 * //=> 53
 */
declare function getISOWeek<DateType extends Date>(date: DateType | number): number;
/**
 * @name startOfISOWeekYear
 * @category ISO Week-Numbering Year Helpers
 * @summary Return the start of an ISO week-numbering year for the given date.
 *
 * @description
 * Return the start of an ISO week-numbering year,
 * which always starts 3 days before the year's first Thursday.
 * The result will be in the local timezone.
 *
 * ISO week-numbering year: http://en.wikipedia.org/wiki/ISO_week_date
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param date - The original date
 *
 * @returns The start of an ISO week-numbering year
 *
 * @example
 * // The start of an ISO week-numbering year for 2 July 2005:
 * const result = startOfISOWeekYear(new Date(2005, 6, 2))
 * //=> Mon Jan 03 2005 00:00:00
 */
declare function startOfISOWeekYear<DateType extends Date>(date: DateType | number): DateType;
/**
 * @name getISOWeekYear
 * @category ISO Week-Numbering Year Helpers
 * @summary Get the ISO week-numbering year of the given date.
 *
 * @description
 * Get the ISO week-numbering year of the given date,
 * which always starts 3 days before the year's first Thursday.
 *
 * ISO week-numbering year: http://en.wikipedia.org/wiki/ISO_week_date
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param date - The given date
 *
 * @returns The ISO week-numbering year
 *
 * @example
 * // Which ISO-week numbering year is 2 January 2005?
 * const result = getISOWeekYear(new Date(2005, 0, 2))
 * //=> 2004
 */
declare function getISOWeekYear<DateType extends Date>(date: DateType | number): number;
/**
 * @name endOfISOWeek
 * @category ISO Week Helpers
 * @summary Return the end of an ISO week for the given date.
 *
 * @description
 * Return the end of an ISO week for the given date.
 * The result will be in the local timezone.
 *
 * ISO week-numbering year: http://en.wikipedia.org/wiki/ISO_week_date
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param date - The original date
 *
 * @returns The end of an ISO week
 *
 * @example
 * // The end of an ISO week for 2 September 2014 11:55:00:
 * const result = endOfISOWeek(new Date(2014, 8, 2, 11, 55, 0))
 * //=> Sun Sep 07 2014 23:59:59.999
 */
declare function endOfISOWeek<DateType extends Date>(date: DateType | number): DateType;
/**
 * The {@link endOfWeek} function options.
 */
interface EndOfWeekOptions extends WeekOptions, LocalizedOptions<"options"> {
}
/**
 * @name endOfWeek
 * @category Week Helpers
 * @summary Return the end of a week for the given date.
 *
 * @description
 * Return the end of a week for the given date.
 * The result will be in the local timezone.
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param date - The original date
 * @param options - An object with options
 *
 * @returns The end of a week
 *
 * @example
 * // The end of a week for 2 September 2014 11:55:00:
 * const result = endOfWeek(new Date(2014, 8, 2, 11, 55, 0))
 * //=> Sat Sep 06 2014 23:59:59.999
 *
 * @example
 * // If the week starts on Monday, the end of the week for 2 September 2014 11:55:00:
 * const result = endOfWeek(new Date(2014, 8, 2, 11, 55, 0), { weekStartsOn: 1 })
 * //=> Sun Sep 07 2014 23:59:59.999
 */
declare function endOfWeek<DateType extends Date>(date: DateType | number, options?: EndOfWeekOptions): DateType;
/**
 * Return the weeks belonging to the given month, adding the "outside days" to
 * the first and last week.
 */
declare function getMonthWeeks(month: Date, options: {
    locale: Locale;
    useFixedWeeks?: boolean;
    weekStartsOn?: 0 | 1 | 2 | 3 | 4 | 5 | 6;
    firstWeekContainsDate?: 1 | 2 | 3 | 4 | 5 | 6 | 7;
    ISOWeek?: boolean;
}): MonthWeek[];
/**
 * The {@link getWeeksInMonth} function options.
 */
interface GetWeeksInMonthOptions extends LocalizedOptions<"options">, WeekOptions {
}
/**
 * @name getWeeksInMonth
 * @category Week Helpers
 * @summary Get the number of calendar weeks a month spans.
 *
 * @description
 * Get the number of calendar weeks the month in the given date spans.
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param date - The given date
 * @param options - An object with options.
 *
 * @returns The number of calendar weeks
 *
 * @example
 * // How many calendar weeks does February 2015 span?
 * const result = getWeeksInMonth(new Date(2015, 1, 8))
 * //=> 4
 *
 * @example
 * // If the week starts on Monday,
 * // how many calendar weeks does July 2017 span?
 * const result = getWeeksInMonth(new Date(2017, 6, 5), { weekStartsOn: 1 })
 * //=> 6
 */
declare function getWeeksInMonth<DateType extends Date>(date: DateType | number, options?: GetWeeksInMonthOptions): number;
/**
 * The {@link differenceInCalendarWeeks} function options.
 */
interface DifferenceInCalendarWeeksOptions extends LocalizedOptions<"options">, WeekOptions {
}
/**
 * @name differenceInCalendarWeeks
 * @category Week Helpers
 * @summary Get the number of calendar weeks between the given dates.
 *
 * @description
 * Get the number of calendar weeks between the given dates.
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param dateLeft - The later date
 * @param dateRight - The earlier date
 * @param options - An object with options.
 *
 * @returns The number of calendar weeks
 *
 * @example
 * // How many calendar weeks are between 5 July 2014 and 20 July 2014?
 * const result = differenceInCalendarWeeks(
 *   new Date(2014, 6, 20),
 *   new Date(2014, 6, 5)
 * )
 * //=> 3
 *
 * @example
 * // If the week starts on Monday,
 * // how many calendar weeks are between 5 July 2014 and 20 July 2014?
 * const result = differenceInCalendarWeeks(
 *   new Date(2014, 6, 20),
 *   new Date(2014, 6, 5),
 *   { weekStartsOn: 1 }
 * )
 * //=> 2
 */
declare function differenceInCalendarWeeks<DateType extends Date>(dateLeft: DateType | number, dateRight: DateType | number, options?: DifferenceInCalendarWeeksOptions): number;
/**
 * The {@link startOfWeek} function options.
 */
interface StartOfWeekOptions extends LocalizedOptions<"options">, WeekOptions {
}
/**
 * @name startOfWeek
 * @category Week Helpers
 * @summary Return the start of a week for the given date.
 *
 * @description
 * Return the start of a week for the given date.
 * The result will be in the local timezone.
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param date - The original date
 * @param options - An object with options
 *
 * @returns The start of a week
 *
 * @example
 * // The start of a week for 2 September 2014 11:55:00:
 * const result = startOfWeek(new Date(2014, 8, 2, 11, 55, 0))
 * //=> Sun Aug 31 2014 00:00:00
 *
 * @example
 * // If the week starts on Monday, the start of the week for 2 September 2014 11:55:00:
 * const result = startOfWeek(new Date(2014, 8, 2, 11, 55, 0), { weekStartsOn: 1 })
 * //=> Mon Sep 01 2014 00:00:00
 */
declare function startOfWeek<DateType extends Date>(date: DateType | number, options?: StartOfWeekOptions): DateType;
/**
 * The localized function options. Used to build function options.
 */
interface LocalizedOptions<LocaleFields extends keyof Locale> {
    /** The locale to use in the function. */
    locale?: Pick<Locale, LocaleFields>;
}
type DefaultOptions = LocalizedOptions<keyof Locale> & WeekOptions & FirstWeekContainsDateOptions;
declare function getDefaultOptions(): DefaultOptions;
/**
 * @name lastDayOfMonth
 * @category Month Helpers
 * @summary Return the last day of a month for the given date.
 *
 * @description
 * Return the last day of a month for the given date.
 * The result will be in the local timezone.
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param date - The original date
 *
 * @returns The last day of a month
 *
 * @example
 * // The last day of a month for 2 September 2014 11:55:00:
 * const result = lastDayOfMonth(new Date(2014, 8, 2, 11, 55, 0))
 * //=> Tue Sep 30 2014 00:00:00
 */
declare function lastDayOfMonth<DateType extends Date>(date: DateType | number): DateType;
/**
 * @name addWeeks
 * @category Week Helpers
 * @summary Add the specified number of weeks to the given date.
 *
 * @description
 * Add the specified number of week to the given date.
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param date - The date to be changed
 * @param amount - The amount of weeks to be added. Positive decimals will be rounded using `Math.floor`, decimals less than zero will be rounded using `Math.ceil`.
 *
 * @returns The new date with the weeks added
 *
 * @example
 * // Add 4 weeks to 1 September 2014:
 * const result = addWeeks(new Date(2014, 8, 1), 4)
 * //=> Mon Sep 29 2014 00:00:00
 */
declare function addWeeks<DateType extends Date>(date: DateType | number, amount: number): DateType;
/**
 * @name addDays
 * @category Day Helpers
 * @summary Add the specified number of days to the given date.
 *
 * @description
 * Add the specified number of days to the given date.
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param date - The date to be changed
 * @param amount - The amount of days to be added. Positive decimals will be rounded using `Math.floor`, decimals less than zero will be rounded using `Math.ceil`.
 *
 * @returns The new date with the days added
 *
 * @example
 * // Add 10 days to 1 September 2014:
 * const result = addDays(new Date(2014, 8, 1), 10)
 * //=> Thu Sep 11 2014 00:00:00
 */
declare function addDays<DateType extends Date>(date: DateType | number, amount: number): DateType;
/**
 * @name endOfMonth
 * @category Month Helpers
 * @summary Return the end of a month for the given date.
 *
 * @description
 * Return the end of a month for the given date.
 * The result will be in the local timezone.
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param date - The original date
 *
 * @returns The end of a month
 *
 * @example
 * // The end of a month for 2 September 2014 11:55:00:
 * const result = endOfMonth(new Date(2014, 8, 2, 11, 55, 0))
 * //=> Tue Sep 30 2014 23:59:59.999
 */
declare function endOfMonth<DateType extends Date>(date: DateType | number): DateType;
/**
 * @name differenceInCalendarDays
 * @category Day Helpers
 * @summary Get the number of calendar days between the given dates.
 *
 * @description
 * Get the number of calendar days between the given dates. This means that the times are removed
 * from the dates and then the difference in days is calculated.
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param dateLeft - The later date
 * @param dateRight - The earlier date
 *
 * @returns The number of calendar days
 *
 * @example
 * // How many calendar days are between
 * // 2 July 2011 23:00:00 and 2 July 2012 00:00:00?
 * const result = differenceInCalendarDays(
 *   new Date(2012, 6, 2, 0, 0),
 *   new Date(2011, 6, 2, 23, 0)
 * )
 * //=> 366
 * // How many calendar days are between
 * // 2 July 2011 23:59:00 and 3 July 2011 00:01:00?
 * const result = differenceInCalendarDays(
 *   new Date(2011, 6, 3, 0, 1),
 *   new Date(2011, 6, 2, 23, 59)
 * )
 * //=> 1
 */
declare function differenceInCalendarDays<DateType extends Date>(dateLeft: DateType | number, dateRight: DateType | number): number;
/**
 * Google Chrome as of 67.0.3396.87 introduced timezones with offset that includes seconds.
 * They usually appear for dates that denote time before the timezones were introduced
 * (e.g. for 'Europe/Prague' timezone the offset is GMT+00:57:44 before 1 October 1891
 * and GMT+01:00:00 after that date)
 *
 * Date#getTimezoneOffset returns the offset in minutes and would return 57 for the example above,
 * which would lead to incorrect calculations.
 *
 * This function returns the timezone offset in milliseconds that takes seconds in account.
 */
declare function getTimezoneOffsetInMilliseconds(date: Date): number;
/**
 * @name startOfDay
 * @category Day Helpers
 * @summary Return the start of a day for the given date.
 *
 * @description
 * Return the start of a day for the given date.
 * The result will be in the local timezone.
 *
 * @typeParam DateType - The `Date` type, the function operates on. Gets inferred from passed arguments. Allows to use extensions like [`UTCDate`](https://github.com/date-fns/utc).
 *
 * @param date - The original date
 *
 * @returns The start of a day
 *
 * @example
 * // The start of a day for 2 September 2014 11:55:00:
 * const result = startOfDay(new Date(2014, 8, 2, 11, 55, 0))
 * //=> Tue Sep 02 2014 00:00:00
 */
declare function startOfDay<DateType extends Date>(date: DateType | number): DateType;
/**
 * @constant
 * @name millisecondsInDay
 * @summary Milliseconds in 1 day.
 */
declare const millisecondsInDay = 86400000;
/**
 * @constant
 * @name millisecondsInWeek
 * @summary Milliseconds in 1 week.
 */
declare const millisecondsInWeek = 604800000;

type CalendarWeek = {
    id: number;
    dates: {
        id: number;
        text: string;
        yyyy: string;
        value: Date;
        time: number;
        is_prev_month: boolean;
        is_next_month: boolean;
        is_today: boolean;
    }[];
};
type CalendarCoreProps = {
    today: Date;
};
declare function CalendarCore(props: CalendarCoreProps): {
    state: {
        readonly day: {
            text: string;
            value: Date;
            time: number;
        };
        readonly month: {
            /** 12月 */
            text: string;
            value: Date;
            time: number;
        };
        readonly year: {
            text: number;
            value: Date;
            time: number;
        };
        readonly weekdays: {
            id: number;
            text: string;
            yyyy: string;
            value: Date;
            time: number;
            is_prev_month: boolean;
            is_next_month: boolean;
            is_today: boolean;
        }[];
        readonly weeks: CalendarWeek[];
        readonly selectedDay: {
            text: string;
            value: Date;
            time: number;
        };
    };
    readonly value: Date;
    selectDay(day: Date): void;
    nextMonth(): void;
    prevMonth(): void;
    buildMonthText: (d: Date) => string;
    onSelectDay(handler: Handler<Date>): () => void;
    onChange(handler: Handler<{
        readonly day: {
            text: string;
            value: Date;
            time: number;
        };
        readonly month: {
            /** 12月 */
            text: string;
            value: Date;
            time: number;
        };
        readonly year: {
            text: number;
            value: Date;
            time: number;
        };
        readonly weekdays: {
            id: number;
            text: string;
            yyyy: string;
            value: Date;
            time: number;
            is_prev_month: boolean;
            is_next_month: boolean;
            is_today: boolean;
        }[];
        readonly weeks: CalendarWeek[];
        readonly selectedDay: {
            text: string;
            value: Date;
            time: number;
        };
    }>): () => void;
};
type CalendarCore = ReturnType<typeof CalendarCore>;

type index_ArrayFieldCore<T extends (count: number) => SingleFieldCore<any> | ArrayFieldCore<any> | ObjectFieldCore<any>> = ArrayFieldCore<T>;
declare const index_ArrayFieldCore: typeof ArrayFieldCore;
type index_ButtonCore<T = unknown> = ButtonCore<T>;
declare const index_ButtonCore: typeof ButtonCore;
type index_ButtonInListCore<T> = ButtonInListCore<T>;
declare const index_ButtonInListCore: typeof ButtonInListCore;
type index_CalendarCore = CalendarCore;
type index_CheckboxCore = CheckboxCore;
declare const index_CheckboxCore: typeof CheckboxCore;
type index_CheckboxGroupCore<T extends any> = CheckboxGroupCore<T>;
declare const index_CheckboxGroupCore: typeof CheckboxGroupCore;
type index_ContextMenuCore = ContextMenuCore;
declare const index_ContextMenuCore: typeof ContextMenuCore;
type index_Day = Day;
type index_DefaultOptions = DefaultOptions;
type index_DialogCore = DialogCore;
declare const index_DialogCore: typeof DialogCore;
type index_DialogProps = DialogProps;
type index_DifferenceInCalendarWeeksOptions = DifferenceInCalendarWeeksOptions;
type index_Direction = Direction;
type index_DismissableLayerCore = DismissableLayerCore;
declare const index_DismissableLayerCore: typeof DismissableLayerCore;
type index_DropdownMenuCore = DropdownMenuCore;
declare const index_DropdownMenuCore: typeof DropdownMenuCore;
type index_EndOfWeekOptions = EndOfWeekOptions;
type index_Era = Era;
type index_FirstWeekContainsDate = FirstWeekContainsDate;
type index_FocusScopeCore = FocusScopeCore;
declare const index_FocusScopeCore: typeof FocusScopeCore;
type index_FormCore<F extends Record<string, FormFieldCore<{
    label: string;
    name: string;
    input: ValueInputInterface<any>;
}>> = {}> = FormCore<F>;
type index_GetWeekOptions = GetWeekOptions;
type index_GetWeekYearOptions = GetWeekYearOptions;
type index_GetWeeksInMonthOptions = GetWeeksInMonthOptions;
type index_ImageCore = ImageCore;
declare const index_ImageCore: typeof ImageCore;
type index_ImageInListCore = ImageInListCore;
declare const index_ImageInListCore: typeof ImageInListCore;
type index_ImageStep = ImageStep;
declare const index_ImageStep: typeof ImageStep;
type index_InputCore<T> = InputCore<T>;
declare const index_InputCore: typeof InputCore;
type index_InputInListCore<K extends string, T> = InputInListCore<K, T>;
declare const index_InputInListCore: typeof InputInListCore;
type index_InputProps<T> = InputProps<T>;
type index_Locale = Locale;
type index_LocalizedOptions<LocaleFields extends keyof Locale> = LocalizedOptions<LocaleFields>;
type index_MenuCore = MenuCore;
declare const index_MenuCore: typeof MenuCore;
type index_MenuItemCore = MenuItemCore;
declare const index_MenuItemCore: typeof MenuItemCore;
type index_Month = Month;
type index_NodeCore = NodeCore;
declare const index_NodeCore: typeof NodeCore;
type index_ObjectFieldCore<T extends Record<string, SingleFieldCore<any> | ArrayFieldCore<any> | ObjectFieldCore<any>>> = ObjectFieldCore<T>;
declare const index_ObjectFieldCore: typeof ObjectFieldCore;
type index_Orientation = Orientation;
type index_PointEvent = PointEvent;
type index_PopoverCore = PopoverCore;
declare const index_PopoverCore: typeof PopoverCore;
type index_PresenceCore = PresenceCore;
declare const index_PresenceCore: typeof PresenceCore;
type index_ProgressCore = ProgressCore;
declare const index_ProgressCore: typeof ProgressCore;
type index_Quarter = Quarter;
type index_RovingFocusCore = RovingFocusCore;
declare const index_RovingFocusCore: typeof RovingFocusCore;
type index_ScrollViewCore = ScrollViewCore;
declare const index_ScrollViewCore: typeof ScrollViewCore;
type index_ScrollViewProps = ScrollViewProps;
type index_SelectCore<T> = SelectCore<T>;
declare const index_SelectCore: typeof SelectCore;
type index_SelectInListCore<K extends string, T> = SelectInListCore<K, T>;
declare const index_SelectInListCore: typeof SelectInListCore;
type index_SingleFieldCore<T extends FormInputInterface<any>> = SingleFieldCore<T>;
declare const index_SingleFieldCore: typeof SingleFieldCore;
type index_StartOfWeekOptions = StartOfWeekOptions;
type index_StartOfWeekYearOptions = StartOfWeekYearOptions;
type index_TabsCore = TabsCore;
declare const index_TabsCore: typeof TabsCore;
type index_ToastCore = ToastCore;
declare const index_ToastCore: typeof ToastCore;
type index_TreeCore = TreeCore;
declare const index_TreeCore: typeof TreeCore;
declare const index_addDays: typeof addDays;
declare const index_addWeeks: typeof addWeeks;
declare const index_clamp: typeof clamp;
declare const index_constructFrom: typeof constructFrom;
declare const index_damping: typeof damping;
declare const index_daysToMonthWeeks: typeof daysToMonthWeeks;
declare const index_differenceInCalendarDays: typeof differenceInCalendarDays;
declare const index_differenceInCalendarMonths: typeof differenceInCalendarMonths;
declare const index_differenceInCalendarWeeks: typeof differenceInCalendarWeeks;
declare const index_endOfISOWeek: typeof endOfISOWeek;
declare const index_endOfMonth: typeof endOfMonth;
declare const index_endOfWeek: typeof endOfWeek;
declare const index_getAngleByPoints: typeof getAngleByPoints;
declare const index_getDefaultOptions: typeof getDefaultOptions;
declare const index_getISOWeek: typeof getISOWeek;
declare const index_getISOWeekYear: typeof getISOWeekYear;
declare const index_getMonthWeeks: typeof getMonthWeeks;
declare const index_getPoint: typeof getPoint;
declare const index_getTimezoneOffsetInMilliseconds: typeof getTimezoneOffsetInMilliseconds;
declare const index_getWeek: typeof getWeek;
declare const index_getWeekYear: typeof getWeekYear;
declare const index_getWeeksInMonth: typeof getWeeksInMonth;
declare const index_lastDayOfMonth: typeof lastDayOfMonth;
declare const index_millisecondsInDay: typeof millisecondsInDay;
declare const index_millisecondsInWeek: typeof millisecondsInWeek;
declare const index_onCreateScrollView: typeof onCreateScrollView;
declare const index_preventDefault: typeof preventDefault;
declare const index_startOfDay: typeof startOfDay;
declare const index_startOfISOWeek: typeof startOfISOWeek;
declare const index_startOfISOWeekYear: typeof startOfISOWeekYear;
declare const index_startOfMonth: typeof startOfMonth;
declare const index_startOfWeek: typeof startOfWeek;
declare const index_startOfWeekYear: typeof startOfWeekYear;
declare const index_toDate: typeof toDate;
declare namespace index {
  export { index_ArrayFieldCore as ArrayFieldCore, index_ButtonCore as ButtonCore, index_ButtonInListCore as ButtonInListCore, index_CheckboxCore as CheckboxCore, index_CheckboxGroupCore as CheckboxGroupCore, index_ContextMenuCore as ContextMenuCore, index_DialogCore as DialogCore, index_DismissableLayerCore as DismissableLayerCore, index_DropdownMenuCore as DropdownMenuCore, index_FocusScopeCore as FocusScopeCore, index_ImageCore as ImageCore, index_ImageInListCore as ImageInListCore, index_ImageStep as ImageStep, index_InputCore as InputCore, index_InputInListCore as InputInListCore, index_MenuCore as MenuCore, index_MenuItemCore as MenuItemCore, index_NodeCore as NodeCore, index_ObjectFieldCore as ObjectFieldCore, index_PopoverCore as PopoverCore, index_PresenceCore as PresenceCore, index_ProgressCore as ProgressCore, index_RovingFocusCore as RovingFocusCore, index_ScrollViewCore as ScrollViewCore, index_SelectCore as SelectCore, index_SelectInListCore as SelectInListCore, index_SingleFieldCore as SingleFieldCore, index_TabsCore as TabsCore, index_ToastCore as ToastCore, index_TreeCore as TreeCore, utils$1 as Utils, index_addDays as addDays, index_addWeeks as addWeeks, index_clamp as clamp, index_constructFrom as constructFrom, index_damping as damping, index_daysToMonthWeeks as daysToMonthWeeks, index_differenceInCalendarDays as differenceInCalendarDays, index_differenceInCalendarMonths as differenceInCalendarMonths, index_differenceInCalendarWeeks as differenceInCalendarWeeks, index_endOfISOWeek as endOfISOWeek, index_endOfMonth as endOfMonth, index_endOfWeek as endOfWeek, index_getAngleByPoints as getAngleByPoints, index_getDefaultOptions as getDefaultOptions, index_getISOWeek as getISOWeek, index_getISOWeekYear as getISOWeekYear, index_getMonthWeeks as getMonthWeeks, index_getPoint as getPoint, index_getTimezoneOffsetInMilliseconds as getTimezoneOffsetInMilliseconds, index_getWeek as getWeek, index_getWeekYear as getWeekYear, index_getWeeksInMonth as getWeeksInMonth, index_lastDayOfMonth as lastDayOfMonth, index_millisecondsInDay as millisecondsInDay, index_millisecondsInWeek as millisecondsInWeek, index_onCreateScrollView as onCreateScrollView, index_preventDefault as preventDefault, index_startOfDay as startOfDay, index_startOfISOWeek as startOfISOWeek, index_startOfISOWeekYear as startOfISOWeekYear, index_startOfMonth as startOfMonth, index_startOfWeek as startOfWeek, index_startOfWeekYear as startOfWeekYear, index_toDate as toDate };
  export type { index_CalendarCore as CalendarCore, index_Day as Day, index_DefaultOptions as DefaultOptions, index_DialogProps as DialogProps, index_DifferenceInCalendarWeeksOptions as DifferenceInCalendarWeeksOptions, index_Direction as Direction, index_EndOfWeekOptions as EndOfWeekOptions, index_Era as Era, index_FirstWeekContainsDate as FirstWeekContainsDate, index_FormCore as FormCore, index_GetWeekOptions as GetWeekOptions, index_GetWeekYearOptions as GetWeekYearOptions, index_GetWeeksInMonthOptions as GetWeeksInMonthOptions, index_InputProps as InputProps, index_Locale as Locale, index_LocalizedOptions as LocalizedOptions, index_Month as Month, index_Orientation as Orientation, index_PointEvent as PointEvent, index_Quarter as Quarter, index_ScrollViewProps as ScrollViewProps, index_StartOfWeekOptions as StartOfWeekOptions, index_StartOfWeekYearOptions as StartOfWeekYearOptions };
}

/**
 * @file 构建 http 请求载荷
 */

type RequestPayload<T> = {
    hostname?: string;
    url: string;
    method: "POST" | "GET" | "DELETE" | "PUT";
    query?: any;
    params?: any;
    body?: any;
    headers?: Record<string, string | number>;
    process?: (v: any) => T;
};
/**
 * GetRespTypeFromRequestPayload
 * T extends RequestPayload
 */
type UnpackedRequestPayload<T> = NonNullable<T extends RequestPayload<infer U> ? (U extends null ? U : U) : T>;
type RequestedResource<T extends (...args: any[]) => any> = UnpackedResult<Unpacked<ReturnType<T>>>;
type TmpRequestResp<T extends (...args: any[]) => any> = Result<UnpackedRequestPayload<RequestedResource<T>>>;
declare function onCreatePostPayload(h: (v: RequestPayload<any>) => void): void;
declare function onCreateGetPayload(h: (v: RequestPayload<any>) => void): void;
/**
 * 并不是真正发出网络请求，仅仅是「构建请求信息」然后交给 HttpClient 发出请求
 * 所以这里构建的请求信息，就要包含
 * 1. 请求地址
 * 2. 请求参数
 * 3. headers
 */
declare const request: {
    get<T>(endpoint: string, query?: Record<string, string | number | boolean | null | undefined>, extra?: Partial<{
        headers: Record<string, string | number>;
    }>): RequestPayload<T>;
    /** 构建请求参数 */
    post<T>(url: unknown, body?: any, extra?: Partial<{
        headers: Record<string, string | number>;
    }>): RequestPayload<T>;
};
declare function request_factory(opt?: Partial<{
    hostnames: Partial<{
        dev: string;
        test: string;
        beta: string;
        prod: string;
    }>;
    debug: boolean;
    headers: Record<string, string | number>;
    process: (v: any) => any;
}>): {
    getHostname(): string;
    setHostname(hostname: string): void;
    setHeaders(headers: Record<string, string | number>): void;
    deleteHeaders(key: string): void;
    appendHeaders(extra: Record<string, string | number>): void;
    setEnv(env: "dev" | "test" | "beta" | "prod"): void;
    setDebug(debug: boolean): void;
    get<T>(endpoint: string, query?: Record<string, string | number | boolean>, extra?: Partial<{
        headers: Record<string, string | number>;
    }>): RequestPayload<T>;
    post<T>(url: unknown, body?: any, extra?: Partial<{
        headers: Record<string, string | number>;
    }>): RequestPayload<T>;
};

type utils_RequestPayload<T> = RequestPayload<T>;
type utils_RequestedResource<T extends (...args: any[]) => any> = RequestedResource<T>;
type utils_TmpRequestResp<T extends (...args: any[]) => any> = TmpRequestResp<T>;
type utils_UnpackedRequestPayload<T> = UnpackedRequestPayload<T>;
declare const utils_onCreateGetPayload: typeof onCreateGetPayload;
declare const utils_onCreatePostPayload: typeof onCreatePostPayload;
declare const utils_request: typeof request;
declare const utils_request_factory: typeof request_factory;
declare namespace utils {
  export { utils_onCreateGetPayload as onCreateGetPayload, utils_onCreatePostPayload as onCreatePostPayload, utils_request as request, utils_request_factory as request_factory };
  export type { utils_RequestPayload as RequestPayload, utils_RequestedResource as RequestedResource, utils_TmpRequestResp as TmpRequestResp, utils_UnpackedRequestPayload as UnpackedRequestPayload };
}

declare enum Events$2 {
    BeforeRequest = 0,
    AfterRequest = 1,
    LoadingChange = 2,
    Success = 3,
    Failed = 4,
    Completed = 5,
    Canceled = 6,
    StateChange = 7,
    ResponseChange = 8
}
type TheTypesOfEvents$2<T> = {
    [Events$2.LoadingChange]: boolean;
    [Events$2.BeforeRequest]: void;
    [Events$2.AfterRequest]: void;
    [Events$2.Success]: T;
    [Events$2.Failed]: BizError;
    [Events$2.Completed]: void;
    [Events$2.Canceled]: void;
    [Events$2.StateChange]: RequestState<T>;
    [Events$2.ResponseChange]: T | null;
};
type RequestState<T> = {
    initial: boolean;
    loading: boolean;
    error: BizError | null;
    response: T | null;
};
type FetchFunction = (...args: any[]) => RequestPayload<any>;
type ProcessFunction<V, P> = (value: V) => Result<P>;
type RequestProps<F extends FetchFunction, P> = {
    _name?: string;
    client?: HttpClientCore;
    loading?: boolean;
    delay?: null | number;
    defaultResponse?: P;
    process?: ProcessFunction<Result<UnpackedRequestPayload<ReturnType<F>>>, P>;
    onSuccess?: (v: UnpackedResult<P>) => void;
    onFailed?: (error: BizError) => void;
    onCompleted?: () => void;
    onCanceled?: () => void;
    beforeRequest?: () => void;
    onLoading?: (loading: boolean) => void;
};
declare function onRequestCreated(h: (v: RequestCore<any>) => void): void;
type TheResponseOfRequestCore<T extends RequestCore<any, any>> = NonNullable<T["response"]>;
type TheResponseOfFetchFunction<T extends FetchFunction> = UnpackedRequestPayload<ReturnType<T>>;
/**
 * 用于接口请求的核心类
 */
declare class RequestCore<F extends FetchFunction, P = UnpackedRequestPayload<ReturnType<F>>> extends BaseDomain<TheTypesOfEvents$2<any>> {
    _name: string;
    debug: boolean;
    defaultResponse: P | null;
    /**
     * 就是
     *
     * ```js
     * function test() {
     *   return request.post('/api/ping');
     * }
     * ```
     *
     * 函数返回 RequestPayload，描述该次请求的地址、参数等
     */
    service: F;
    process?: ProcessFunction<Result<UnpackedRequestPayload<ReturnType<F>>>, P>;
    client?: HttpClientCore;
    delay: number | null;
    loading: boolean;
    initial: boolean;
    /** 处于请求中的 promise */
    pending: Promise<Result<P>> | null;
    /** 调用 run 方法暂存的参数 */
    args: Parameters<F>;
    /** 请求的响应 */
    response: P | null;
    /** 请求失败，保存错误信息 */
    error: BizError | null;
    id: string;
    get state(): RequestState<P>;
    constructor(fn: F, props?: RequestProps<F, P>);
    /** 执行 service 函数 */
    run(...args: Parameters<F>): Promise<Result<P>>;
    /** 使用当前参数再请求一次 */
    reload(): void;
    cancel(): Resp<null>;
    clear(): void;
    setError(err: BizError): void;
    modifyResponse(fn: (resp: P) => P): void;
    onLoadingChange(handler: Handler<TheTypesOfEvents$2<UnpackedResult<P>>[Events$2.LoadingChange]>): () => void;
    beforeRequest(handler: Handler<TheTypesOfEvents$2<UnpackedResult<P>>[Events$2.BeforeRequest]>): () => void;
    onSuccess(handler: Handler<TheTypesOfEvents$2<UnpackedResult<P>>[Events$2.Success]>): () => void;
    onFailed(handler: Handler<TheTypesOfEvents$2<UnpackedResult<P>>[Events$2.Failed]>, opt?: Partial<{
        /** 清除其他 failed 监听 */
        override: boolean;
    }>): () => void;
    onCanceled(handler: Handler<TheTypesOfEvents$2<UnpackedResult<P>>[Events$2.Canceled]>): () => void;
    /** 建议使用 onFailed */
    onError(handler: Handler<TheTypesOfEvents$2<UnpackedResult<P>>[Events$2.Failed]>): () => void;
    onCompleted(handler: Handler<TheTypesOfEvents$2<UnpackedResult<P>>[Events$2.Completed]>): () => void;
    onStateChange(handler: Handler<TheTypesOfEvents$2<UnpackedResult<P>>[Events$2.StateChange]>): () => void;
    onResponseChange(handler: Handler<TheTypesOfEvents$2<UnpackedResult<P>>[Events$2.ResponseChange]>): () => void;
}

/**
 * 请求原始响应
 */
type OriginalResponse = {
    list: unknown[];
};
/**
 * 查询参数
 */
type Search = {
    [x: string]: string | number | boolean | null | undefined;
};
/**
 * 请求参数
 */
interface FetchParams {
    page: number;
    pageSize: number;
    next_marker: string;
}
/**
 * 对外暴露的响应值
 */
interface Response<T> {
    /**
     * 列表数据
     */
    dataSource: T[];
    /**
     * 当前页码
     * @default 0
     */
    page: number;
    /**
     * 每页数量
     * @default 10
     */
    pageSize: number;
    /**
     * 记录总数
     * @default 0
     */
    total: number;
    /**
     * 查询参数
     */
    search: Search;
    /**
     * 是否初始化（用于展示骨架屏）
     */
    initial: boolean;
    /**
     * 没有更多数据了
     */
    noMore: boolean;
    /**
     * 是否请求中，initial 时 loading 为 false
     */
    loading: boolean;
    /** 是否为空（用于展示空状态） */
    empty: boolean;
    /**
     * 是否正在刷新（用于移动端下拉刷新）
     */
    refreshing: boolean | null;
    /**
     * 请求是否出错
     */
    error: Error | null;
}
/**
 * 参数处理器
 */
type ParamsProcessor = (params: FetchParams, currentParams: any) => FetchParams;
interface ListProps<T> {
    /**
     * 是否打开 debug
     */
    debug?: boolean;
    /**
     * dataSource 中元素唯一 key
     * @default id
     */
    rowKey?: string;
    /**
     * 参数处理器
     * 建议在 service 函数中直接处理
     */
    beforeRequest?: ParamsProcessor;
    /**
     * 响应处理器
     * 建议在 service 函数中直接处理
     */
    processor?: <T>(response: Response<T>, originalResponse: OriginalResponse | null) => Response<T>;
    /**
     * 默认已存在的数据
     */
    dataSource?: T[];
    /**
     * 默认查询条件
     */
    search?: Search;
    /**
     * 默认当前页
     */
    page?: number;
    /**
     * 默认每页数量
     */
    pageSize?: number;
    /**
     * 额外的默认 response
     */
    extraDefaultResponse?: Record<string, unknown>;
    extraDataSource?: T[];
    /** 初始状态，默认该值为 true，可以通过该值判断是否展示骨架屏 */
    initial?: boolean;
    onLoadingChange?: (loading: boolean) => void;
    onStateChange?: (state: Response<T>) => void;
    beforeSearch?: () => void;
    afterSearch?: () => void;
}

/**
 * 移除指定字段
 * 这个方法 lodash 中有，但是为了 Pagination 不包含任何依赖，所以这里自己实现了
 * @param data
 * @param keys
 */
declare function omit(data: {
    [key: string]: any;
}, keys: string[]): {
    [key: string]: any;
};
declare function noop(): void;
declare function merge<T extends Record<string, any> = {
    [key: string]: any;
}>(current: T, defaultObj: T, override?: boolean): {
    [x: string]: any;
};
declare const qs: {
    parse(search: string): {
        [x: string]: string;
    };
    stringify(obj: Record<string, any>): string;
};

declare enum Events$1 {
    LoadingChange = 0,
    BeforeSearch = 1,
    AfterSearch = 2,
    ParamsChange = 3,
    DataSourceChange = 4,
    DataSourceAdded = 5,
    StateChange = 6,
    Error = 7,
    /** 一次请求结束 */
    Completed = 8
}
type TheTypesOfEvents$1<T> = {
    [Events$1.LoadingChange]: boolean;
    [Events$1.BeforeSearch]: void;
    [Events$1.AfterSearch]: {
        params: any;
    };
    [Events$1.ParamsChange]: FetchParams;
    [Events$1.DataSourceAdded]: T[];
    [Events$1.DataSourceChange]: {
        dataSource: T[];
        reason: "init" | "goto" | "next" | "prev" | "clear" | "refresh" | "search" | "load_more" | "reload" | "reset" | "manually";
    };
    [Events$1.StateChange]: ListState<T>;
    [Events$1.Error]: Error;
    [Events$1.Completed]: void;
};
interface ListState<T> extends Response<T> {
}
/**
 * 分页类
 */
declare class ListCore<S extends RequestCore<(...args: any[]) => RequestPayload<any>>, T = NonNullable<S["response"]>["list"][number]> extends BaseDomain<TheTypesOfEvents$1<T>> {
    debug: boolean;
    static defaultResponse: <T_1>() => Response<T_1>;
    static commonProcessor: <T_1>(originalResponse: OriginalResponse | null) => {
        dataSource: T_1[];
        page: number;
        pageSize: number;
        total: number;
        empty: boolean;
        noMore: boolean;
        error: Error | null;
    };
    /** 原始请求方法 */
    request: S;
    /** 支持请求前对参数进行处理（formToBody） */
    private beforeRequest;
    /** 响应处理器 */
    private processor;
    /** 初始查询参数 */
    private initialParams;
    private extraResponse;
    /** 额外的数据 */
    private extraDatSource;
    private insertExtraDataSource;
    params: FetchParams;
    response: Response<T>;
    rowKey: string;
    constructor(fetch: S, options?: ListProps<T>);
    private initialize;
    /**
     * 手动修改当前实例的查询参数
     * @param {import('./typing').FetchParams} nextParams 查询参数或设置函数
     */
    setParams(nextParams: Partial<FetchParams> | ((p: FetchParams) => FetchParams)): void;
    setDataSource(dataSources: T[]): void;
    /**
     * 调用接口进行请求
     * 外部不应该直接调用该方法
     * @param {import('./typing').FetchParams} nextParams - 查询参数
     */
    fetch(params: Partial<FetchParams>, ...restArgs: any[]): Promise<Result<Response<T>>>;
    /**
     * 使用初始参数请求一次，初始化时请调用该方法
     */
    init(params?: {}): Promise<Result<{
        dataSource: T[];
        page: number;
        pageSize: number;
        total: number;
        search: Search;
        initial: boolean;
        noMore: boolean;
        loading: boolean;
        empty: boolean;
        refreshing: boolean | null;
        error: Error | null;
    }>>;
    /** 无论如何都会触发一次 state change */
    initAny(): Promise<Resp<Response<T>>>;
    /**
     * 下一页
     */
    next(): Promise<Result<{
        dataSource: T[];
        page: number;
        pageSize: number;
        total: number;
        search: Search;
        initial: boolean;
        noMore: boolean;
        loading: boolean;
        empty: boolean;
        refreshing: boolean | null;
        error: Error | null;
    }>>;
    /**
     * 返回上一页
     */
    prev(): Promise<Result<{
        dataSource: T[];
        page: number;
        pageSize: number;
        total: number;
        search: Search;
        initial: boolean;
        noMore: boolean;
        loading: boolean;
        empty: boolean;
        refreshing: boolean | null;
        error: Error | null;
    }>>;
    nextWithCursor(): void;
    /** 强制请求下一页，如果下一页没有数据，page 不改变 */
    loadMoreForce(): Promise<Result<{
        dataSource: T[];
        page: number;
        pageSize: number;
        total: number;
        search: Search;
        initial: boolean;
        noMore: boolean;
        loading: boolean;
        empty: boolean;
        refreshing: boolean | null;
        error: Error | null;
    }>>;
    /**
     * 无限加载时使用的下一页
     */
    loadMore(): Promise<Result<{
        dataSource: T[];
        page: number;
        pageSize: number;
        total: number;
        search: Search;
        initial: boolean;
        noMore: boolean;
        loading: boolean;
        empty: boolean;
        refreshing: boolean | null;
        error: Error | null;
    }>>;
    /**
     * 前往指定页码
     * @param {number} page - 要前往的页码
     * @param {number} [pageSize] - 每页数量
     */
    goto(targetPage: number, targetPageSize: number): Promise<Result<{
        dataSource: T[];
        page: number;
        pageSize: number;
        total: number;
        search: Search;
        initial: boolean;
        noMore: boolean;
        loading: boolean;
        empty: boolean;
        refreshing: boolean | null;
        error: Error | null;
    }>>;
    search(...args: Parameters<S["service"]>): Promise<Result<{
        dataSource: T[];
        page: number;
        pageSize: number;
        total: number;
        search: Search;
        initial: boolean;
        noMore: boolean;
        loading: boolean;
        empty: boolean;
        refreshing: boolean | null;
        error: Error | null;
    }>>;
    searchDebounce: (...args: Parameters<S["service"]>) => void;
    /**
     * 使用初始参数请求一次，「重置」操作时调用该方法
     */
    reset(params?: Partial<FetchParams>): Promise<Result<{
        dataSource: T[];
        page: number;
        pageSize: number;
        total: number;
        search: Search;
        initial: boolean;
        noMore: boolean;
        loading: boolean;
        empty: boolean;
        refreshing: boolean | null;
        error: Error | null;
    }>>;
    /**
     * 使用当前参数重新请求一次，PC 端「刷新」操作时调用该方法
     */
    reload(): Promise<Result<Response<T>>>;
    /**
     * 页码置为 1，其他参数保留，重新请求一次。移动端「刷新」操作时调用该方法
     */
    refresh(): Promise<Result<{
        dataSource: T[];
        page: number;
        pageSize: number;
        total: number;
        search: Search;
        initial: boolean;
        noMore: boolean;
        loading: boolean;
        empty: boolean;
        refreshing: boolean | null;
        error: Error | null;
    }>>;
    clear(): void;
    deleteItem(fn: (item: T) => boolean): void;
    /**
     * 移除列表中的多项（用在删除场景）
     * @param {T[]} items 要删除的元素列表
     */
    deleteItems(items: T[]): Promise<void>;
    modifyItem(fn: (item: T) => T): void;
    replaceDataSource(dataSource: T[]): void;
    /**
     * 手动修改当前 dataSource
     * @param fn
     */
    modifyDataSource(fn: (v: T) => T): void;
    /**
     * 手动修改当前 response
     * @param fn
     */
    modifyResponse(fn: (v: Response<T>) => Response<T>): void;
    /**
     * 手动修改当前 params
     */
    modifyParams(fn: (v: FetchParams) => FetchParams): void;
    /**
     * 手动修改当前 search
     */
    modifySearch(fn: (v: FetchParams) => FetchParams): void;
    onStateChange(handler: Handler<TheTypesOfEvents$1<T>[Events$1.StateChange]>): () => void;
    onLoadingChange(handler: Handler<TheTypesOfEvents$1<T>[Events$1.LoadingChange]>): () => void;
    onBeforeSearch(handler: Handler<TheTypesOfEvents$1<T>[Events$1.BeforeSearch]>): () => void;
    onAfterSearch(handler: Handler<TheTypesOfEvents$1<T>[Events$1.AfterSearch]>): () => void;
    onDataSourceChange(handler: Handler<TheTypesOfEvents$1<T>[Events$1.DataSourceChange]>): () => void;
    onDataSourceAdded(handler: Handler<TheTypesOfEvents$1<T>[Events$1.DataSourceAdded]>): () => void;
    onError(handler: Handler<TheTypesOfEvents$1<T>[Events$1.Error]>): () => void;
    onComplete(handler: Handler<TheTypesOfEvents$1<T>[Events$1.Completed]>): () => void;
}

/**
 * @file 列表中多选
 */

declare enum Events {
    Change = 0,
    StateChange = 1
}
type TheTypesOfEvents<T> = {
    [Events.Change]: T[];
    [Events.StateChange]: MultipleSelectionState<T>;
};
type SelectionProps<T> = {
    defaultValue: T[];
    options: {
        label: string;
        value: T;
    }[];
    onChange?: (v: T[]) => void;
};
type MultipleSelectionState<T> = {
    value: T[];
    options: {
        label: string;
        value: T;
    }[];
};
declare class MultipleSelectionCore<T> extends BaseDomain<TheTypesOfEvents<T>> {
    shape: "multiple-select";
    value: T[];
    defaultValue: T[];
    options: {
        label: string;
        value: T;
    }[];
    get state(): MultipleSelectionState<T>;
    constructor(props: SelectionProps<T>);
    setValue(value: T[]): void;
    toggle(value: T): void;
    select(value: T): void;
    remove(value: T): void;
    /** 暂存的值是否为空 */
    isEmpty(): boolean;
    clear(): void;
    onChange(handler: Handler<TheTypesOfEvents<T>[Events.Change]>): () => void;
    onStateChange(handler: Handler<TheTypesOfEvents<T>[Events.StateChange]>): () => void;
}

type CanvasDrawingProps = {
    width: number;
    height: number;
    ctx: CanvasRenderingContext2D;
};
interface QRCode {
    getModuleCount(): number;
    isDark(x: number, y: number): boolean;
}
/**
 * Drawing QRCode by using canvas
 *
 * @constructor
 * @param {Object} htOption QRCode Options
 */
declare class CanvasDrawing {
    /** canvas 绘制上下文 */
    ctx: CanvasRenderingContext2D;
    /** 是否绘制完成 */
    isPainted: boolean;
    options: {
        width: number;
        height: number;
    };
    constructor(options: CanvasDrawingProps);
    /**
     * 绘制 logo
     */
    /**
     * Draw the QRCode
     *
     * @param {QRCode} model
     */
    draw(model: QRCode): void;
    /**
     * Make the image from Canvas if the browser supports Data URI.
     */
    /**
     * Clear the QRCode
     */
    clear(): void;
    /**
     * @private
     * @param {Number} nNumber
     */
    round(nNumber: number): number;
}
declare function createQRCode(text: string, options: CanvasDrawingProps): Promise<void>;

declare class CurSystem {
    connection: string;
    constructor();
    query_network(): any;
}
declare const system: CurSystem;

/**
 * @file 路径
 * 可以根据给定的 number[] 从树上找到指定节点
 */

type SlatePoint = {
    path: number[];
    offset: number;
};

declare enum SlateDescendantType {
    Text = "text",
    Paragraph = "paragraph"
}
type SlateParagraph = {
    children: SlateDescendant[];
};
type SlateText = {
    text: string;
};
type SlateDescendant = MutableRecord2<{
    [SlateDescendantType.Text]: SlateText;
    [SlateDescendantType.Paragraph]: SlateParagraph;
}>;
declare enum SlateOperationType {
    InsertText = "insert_text",
    ReplaceText = "replace_text",
    RemoveText = "remove_text",
    InsertLines = "insert_line",
    RemoveLines = "remove_lines",
    MergeNode = "merge_node",
    SplitNode = "split_node",
    SetSelection = "set_selection",
    Unknown = "unknown"
}
type SlateOperationInsertText = {
    /** 插入的文本 */
    text: string;
    original_text: string;
    path: number[];
    offset: number;
};
type SlateOperationReplaceText = {
    /** 替换的文本 */
    text: string;
    original_text: string;
    path: number[];
    offset: number;
};
type SlateOperationRemoveText = {
    /** 删除的文本 */
    text: string;
    original_text: string;
    ignore?: boolean;
    path: number[];
    offset: number;
};
type SlateOperationInsertLines = {
    /** 插入的位置，取第一个元素，就是 line index */
    path: number[];
    node: (SlateDescendant & {
        key: number;
    })[];
};
type SlateOperationRemoveLines = {
    /** 插入的位置，取第一个元素，就是 line index */
    path: number[];
    node: (SlateDescendant & {
        key: number;
    })[];
};
type SlateOperationMergeNode = {
    /** 前一个节点的位置 */
    path: number[];
    offset: number;
    /** 当前光标的位置，只用 end */
    start: SlatePoint;
    end: SlatePoint;
    compositing?: boolean;
};
type SlateOperationSplitNode = {
    path: number[];
    offset: number;
    /** 分割完成后的光标位置 */
    start: SlatePoint;
    /** 分割完成后的光标位置 */
    end: SlatePoint;
};
/** 设置选区/光标位置 */
type SlateOperationSetSelection = {
    start: SlatePoint;
    end: SlatePoint;
};
type SlateOperation = MutableRecord2<{
    [SlateOperationType.InsertText]: SlateOperationInsertText;
    [SlateOperationType.ReplaceText]: SlateOperationReplaceText;
    [SlateOperationType.RemoveText]: SlateOperationRemoveText;
    [SlateOperationType.InsertLines]: SlateOperationInsertLines;
    [SlateOperationType.RemoveLines]: SlateOperationRemoveLines;
    [SlateOperationType.MergeNode]: SlateOperationMergeNode;
    [SlateOperationType.SplitNode]: SlateOperationSplitNode;
    [SlateOperationType.SetSelection]: SlateOperationSetSelection;
}>;

type BeforeInputEvent = {
    preventDefault(): void;
    data: unknown;
};
type BlurEvent = {};
type FocusEvent = {};
type CompositionEndEvent = {
    data: unknown;
    preventDefault(): void;
};
type CompositionUpdateEvent = {
    preventDefault(): void;
};
type CompositionStartEvent = {
    preventDefault(): void;
};
type KeyDownEvent = {
    code: string;
    preventDefault(): void;
};
type KeyUpEvent = {
    code: string;
    preventDefault(): void;
};
type TextInsertTextOptions = {
    at?: any;
    voids?: boolean;
};
declare function SlateEditorModel(props: {
    defaultValue?: SlateDescendant[];
}): {
    methods: {
        refresh(): void;
        emitSelectionChange(v: {
            type?: SlateOperationType;
        }): void;
        apply(operations: SlateOperation[]): void;
        findNodeByPath(path: number[]): SlateDescendant;
        getDefaultInsertLocation(): number[];
        /** 输入文本内容 */
        insertText(text: string, options?: TextInsertTextOptions): void;
        /** 前面新增行 */
        insertLineBefore(): void;
        /** 后面新增行 */
        insertLineAfter(): void;
        /** 拆分当前行 */
        splitLine(): void;
        mergeLines(start: SlatePoint): void;
        removeLines(start: SlatePoint): void;
        /** 删除指定位置的文本 */
        removeText(node: SlateText, point: SlatePoint): void;
        removeContentCrossLines(node1: SlateText, node2: SlateText, start: SlatePoint, end: SlatePoint): {
            op_start_delete_text: {
                type: SlateOperationType.RemoveText;
            } & SlateOperationRemoveText;
            op_end_delete_text: {
                type: SlateOperationType.RemoveText;
            } & SlateOperationRemoveText;
        };
        /** 跨行操作，删除、回车、输入 时，要不要删除中间的行，删除后 end 坐标应该在哪 */
        removeLinesCrossLines(start: SlatePoint, end: SlatePoint): {
            end: SlatePoint;
            op_remove_lines: {
                type: SlateOperationType.RemoveLines;
            } & SlateOperationRemoveLines;
        };
        /** 选中的文本后删除，可能是跨节点 */
        removeSelectedTextsCrossNodes(node: SlateText, arr: {
            start: SlatePoint;
            end: SlatePoint;
        }): void;
        handleBackward(param?: Partial<{
            unit: "character";
        }>): void;
        mapNodeWithKey(key?: string): SlateDescendant;
        checkIsSelectAll(): void;
        collapse(): void;
        /** 移动光标 */
        move(opts: {
            unit: "line";
            edge?: "focus";
            reverse?: boolean;
        }): void;
        getCaretPosition(): void;
        setCaretPosition(arg: {
            start: SlatePoint;
            end: SlatePoint;
        }): void;
        handleBeforeInput(event: BeforeInputEvent): void;
        handleInput(event: InputEvent): void;
        handleBlur(event: BlurEvent): void;
        handleFocus(event: FocusEvent): void;
        handleClick(): void;
        handleCompositionEnd(event: CompositionEndEvent): void;
        handleCompositionUpdate(event: CompositionUpdateEvent): void;
        handleCompositionStart(event: CompositionStartEvent): void;
        handleKeyDown(event: KeyDownEvent): void;
        handleKeyUp(event: KeyUpEvent): void;
        handleSelectionChange(): void;
    };
    ui: {
        $selection: {
            methods: {
                refresh(): void;
                moveForward(param?: Partial<{
                    step: number;
                    min: number;
                    collapse: boolean;
                }>): void;
                moveBackward(param?: Partial<{
                    step: number;
                    min: number;
                }>): void;
                moveToPrevLineHead(): void;
                calcNextLineHead(): {
                    start: {
                        path: number[];
                        offset: number;
                    };
                    end: {
                        path: number[];
                        offset: number;
                    };
                };
                moveToNextLineHead(): void;
                collapseToHead(): void;
                collapseToEnd(): void;
                collapseToOffset(param: {
                    offset: number;
                }): void;
                setToHead(): void;
                setStartAndEnd(param: {
                    start: SlatePoint;
                    end: SlatePoint;
                }): void;
                handleChange(event: {
                    start: SlatePoint;
                    end: SlatePoint;
                    collapsed: boolean;
                }): void;
            };
            ui: {};
            state: {
                readonly start: {
                    line: number;
                    path: number[];
                    offset: number;
                };
                readonly end: {
                    line: number;
                    path: number[];
                    offset: number;
                };
                readonly collapsed: boolean;
                readonly dirty: boolean;
            };
            readonly dirty: boolean;
            readonly start: {
                line: number;
                path: number[];
                offset: number;
            };
            readonly end: {
                line: number;
                path: number[];
                offset: number;
            };
            readonly collapsed: boolean;
            print(): {
                start: string;
                end: string;
            };
            ready(): void;
            destroy(): void;
            onStateChange(handler: Handler<{
                readonly start: {
                    line: number;
                    path: number[];
                    offset: number;
                };
                readonly end: {
                    line: number;
                    path: number[];
                    offset: number;
                };
                readonly collapsed: boolean;
                readonly dirty: boolean;
            }>): () => void;
            onError(handler: Handler<BizError>): () => void;
        };
        $history: {
            methods: {
                refresh(): void;
                mark(): void;
                push(ops: SlateOperation[], selection: {
                    start: SlatePoint;
                    end: SlatePoint;
                }): void;
                undo(): {
                    operations: SlateOperation[];
                    selection: {
                        start: SlatePoint;
                        end: SlatePoint;
                    };
                };
            };
            ui: {};
            state: {
                readonly stacks: {
                    type: SlateOperationType;
                    created_at: string;
                }[];
            };
            ready(): void;
            destroy(): void;
            onStateChange(handler: Handler<{
                readonly stacks: {
                    type: SlateOperationType;
                    created_at: string;
                }[];
            }>): () => void;
            onError(handler: Handler<BizError>): () => void;
        };
        $shortcut: {
            methods: {
                refresh(): void;
                register(handlers: Record<string, (event: {
                    code: string;
                    preventDefault: () => void;
                } & {
                    step?: "keydown" | "keyup";
                }) => void>): void;
                clearPressedKeys(): void;
                invokeHandlers(event: {
                    code: string;
                    preventDefault: () => void;
                }, key: string): void;
                buildShortcut(): {
                    key1: string;
                    key2: string;
                };
                setRecordingCodes(codes: string): void;
                reset(): void;
                testShortcut(opt: {
                    key1: string;
                    key2: string;
                    step: "keydown" | "keyup";
                }, event: {
                    code: string;
                    preventDefault: () => void;
                }): void;
                handleKeydown(event: {
                    code: string;
                    preventDefault: () => void;
                }): void;
                handleKeyup(event: {
                    code: string;
                    preventDefault: () => void;
                }, opt?: Partial<{
                    fake: boolean;
                }>): void;
            };
            ui: {};
            state: {
                readonly codes: string[];
                readonly codes2: string[];
            };
            ready(): void;
            destroy(): void;
            onShortcut(handler: Handler<{
                key: string;
            }>): () => void;
            onShortcutComplete(handler: Handler<{
                codes: string[];
            }>): () => void;
            onStateChange(handler: Handler<{
                readonly codes: string[];
                readonly codes2: string[];
            }>): () => void;
            onError(handler: Handler<BizError>): () => void;
        };
    };
    state: {
        readonly children: SlateDescendant[];
        readonly isFocus: boolean;
        readonly JSON: string;
    };
    readonly isFocus: boolean;
    ready(): void;
    destroy(): void;
    onAction(handler: Handler<SlateOperation[]>): () => void;
    onSelectionChange(handler: Handler<{
        type?: SlateOperationType;
        start: SlatePoint;
        end: SlatePoint;
    }>): () => void;
    onStateChange(handler: Handler<{
        readonly children: SlateDescendant[];
        readonly isFocus: boolean;
        readonly JSON: string;
    }>): () => void;
    onError(handler: Handler<BizError>): () => void;
};
type SlateEditorModel = ReturnType<typeof SlateEditorModel>;

declare const SlateNodeOperations: {
    insertText(nodes: SlateDescendant[], op: SlateOperation): SlateDescendant[];
    replaceText(nodes: SlateDescendant[], op: SlateOperation): SlateDescendant[];
    removeText(nodes: SlateDescendant[], op: SlateOperation): SlateDescendant[];
    splitNode(nodes: SlateDescendant[], op: SlateOperation): SlateDescendant[];
    mergeNode(nodes: SlateDescendant[], op: SlateOperation): SlateDescendant[];
    insertLines(nodes: SlateDescendant[], op: SlateOperation): SlateDescendant[];
    removeLines(nodes: SlateDescendant[], op: SlateOperation): SlateDescendant[];
    exec(nodes: SlateDescendant[], op: SlateOperation): SlateDescendant[];
};
declare function findNodeByPathWithNode(nodes: SlateDescendant[], path: number[]): SlateDescendant;

declare const op_node_SlateNodeOperations: typeof SlateNodeOperations;
declare const op_node_findNodeByPathWithNode: typeof findNodeByPathWithNode;
declare namespace op_node {
  export {
    op_node_SlateNodeOperations as SlateNodeOperations,
    op_node_findNodeByPathWithNode as findNodeByPathWithNode,
  };
}

declare const SlateDOMOperations: {
    insertText($input: Element, op: SlateOperation): void;
    replaceText($input: Element, op: SlateOperation): void;
    removeText($input: Element, op: SlateOperation): void;
    splitNode($input: Element, op: SlateOperation): void;
    mergeNode($input: Element, op: SlateOperation): void;
    insertLines($input: Element, op: SlateOperation): void;
    removeLines($input: Element, op: SlateOperation): void;
    exec($input: Element, op: SlateOperation): void;
};
declare function renderText(node: SlateDescendant & {
    key?: number;
}, extra?: {
    text?: boolean;
}): Element | null;
declare function renderElement(node: SlateDescendant & {
    key?: number;
}, extra?: {
    text?: boolean;
}): Element | null;
declare function buildInnerHTML(nodes: SlateDescendant[], parents?: number[], level?: number): DocumentFragment;
declare function getNodeText($node: Element): string;
declare function formatInnerHTML(v: string): string;
declare function renderHTML(v: string): string;
declare function renderNodeThenInsertLine($input: Element, op: {
    node: SlateDescendant;
    path: number[];
}): void;
declare function renderLineNodesThenInsert($input: Element, op: {
    node: SlateDescendant[];
    path: number[];
}): void;
declare function findInnerTextNode($node?: any): any;
declare function getNodePath(targetNode: Element, rootNode: Element): number[];
declare function findNodeByPathWithElement($elm: Element, path: number[]): Element | null;
declare function refreshSelection($editor: Element, start: SlatePoint, end: SlatePoint): void;

declare const op_dom_SlateDOMOperations: typeof SlateDOMOperations;
declare const op_dom_buildInnerHTML: typeof buildInnerHTML;
declare const op_dom_findInnerTextNode: typeof findInnerTextNode;
declare const op_dom_findNodeByPathWithElement: typeof findNodeByPathWithElement;
declare const op_dom_formatInnerHTML: typeof formatInnerHTML;
declare const op_dom_getNodePath: typeof getNodePath;
declare const op_dom_getNodeText: typeof getNodeText;
declare const op_dom_refreshSelection: typeof refreshSelection;
declare const op_dom_renderElement: typeof renderElement;
declare const op_dom_renderHTML: typeof renderHTML;
declare const op_dom_renderLineNodesThenInsert: typeof renderLineNodesThenInsert;
declare const op_dom_renderNodeThenInsertLine: typeof renderNodeThenInsertLine;
declare const op_dom_renderText: typeof renderText;
declare namespace op_dom {
  export {
    op_dom_SlateDOMOperations as SlateDOMOperations,
    op_dom_buildInnerHTML as buildInnerHTML,
    op_dom_findInnerTextNode as findInnerTextNode,
    op_dom_findNodeByPathWithElement as findNodeByPathWithElement,
    op_dom_formatInnerHTML as formatInnerHTML,
    op_dom_getNodePath as getNodePath,
    op_dom_getNodeText as getNodeText,
    op_dom_refreshSelection as refreshSelection,
    op_dom_renderElement as renderElement,
    op_dom_renderHTML as renderHTML,
    op_dom_renderLineNodesThenInsert as renderLineNodesThenInsert,
    op_dom_renderNodeThenInsertLine as renderNodeThenInsertLine,
    op_dom_renderText as renderText,
  };
}

type KeyboardEvent = {
    code: string;
    preventDefault: () => void;
};
declare function ShortcutModel(props?: Partial<{
    mode?: "normal" | "recording";
}>): {
    methods: {
        refresh(): void;
        register(handlers: Record<string, (event: KeyboardEvent & {
            step?: "keydown" | "keyup";
        }) => void>): void;
        clearPressedKeys(): void;
        invokeHandlers(event: KeyboardEvent, key: string): void;
        buildShortcut(): {
            key1: string;
            key2: string;
        };
        setRecordingCodes(codes: string): void;
        reset(): void;
        testShortcut(opt: {
            /** 存在 pressing 时，进行拼接后的字符串，用于「组合快捷键」 */
            key1: string;
            /** 没有其他出于 pressing 状态的情况下，按下的按键拼接后的字符串，用于「单个快捷键或连按」 */
            key2: string;
            step: "keydown" | "keyup";
        }, event: KeyboardEvent): void;
        handleKeydown(event: {
            code: string;
            preventDefault: () => void;
        }): void;
        handleKeyup(event: {
            code: string;
            preventDefault: () => void;
        }, opt?: Partial<{
            fake: boolean;
        }>): void;
    };
    ui: {};
    state: {
        readonly codes: string[];
        readonly codes2: string[];
    };
    ready(): void;
    destroy(): void;
    onShortcut(handler: Handler<{
        key: string;
    }>): () => void;
    onShortcutComplete(handler: Handler<{
        codes: string[];
    }>): () => void;
    onStateChange(handler: Handler<{
        readonly codes: string[];
        readonly codes2: string[];
    }>): () => void;
    onError(handler: Handler<BizError>): () => void;
};

/**
 * @todo 如果删除当前选中的文件夹，子文件夹在视图上也要同步移除
 */

type BizFile = {
    id: string;
    filepath: string;
    filename: string;
    type: BizFileType;
};
declare enum BizFileType {
    File = 1,
    Folder = 2
}
type BizFileFetchService = (...args: any[]) => RequestPayload<BizFile[]>;
type FileColumn = {
    list: ListCore<RequestCore<BizFileFetchService, any>>;
    view: ScrollViewCore;
};
declare function FileBrowserModel(props: {
    paths?: BizFile[];
    service: BizFileFetchService;
    /** 调用接口时的额外参数 */
    extra?: Record<string, unknown>;
    client: HttpClientCore;
    onError?: (err: BizError) => void;
}): {
    methods: {
        refresh(): void;
        createColumn(folder: BizFile): FileColumn;
        appendColumn(folder: BizFile): void;
        replaceColumn(folder: BizFile, index: number): void;
        clearFolderColumns(): void;
        /** 选中文件/文件夹 */
        select(folder: BizFile, index: [number, number]): void;
        virtualSelect(folder: BizFile, position: [number, number]): void;
        clear(): void;
        clearVirtualSelected(): void;
    };
    state: {
        readonly loading: boolean;
        readonly initialized: boolean;
        readonly curFolder: BizFile;
        readonly paths: BizFile[];
        readonly columns: FileColumn[];
    };
    onFolderColumnChange(handler: Handler<FileColumn[]>): () => void;
    onPathsChange(handler: Handler<{
        file_id: string;
        name: string;
    }[]>): () => void;
    onSelectFolder(handler: Handler<[BizFile, [number, number]]>): () => void;
    onLoadingChange(handler: Handler<boolean>): () => void;
    onError(handler: Handler<BizError>): () => void;
    onStateChange(handler: Handler<{
        readonly loading: boolean;
        readonly initialized: boolean;
        readonly curFolder: BizFile;
        readonly paths: BizFile[];
        readonly columns: FileColumn[];
    }>): () => void;
};
type FileBrowserModel = ReturnType<typeof FileBrowserModel>;

export { ApplicationModel, BaseDomain, BaseEvents, BizError, BizFileType, CanvasDrawing, CurSystem, FileBrowserModel, HistoryCore, HttpClientCore, ListCore, MultipleSelectionCore, NavigatorCore, RequestCore, Result, RouteMenusModel, RouteViewCore, ShortcutModel, op_dom as SlateDOM, SlateEditorModel, op_node as SlateNode, StorageCore, applyMixins, base, build, buildUrl, createQRCode, merge, noop, omit, onRequestCreated, onViewCreated, qs, utils as rutil, system, index as ui };
export type { BizFile, BizFileFetchService, Handler, OriginalRouteConfigure, PageKeysType, PathnameKey, Resp, RouteAction, RouteConfig, TheResponseOfFetchFunction, TheResponseOfRequestCore, UnpackedResult };

}