require('dotenv').config();
const express = require('express');
const cors = require('cors');
const { Pool } = require('pg');
const swaggerJsdoc = require('swagger-jsdoc');
const swaggerUi = require('swagger-ui-express');

const app = express();
app.use(cors());
app.use(express.json());

// conexion a la base de datos Neon (PostgreSQL en la nube)
const pool = new Pool({
    connectionString: process.env.DATABASE_URL,
    ssl: { rejectUnauthorized: false }, // Neon exige SSL
});

// Swagger necesita la URL absoluta real del servidor (una relativa como '/'
// hace que el boton "Try it out" arme mal la peticion y el navegador la
// bloquee por CORS). Render define sola la variable RENDER_EXTERNAL_URL con
// la URL publica del servicio, asi que la usamos si existe.
const servers = [];
if (process.env.RENDER_EXTERNAL_URL) {
    servers.push({ url: process.env.RENDER_EXTERNAL_URL, description: 'Render (produccion)' });
}
servers.push({ url: `http://localhost:${process.env.PORT || 3000}`, description: 'Local' });

// ---------- Configuracion de Swagger ----------
const swaggerSpec = swaggerJsdoc({
    definition: {
        openapi: '3.0.0',
        info: {
            title: 'API - Microservicio de Curiosidades',
            version: '1.0.0',
            description:
                'Microservicio en Node.js/Express que consulta curiosidades de mascotas ' +
                'guardadas en una base de datos PostgreSQL en Neon. Lo consume la vista ' +
                '"ver_curiosidades" de la app Django de mascotas virtuales.',
        },
        servers,
        components: {
            schemas: {
                Curiosidad: {
                    type: 'object',
                    properties: {
                        id: { type: 'integer', example: 1 },
                        especie: { type: 'string', example: 'perro' },
                        texto: { type: 'string', example: 'Los perros pueden aprender mas de 100 palabras.' },
                    },
                },
                Error: {
                    type: 'object',
                    properties: {
                        error: { type: 'string', example: 'Error consultando la base de datos' },
                    },
                },
                Monedas: {
                    type: 'object',
                    properties: {
                        mascota_id: { type: 'integer', example: 1 },
                        cantidad: { type: 'integer', example: 30 },
                    },
                },
                Mascota: {
                    type: 'object',
                    properties: {
                        id: { type: 'integer', example: 1 },
                        nombre: { type: 'string', example: 'Firu' },
                        especie: { type: 'string', example: 'perro' },
                        hambre: { type: 'integer', example: 35 },
                        felicidad: { type: 'integer', example: 68 },
                        actualizado_en: { type: 'string', format: 'date-time' },
                    },
                },
                MascotaEntrada: {
                    type: 'object',
                    required: ['id', 'nombre', 'especie'],
                    properties: {
                        id: { type: 'integer', example: 1 },
                        nombre: { type: 'string', example: 'Firu' },
                        especie: { type: 'string', example: 'perro' },
                        hambre: { type: 'integer', example: 35 },
                        felicidad: { type: 'integer', example: 68 },
                    },
                },
            },
        },
    },
    apis: ['./index.js'], // aqui mismo estan los comentarios @swagger de cada ruta
});

app.use('/api-docs', swaggerUi.serve, swaggerUi.setup(swaggerSpec));

// la raiz redirige a la documentacion swagger, para que sea lo primero que
// se ve al entrar a la URL del servicio (ej: https://tu-servicio.onrender.com/)
app.get('/', (req, res) => {
    res.redirect('/api-docs');
});

/**
 * @swagger
 * /health:
 *   get:
 *     summary: Health check
 *     description: Confirma que el microservicio esta corriendo (util para verificar el deploy en Render).
 *     responses:
 *       200:
 *         description: El servicio esta arriba
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 status:
 *                   type: string
 *                   example: ok
 *                 mensaje:
 *                   type: string
 *                   example: microservicio-curiosidades funcionando
 */
app.get('/health', (req, res) => {
    res.json({ status: 'ok', mensaje: 'microservicio-curiosidades funcionando' });
});

