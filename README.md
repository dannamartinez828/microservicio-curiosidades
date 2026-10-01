# Microservicios de Mascotas Virtuales
12 CRUD independientes: 4 lenguajes (Node.js, Python, Java, PHP) x 3 operaciones (insertar, actualizar, eliminar).
Además: consulta-python y consulta-node para resiliencia.
Tabla PostgreSQL/Neon: `mascotas_mascota`.
Rutas CRUD: POST `/api/mascotas`, PUT `/api/mascotas/:id`, DELETE `/api/mascotas/:id`, GET `/health`.
Cuerpo POST/PUT: `{"nombre":"Luna","especie":"gato","hambre":50,"felicidad":50}`.
