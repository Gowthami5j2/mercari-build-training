CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE
);

CREATE TABLE IF NOT EXISTS items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE,
    category TEXT NOT NULL,
    image_name TEXT NOT NULL
    category_id TEXT NOT NULL,
    image_name TEXT NOT NULL,
    FOREIGN KEY (category_id) REFERENCES categories(id)
);