/**
 * @swagger
 * /api/curiosidades/{especie}:
 *   get:
 *     summary: Curiosidades filtradas por especie
 *     description: Ruta dinamica que recibe la especie como parametro y devuelve solo esas curiosidades.
 *     parameters:
 *       - in: path
 *         name: especie
 *         required: true
 *         schema:
 *           type: string
 *           example: perro
 *         description: Especie a filtrar (perro, gato, dragon, robot)
 *     responses:
 *       200:
 *         description: Lista de curiosidades de esa especie
 *         content:
 *           application/json:
 *             schema:
 *               type: array
 *               items:
 *                 $ref: '#/components/schemas/Curiosidad'
 *       500:
 *         description: Error consultando la base de datos
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Error'
 */
app.get('/api/curiosidades/:especie', async (req, res) => {
    const { especie } = req.params;
    try {
        const resultado = await pool.query(
            'SELECT id, especie, texto FROM curiosidades WHERE especie = $1',
            [especie]
        );
        res.json(resultado.rows);
    } catch (error) {
        console.error(error);
        res.status(500).json({ error: 'Error consultando la base de datos' });
    }
});

/**
 * @swagger
 * /api/curiosidades:
 *   get:
 *     summary: Todas las curiosidades
 *     description: Devuelve todas las curiosidades guardadas, sin filtrar por especie.
 *     responses:
 *       200:
 *         description: Lista completa de curiosidades
 *         content:
 *           application/json:
 *             schema:
 *               type: array
 *               items:
 *                 $ref: '#/components/schemas/Curiosidad'
 *       500:
 *         description: Error consultando la base de datos
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Error'
 */
app.get('/api/curiosidades', async (req, res) => {
    try {
        const resultado = await pool.query('SELECT id, especie, texto FROM curiosidades');
        res.json(resultado.rows);
    } catch (error) {
        console.error(error);
        res.status(500).json({ error: 'Error consultando la base de datos' });
    }
});

/**
 * @swagger
 * /api/monedas/{mascota_id}:
 *   get:
 *     summary: Saldo de monedas de una mascota
 *     description: Devuelve cuantas monedas tiene guardada una mascota. Si nunca ha jugado, devuelve 0.
 *     parameters:
 *       - in: path
 *         name: mascota_id
 *         required: true
 *         schema:
 *           type: integer
 *         description: Id de la mascota (el mismo id que usa Django)
 *     responses:
 *       200:
 *         description: Saldo actual
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 mascota_id: { type: integer, example: 1 }
 *                 cantidad: { type: integer, example: 30 }
 */
app.get('/api/monedas/:mascota_id', async (req, res) => {
    const mascotaId = req.params.mascota_id;
    try {
        const resultado = await pool.query(
            'SELECT cantidad FROM monedas WHERE mascota_id = $1',
            [mascotaId]
        );
        const cantidad = resultado.rows.length ? resultado.rows[0].cantidad : 0;
        res.json({ mascota_id: Number(mascotaId), cantidad });
    } catch (error) {
        console.error(error);
        res.status(500).json({ error: 'Error consultando la base de datos' });
    }
});

/**
 * @swagger
 * /api/monedas/{mascota_id}/ganar:
 *   post:
 *     summary: Sumar monedas (ganadas en el minijuego)
 *     description: Suma una cantidad de monedas al saldo de la mascota. Se usa cuando se gana el minijuego.
 *     parameters:
 *       - in: path
 *         name: mascota_id
 *         required: true
 *         schema:
 *           type: integer
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             properties:
 *               cantidad: { type: integer, example: 10 }
 *     responses:
 *       200:
 *         description: Nuevo saldo despues de sumar
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 mascota_id: { type: integer, example: 1 }
 *                 cantidad: { type: integer, example: 40 }
 */
