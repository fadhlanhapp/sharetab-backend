-- migrations/schema.sql

-- Drop tables if they exist (for clean setup)
DROP TABLE IF EXISTS expenses_items;
DROP TABLE IF EXISTS expense_participants;
DROP TABLE IF EXISTS expenses;
DROP TABLE IF EXISTS trip_participants;
DROP TABLE IF EXISTS trips;

-- Create trips table
CREATE TABLE trips (
    id VARCHAR(36) PRIMARY KEY,
    code VARCHAR(10) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    creation_time BIGINT NOT NULL
);

-- Create trip_participants table
CREATE TABLE trip_participants (
    trip_id VARCHAR(36) REFERENCES trips(id) ON DELETE CASCADE,
    participant VARCHAR(255) NOT NULL,
    PRIMARY KEY (trip_id, participant)
);

-- Create expenses table
CREATE TABLE expenses (
    id VARCHAR(36) PRIMARY KEY,
    trip_id VARCHAR(36) REFERENCES trips(id) ON DELETE CASCADE,
    description VARCHAR(255) NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    subtotal DECIMAL(10, 2) NOT NULL,
    tax DECIMAL(10, 2) NOT NULL,
    service_charge DECIMAL(10, 2) NOT NULL,
    total_discount DECIMAL(10, 2) NOT NULL,
    paid_by VARCHAR(255) NOT NULL,
    split_type VARCHAR(50) NOT NULL,
    creation_time BIGINT NOT NULL,
    receipt_image VARCHAR(255)
);

-- Create expense_participants table (for equal splits)
CREATE TABLE expense_participants (
    expense_id VARCHAR(36) REFERENCES expenses(id) ON DELETE CASCADE,
    participant VARCHAR(255) NOT NULL,
    PRIMARY KEY (expense_id, participant)
);

-- Create expenses_items table (for item-based splits)
CREATE TABLE expenses_items (
    id SERIAL PRIMARY KEY,
    expense_id VARCHAR(36) REFERENCES expenses(id) ON DELETE CASCADE,
    description VARCHAR(255) NOT NULL,
    unit_price DECIMAL(10, 2) NOT NULL,
    quantity INT NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    item_discount DECIMAL(10, 2) NOT NULL,
    paid_by VARCHAR(255) NOT NULL
);

-- Create item_consumers table
CREATE TABLE item_consumers (
    item_id INT REFERENCES expenses_items(id) ON DELETE CASCADE,
    consumer VARCHAR(255) NOT NULL,
    PRIMARY KEY (item_id, consumer)
);

-- Create payments table
CREATE TABLE payments (
    id SERIAL PRIMARY KEY,
    trip_id VARCHAR(36) NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    from_person VARCHAR(255) NOT NULL,
    to_person VARCHAR(255) NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    description TEXT,
    payment_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for faster queries
CREATE INDEX idx_trips_code ON trips(code);
CREATE INDEX idx_expenses_trip_id ON expenses(trip_id);
CREATE INDEX idx_expense_participants_expense_id ON expense_participants(expense_id);
CREATE INDEX idx_expenses_items_expense_id ON expenses_items(expense_id);
CREATE INDEX idx_item_consumers_item_id ON item_consumers(item_id);
CREATE INDEX idx_payments_trip_id ON payments(trip_id);
CREATE INDEX idx_payments_from_person ON payments(from_person);
CREATE INDEX idx_payments_to_person ON payments(to_person);