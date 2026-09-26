-- Ejecutar este script en la base de datos de Neon (por ejemplo desde el SQL editor de Neon
-- o con psql usando el connection string que da Neon).

CREATE TABLE IF NOT EXISTS curiosidades (
    id SERIAL PRIMARY KEY,
    especie VARCHAR(20) NOT NULL,
    texto TEXT NOT NULL
);

INSERT INTO curiosidades (especie, texto) VALUES
('perro', 'Los perros pueden aprender mas de 100 palabras y gestos.'),
('perro', 'La nariz de un perro es unica, como una huella digital.'),
('perro', 'Los perros sudan solo por las patas.'),
('gato', 'Los gatos pasan cerca del 70% de su vida durmiendo.'),
('gato', 'Un gato puede saltar hasta 6 veces su propia longitud.'),
('gato', 'Los gatos no pueden saborear lo dulce.'),
('dragon', 'En esta app, los dragones son la especie mas dificil de mantener feliz.'),
('dragon', 'A los dragones virtuales les encanta el pastel.'),
('robot', 'Las mascotas robot no sienten hambre real, pero igual hay que "alimentarlas" por diversion.'),
('robot', 'Un robot mascota nunca duerme, pero si se le puede bajar el nivel de energia (felicidad).');

-- Monedas del juego: cada mascota (identificada por su id en la base de datos
-- de Django/SQLite) tiene un saldo guardado aca en Neon. Se ganan jugando el
-- minijuego y se gastan en la tienda.
CREATE TABLE IF NOT EXISTS monedas (
    mascota_id INTEGER PRIMARY KEY,
    cantidad INTEGER NOT NULL DEFAULT 0
);