app.post('/api/monedas/:mascota_id/ganar', async (req, res) => {
    const mascotaId = req.params.mascota_id;
    const cantidad = Number(req.body.cantidad) || 0;
    try {
        await pool.query(
            'INSERT INTO monedas (mascota_id, cantidad) VALUES ($1, 0) ON CONFLICT (mascota_id) DO NOTHING',
            [mascotaId]
        );
        const actual = await pool.query('SELECT cantidad FROM monedas WHERE mascota_id = $1', [mascotaId]);
        const nuevoSaldo = actual.rows[0].cantidad + cantidad;

        const resultado = await pool.query(
            'UPDATE monedas SET cantidad = $2 WHERE mascota_id = $1 RETURNING cantidad',
            [mascotaId, nuevoSaldo]
        );
        res.json({ mascota_id: Number(mascotaId), cantidad: resultado.rows[0].cantidad });
    } catch (error) {
        console.error(error);
        res.status(500).json({ error: 'Error actualizando la base de datos' });
    }
});

/**
 * @swagger
 * /api/monedas/{mascota_id}/gastar:
 *   post:
 *     summary: Gastar monedas (comprar en la tienda)
 *     description: Resta monedas del saldo de la mascota, solo si tiene suficientes.
 *     parameters:
 *       - in: path
 *         name: mascota_id
 *         required: true
 *         schema:
 *           type: integer
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             properties:
 *               cantidad: { type: integer, example: 10 }
 *     responses:
 *       200:
 *         description: Resultado del intento de gasto
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 ok: { type: boolean, example: true }
 *                 mascota_id: { type: integer, example: 1 }
 *                 cantidad: { type: integer, example: 20 }
 */
app.post('/api/monedas/:mascota_id/gastar', async (req, res) => {
    const mascotaId = req.params.mascota_id;
    const cantidad = Number(req.body.cantidad) || 0;
    try {
        await pool.query(
            'INSERT INTO monedas (mascota_id, cantidad) VALUES ($1, 0) ON CONFLICT (mascota_id) DO NOTHING',
            [mascotaId]
        );
        const actual = await pool.query('SELECT cantidad FROM monedas WHERE mascota_id = $1', [mascotaId]);
        const saldo = actual.rows[0].cantidad;

        if (saldo < cantidad) {
            return res.json({ ok: false, mascota_id: Number(mascotaId), cantidad: saldo });
        }

        const nuevoSaldo = saldo - cantidad;
        const resultado = await pool.query(
            'UPDATE monedas SET cantidad = $2 WHERE mascota_id = $1 RETURNING cantidad',
            [mascotaId, nuevoSaldo]
        );
        res.json({ ok: true, mascota_id: Number(mascotaId), cantidad: resultado.rows[0].cantidad });
    } catch (error) {
        console.error(error);
        res.status(500).json({ error: 'Error actualizando la base de datos' });
    }
});

/**
 * @swagger
 * /api/mascotas:
 *   get:
 *     summary: Listar todas las mascotas
 *     description: Devuelve el espejo en Neon de todas las mascotas administradas por Django. Endpoint publico pensado para que una IA externa u otro consumidor pueda leer el estado real y actual de las mascotas.
 *     responses:
 *       200:
 *         description: Lista de mascotas
 *         content:
 *           application/json:
 *             schema:
 *               type: array
 *               items:
 *                 $ref: '#/components/schemas/Mascota'
 *   post:
 *     summary: Crear o sincronizar una mascota (upsert)
 *     description: Inserta una mascota con el id que le asigno Django, o actualiza sus datos si ese id ya existe (upsert por id). Lo usa Django cada vez que se crea, edita, alimenta o hace jugar a una mascota, para mantener Neon sincronizado.
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/MascotaEntrada'
 *     responses:
 *       200:
 *         description: Mascota creada o actualizada
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Mascota'
 *       400:
 *         description: Faltan campos obligatorios (id, nombre, especie)
 */
app.get('/api/mascotas', async (req, res) => {
    try {
        const resultado = await pool.query(
            'SELECT id, nombre, especie, hambre, felicidad, actualizado_en FROM mascotas ORDER BY id'
        );
        res.json(resultado.rows);
    } catch (error) {
        console.error(error);
        res.status(500).json({ error: 'Error consultando la base de datos' });
    }
});

