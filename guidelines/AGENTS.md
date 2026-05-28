# Directrices de Desarrollo (Agents)

El proyecto **Cifras y Letras** se desarrolla utilizando las siguientes tecnologías, patrones de diseño y mejores prácticas. Si eres un agente de IA colaborando en esta base de código, debes seguir rigurosamente estas pautas para mantener la cohesión del sistema.

---

## 1. Stack Tecnológico de Referencia

### 1.1. Backend (Go 1.26)
*   **Patrón de Concurrencia:** **Actor Model** (vía goroutines y canales).
    *   **Regla de Oro:** **Evita el uso de `sync.Mutex`** para proteger o mutar el estado de la sala de juego (`SyncData` / `GameRoom`).
    *   Todas las operaciones de la sala (uniones, desconexiones, cambios de nombre, envíos de respuestas) deben ser canalizadas secuencialmente a través del canal central de acciones (`ActionChan chan GameAction`) de la clase `GameRoom`.
    *   El bucle de ejecución asíncrono único en `GameRoom.Run()` (`backend/room.go`) es la única goroutine autorizada para leer y mutar el estado, eliminando de raíz las condiciones de carrera de forma natural.
*   **Comunicaciones:** WebSockets nativos eficientes utilizando la biblioteca `github.com/coder/websocket` (`backend/server.go`).
    *   Mantén el ciclo de vida de los clientes estructurado en tres goroutines independientes: `readPump()`, `writePump()`, y el bucle de latencia `pingLoop()`.

### 1.2. Frontend (Svelte 5)
*   **Reactividad:** Paradigma moderno de **Runes** de Svelte 5.
    *   **Prohibido:** No utilices la API clásica de Svelte 4 (`writable`, `readable`, `derived` de `'svelte/store'`) ni la asignación reactiva con etiquetas `$:` en componentes.
    *   **Uso Obligatorio:** Emplea de forma exclusiva las runes nativas:
        *   `$state(...)` para datos mutables reactivos simples o complejos.
        *   `$derived(...)` o `$derived.by(...)` para cálculos en tiempo de ejecución (como el filtrado y ordenación dinámica del ranking y tablas de resultados).
        *   `$effect(...)` para lógica externa que reacciona a cambios de estado, tales como temporizadores de reloj, reconexiones o reproducción rítmica de sonido.
    *   Consulta la documentación actualizada de Svelte 5 en: https://svelte.dev/llms.txt
*   **Diseño UI/UX:** Fluent Design Oscuro (estilo Microsoft Modern UI).
*   **Estilos:** **Vanilla CSS nativo.**
    *   **Prohibido:** No instales frameworks CSS externos ni TailwindCSS. Toda la lógica de presentación debe residir en `app.css` y las etiquetas `<style>` individuales de los componentes Svelte.
    *   Esto garantiza el control milimétrico sobre las animaciones complejas, transiciones de desvanecimiento (`fade`), y los filtros avanzados de vidrio esmerilado (`backdrop-filter`).

---

## 2. Pautas Técnicas Críticas para el Desarrollo

### 2.1. Síntesis de Sonido en el Cliente (Web Audio API)
*   **Directriz de Rendimiento:** El proyecto no carga archivos `.mp3` ni `.wav` por red para evitar congestión y latencia de carga.
*   **Implementación:** Cualquier sonido adicional debe ser sintetizado en tiempo real usando osciladores (`OscillatorNode`) y envolventes de volumen de la Web Audio API del navegador, tal y como se hace en `frontend/src/lib/sound.svelte.ts`.

### 2.2. Validaciones en Cifras y Letras
*   **Cifras (Matemáticas):**
    *   Las operaciones válidas son sumas, restas, multiplicaciones y divisiones.
    *   **Límites:** El resultado de cada operación intermedia debe ser entero positivo ($> 0$) y las divisiones deben ser exactas sin decimales.
    *   El resolvedor de cifras (`backend/solver.go`) es un algoritmo de backtracking DFS que **debe ejecutarse con un límite de contexto de 1 segundo** y un máximo estricto de **1.000.000 de pasos** para evitar la sobrecarga de CPU en el servidor.
*   **Letras (Vocabulario):**
    *   Las palabras propuestas deben ser constructibles con las letras de la ronda y existir en el diccionario en memoria (`backend/dictionary.go`).
    *   El preprocesamiento del diccionario indexa las palabras por longitud en tiempo $O(1)$ para acelerar la búsqueda de las mejores soluciones sin impactar la latencia.

---

## 3. Despliegue y Build

### 3.1. Contenedores (Docker)
El proyecto incluye un `Dockerfile` multistage para producción:
*   **Stage 1:** Compila el frontend con la imagen oficial `node:22-alpine`.
*   **Stage 2:** Compila el backend Go (`golang:1.26-alpine`), embebiendo el build del frontend.
*   **Stage 3:** Imagen final mínima basada en `scratch` con el binario Go y los assets estáticos.

Para levantar localmente:
```bash
docker compose up --build
```

### 3.2. Diccionario Embebido
A partir de la versión actual, el diccionario (`assets/diccionario.txt`) se embebe en el binario Go mediante `//go:embed` en `backend/dictionary.go`. Esto elimina dependencias de rutas de archivos en tiempo de ejecución.

### 3.3. Archivos Estáticos
El servidor resuelve la carpeta `frontend/build` de forma robusta:
1.  Primero consulta la variable de entorno `STATIC_DIR`.
2.  Luego prueba rutas relativas al ejecutable y al directorio de trabajo.
3.  El fallback final es `./frontend/build`.

---

## 4. Tests y Calidad

### 4.1. Backend (Go)
El backend dispone de tests unitarios en:
*   `backend/solver_test.go`: Verifica la búsqueda exacta, límites de contexto/pasos y aproximaciones.
*   `backend/dictionary_test.go`: Valida la aleatoriedad de desempate y el límite máximo de 5 palabras en `GetBestWords`.
*   `backend/room_test.go`: Prueba transiciones de estado, envío de respuestas, puntuaciones de cifras y letras, y temporizadores del lobby.

Ejecutar tests:
```bash
cd backend && go test -v ./...
```

### 4.2. Linter y Vet
Ejecutar `go vet ./...` antes de cualquier commit para detectar errores comunes.

---

## 5. Documentación Técnica de Referencia
Para comprender en profundidad los detalles de funcionamiento, consulta la documentación interna del proyecto:
1.  [Architecture.md](guidelines/Architecture.md): Contiene los diagramas del Actor Model de Go, la arquitectura de goroutines de WebSockets y los límites computacionales de los resolvedores.
2.  [Design.md](guidelines/Design.md): Detalla las variables CSS del Fluent Mode, el ciclo de vida reactivo de las runes de Svelte 5 y las frecuencias exactas de la síntesis de audio.
3.  [GDD.md](guidelines/GDD.md): Expone el flujo completo del concurso, los tiempos de transición, y el reglamento de asignación de puntuación.