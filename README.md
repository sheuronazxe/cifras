# Cifras y Letras

Juego multiplayer en tiempo real inspirado en el concurso de televisión. Construido con Go y Svelte 5.

## Reglas del Juego

### Cifras
- Se presenta un número objetivo y 6 cifras (4 pequeñas + 2 grandes)
- Combina los números usando sumas, restas, multiplicaciones y divisiones
- El objetivo es alcanzar el número exacto (10 puntos) o quedar lo más cerca posible (7 puntos si eres el más cercano)

### Letras
- Se eligen entre 3-6 vocales y el resto consonantes
- Forma la palabra más larga posible (mínimo 5 letras)
- Cada letra = 1 punto

## Tecnologías

- **Backend:** Go 1.26 con github.com/coder/websocket
- **Frontend:** Svelte 5 (Runes) con Fluent Design Oscuro
- **Build:** Bun para desarrollo, Node.js para producción (Docker)
- **Contenedor:** Dockerfile multistage (Go + Node → scratch)
- **Comunicación:** WebSocket en tiempo real
- **Diccionario:** ~647k palabras en español (embebido en el binario)

## Desarrollo Local

```bash
# Arrancar frontend (desde /frontend)
bun run dev
# o con npm:
npm run dev
```

Accede a `http://localhost:5173` para el desarrollo.

## Producción con Docker

```bash
# Construir y levantar
docker compose up --build

# O manualmente
docker build -t cifras .
docker run -p 8080:8080 cifras
```

El servidor estará disponible en `http://localhost:8080`.

### Variables de entorno

| Variable | Default | Descripción |
|----------|---------|-------------|
| `PORT` | `8080` | Puerto del servidor |
| `STATIC_DIR` | (auto) | Ruta a los archivos estáticos del frontend |

## Características

- Partidas multijugador simultáneas (hasta 20 jugadores)
- Sistema de ranking con puntuación acumulada
- Efectos de sonido sintetizados con Web Audio API
- Temporizador en tiempo real sincronizado con el servidor
- Solver automático que encuentra soluciones exactas
- Selector de vocales en modo Letras
- UI responsiva con Fluent Design Oscuro

## Estructura del Proyecto

```
cifras/
├── Dockerfile           # Build multistage
├── docker-compose.yml   # Orquestación
├── backend/
│   ├── main.go          # Servidor HTTP y arranque
│   ├── room.go          # GameRoom (Actor Model)
│   ├── server.go        # WebSocket handling
│   ├── solver.go        # Resolvedor de cifras
│   ├── dictionary.go    # Validación de palabras (embebido)
│   ├── models.go        # GameAction
│   ├── internal/
│   │   └── types/
│   │       └── types.go # Tipos compartidos (Player, SyncData, etc.)
│   └── cmd/
│       └── debug/
│           └── main.go  # Herramienta de depuración de UI
└── frontend/
    ├── src/
    │   ├── routes/      # Páginas SvelteKit
    │   └── lib/         # Componentes y stores
    ├── static/          # Assets estáticos
    ├── package.json     # Dependencias
    └── bun.lock         # Lockfile (Bun para desarrollo, npm para producción)
```

## Licencia

MIT