app.post('/api/mascotas', async (req, res) => {
    const { id, nombre, especie, hambre, felicidad } = req.body;
    if (!id || !nombre || !especie) {
        return res.status(400).json({ error: 'id, nombre y especie son obligatorios' });
    }
    try {
        const resultado = await pool.query(
            `INSERT INTO mascotas (id, nombre, especie, hambre, felicidad, actualizado_en)
             VALUES ($1, $2, $3, COALESCE($4, 50), COALESCE($5, 50), now())
             ON CONFLICT (id) DO UPDATE
                SET nombre = EXCLUDED.nombre,
                    especie = EXCLUDED.especie,
                    hambre = EXCLUDED.hambre,
                    felicidad = EXCLUDED.felicidad,
                    actualizado_en = now()
             RETURNING id, nombre, especie, hambre, felicidad, actualizado_en`,
            [id, nombre, especie, hambre, felicidad]
        );
        res.json(resultado.rows[0]);
    } catch (error) {
        console.error(error);
        res.status(500).json({ error: 'Error guardando la mascota' });
    }
});

/**
 * @swagger
 * /api/mascotas/{id}:
 *   get:
 *     summary: Detalle de una mascota
 *     description: Devuelve el estado real y actual (hambre, felicidad) de una mascota puntual, guardado en Neon.
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema: { type: integer }
 *     responses:
 *       200:
 *         description: Datos de la mascota
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Mascota'
 *       404:
 *         description: No existe una mascota con ese id
 *   put:
 *     summary: Actualizar una mascota existente
 *     description: Actualiza nombre, especie, hambre y/o felicidad de una mascota ya existente. Lo usa Django al editar, alimentar o hacer jugar a una mascota.
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema: { type: integer }
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/MascotaEntrada'
 *     responses:
 *       200:
 *         description: Mascota actualizada
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Mascota'
 *       404:
 *         description: No existe una mascota con ese id
 *   delete:
 *     summary: Eliminar una mascota
 *     description: Borra la mascota de Neon (y su saldo de monedas, si tenia). Lo usa Django cuando se elimina una mascota desde la app.
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema: { type: integer }
 *     responses:
 *       200:
 *         description: Confirmacion del borrado
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 ok: { type: boolean, example: true }
 */
app.get('/api/mascotas/:id', async (req, res) => {
    try {
        const resultado = await pool.query(
            'SELECT id, nombre, especie, hambre, felicidad, actualizado_en FROM mascotas WHERE id = $1',
            [req.params.id]
        );
        if (!resultado.rows.length) {
            return res.status(404).json({ error: 'No existe una mascota con ese id' });
        }
        res.json(resultado.rows[0]);
    } catch (error) {
        console.error(error);
        res.status(500).json({ error: 'Error consultando la base de datos' });
    }
});

app.put('/api/mascotas/:id', async (req, res) => {
    const { nombre, especie, hambre, felicidad } = req.body;
    try {
        const resultado = await pool.query(
            `UPDATE mascotas
                SET nombre = COALESCE($2, nombre),
                    especie = COALESCE($3, especie),
                    hambre = COALESCE($4, hambre),
                    felicidad = COALESCE($5, felicidad),
                    actualizado_en = now()
             WHERE id = $1
             RETURNING id, nombre, especie, hambre, felicidad, actualizado_en`,
            [req.params.id, nombre, especie, hambre, felicidad]
        );
        if (!resultado.rows.length) {
            return res.status(404).json({ error: 'No existe una mascota con ese id' });
        }
        res.json(resultado.rows[0]);
    } catch (error) {
        console.error(error);
        res.status(500).json({ error: 'Error actualizando la mascota' });
    }
});

app.delete('/api/mascotas/:id', async (req, res) => {
    try {
        await pool.query('DELETE FROM monedas WHERE mascota_id = $1', [req.params.id]);
        await pool.query('DELETE FROM mascotas WHERE id = $1', [req.params.id]);
        res.json({ ok: true });
    } catch (error) {
        console.error(error);
        res.status(500).json({ error: 'Error eliminando la mascota' });
    }
});

const PUERTO = process.env.PORT || 3000;
app.listen(PUERTO, () => {
    console.log(`microservicio-curiosidades escuchando en el puerto ${PUERTO}`);
    console.log(`Documentacion Swagger en http://localhost:${PUERTO}/api-docs`);
});
