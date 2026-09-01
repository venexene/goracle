(() => {
  const movedRoutes = [
    ["/go-fundamentals/Компиляция и запуск Go/compilation", "/go-fundamentals/Компиляция и оптимизации Go/compilation"],
    ["/go-fundamentals/Компиляция и запуск Go/compilation-and-startup", "/go-fundamentals/Компиляция и оптимизации Go/compilation-and-startup"],
    ["/go-fundamentals/Escape Analysis в Go/escape-analysis", "/go-fundamentals/Компиляция и оптимизации Go/Анализ времени жизни/escape-analysis"],
    ["/go-fundamentals/Escape Analysis в Go/compiler-reference", "/go-fundamentals/Компиляция и оптимизации Go/Анализ времени жизни/compiler-reference"],
    ["/go-fundamentals/Инлайн в Go/inlining", "/go-fundamentals/Компиляция и оптимизации Go/Встраивание функций/inlining"],
    ["/go-fundamentals/Инлайн в Go/compiler-reference", "/go-fundamentals/Компиляция и оптимизации Go/Встраивание функций/compiler-reference"],
    ["/go-fundamentals/Выравнивание в Go/alignment", "/go-fundamentals/Память в Go/Выравнивание/alignment"],
  ];

  const url = new URL(window.location.href);
  const pathname = decodeURIComponent(url.pathname).replace(/\/$/, "").replace(/\.html$/, "");
  for (const [oldRoute, newRoute] of movedRoutes) {
    if (!pathname.endsWith(oldRoute)) continue;
    const prefix = pathname.slice(0, -oldRoute.length);
    url.pathname = `${prefix}${newRoute}/`;
    window.location.replace(url);
    return;
  }

  if (!window.location.hash) return;

  const routes = [
    ["/architecture/HTTP/http", "/architecture/HTTP/reference"],
    ["/tools-and-practice/Базы данных в Go/db", "/tools-and-practice/Базы данных в Go/reference"],
    ["/go-details/Синхронизация в Go/synchronization", "/go-details/Синхронизация в Go/reference"],
    ["/go-fundamentals/Планировщик Go/scheduler", "/go-fundamentals/Планировщик Go/reference"],
    ["/go-fundamentals/Сбощик мусора в Go/garbage-collector", "/go-fundamentals/Сбощик мусора в Go/runtime-reference"],
    ["/go-fundamentals/Компиляция и оптимизации Go/Анализ времени жизни/escape-analysis", "/go-fundamentals/Компиляция и оптимизации Go/Анализ времени жизни/compiler-reference"],
    ["/go-fundamentals/Компиляция и оптимизации Go/Встраивание функций/inlining", "/go-fundamentals/Компиляция и оптимизации Go/Встраивание функций/compiler-reference"],
    ["/computer-science/Конкурентные паттерны в Go/concurrency-patterns", "/computer-science/Конкурентные паттерны в Go/reference"],
  ];

  for (const [oldRoute, newRoute] of routes) {
    if (!pathname.endsWith(oldRoute)) continue;
    const prefix = pathname.slice(0, -oldRoute.length);
    url.pathname = `${prefix}${newRoute}/`;
    window.location.replace(url);
    return;
  }
})();
