-- Revert customer role back to developer
UPDATE users SET role = 'developer' WHERE role = 'customer';
