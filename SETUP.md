# Setup Local — Pulse

**Guía paso a paso para levantar todo en tu máquina.**

---

## Requisitos previos

- **PostgreSQL** 14+ — tu base de datos principal
- **Redis** — para caché y sesiones
- **Node.js** 18+ — para frontend
- **Go** 1.21+ — para backend (solo si no usas Nix)
- **Nix** (opcional) — si quieres que Nix maneje todas las dependencias

### Opción A: Con Nix (recomendado)

Si tienes Nix instalado:

```bash
make services       # Inicia PostgreSQL + Redis en background
```

### Opción B: Sin Nix (manual)

Asegúrate de que tengas corriendo:

```bash
# Terminal 1: PostgreSQL
postgres -D /usr/local/var/postgres

# Terminal 2: Redis
redis-server
```

---

## Paso 1: Configurar variables de entorno

En `/stone/.env` (crea el archivo si no existe):

```env
# Database & Cache
DATABASE_URL=postgres://pulse@127.0.0.1:5433/pulse_dev?sslmode=disable
REDIS_URL=redis://127.0.0.1:6379/0

# Authentication
JWT_SECRET=your-secret-key-change-this-in-production-at-least-32-chars

# API
CORS_ORIGINS=http://localhost:5173,http://127.0.0.1:5173,http://localhost:5174,http://127.0.0.1:5174
WS_ORIGINS=http://localhost:5173,http://127.0.0.1:5173,http://localhost:5174,http://127.0.0.1:5174

# Storage
STORAGE_PATH=./uploads
STORAGE_BASE_URL=/uploads

# Ollama (opcional)
# OLLAMA_BASE_URL=http://localhost:11434
# OLLAMA_MODEL=qwen3-embedding
```

**Nota:** si usas `make services`, la base local `pulse_dev` se crea automáticamente en el PostgreSQL administrado por Nix, en el puerto `5433`.

Si usas PostgreSQL manual en otro puerto, crea la base y ajusta `DATABASE_URL`:

```bash
createdb pulse_dev
```

---

## Paso 2: Instalar dependencias frontend

```bash
# Tipos compartidos
cd drift/ts && npm install

# PWA principal
cd ../../ember && npm install

# Landing page
cd ../grove && npm install
```

---

## Paso 3: Ejecutar migraciones

Las migraciones crean todas las tablas en PostgreSQL.

```bash
make migrate
```

Verás output como:
```
Running migrations...
Migration 001_initial_schema.sql completed
Migration 002_reactions.sql completed
...
✓ All migrations completed
```

---

## Paso 4: Poblar base de datos con datos de prueba

El fixture incluye **10 usuarios** con datos de ejemplo: posts, tags, reacciones, etc.

```bash
make seed
```

**Usuarios de prueba creados:**

| Handle | Email | Password | Bio |
|--------|-------|----------|-----|
| `lunanova` | luna@example.com | `pulse-demo-2024` | Night photographer. Chasing city lights. |
| `marcelo.wav` | marcelo@example.com | `pulse-demo-2024` | Lofi producer & visual storyteller. |
| `iris.analog` | iris@example.com | `pulse-demo-2024` | Film photography only. 35mm dreams. |
| `kai` | kai@example.com | `pulse-demo-2024` | Street photographer. |
| `solene` | solene@example.com | `pulse-demo-2024` | Color theory obsessed. |
| `driftwood` | driftwood@example.com | `pulse-demo-2024` | Nature videographer. |
| `noor` | noor@example.com | `pulse-demo-2024` | Architecture and geometry. |
| `pixel.witch` | pixelwitch@example.com | `pulse-demo-2024` | Digital artist & glitch aesthetic. |
| `ravi.lens` | ravi@example.com | `pulse-demo-2024` | Documentary photographer. |
| `ama.sky` | ama@example.com | `pulse-demo-2024` | Fashion meets street culture. |

**Para resetear (limpiar todo antes de insertar):**

```bash
make seed-reset
```

---

## Paso 5: Iniciar los servicios

### Opción A: Todo en una terminal (recomendado)

```bash
make local
```

Esto abre un TUI (mprocs) que ejecuta:
- Backend (`:8080`)
- Frontend PWA (`:5173`)
- Landing page (`:5174`)

