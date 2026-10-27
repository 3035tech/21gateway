-- API requests without immediate dispatch still consume the per-account request budget.
ALTER TABLE send_reservations DROP CONSTRAINT send_reservations_messages_check;
ALTER TABLE send_reservations ADD CONSTRAINT send_reservations_messages_check CHECK (messages BETWEEN 0 AND 200);
