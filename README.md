# Cifras y Letras

Juego multiplayer en tiempo real inspirado en el concurso de televisión. Construido con Go y WebSockets.

## Reglas del Juego

### Cifras
- Se presenta un número objetivo y 6 cifras (4 pequeñas + 2 grandes)
- Combina los números usando sumas, restas, multiplicaciones y divisiones
- El objetivo es alcanzar el número exacto (10 puntos) o quedar lo más cerca posible (7 puntos si eres el más cercano)

### Letras
- Se eligen entre 3-5 vocales y el resto consonantes
- Forma la palabra más larga posible (mínimo 5 letras)
- Cada letra = 1 punto

## Tecnologías

- **Backend:** Go con gorilla/websocket
- **Frontend:** HTML, CSS, JavaScript vanilla
- **Comunicación:** WebSocket en tiempo real
- **Diccionario:** ~647k palabras en español

## Instalación

```bash
git clone https://github.com/sheuronazxe/cifras.git
cd cifras
go mod download
go run main.go
```

Abre `http://localhost:8080` en tu navegador.

## Ejecución en Producción

```bash
PORT=80 go run main.go
```

## Características

- Partidas multijugador simultáneas (hasta 20 jugadores)
- Sistema de ranking con puntuación acumulada
- Reconexión automática ante desconexiones
- Efectos de sonido
- Temporizador en tiempo real sincronizado con el servidor
- Solver automático que encuentra soluciones exactas
- Selector de vocales en modo Letras

## Estructura del Proyecto

```
cifras/
├── main.go          # Servidor Go con WebSocket
├── public/
│   ├── index.html   # Interfaz del juego
│   ├── app.js       # Lógica del cliente
│   └── style.css    # Estilos
└── assets/
    └── diccionario.txt  # Diccionario español
```

## Licencia

MIT