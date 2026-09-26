# microservicio-curiosidades

Microservicio en Node.js/Express que expone curiosidades de mascotas,
guardadas en una base de datos PostgreSQL alojada en Neon. Lo consume la
vista `ver_curiosidades` de la app Django de mascotas virtuales.

## Endpoints

- `GET /` -> health check.
- `GET /api/curiosidades` -> todas las curiosidades.
- `GET /api/curiosidades/:especie` -> curiosidades filtradas por especie
  (perro, gato, dragon, robot).
- `GET /api-docs` -> documentación interactiva (Swagger UI), donde se
  pueden probar los endpoints desde el navegador.

## 1. Crear la base de datos en Neon

1. Crear un proyecto en https://neon.tech (tiene plan gratuito).
2. Copiar el "Connection string" que da el dashboard.
3. Ejecutar el archivo `schema.sql` de esta carpeta en el SQL editor de
   Neon (o con `psql "connection_string" -f schema.sql`).

## 2. Correr el microservicio en local

```bash
npm install
cp .env.example .env
# pegar el connection string de Neon en DATABASE_URL dentro de .env
npm start
```

Probar en el navegador: http://localhost:3000/api/curiosidades/perro

También se puede abrir http://localhost:3000/api-docs para ver la
documentación Swagger y probar los endpoints ahí mismo, sin necesidad de
curl ni Postman.

## 3. Desplegar en Render

1. Subir esta carpeta a un repositorio de GitHub.
2. En Render: New -> Web Service -> conectar el repo.
3. Build command: `npm install`. Start command: `npm start`.
4. En Environment agregar la variable `DATABASE_URL` con el connection
   string de Neon.
5. Una vez desplegado, Render da una URL publica, por ejemplo:
   `https://microservicio-curiosidades.onrender.com`

## 4. Conectar con la app Django

En el proyecto Django, definir la variable de entorno `MICROSERVICIO_URL`
con la URL de Render (sin barra al final), por ejemplo:

```bash
export MICROSERVICIO_URL=https://microservicio-curiosidades.onrender.com
```

Si no se define, la app Django usa `http://localhost:3000` por defecto.
