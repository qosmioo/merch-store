CREATE TABLE IF NOT EXISTS inventory (
    employee_id INT NOT NULL,
    type VARCHAR(255) NOT NULL,
    quantity INT NOT NULL,
    PRIMARY KEY (employee_id, type),
    FOREIGN KEY (employee_id) REFERENCES employees(id)
); 