Presiona `q` para salir, `h` para ver controles.

### Opción B: Terminales separadas

**Terminal 1 — Backend:**
```bash
make dev-stone
```
Backend correrá en `http://localhost:8080`

**Terminal 2 — Frontend PWA:**
```bash
make dev-ember
```
App correrá en `http://localhost:5173`

**Terminal 3 — Landing page:**
```bash
make dev-grove
```
Landing page en `http://localhost:5174`

---

## Acceso a la app

1. **Landing page:** http://localhost:5174
2. **App principal (PWA):** http://localhost:5173

### Iniciar sesión

- **Handle:** `lunanova` (cualquiera de la tabla arriba)
- **Password:** `pulse-demo-2024`

O regístrate con un nuevo usuario.

---

## Demo completo

Para dejar el entorno en estado conocido y abrir el demo en un solo comando:

```bash
make demo
```

Esto inicia PostgreSQL + Redis, aplica migraciones, resetea e inserta el fixture demo, y abre el TUI con backend + frontend + landing.

---

## Desarrollar

### Backend (Go)

- Ubicación: `./stone`
- Hot reload: Cambios en `.go` se detectan automáticamente con `make dev-stone`
- Tests: `cd stone && go test ./...`

### Frontend (React)

- Ubicación: `./ember`
- Hot reload: Vite detecta cambios en `.tsx` / `.ts` / `.css`
- Build: `cd ember && npm run build`

### Landing (React)

- Ubicación: `./grove`
- Hot reload: Igual que Ember
- Build: `cd grove && npm run build`

---

## Troubleshooting

### ❌ "Connection refused" en backend
- ¿Está corriendo PostgreSQL? → `make services` (o inicia manualmente)
- ¿Está corriendo Redis? → `make services` (o inicia `redis-server`)
- ¿Existe la base de datos? → `createdb pulse_dev`

### ❌ "DATABASE_URL not found"
- Asegúrate de que `stone/.env` existe y está configurado
- `make dev-stone` leerá automáticamente las variables

### ❌ "Failed to embed content" (logs de Ollama)
- Ollama es opcional. El backend usa un embedder local por defecto.
- Si quieres embeddings semánticos, instala [Ollama](https://ollama.ai) y corre:
  ```bash
  ollama run qwen3-embedding
  ```

### ❌ Elementos no se ven en la app
- El fixture no incluye archivos de media (imágenes/videos reales)
- Los posts se crean pero apuntan a URLs que no existen
- Para desarrollo local, puedes subir imágenes manualmente desde la app

---

## Archivo de fixture

**Ubicación:** `stone/internal/db/seed/demo/fixture.json`

Contiene:
- 10 usuarios
- ~50+ posts (imágenes, videos, texto)
- Reacciones semánticas
- Follows/followers
- Rooms (mood clusters)
- Paths (curated journeys)

Puedes editar este JSON para customizar los datos de prueba. Para aplicar cambios:

```bash
make seed-reset  # Limpia y reinserta
```

---

## Stack técnico

| Componente | Tech | Port |
|-----------|------|------|
| **Backend** | Go + Gin + GORM + PostgreSQL | :8080 |
| **Frontend** | React + Vite + TypeScript + Tailwind v4 | :5173 |
| **Landing** | React + Vite + TypeScript + Tailwind v4 | :5174 |
| **Database** | PostgreSQL 14+ | :5432 |
| **Cache** | Redis | :6379 |
| **Types** | TypeScript (shared) | — |

---

## Próximos pasos

Una vez que todo está corriendo:

1. **Explora la app:** Crea posts, reacciona, sigue usuarios
2. **Mira el WebSocket:** Abre dos navegadores y ve las rooms en tiempo real
3. **Revisa los logs:** `make dev-stone` y `make dev-ember` muestran requests/errors
4. **Lee los docs:** `docs/ideas.md` explica la visión del proyecto

---

## ¿Necesitas ayuda?

- **Preguntas técnicas:** Revisa `CLAUDE.md` para arquitectura
- **Errores:** Mira los logs en la terminal donde corre `make local`
- **Variables de entorno:** Todas están en `stone/internal/config/config.go`

¡Listo! Tu ambiente local está funcionando. 🚀
